package types

type FsListData struct {
	Path  string `json:"path"`
	Items []struct {
		Name  string `json:"name"`
		IsDir bool   `json:"is_dir"`
		Size  int64  `json:"size"`
	} `json:"items"`
}

func NewFsListData(path string, items []struct {
	Name  string `json:"name"`
	IsDir bool   `json:"is_dir"`
	Size  int64  `json:"size"`
}) *ActionOutput {
	return NewActionOutput("success", "list dir success", FsListData{
		Path:  path,
		Items: items,
	}, nil)
}
