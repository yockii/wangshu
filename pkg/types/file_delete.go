package types

type DeleteFileData struct {
	Path string `json:"path"`
}

func NewDeleteFileData(path string) *ActionOutput {
	return NewActionOutput("success", "delete file success", DeleteFileData{
		Path: path,
	}, nil)
}
