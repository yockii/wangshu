package types

type RunData struct {
	Output   string `json:"output"`    // The output of the command
	ExitCode int    `json:"exit_code"` // The exit code of the command
}

func NewRunData(output string, exit_code int) *ActionOutput {
	return NewActionOutput("success", "run command success", RunData{
		Output:   output,
		ExitCode: exit_code,
	}, nil)
}
