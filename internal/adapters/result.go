package adapters

// Result holds the output from a single agent execution
type Result struct {
	TaskID   string
	Agent    string
	Output   string
	Status   string  // "success" | "failed" | "timeout" | "no_agent"
	Duration float64 // seconds
	ExitCode int
	Error    error
}

// IsSuccess returns true if the execution completed without error.
func (r *Result) IsSuccess() bool {
	return r.Status == "success"
}
