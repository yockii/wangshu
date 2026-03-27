package types

type MessageSendData struct {
	Channel   string `json:"channel"`
	MessageID string `json:"message_id"`
	Timestamp string `json:"timestamp"`
}

func NewMessageSendData(channel string, message_id string, timestamp string) *ActionOutput {
	return NewActionOutput("success", "message send success", MessageSendData{
		Channel:   channel,
		MessageID: message_id,
		Timestamp: timestamp,
	}, nil)
}
