package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/yechua-silva/zyrocli/internal/apply"
	dbhelix "github.com/yechua-silva/zyrocli/internal/db/helix"
	"github.com/yechua-silva/zyrocli/internal/boomerang"
	"github.com/yechua-silva/zyrocli/internal/setup"
	"github.com/spf13/cobra"
)

// JSON-RPC 2.0 types
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// MCP Initialize params
type InitializeParams struct {
	ProtocolVersion string          `json:"protocolVersion"`
	Capabilities    json.RawMessage `json:"capabilities"`
	ClientInfo      struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"clientInfo"`
}

// MCP Tool definitions
type ToolDefinition struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

type ToolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type ToolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// MCP Server global state
var taskManager *boomerang.TaskManager
var helixClient *dbhelix.Client

var mcpServerCmd = &cobra.Command{
	Use:    "mcp-server",
	Short:  "Start MCP server for task board tools",
	Long:   `Starts a stdio-based MCP server exposing task board tools (dispatch_task, check_task_status, wait_phase, list_tasks, cancel_task) for OpenCode orchestration.`,
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMCPServer()
	},
}

func runMCPServer() error {
	applyRunner := apply.NewRunner(apply.DefaultPoolConfig())
	taskManager = boomerang.NewTaskManagerWithRunner(5, applyRunner)

	var err error
	helixClient, err = dbhelix.NewClient(context.Background(),
		dbhelix.WithBaseURL(setup.GetHelixDBURL()),
		dbhelix.WithProjectID("zyrocli"),
	)
	if err != nil {
		log.Printf("[mcp] HelixDB no disponible: %v", err)
		// No es fatal — el MCP server funciona sin HelixDB
	}

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			sendError(nil, -32700, "Parse error", err.Error())
			continue
		}

		response := handleRequest(req)
		if response != nil {
			data, _ := json.Marshal(response)
			fmt.Println(string(data))
		}
	}

	return scanner.Err()
}

func handleRequest(req JSONRPCRequest) *JSONRPCResponse {
	switch req.Method {
	case "initialize":
		return handleInitialize(req)
	case "notifications/initialized":
		// No response needed for notifications
		return nil
	case "tools/list":
		return handleToolsList(req)
	case "tools/call":
		return handleToolsCall(req)
	default:
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      parseID(req.ID),
			Error: &RPCError{
				Code:    -32601,
				Message: fmt.Sprintf("Method not found: %s", req.Method),
			},
		}
	}
}

func parseID(raw json.RawMessage) interface{} {
	if len(raw) == 0 {
		return nil
	}
	// Try string first
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	// Try number
	var n float64
	if err := json.Unmarshal(raw, &n); err == nil {
		return n
	}
	return nil
}

func sendError(id interface{}, code int, message string, data string) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &RPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	jsonData, _ := json.Marshal(resp)
	fmt.Println(string(jsonData))
}

func handleInitialize(req JSONRPCRequest) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      parseID(req.ID),
		Result: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]bool{},
			},
			"serverInfo": map[string]string{
				"name":    "zyro-task-board",
				"version": "1.0.0",
			},
		},
	}
}

func handleToolsList(req JSONRPCRequest) *JSONRPCResponse {
	tools := []ToolDefinition{
		{
			Name:        "dispatch_task",
			Description: "Crea una tarea en estado running. La tarea NO se ejecuta automáticamente — el orquestador debe llamar task_complete cuando termine.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Nombre descriptivo de la tarea (ej: patterns, libraries, skills)",
					},
					"agent": map[string]interface{}{
						"type":        "string",
						"description": "Nombre del agente a ejecutar (ej: zyro-phase-0-patterns)",
					},
					"phase": map[string]interface{}{
						"type":        "string",
						"description": "Fase SDD (PRE-F0, F0, F1, F2, F3, F4)",
					},
					"params": map[string]interface{}{
						"type":        "object",
						"description": "Parámetros adicionales para el agente",
						"additionalProperties": map[string]interface{}{
							"type": "string",
						},
					},
				},
				"required": []string{"name", "agent", "phase"},
			},
		},
		{
			Name:        "check_task_status",
			Description: "Consulta el estado de una tarea por su ID.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{
						"type":        "string",
						"description": "ID de la tarea (ej: F0-patterns-1)",
					},
				},
				"required": []string{"task_id"},
			},
		},
		{
			Name:        "wait_phase",
			Description: "BLOQUEA hasta que TODAS las tareas de una fase estén completadas. Usar SOLO para sincronizar entre fases, no para consultar progreso.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"phase": map[string]interface{}{
						"type":        "string",
						"description": "Fase SDD (PRE-F0, F0, F1, F2, F3, F4)",
						"enum":        []string{"F0", "F1", "F2", "F3", "F4"},
					},
					"timeout": map[string]interface{}{
						"type":        "integer",
						"description": "Timeout en segundos (default: 600 = 10 min)",
					},
				},
				"required": []string{"phase"},
			},
		},
		{
			Name:        "list_tasks",
			Description: "Lista todas las tareas, opcionalmente filtradas por fase.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"phase": map[string]interface{}{
						"type":        "string",
						"description": "Filtro opcional por fase (PRE-F0, F0, F1, F2, F3, F4). Vacío = todas.",
					},
				},
			},
		},
		{
			Name:        "cancel_task",
			Description: "Cancela una tarea pendiente o en ejecución.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{
						"type":        "string",
						"description": "ID de la tarea a cancelar",
					},
				},
				"required": []string{"task_id"},
			},
		},
		{
			Name:        "complete_task",
			Description: "Marca una tarea como completada. La llama el orquestador cuando termina de ejecutar la tarea.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"task_id": map[string]interface{}{
						"type":        "string",
						"description": "ID de la tarea a completar (ej: F0-patterns-1)",
					},
					"output": map[string]interface{}{
						"type":        "string",
						"description": "Output de la tarea completada",
					},
				},
				"required": []string{"task_id"},
			},
		},
	}

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      parseID(req.ID),
		Result: map[string]interface{}{
			"tools": tools,
		},
	}
}

func handleToolsCall(req JSONRPCRequest) *JSONRPCResponse {
	var params ToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      parseID(req.ID),
			Error: &RPCError{
				Code:    -32602,
				Message: "Invalid params",
				Data:    err.Error(),
			},
		}
	}

	switch params.Name {
	case "task_create":
		return handleTaskCreate(req.ID, params.Arguments)
	case "task_status":
		return handleTaskStatus(req.ID, params.Arguments)
	case "task_wait":
		return handleTaskWait(req.ID, params.Arguments)
	case "task_list":
		return handleTaskList(req.ID, params.Arguments)
	case "task_cancel":
		return handleTaskCancel(req.ID, params.Arguments)
	case "task_complete":
		return handleTaskComplete(req.ID, params.Arguments)
	default:
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      parseID(req.ID),
			Error: &RPCError{
				Code:    -32601,
				Message: fmt.Sprintf("Tool not found: %s", params.Name),
			},
		}
	}
}

func handleTaskCreate(rawID json.RawMessage, args json.RawMessage) *JSONRPCResponse {
	var input struct {
		Name   string            `json:"name"`
		Agent  string            `json:"agent"`
		Phase  string            `json:"phase"`
		Params map[string]string `json:"params,omitempty"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return errorResponse(rawID, -32602, "Invalid arguments", err.Error())
	}
	if input.Name == "" || input.Agent == "" || input.Phase == "" {
		return errorResponse(rawID, -32602, "Missing required fields", "name, agent, and phase are required")
	}

	id := taskManager.CreateTask(context.Background(), input.Name, input.Agent, input.Phase, input.Params)

	// ── Guardar nodo Task en HelixDB (sin criteria aún — se asignan desde el orquestador) ──
	go saveTaskToHelix(string(id), input.Name, input.Agent, input.Phase, nil)

	result, _ := json.Marshal(map[string]string{
		"task_id": string(id),
		"status":  "dispatched",
	})

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      parseID(rawID),
		Result: map[string]interface{}{
			"content": []ToolContent{
				{Type: "text", Text: string(result)},
			},
		},
	}
}

