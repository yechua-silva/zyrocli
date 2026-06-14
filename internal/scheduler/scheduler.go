package scheduler

// Scheduler manages phase execution as a DAG of ordered tasks.
// TODO: implement DAG executor with goroutines + channels for parallel phase
// scheduling. Each phase maps to a node with edges to dependents.
type Scheduler struct {
	Phases []string
}
