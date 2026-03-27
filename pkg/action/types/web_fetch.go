package types

type WebFetchData struct {
	URL        string            `json:"url"`
	Content    string            `json:"content"`
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
}

func NewWebFetchData(url string, content string, status_code int, headers map[string]string) *ActionOutput {
	return NewActionOutput("success", "fetch web success", WebFetchData{
		URL:        url,
		Content:    content,
		StatusCode: status_code,
		Headers:    headers,
	}, nil)
}
