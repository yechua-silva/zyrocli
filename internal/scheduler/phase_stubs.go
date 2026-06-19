package scheduler

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/secko/zyrocli/internal/db/helix"
	"github.com/secko/zyrocli/internal/handoff"
)

// PREF0Runner implements PhaseRunner for PRE-F0: Domain alignment.
// Opens OpenCode and runs zyro-pre-f0 skill for grill-me and domain-model.
type PREF0Runner struct{}

func (r *PREF0Runner) Run(ctx context.Context, cfg *Config) (*Result, error) {
	select {
	case <-ctx.Done():
		return &Result{Phase: PhasePREF0, Status: StatusFail, Summary: "tiempo agotado"}, ctx.Err()
	default:
	}

	payload, err := handoff.Parse("handoff.yaml")
	if err != nil {
		return &Result{Phase: PhasePREF0, Status: StatusFail, Summary: "No se pudo leer handoff.yaml"}, nil
	}

	projectDir := payload.Project.Name
	if projectDir == "" {
		projectDir = "."
	}

	// Escribir .zyro/task.yaml con contexto para PRE-F0
	taskDir := filepath.Join(projectDir, ".zyro")
	os.MkdirAll(taskDir, 0755)
	taskYAML := `phase: "PRE-F0"
agent: "zyro-pre-f0"
required_output:
  alignment: true
  domain_model: true
  handoff: true
`
	os.WriteFile(filepath.Join(taskDir, "task.yaml"), []byte(taskYAML), 0644)

	fmt.Printf("\n  ▶ Fase PRE-F0: Alineación de dominio\n")
	fmt.Printf("  El agente zyro-pre-f0 va a realizar grill-me y domain-model.\n")
	fmt.Printf("  Cuando termines de alinear, cerrá el editor para continuar.\n\n")

	if os.Getenv("ZYRO_TEST") == "" {
		if _, err := exec.LookPath("opencode"); err == nil {
			openCmd := exec.CommandContext(ctx, "opencode", projectDir)
			openCmd.Stdin = os.Stdin
			openCmd.Stdout = os.Stdout
			openCmd.Stderr = os.Stderr
			_ = openCmd.Run()
		} else {
			fmt.Println("  ⚠ opencode no encontrado. Abrí manualmente:", projectDir)
		}
	} else {
		fmt.Println("  [ZYRO_TEST] saltando apertura de OpenCode")
	}

	// Verificar post-condiciones: alignment.md debe existir
	fmt.Print("  Verificando alineación...")
	alignmentPath := filepath.Join(projectDir, "openspec", "alignment.md")
	if _, err := os.Stat(alignmentPath); os.IsNotExist(err) {
		fmt.Println()
		return &Result{Phase: PhasePREF0, Status: StatusFail,
			Summary: "No se encontró openspec/alignment.md. La alineación no se completó."}, nil
	}
	fmt.Println(" OK")

	return &Result{
		Phase:   PhasePREF0,
		Status:  StatusSuccess,
		Summary: "PRE-F0 completada: alignment.md generado, dominio alineado",
	}, nil
}

func (r *PREF0Runner) Name() Phase { return PhasePREF0 }

// F1Runner implements PhaseRunner for F1: Planificación con state-gating.
type F1Runner struct{}