func handleTaskStatus(rawID json.RawMessage, args json.RawMessage) *JSONRPCResponse {
	var input struct {
		TaskID string `json:"task_id"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return errorResponse(rawID, -32602, "Invalid arguments", err.Error())
	}

	task, ok := taskManager.GetTask(boomerang.TaskID(input.TaskID))
	if !ok {
		return errorResponse(rawID, -32602, "Task not found",
			fmt.Sprintf("No task with ID %s. Use list_tasks to see available tasks.", input.TaskID))
	}

	// Build a clear LLM-readable response with explicit status string
	result, _ := json.Marshal(map[string]interface{}{
		"task_id":    string(task.ID),
		"name":       task.Name,
		"agent":      task.Agent,
		"phase":      task.Phase,
		"status":     task.Status.String(), // explicit string "running", "done", etc.
		"output":     task.Output,
		"error":      task.Error,
		"started_at": task.StartedAt.Format(time.RFC3339),
		"done_at":    formatTimeOrNil(task.DoneAt),
	})

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      parseID(rawID),
		Result: map[string]interface{}{
			"content": []ToolContent{
				{Type: "text", Text: string(result)},
			},
		},
	}
}

func handleTaskWait(rawID json.RawMessage, args json.RawMessage) *JSONRPCResponse {
	var input struct {
		Phase   string `json:"phase"`
		Timeout int    `json:"timeout,omitempty"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return errorResponse(rawID, -32602, "Invalid arguments", err.Error())
	}

	// Validar que la fase sea conocida
	validPhases := map[string]bool{
		"PRE-F0": true, "F0": true, "F1": true,
		"F2": true, "F3": true, "F4": true,
	}
	if !validPhases[input.Phase] {
		return errorResponse(rawID, -32602, "Invalid phase",
			fmt.Sprintf("Phase %q is not valid. Valid phases: PRE-F0, F0, F1, F2, F3, F4", input.Phase))
	}

	timeout := 600 * time.Second // default 10 min
	if input.Timeout > 0 {
		timeout = time.Duration(input.Timeout) * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	err := taskManager.WaitPhase(ctx, input.Phase)
	if err != nil {
		return errorResponse(rawID, -32602, "Phase wait failed", err.Error())
	}

	summary := taskManager.PhaseSummary(input.Phase)

	// Determine overall status
	status := "completed"
	if summary.Failed > 0 {
		status = "completed_with_errors"
	}

	// Build descriptive message for the LLM
	statusMsg := fmt.Sprintf("Fase %s completada. %d tareas: %d exitosas, %d fallaron, %d canceladas, %d pendientes.",
		input.Phase, summary.Total, summary.Done, summary.Failed, summary.Cancelled, summary.Pending)

	result, _ := json.Marshal(map[string]interface{}{
		"phase":   input.Phase,
		"status":  status,
		"message": statusMsg,
		"summary": summary,
	})

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      parseID(rawID),
		Result: map[string]interface{}{
			"content": []ToolContent{
				{Type: "text", Text: string(result)},
			},
		},
	}
}

