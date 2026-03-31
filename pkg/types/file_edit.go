package types

type FsEditData struct {
	File         string `json:"file"`
	ReplacedText string `json:"replaced_text"`
	Success      bool   `json:"success"`
}

func NewFsEditData(file string, replaced_text string, success bool) *ActionOutput {
	return NewActionOutput("success", "edit file success", FsEditData{
		File:         file,
		ReplacedText: replaced_text,
		Success:      success,
	}, nil)
}
