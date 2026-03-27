package types

type FsGrepData struct {
	Pattern string `json:"pattern"`
	Matches []struct {
		Path string `json:"path"`
		Line int    `json:"line"`
		Text string `json:"text"`
	} `json:"matches"`
}

func NewFsGrepData(pattern string, matches []struct {
	Path string `json:"path"`
	Line int    `json:"line"`
	Text string `json:"text"`
}) *ActionOutput {
	return NewActionOutput("success", "grep search success", FsGrepData{
		Pattern: pattern,
		Matches: matches,
	}, nil)
}
