package types

type FsMoveData struct {
	OldPath string `json:"old_path"`
	NewPath string `json:"new_path"`
	Success bool   `json:"success"`
}

func NewMoveData(old_path string, new_path string, success bool) *ActionOutput {
	return NewActionOutput("success", "rename file success", FsMoveData{
		OldPath: old_path,
		NewPath: new_path,
		Success: success,
	}, nil)
}
