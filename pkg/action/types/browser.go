package types

type BrowserOpenData struct {
	URL      string        `json:"url"`
	Elements []ElementInfo `json:"elements"`
}

func NewBrowserOpenData(url string, elements []ElementInfo) *ActionOutput {
	return NewActionOutput("success", "browser open success", BrowserOpenData{
		URL:      url,
		Elements: elements,
	}, nil)
}

type TextData struct {
	Elements []ElementInfo `json:"elements"`
}

type ClickData TextData
type FillData TextData

func NewBrowserTextData(elements []ElementInfo) *ActionOutput {
	return NewActionOutput("success", "text success", TextData{
		Elements: elements,
	}, nil)
}

func NewBrowserClickData(elements []ElementInfo) *ActionOutput {
	return NewActionOutput("success", "click success", ClickData{
		Elements: elements,
	}, nil)
}

func NewBrowserFillData(elements []ElementInfo) *ActionOutput {
	return NewActionOutput("success", "fill success", FillData{
		Elements: elements,
	}, nil)
}

type HTMLData struct {
	Format    string `json:"format"`
	Start     int    `json:"start"`
	MaxLength int    `json:"max_length"`
	Content   string `json:"content"`
	NextStart int    `json:"next_start"`
}

func NewBrowserHTMLData(format string, start int, max_length int, content string, next_start int) *ActionOutput {
	return NewActionOutput("success", "html success", HTMLData{
		Format:    format,
		Start:     start,
		MaxLength: max_length,
		Content:   content,
		NextStart: next_start,
	}, nil)
}

type ElementInfo struct {
	// 基本信息
	Tag      string `json:"tag"`
	Visible  bool   `json:"visible"`
	Enabled  bool   `json:"enabled"`
	Editable bool   `json:"editable"`

	// 各种选择器
	IDSelector    string            `json:"id_selector,omitempty"`
	NameSelector  string            `json:"name_selector,omitempty"`
	ClassSelector string            `json:"class_selector,omitempty"`
	XPathSelector string            `json:"xpath_selector,omitempty"`
	DataSelectors map[string]string `json:"data_selectors,omitempty"` // data-testid, data-test-id等

	// 元素属性
	Type        string `json:"type,omitempty"`
	Name        string `json:"name,omitempty"`
	Placeholder string `json:"placeholder,omitempty"`
	Value       string `json:"value,omitempty"`
	Text        string `json:"text,omitempty"`
	Href        string `json:"href,omitempty"`
	ARIALabel   string `json:"aria_label,omitempty"`

	// 表单特定属性
	ReadOnly bool `json:"readonly,omitempty"`
	Required bool `json:"required,omitempty"`
	Checked  bool `json:"checked,omitempty"`
}

type BrowserScreenshotData struct {
	Path string `json:"path"`
}

func NewBrowserScreenshotData(path string) *ActionOutput {
	return NewActionOutput("success", "browser screenshot success", BrowserScreenshotData{
		Path: path,
	}, nil)
}

type BrowserListTabsData struct {
	Tabs []struct {
		Title string `json:"title"`
		URL   string `json:"url"`
	} `json:"tabs"`
}

func NewBrowserListTabsData(tabs []struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}) *ActionOutput {
	return NewActionOutput("success", "list tabs success", BrowserListTabsData{
		Tabs: tabs,
	}, nil)
}

type BrowserRunTaskData struct {
	Result map[string]any `json:"result"`
}

func NewBrowserRunTaskData(result map[string]any) *ActionOutput {
	return NewActionOutput("success", "run task success", BrowserRunTaskData{
		Result: result,
	}, nil)
	return NewActionOutput("success", "run task success", BrowserRunTaskData{
		Result: result,
	}, nil)
}