func handleTaskList(rawID json.RawMessage, args json.RawMessage) *JSONRPCResponse {
	var input struct {
		Phase string `json:"phase,omitempty"`
	}
	json.Unmarshal(args, &input) // ignore error, phase is optional

	tasks := taskManager.ListTasks(input.Phase)
	result, _ := json.Marshal(map[string]interface{}{
		"tasks": tasks,
	})

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      parseID(rawID),
		Result: map[string]interface{}{
			"content": []ToolContent{
				{Type: "text", Text: string(result)},
			},
		},
	}
}

func handleTaskCancel(rawID json.RawMessage, args json.RawMessage) *JSONRPCResponse {
	var input struct {
		TaskID string `json:"task_id"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return errorResponse(rawID, -32602, "Invalid arguments", err.Error())
	}

	cancelled := taskManager.CancelTask(boomerang.TaskID(input.TaskID))
	result, _ := json.Marshal(map[string]interface{}{
		"task_id":   input.TaskID,
		"cancelled": cancelled,
	})

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      parseID(rawID),
		Result: map[string]interface{}{
			"content": []ToolContent{
				{Type: "text", Text: string(result)},
			},
		},
	}
}

func handleTaskComplete(rawID json.RawMessage, args json.RawMessage) *JSONRPCResponse {
	var input struct {
		TaskID string `json:"task_id"`
		Output string `json:"output,omitempty"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return errorResponse(rawID, -32602, "Invalid arguments", err.Error())
	}

	ok := taskManager.CompleteTask(boomerang.TaskID(input.TaskID), input.Output)
	result, _ := json.Marshal(map[string]interface{}{
		"task_id":   input.TaskID,
		"completed": ok,
	})

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      parseID(rawID),
		Result: map[string]interface{}{
			"content": []ToolContent{
				{Type: "text", Text: string(result)},
			},
		},
	}
}