func (r *F1Runner) Run(ctx context.Context, cfg *Config) (*Result, error) {
	select {
	case <-ctx.Done():
		return &Result{Phase: PhaseF1, Status: StatusFail, Summary: "timeout"}, ctx.Err()
	default:
	}

	// 1. Parse handoff
	payload, err := handoff.Parse("handoff.yaml")
	if err != nil {
		return &Result{Phase: PhaseF1, Status: StatusFail, Summary: fmt.Sprintf("No se pudo leer el archivo de configuración: %v", err)}, nil
	}

	projectDir := payload.Project.Name
	fmt.Printf("  Proyecto: %s\n", projectDir)

	// 2. Conectar a la base de datos
	helixClient, err := helix.NewClient(ctx)
	if err != nil {
		return &Result{Phase: PhaseF1, Status: StatusFail, Summary: fmt.Sprintf("No se pudo conectar a la base de datos: %v", err)}, nil
	}
	defer helixClient.Close()

	// 3. Buscar el proyecto en la base de datos
	projectNodes, err := helixClient.TextSearch(ctx, "Project", "name", payload.Project.Name, 1)
	if err != nil || len(projectNodes) == 0 {
		return &Result{Phase: PhaseF1, Status: StatusFail, Summary: "Proyecto no encontrado. Ejecutá 'zyrocli init' primero."}, nil
	}
	projectID := projectNodes[0].ID

	// 4. Verificar que Fase 0 esté completa
	fmt.Print("  Revisando información de Fase 0...")
	patterns, _ := helixClient.GetOutgoing(ctx, projectID, "HAS_PATTERN")
	libs, _ := helixClient.GetOutgoing(ctx, projectID, "USES_LIB")

	var incompleto []string
	if len(patterns) == 0 {
		incompleto = append(incompleto, "patrones de referencia")
	}
	if len(libs) == 0 {
		incompleto = append(incompleto, "librerías recomendadas")
	}

	if len(incompleto) > 0 {
		fmt.Println()
		return &Result{Phase: PhaseF1, Status: StatusFail, Summary: fmt.Sprintf("Falta información de Fase 0: %v. Ejecutá Fase 0 primero.", incompleto)}, nil
	}
	fmt.Println(" OK")

	// 5. Preparar contexto para el agente de especificación
	taskDir := filepath.Join(projectDir, ".zyro")
	os.MkdirAll(taskDir, 0755)

	patternNames := make([]string, len(patterns))
	for i, p := range patterns {
		patternNames[i] = fmt.Sprintf("%s", p.Properties["name"])
	}
	libNames := make([]string, len(libs))
	for i, l := range libs {
		libNames[i] = fmt.Sprintf("%s", l.Properties["name"])
	}

	taskYAML := fmt.Sprintf(`phase: "F1"
agent: "zyro-sdd-spec"
project_id: %d
patterns:
%s
libraries:
%s
required_output:
  spec:
    architecture: required
    modules: required
    dependencies: required
    testing_strategy: required
`, projectID,
		func() string { var s string; for _, n := range patternNames { s += fmt.Sprintf("  - \"%s\"\n", n) }; return s }(),
		func() string { var s string; for _, n := range libNames { s += fmt.Sprintf("  - \"%s\"\n", n) }; return s }())

	if err := os.WriteFile(filepath.Join(taskDir, "task.yaml"), []byte(taskYAML), 0644); err != nil {
		fmt.Printf("  ⚠ advertencia: no se pudo guardar contexto temporal: %v\n", err)
	}
	fmt.Printf("  Información de Fase 0 lista: %d patrones, %d librerías\n", len(patterns), len(libs))

	// 6. Abrir OpenCode para la especificación
	fmt.Printf("\n  Abriendo editor para definir la especificación técnica...\n")
	fmt.Printf("  El asistente va a diseñar la arquitectura del proyecto.\n")
	fmt.Printf("  Cuando termines de revisar, cerrá el editor para continuar.\n\n")

	if _, err := exec.LookPath("opencode"); err == nil {
		openCmd := exec.CommandContext(ctx, "opencode", projectDir)
		openCmd.Stdin = os.Stdin
		openCmd.Stdout = os.Stdout
		openCmd.Stderr = os.Stderr
		_ = openCmd.Run()
	} else {
		fmt.Println("  ⚠ opencode no encontrado. Instalalo y ejecutá: opencode", projectDir)
	}

	// 7. Verificar que la especificación se haya creado
	fmt.Print("  Verificando especificación...")
	specNodes, err := helixClient.TextSearch(ctx, "Spec", "project_id", fmt.Sprintf("%d", projectID), 1)
	if err != nil || len(specNodes) == 0 {
		fmt.Println()
		return &Result{Phase: PhaseF1, Status: StatusFail, Summary: "No se encontró la especificación. La Fase 1 no completó su trabajo."}, nil
	}

	spec := specNodes[0].Properties
	var missing []string
	for _, field := range []string{"architecture", "modules", "dependencies", "testing_strategy"} {
		if spec[field] == nil {
			missing = append(missing, field)
		}
	}
	if len(missing) > 0 {
		fmt.Println()
		return &Result{Phase: PhaseF1, Status: StatusFail, Summary: fmt.Sprintf("La especificación está incompleta. Faltan: %v", missing)}, nil
	}

	arch, _ := spec["architecture"].(string)
	fmt.Println(" OK")
	fmt.Printf("\n  Especificación lista: arquitectura %s\n", arch)

	summary := fmt.Sprintf("Arquitectura: %s | Módulos: %v", arch, spec["modules"])
	return &Result{Phase: PhaseF1, Status: StatusSuccess, Summary: summary}, nil
}

