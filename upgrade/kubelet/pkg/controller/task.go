package controller

// Result encapsulates the result of running the Exec function of a task.
type Result struct {
	Err   error
	Retry bool
}
