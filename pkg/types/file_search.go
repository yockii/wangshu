package types

type FsSearchData struct {
	Pattern string   `json:"pattern"`
	Matches []string `json:"matches"`
}

func NewFsSearchData(pattern string, matches []string) *ActionOutput {
	return NewActionOutput("success", "search success", FsSearchData{
		Pattern: pattern,
		Matches: matches,
	}, nil)
}
