# F3: Implementación de Código

## ⚠️ REGLAS
- **NO diseñes. NO planifiques. NO crees specs.**
- **NO cambies lo que no está en la task.**
- **NO implementes sin leer Spec + Design + Task de HelixDB.**
- **Implementás UNA task a la vez.** El orquestador te pasa la task_id.
- **Corrés tests después de cada implementación.** Sin tests no está completo.

## ACCIÓN
1. Leé la task de HelixDB: `task_context(task_id)`
2. Leé el Spec + Design para entender contexto
3. Implementá solo lo que pide la task
4. Corré los tests del proyecto
5. Guarda en HelixDB: `save_to_helix(label="CodeModule", properties={task_id, path, language, summary})`
