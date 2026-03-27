package types

type TimeNowData struct {
	Timestamp string `json:"timestamp"`
}

func NewTimeNowData(timestamp string) *ActionOutput {
	return NewActionOutput("success", "get time now success", TimeNowData{
		Timestamp: timestamp,
	}, nil)
}
