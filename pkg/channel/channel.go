package channel

import (
	"context"

	"github.com/yockii/wangshu/pkg/bus"
)

// Channel 渠道接口
type Channel interface {
	// 基础生命周期
	Start() error
	Stop() error
	GetName() string

	// 能力查询
	Capabilities() ChannelCapabilities
	Supports(capability ChannelCapability) bool

	// 消息发送
	SendMessage(ctx context.Context, msg *bus.Message) error
	SendText(ctx context.Context, chatID, text string) error
	SendMedia(ctx context.Context, chatID string, media *bus.MediaContent, caption string) error

	// 高级操作（如果Channel不支持，返回错误）
	EditMessage(ctx context.Context, chatID, messageID, content string) error
	DeleteMessage(ctx context.Context, chatID, messageID string) error
	PinMessage(ctx context.Context, chatID, messageID string) error
	UnpinMessage(ctx context.Context, chatID, messageID string) error

	// 消息交互
	SendKeyboard(ctx context.Context, chatID, text string, keyboard *bus.Keyboard) error
	AnswerCallback(ctx context.Context, callbackID, text string) error

	// 聊天操作
	GetChatInfo(ctx context.Context, chatID string) (*ChatInfo, error)
	// GetChatMembers(ctx context.Context, chatID string) ([]ChatMember, error)
}

// ChannelCapability 单个能力标识
type ChannelCapability int

const (
	// 发送能力
	CanSendText ChannelCapability = iota
	CanSendImage
	CanSendVideo
	CanSendAudio
	CanSendFile
	CanSendLocation
	CanSendSticker
	CanSendRichMedia
	CanSendKeyboard

	// 接收能力
	CanReceiveText
	CanReceiveImage
	CanReceiveVideo
	CanReceiveAudio
	CanReceiveFile
	CanReceiveLocation
	CanReceiveSticker

	// 消息操作能力
	CanEditMessage
	CanDeleteMessage
	CanPinMessage
	CanReplyMessage
	CanForwardMessage
	CanMentionUsers // @用户
	CanMentionAll   // @全体成员

	// 聊天能力
	CanGetChatInfo
	CanGetMembers
	CanKickMembers
	CanInviteMembers

	// 连接方式
	SupportsWebhook
	SupportsPolling
	SupportsStreaming
)

// ChannelCapabilities 渠道能力集合
type ChannelCapabilities struct {
	// 发送能力
	CanSendText      bool
	CanSendImage     bool
	CanSendVideo     bool
	CanSendAudio     bool
	CanSendFile      bool
	CanSendLocation  bool
	CanSendSticker   bool
	CanSendRichMedia bool
	CanSendKeyboard  bool // 支持交互键盘

	// 接收能力
	CanReceiveText     bool
	CanReceiveImage    bool
	CanReceiveVideo    bool
	CanReceiveAudio    bool
	CanReceiveFile     bool
	CanReceiveLocation bool
	CanReceiveSticker  bool

	// 消息操作能力
	CanEditMessage    bool
	CanDeleteMessage  bool
	CanPinMessage     bool
	CanReplyMessage   bool
	CanForwardMessage bool
	CanMentionUsers   bool // @用户
	CanMentionAll     bool // @全体成员

	// 聊天能力
	CanGetChatInfo   bool
	CanGetMembers    bool
	CanKickMembers   bool
	CanInviteMembers bool

	// 连接方式
	SupportsWebhook   bool
	SupportsPolling   bool
	SupportsStreaming bool
}

// ChatType 聊天类型
type ChatType string

const (
	ChatTypePrivate    ChatType = "private"    // 私聊
	ChatTypeGroup      ChatType = "group"      // 群聊
	ChatTypeChannel    ChatType = "channel"    // 频道
	ChatTypeSupergroup ChatType = "supergroup" // 超级群组
)

// ChatInfo 聊天信息
type ChatInfo struct {
	ID          string   `json:"id"`
	Type        ChatType `json:"type"`
	Title       string   `json:"title,omitempty"`
	Description string   `json:"description,omitempty"`
	MemberCount int      `json:"member_count,omitempty"`
	CreatedAt   int64    `json:"created_at,omitempty"`
}

// MemberRole 成员角色
type MemberRole string

const (
	MemberRoleOwner  MemberRole = "owner"
	MemberRoleAdmin  MemberRole = "admin"
	MemberRoleMember MemberRole = "member"
)

// ChatMember 聊天成员
type ChatMember struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	DisplayName string     `json:"display_name,omitempty"`
	Role        MemberRole `json:"role"`
	JoinedAt    int64      `json:"joined_at,omitempty"`
}