func (r *F1Runner) Name() Phase { return PhaseF1 }

// F2Runner: verifica Spec, lanza design → tasks, verifica output.
type F2Runner struct{}

func (r *F2Runner) Run(ctx context.Context, cfg *Config) (*Result, error) {
	select {
	case <-ctx.Done():
		return &Result{Phase: PhaseF2, Status: StatusFail, Summary: "tiempo de espera agotado"}, ctx.Err()
	default:
	}

	helixClient, err := helix.NewClient(ctx)
	if err != nil {
		return &Result{Phase: PhaseF2, Status: StatusFail, Summary: "No se pudo conectar a la base de datos"}, nil
	}
	defer helixClient.Close()

	payload, err := handoff.Parse("handoff.yaml")
	if err != nil {
		return &Result{Phase: PhaseF2, Status: StatusFail, Summary: "No se pudo leer handoff.yaml"}, nil
	}

	projectNodes, _ := helixClient.TextSearch(ctx, "Project", "name", payload.Project.Name, 1)
	if len(projectNodes) == 0 {
		return &Result{Phase: PhaseF2, Status: StatusFail, Summary: "Proyecto no encontrado"}, nil
	}
	projectID := projectNodes[0].ID

	// Verificar Spec
	fmt.Print("  Verificando especificación de F1...")
	specNodes, _ := helixClient.TextSearch(ctx, "Spec", "project_id", fmt.Sprintf("%d", projectID), 1)
	if len(specNodes) == 0 {
		fmt.Println()
		return &Result{Phase: PhaseF2, Status: StatusFail, Summary: "F1 incompleta: no se encontró la especificación"}, nil
	}
	fmt.Println(" OK")

	// Fase 2a: Diseño
	fmt.Println("\n  Paso 1: Diseño técnico — abriendo editor...")
	fmt.Print("  El asistente va a diseñar los componentes y el flujo de datos.")
	fmt.Println("  Cuando termines, cerrá el editor para continuar.")

	projectDir := payload.Project.Name
	if _, err := exec.LookPath("opencode"); err == nil {
		openCmd := exec.CommandContext(ctx, "opencode", projectDir)
		openCmd.Stdin = os.Stdin
		openCmd.Stdout = os.Stdout
		openCmd.Stderr = os.Stderr
		_ = openCmd.Run()
	} else {
		fmt.Println("  ⚠ opencode no encontrado. Ejecutá: opencode", projectDir)
	}

	// Verificar Design node
	fmt.Print("  Verificando diseño...")
	designNodes, _ := helixClient.TextSearch(ctx, "Design", "project_id", fmt.Sprintf("%d", projectID), 1)
	if len(designNodes) == 0 {
		fmt.Println()
		return &Result{Phase: PhaseF2, Status: StatusFail, Summary: "No se encontró el diseño. La Fase 2a no completó su trabajo."}, nil
	}
	fmt.Println(" OK")

	// Fase 2b: Tareas
	fmt.Println("\n  Paso 2: Planificación de tareas — abriendo editor...")
	fmt.Print("  El asistente va a dividir el diseño en tareas atómicas.")
	fmt.Println("  Cuando termines, cerrá el editor para continuar.")

	if _, err := exec.LookPath("opencode"); err == nil {
		openCmd := exec.CommandContext(ctx, "opencode", projectDir)
		openCmd.Stdin = os.Stdin
		openCmd.Stdout = os.Stdout
		openCmd.Stderr = os.Stderr
		_ = openCmd.Run()
	} else {
		fmt.Println("  ⚠ opencode no encontrado. Ejecutá: opencode", projectDir)
	}

	// Verificar Task nodes
	fmt.Print("  Verificando tareas...")
	taskNodes, _ := helixClient.TextSearch(ctx, "Task", "project_id", fmt.Sprintf("%d", projectID), 10)
	if len(taskNodes) == 0 {
		fmt.Println()
		return &Result{Phase: PhaseF2, Status: StatusFail, Summary: "No se encontraron tareas. La Fase 2b no completó su trabajo."}, nil
	}
	fmt.Printf(" OK (%d tareas)\n", len(taskNodes))

	summary := fmt.Sprintf("Diseño completado | %d tareas planificadas", len(taskNodes))
	return &Result{Phase: PhaseF2, Status: StatusSuccess, Summary: summary}, nil
}