// saveTaskToHelix guarda un nodo Task en HelixDB mediante el SDK con project isolation.
// Si criteria no está vacío, serializa cada AcceptanceCriteria a map[string]any y lo
// incluye en las properties del nodo como acceptance_criteria.
func saveTaskToHelix(taskID, name, agent, phase string, criteria []boomerang.AcceptanceCriteria) {
	if helixClient == nil {
		log.Printf("[mcp] HelixDB no disponible, saltando persistencia de Task %s", taskID)
		return
	}
	props := map[string]interface{}{
		"task_id": taskID,
		"name":    name,
		"agent":   agent,
		"phase":   phase,
		"status":  "running",
	}

	if len(criteria) > 0 {
		criteriaData := make([]map[string]any, len(criteria))
		for i, c := range criteria {
			criteriaData[i] = map[string]any{
				"id":          c.ID,
				"description": c.Description,
				"phase":       c.Phase,
				"status":      string(c.Status),
				"source":      c.Source,
				"task_id":     c.TaskID,
			}
		}
		props["acceptance_criteria"] = criteriaData
	}

	nodeID, err := helixClient.CreateNode(context.Background(), "Task", props)
	if err != nil {
		log.Printf("[mcp] Error creando nodo Task %s en HelixDB: %v", taskID, err)
		return
	}
	log.Printf("[mcp] Nodo Task creado: id=%d, task_id=%s", nodeID, taskID)
}

// deserializeCriteria convierte un slice raw de interface{} (proveniente de HelixDB)
// a un slice tipado de AcceptanceCriteria.
func deserializeCriteria(raw []interface{}) []boomerang.AcceptanceCriteria {
	var result []boomerang.AcceptanceCriteria
	for _, item := range raw {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		c := boomerang.AcceptanceCriteria{
			ID:          getString(m, "id"),
			Description: getString(m, "description"),
			Phase:       getString(m, "phase"),
			Status:      boomerang.CriteriaStatus(getString(m, "status")),
			Source:      getString(m, "source"),
			TaskID:      getString(m, "task_id"),
		}
		result = append(result, c)
	}
	return result
}

// getString extrae un string de un map, retornando "" si la key no existe o no es string.
func getString(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}


// formatTimeOrNil returns the time formatted as RFC3339, or nil if zero.
func formatTimeOrNil(t time.Time) interface{} {
	if t.IsZero() {
		return nil
	}
	return t.Format(time.RFC3339)
}

func errorResponse(rawID json.RawMessage, code int, message string, data string) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      parseID(rawID),
		Error: &RPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
}

func init() {
	rootCmd.AddCommand(mcpServerCmd)
}
