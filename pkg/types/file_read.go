package types

type FsReadData struct {
	File    string `json:"file"`
	Content string `json:"content"`
	Type    string `json:"type"`
}

func NewFsReadData(file string, content string, t string) *ActionOutput {
	return NewActionOutput("success", "read file success", FsReadData{
		File:    file,
		Content: content,
		Type:    t,
	}, nil)
}