func (r *F2Runner) Name() Phase { return PhaseF2 }

// F3Runner: loop apply→verify para cada tarea, con reintentos.
type F3Runner struct{}

func (r *F3Runner) Run(ctx context.Context, cfg *Config) (*Result, error) {
	select {
	case <-ctx.Done():
		return &Result{Phase: PhaseF3, Status: StatusFail, Summary: "tiempo de espera agotado"}, ctx.Err()
	default:
	}

	helixClient, err := helix.NewClient(ctx)
	if err != nil {
		return &Result{Phase: PhaseF3, Status: StatusFail, Summary: "No se pudo conectar a la base de datos"}, nil
	}
	defer helixClient.Close()

	payload, err := handoff.Parse("handoff.yaml")
	if err != nil {
		return &Result{Phase: PhaseF3, Status: StatusFail, Summary: "No se pudo leer handoff.yaml"}, nil
	}

	projectNodes, _ := helixClient.TextSearch(ctx, "Project", "name", payload.Project.Name, 1)
	if len(projectNodes) == 0 {
		return &Result{Phase: PhaseF3, Status: StatusFail, Summary: "Proyecto no encontrado"}, nil
	}
	projectID := projectNodes[0].ID

	// Verificar precondiciones
	fmt.Print("  Verificando Spec + Design + Tasks...")
	specNodes, _ := helixClient.TextSearch(ctx, "Spec", "project_id", fmt.Sprintf("%d", projectID), 1)
	designNodes, _ := helixClient.TextSearch(ctx, "Design", "project_id", fmt.Sprintf("%d", projectID), 1)
	taskNodes, _ := helixClient.TextSearch(ctx, "Task", "project_id", fmt.Sprintf("%d", projectID), 10)
	if len(specNodes) == 0 || len(designNodes) == 0 || len(taskNodes) == 0 {
		fmt.Println()
		return &Result{Phase: PhaseF3, Status: StatusFail, Summary: "Fases anteriores incompletas: faltan Spec, Design o Tasks"}, nil
	}
	fmt.Printf(" OK (%d tareas pendientes)\n", len(taskNodes))

	projectDir := payload.Project.Name
	maxLoops := cfg.MaxLoops
	if maxLoops == 0 {
		maxLoops = 5
	}

	// Loop apply → verify por cada tarea
	for attempt := 0; attempt < maxLoops; attempt++ {
		fmt.Printf("\n  Intento %d de %d\n", attempt+1, maxLoops)

		// Apply
		fmt.Println("  Abriendo editor para implementación...")
		fmt.Print("  El asistente va a implementar las tareas pendientes.")
		fmt.Println("  Cuando termines, cerrá el editor para continuar.")

		if _, err := exec.LookPath("opencode"); err == nil {
			openCmd := exec.CommandContext(ctx, "opencode", projectDir)
			openCmd.Stdin = os.Stdin
			openCmd.Stdout = os.Stdout
			openCmd.Stderr = os.Stderr
			_ = openCmd.Run()
		} else {
			fmt.Println("  ⚠ opencode no encontrado. Ejecutá: opencode", projectDir)
		}

		// Verify
		fmt.Print("  Verificando implementación...")
		reviewNodes, _ := helixClient.TextSearch(ctx, "Review", "project_id", fmt.Sprintf("%d", projectID), 10)

		// Check if all reviews pass
		allPass := true
		var failCount int
		for _, rn := range reviewNodes {
			if status, ok := rn.Properties["status"].(string); ok && status == "fail" {
				allPass = false
				failCount++
			}
		}

		if allPass && len(reviewNodes) > 0 {
			fmt.Printf(" OK (%d revisiones pasaron)\n", len(reviewNodes))
			summary := fmt.Sprintf("Implementación completa | %d tareas | %d intentos", len(taskNodes), attempt+1)
			return &Result{Phase: PhaseF3, Status: StatusSuccess, Summary: summary}, nil
		}

		if failCount > 0 {
			fmt.Printf(" %d tareas con errores\n", failCount)
		} else if len(reviewNodes) == 0 {
			fmt.Println(" no se encontraron revisiones")
		}

		if attempt+1 < maxLoops {
			fmt.Println("  Reintentando...")
		}
	}

	return &Result{Phase: PhaseF3, Status: StatusFail, Summary: fmt.Sprintf("Se agotaron los %d intentos. Revisar las tareas manualmente.", maxLoops)}, nil
}

