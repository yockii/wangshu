package feishu

type CardV2 struct {
	Schema string   `json:"schema"` // 2.0
	Body   CardBody `json:"body"`
}

type CardBody struct {
	Elements []CardElement `json:"elements"`
}

type CardElement struct {
	Tag       string `json:"tag"`
	Content   string `json:"content"`
	ElementID string `json:"element_id"`
}
