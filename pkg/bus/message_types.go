package bus

import "time"

// MessageType 消息类型
type MessageType string

const (
	MessageTypeText      MessageType = "text"       // 纯文本
	MessageTypeImage     MessageType = "image"      // 图片
	MessageTypeVideo     MessageType = "video"      // 视频
	MessageTypeAudio     MessageType = "audio"      // 音频
	MessageTypeVoice     MessageType = "voice"      // 语音消息
	MessageTypeFile      MessageType = "file"       // 文件
	MessageTypeLocation  MessageType = "location"   // 位置
	MessageTypeSticker   MessageType = "sticker"    // 贴纸
	MessageTypeRichMedia MessageType = "rich_media" // 富媒体（图文、卡片等）
)

// EntityType 实体类型
type EntityType string

const (
	EntityTypeMention    EntityType = "mention"     // @用户
	EntityTypeMentionAll EntityType = "mention_all" // @全体成员
	EntityTypeLink       EntityType = "link"        // 链接
	EntityTypeHashtag    EntityType = "hashtag"     // 话题标签
	EntityTypeBotCommand EntityType = "bot_command" // 机器人命令
	EntityTypeEmail      EntityType = "email"       // 邮箱地址
)

// MessageEntity 消息实体，表示文本中的特殊元素
type MessageEntity struct {
	Type    EntityType `json:"type"`              // 实体类型
	Offset  int        `json:"offset"`            // 在文本中的起始位置
	Length  int        `json:"length"`            // 长度
	URL     string     `json:"url,omitempty"`     // 链接URL
	UserID  string     `json:"user_id,omitempty"` // @的用户ID
	Mention string     `json:"mention,omitempty"` // @的内容（如@全体成员）
}

// ReferenceType 引用类型
type ReferenceType string

const (
	ReferenceTypeReply   ReferenceType = "reply"   // 回复
	ReferenceTypeForward ReferenceType = "forward" // 转发
	ReferenceTypeQuote   ReferenceType = "quote"   // 引用
)

// MessageReference 消息引用
type MessageReference struct {
	MessageID     string        `json:"message_id"`     // 被引用的消息ID
	ChatID        string        `json:"chat_id"`        // 所属聊天ID
	SenderID      string        `json:"sender_id"`      // 发送者ID
	SenderName    string        `json:"sender_name"`    // 发送者名称
	Content       string        `json:"content"`        // 内容摘要
	MessageType   MessageType   `json:"message_type"`   // 消息类型
	ReferenceType ReferenceType `json:"reference_type"` // 引用类型
}

// MediaType 媒体类型
type MediaType string

const (
	MediaTypeImage MediaType = "image"
	MediaTypeVideo MediaType = "video"
	MediaTypeAudio MediaType = "audio"
	MediaTypeFile  MediaType = "file"
)

// MediaContent 媒体内容
type MediaContent struct {
	Type      MediaType `json:"type"`                // 媒体类型
	URL       string    `json:"url,omitempty"`       // 媒体URL
	FilePath  string    `json:"file_path,omitempty"` // 本地文件路径
	Thumbnail string    `json:"thumbnail,omitempty"` // 缩略图URL
	Size      int64     `json:"size,omitempty"`      // 文件大小
	Duration  int       `json:"duration,omitempty"`  // 时长（秒，音视频）
	Width     int       `json:"width,omitempty"`     // 宽度（图片/视频）
	Height    int       `json:"height,omitempty"`    // 高度（图片/视频）
	FileName  string    `json:"file_name,omitempty"` // 文件名
	MimeType  string    `json:"mime_type,omitempty"` // MIME类型
}

// LocationInfo 位置信息
type LocationInfo struct {
	Latitude  float64 `json:"latitude"`          // 纬度
	Longitude float64 `json:"longitude"`         // 经度
	Title     string  `json:"title,omitempty"`   // 位置名称
	Address   string  `json:"address,omitempty"` // 详细地址
}

// Keyboard 交互键盘
type Keyboard struct {
	Inline     bool          `json:"inline"` // 是否为内联键盘（显示在消息下方）
	Rows       []KeyboardRow `json:"rows"`
	Resizeable bool          `json:"resizeable,omitempty"` // 是否可调整大小
	OneTime    bool          `json:"one_time,omitempty"`   // 是否一次性键盘
}

// KeyboardRow 键盘行
type KeyboardRow struct {
	Buttons []KeyboardButton `json:"buttons"`
}

// KeyboardButton 键盘按钮
type KeyboardButton struct {
	Text            string `json:"text"`                       // 按钮文本
	Data            string `json:"data,omitempty"`             // 回调数据
	URL             string `json:"url,omitempty"`              // 链接URL
	RequestContact  bool   `json:"request_contact,omitempty"`  // 请求联系人
	RequestLocation bool   `json:"request_location,omitempty"` // 请求位置
}

// MessageForwardInfo 转发信息
type MessageForwardInfo struct {
	FromChatID     string `json:"from_chat_id"`     // 来源聊天ID
	FromMessageID  string `json:"from_message_id"`  // 来源消息ID
	FromSenderName string `json:"from_sender_name"` // 来源发送者名称
	FromSenderID   string `json:"from_sender_id"`   // 来源发送者ID
}

// MessageMetadata 消息元数据
type MessageMetadata struct {
	MessageID     string              `json:"message_id"`               // 消息ID
	SenderID      string              `json:"sender_id"`                // 发送者ID
	SenderName    string              `json:"sender_name"`              // 发送者名称
	ChatID        string              `json:"chat_id"`                  // 聊天ID
	ChatName      string              `json:"chat_name"`                // 聊天名称
	ChatType      string              `json:"chat_type"`                // 聊天类型, p2p group topic?
	Timestamp     time.Time           `json:"timestamp"`                // 发送时间
	EditTimestamp *time.Time          `json:"edit_timestamp,omitempty"` // 编辑时间
	ForwardFrom   *MessageForwardInfo `json:"forward_from,omitempty"`   // 转发信息
	ReplyToID     string              `json:"reply_to_id,omitempty"`    // 回复的消息ID
	Pinned        bool                `json:"pinned,omitempty"`         // 是否被置顶
	Views         int                 `json:"views,omitempty"`          // 查看次数
	Forwards      int                 `json:"forwards,omitempty"`       // 转发次数
	Channel       string              `json:"channel"`                  // 渠道标识

	// 会话进度
	SessionPercent float64 `json:"session_percent,omitempty"` // 会话进度（0-1）
}

// Message 统一的消息结构
type Message struct {
	// 基础信息
	Type     MessageType     `json:"type"`               // 消息类型
	Content  string          `json:"content"`            // 文本内容
	Entities []MessageEntity `json:"entities,omitempty"` // 消息实体

	// 媒体内容
	Media *MediaContent `json:"media,omitempty"` // 媒体内容

	// 关系信息
	Reference *MessageReference `json:"reference,omitempty"` // 消息引用（回复、转发）

	// 位置信息（当Type=MessageTypeLocation时）
	Location *LocationInfo `json:"location,omitempty"`

	// 交互元素（键盘、按钮等）
	Keyboard *Keyboard `json:"keyboard,omitempty"`

	// 元数据
	Metadata MessageMetadata `json:"metadata"`
}