func (r *F3Runner) Name() Phase { return PhaseF3 }

// F4Runner verifica que todo esté completado y archiva el proyecto.
type F4Runner struct{}

func (r *F4Runner) Run(ctx context.Context, cfg *Config) (*Result, error) {
	select {
	case <-ctx.Done():
		return &Result{Phase: PhaseF4, Status: StatusFail, Summary: "tiempo de espera agotado"}, ctx.Err()
	default:
	}

	helixClient, err := helix.NewClient(ctx)
	if err != nil {
		return &Result{Phase: PhaseF4, Status: StatusFail, Summary: "No se pudo conectar a la base de datos"}, nil
	}
	defer helixClient.Close()

	payload, err := handoff.Parse("handoff.yaml")
	if err != nil {
		return &Result{Phase: PhaseF4, Status: StatusFail, Summary: "No se pudo leer handoff.yaml"}, nil
	}

	projectNodes, _ := helixClient.TextSearch(ctx, "Project", "name", payload.Project.Name, 1)
	if len(projectNodes) == 0 {
		return &Result{Phase: PhaseF4, Status: StatusFail, Summary: "Proyecto no encontrado"}, nil
	}
	projectID := projectNodes[0].ID
	projectDir := payload.Project.Name

	// Verificar que no haya tareas pendientes
	fmt.Print("  Verificando tareas completadas...")
	taskNodes, _ := helixClient.TextSearch(ctx, "Task", "project_id", fmt.Sprintf("%d", projectID), 50)
	pendingCount := 0
	for _, t := range taskNodes {
		if status, ok := t.Properties["status"].(string); ok && status != "completed" {
			pendingCount++
		}
	}
	if pendingCount > 0 {
		fmt.Println()
		return &Result{Phase: PhaseF4, Status: StatusFail, Summary: fmt.Sprintf("Hay %d tareas sin completar. Completalas antes de archivar.", pendingCount)}, nil
	}
	fmt.Printf(" OK (%d completadas)\n", len(taskNodes))

	// Actualizar Project node
	fmt.Print("  Archivando proyecto...")
	helixClient.UpdateNode(ctx, int64(projectID), map[string]any{
		"status": "archived",
	})
	fmt.Println(" OK")

	// Guardar Archive node
	fmt.Print("  Guardando registro de cierre...")
	summary := fmt.Sprintf("Proyecto %s completado. %d tareas realizadas.", payload.Project.Name, len(taskNodes))
	helixClient.CreateNode(ctx, "Archive", map[string]any{
		"project_id":      projectID,
		"summary":         summary,
		"tasks_completed": len(taskNodes),
		"status":          "archived",
	})
	fmt.Println(" OK")

	// Limpiar temporales de .zyro/
	fmt.Print("  Limpiando archivos temporales...")
	os.Remove(filepath.Join(projectDir, ".zyro", "task.yaml"))
	os.Remove(filepath.Join(projectDir, ".zyro", "result.yaml"))
	fmt.Println(" OK")

	fmt.Printf("\n  Proyecto archivado. Resumen: %s\n", summary)
	return &Result{Phase: PhaseF4, Status: StatusSuccess, Summary: summary}, nil
}

