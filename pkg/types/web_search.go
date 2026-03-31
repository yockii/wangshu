package types

type WebSearchData struct {
	Query   string `json:"query"`
	Results []struct {
		Title   string `json:"title"`
		URL     string `json:"url"`
		Snippet string `json:"snippet"`
	} `json:"results"`
}

func NewWebSearchData(query string, results []struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}) *ActionOutput {
	return NewActionOutput("success", "search web success", WebSearchData{
		Query:   query,
		Results: results,
	}, nil)
}
