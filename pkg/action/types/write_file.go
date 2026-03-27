package types

type FsWriteData struct {
	File           string `json:"file"`
	ContentWritten string `json:"content_written"`
	Created        bool   `json:"created"`
}

func NewFsWriteData(file string, content_written string, created bool) *ActionOutput {
	return NewActionOutput("success", "write file success", FsWriteData{
		File:           file,
		ContentWritten: content_written,
		Created:        created,
	}, nil)
}