func (r *F4Runner) Name() Phase { return PhaseF4 }

// F0Runner abre OpenCode para Fase 0 y verifica post-condiciones en HelixDB.
type F0Runner struct{}

func (r *F0Runner) Run(ctx context.Context, cfg *Config) (*Result, error) {
	select {
	case <-ctx.Done():
		return &Result{Phase: PhaseF0, Status: StatusFail, Summary: "tiempo agotado"}, ctx.Err()
	default:
	}

	payload, err := handoff.Parse("handoff.yaml")
	if err != nil {
		return &Result{Phase: PhaseF0, Status: StatusFail, Summary: "No se pudo leer handoff.yaml"}, nil
	}

	projectDir := payload.Project.Name
	if projectDir == "" {
		projectDir = "."
	}

	// Escribir .zyro/task.yaml con contexto mínimo
	taskDir := filepath.Join(projectDir, ".zyro")
	os.MkdirAll(taskDir, 0755)
	taskYAML := fmt.Sprintf(`phase: "F0"
required_output:
  patterns: true
  libraries: true
  skills: true
`)
	os.WriteFile(filepath.Join(taskDir, "task.yaml"), []byte(taskYAML), 0644)

	fmt.Printf("  Abriendo editor para Fase 0 (investigación del proyecto)...\n")
	if os.Getenv("ZYRO_TEST") == "" {
		if _, err := exec.LookPath("opencode"); err == nil {
			openCmd := exec.CommandContext(ctx, "opencode", projectDir)
			openCmd.Stdin = os.Stdin
			openCmd.Stdout = os.Stdout
			openCmd.Stderr = os.Stderr
			_ = openCmd.Run()
		} else {
			fmt.Println("  ⚠ opencode no encontrado. Abrí manualmente:", projectDir)
		}
	} else {
		fmt.Println("  [ZYRO_TEST] saltando apertura de OpenCode")
	}

	// Verificar post-condiciones en HelixDB (no bloqueante — si falla, solo advierte)
	helixClient, err := helix.NewClient(ctx)
	if err == nil {
		defer helixClient.Close()
		fmt.Print("  Verificando resultados de Fase 0...")

		projectNodes, _ := helixClient.TextSearch(ctx, "Project", "name", payload.Project.Name, 1)
		if len(projectNodes) == 0 {
			fmt.Println(" ⚠ proyecto no encontrado en base de datos")
			return &Result{Phase: PhaseF0, Status: StatusSuccess, Summary: "Fase 0 completada (pendiente de registro en HelixDB)"}, nil
		}

		projectID := projectNodes[0].ID
		var faltan []string

		patterns, _ := helixClient.GetOutgoing(ctx, projectID, "HAS_PATTERN")
		if len(patterns) == 0 {
			faltan = append(faltan, "patrones")
		}

		libs, _ := helixClient.GetOutgoing(ctx, projectID, "USES_LIB")
		if len(libs) == 0 {
			faltan = append(faltan, "librerías")
		}

		skills, _ := helixClient.GetOutgoing(ctx, projectID, "REQUIRES_SKILL")
		if len(skills) == 0 {
			faltan = append(faltan, "skills")
		}

		if len(faltan) > 0 {
			fmt.Printf(" faltan: %v\n", faltan)
			return &Result{Phase: PhaseF0, Status: StatusFail, Summary: fmt.Sprintf("Fase 0 incompleta. Faltan en HelixDB: %v", faltan)}, nil
		}

		fmt.Println(" OK")
		return &Result{Phase: PhaseF0, Status: StatusSuccess, Summary: "Fase 0 completada: patrones, librerías y skills verificados en HelixDB"}, nil
	}

	return &Result{Phase: PhaseF0, Status: StatusSuccess, Summary: "Fase 0 completada"}, nil
}

func (r *F0Runner) Name() Phase { return PhaseF0 }
