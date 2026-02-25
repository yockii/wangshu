package bus

type InboundMessage struct {
	Channel    string            // 消息通道标识，如"telegram/wechat等
	SenderID   string            // 发送者ID，根据不同通道有不同的格式
	ChatID     string            // 聊天或会话的ID
	Content    string            // 消息内容
	Media      []MediaAttachment // 可选的媒体附件，如图片、文件等
	SessionKey string            // 会话key，状态管理
	Metadata   map[string]string // 可选的元数据，如消息ID、时间戳等
}

// MediaAttachment 媒体附件
type MediaAttachment struct {
	Type      string // 媒体类型，如"image"、"file"等
	URL       string // 媒体文件的URL
	MimeType  string // 媒体文件的MIME类型
	FileSize  int64  // 媒体文件的大小，单位字节
	Thumbnail string // 可选的缩略图URL
}

type OutboundMessage struct {
	Channel string            // 消息通道标识，如"telegram/wechat等
	ChatID  string            // 聊天或会话的ID
	Content string            // 消息内容
	Media   []MediaAttachment // 可选的媒体附件，如图片、文件等
}
