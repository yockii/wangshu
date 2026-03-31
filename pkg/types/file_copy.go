package types

type FsCopyData struct {
	Src     string `json:"src"`
	Dest    string `json:"dest"`
	Success bool   `json:"success"`
}

func NewFsCopyData(src string, dest string, success bool) *ActionOutput {
	return NewActionOutput("success", "copy file success", FsCopyData{
		Src:     src,
		Dest:    dest,
		Success: success,
	}, nil)
}
