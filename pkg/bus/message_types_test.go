package bus

import (
	"encoding/json"
	"testing"
	"time"
)

func TestMessageTypeValues(t *testing.T) {
	// 测试所有消息类型常量是否正确定义
	tests := []struct {
		name  string
		value MessageType
	}{
		{"Text", MessageTypeText},
		{"Image", MessageTypeImage},
		{"Video", MessageTypeVideo},
		{"Audio", MessageTypeAudio},
		{"Voice", MessageTypeVoice},
		{"File", MessageTypeFile},
		{"Location", MessageTypeLocation},
		{"Sticker", MessageTypeSticker},
		{"RichMedia", MessageTypeRichMedia},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.value) == "" {
				t.Errorf("MessageType %s should not be empty", tt.name)
			}
		})
	}
}

func TestEntityTypeValues(t *testing.T) {
	// 测试所有实体类型常量是否正确定义
	tests := []struct {
		name  string
		value EntityType
	}{
		{"Mention", EntityTypeMention},
		{"MentionAll", EntityTypeMentionAll},
		{"Link", EntityTypeLink},
		{"Hashtag", EntityTypeHashtag},
		{"BotCommand", EntityTypeBotCommand},
		{"Email", EntityTypeEmail},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.value) == "" {
				t.Errorf("EntityType %s should not be empty", tt.name)
			}
		})
	}
}

func TestMessageEntity(t *testing.T) {
	// 测试消息实体创建和序列化
	entity := MessageEntity{
		Type:   EntityTypeMention,
		Offset: 0,
		Length: 8,
		UserID: "user123",
	}

	// 测试JSON序列化
	data, err := json.Marshal(entity)
	if err != nil {
		t.Fatalf("Failed to marshal MessageEntity: %v", err)
	}

	var decoded MessageEntity
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal MessageEntity: %v", err)
	}

	if decoded.Type != EntityTypeMention {
		t.Errorf("Expected type %s, got %s", EntityTypeMention, decoded.Type)
	}

	if decoded.UserID != "user123" {
		t.Errorf("Expected UserID user123, got %s", decoded.UserID)
	}
}

func TestMessageEntityMentionAll(t *testing.T) {
	// 测试@全体成员的实体
	entity := MessageEntity{
		Type:    EntityTypeMentionAll,
		Offset:  0,
		Length:  3,
		Mention: "all",
	}

	data, err := json.Marshal(entity)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded MessageEntity
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Type != EntityTypeMentionAll {
		t.Errorf("Expected type %s, got %s", EntityTypeMentionAll, decoded.Type)
	}
}

func TestReferenceTypeValues(t *testing.T) {
	// 测试引用类型常量
	tests := []struct {
		name  string
		value ReferenceType
	}{
		{"Reply", ReferenceTypeReply},
		{"Forward", ReferenceTypeForward},
		{"Quote", ReferenceTypeQuote},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.value) == "" {
				t.Errorf("ReferenceType %s should not be empty", tt.name)
			}
		})
	}
}

func TestMessageReference(t *testing.T) {
	// 测试消息引用
	ref := MessageReference{
		MessageID:     "msg123",
		ChatID:        "chat456",
		SenderID:      "user789",
		SenderName:    "张三",
		Content:       "这是一条被引用的消息",
		MessageType:   MessageTypeText,
		ReferenceType: ReferenceTypeReply,
	}

	// 测试JSON序列化
	data, err := json.Marshal(ref)
	if err != nil {
		t.Fatalf("Failed to marshal MessageReference: %v", err)
	}

	var decoded MessageReference
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal MessageReference: %v", err)
	}

	if decoded.MessageID != "msg123" {
		t.Errorf("Expected MessageID msg123, got %s", decoded.MessageID)
	}

	if decoded.ReferenceType != ReferenceTypeReply {
		t.Errorf("Expected ReferenceType %s, got %s", ReferenceTypeReply, decoded.ReferenceType)
	}
}

func TestMediaTypeValues(t *testing.T) {
	// 测试媒体类型常量
	tests := []struct {
		name  string
		value MediaType
	}{
		{"Image", MediaTypeImage},
		{"Video", MediaTypeVideo},
		{"Audio", MediaTypeAudio},
		{"File", MediaTypeFile},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.value) == "" {
				t.Errorf("MediaType %s should not be empty", tt.name)
			}
		})
	}
}

func TestMediaContent(t *testing.T) {
	// 测试媒体内容
	media := &MediaContent{
		Type:      MediaTypeImage,
		URL:       "https://example.com/image.jpg",
		FilePath:  "/tmp/image.jpg",
		Thumbnail: "https://example.com/thumb.jpg",
		Size:      102400,
		Width:     1920,
		Height:    1080,
		FileName:  "image.jpg",
		MimeType:  "image/jpeg",
	}

	// 测试JSON序列化
	data, err := json.Marshal(media)
	if err != nil {
		t.Fatalf("Failed to marshal MediaContent: %v", err)
	}

	var decoded MediaContent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal MediaContent: %v", err)
	}

	if decoded.Type != MediaTypeImage {
		t.Errorf("Expected type %s, got %s", MediaTypeImage, decoded.Type)
	}

	if decoded.Width != 1920 {
		t.Errorf("Expected Width 1920, got %d", decoded.Width)
	}

	if decoded.Size != 102400 {
		t.Errorf("Expected Size 102400, got %d", decoded.Size)
	}
}

func TestMediaContentWithDuration(t *testing.T) {
	// 测试带时长的媒体（音频/视频）
	media := &MediaContent{
		Type:     MediaTypeVideo,
		URL:      "https://example.com/video.mp4",
		Size:     5120000,
		Duration: 120, // 2分钟
		Width:    1280,
		Height:   720,
		FileName: "video.mp4",
		MimeType: "video/mp4",
	}

	data, err := json.Marshal(media)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded MediaContent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Duration != 120 {
		t.Errorf("Expected Duration 120, got %d", decoded.Duration)
	}
}

func TestLocationInfo(t *testing.T) {
	// 测试位置信息
	loc := &LocationInfo{
		Latitude:  39.9042,
		Longitude: 116.4074,
		Title:     "北京市天安门广场",
		Address:   "北京市东城区长安街",
	}

	data, err := json.Marshal(loc)
	if err != nil {
		t.Fatalf("Failed to marshal LocationInfo: %v", err)
	}

	var decoded LocationInfo
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal LocationInfo: %v", err)
	}

	if decoded.Title != "北京市天安门广场" {
		t.Errorf("Expected Title '北京市天安门广场', got %s", decoded.Title)
	}

	// 测试经纬度精度（允许小误差）
	if decoded.Latitude < 39.9041 || decoded.Latitude > 39.9043 {
		t.Errorf("Latitude out of expected range: %f", decoded.Latitude)
	}
}

func TestKeyboard(t *testing.T) {
	// 测试键盘
	keyboard := &Keyboard{
		Inline:     true,
		Resizeable: true,
		OneTime:    false,
		Rows: []KeyboardRow{
			{
				Buttons: []KeyboardButton{
					{
						Text: "是",
						Data: "yes",
					},
					{
						Text: "否",
						Data: "no",
					},
				},
			},
			{
				Buttons: []KeyboardButton{
					{
						Text: "取消",
						Data: "cancel",
					},
				},
			},
		},
	}

	data, err := json.Marshal(keyboard)
	if err != nil {
		t.Fatalf("Failed to marshal Keyboard: %v", err)
	}

	var decoded Keyboard
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal Keyboard: %v", err)
	}

	if !decoded.Inline {
		t.Error("Expected Inline to be true")
	}

	if len(decoded.Rows) != 2 {
		t.Errorf("Expected 2 rows, got %d", len(decoded.Rows))
	}

	if len(decoded.Rows[0].Buttons) != 2 {
		t.Errorf("Expected 2 buttons in first row, got %d", len(decoded.Rows[0].Buttons))
	}

	if decoded.Rows[0].Buttons[0].Text != "是" {
		t.Errorf("Expected button text '是', got %s", decoded.Rows[0].Buttons[0].Text)
	}
}

func TestKeyboardButtonWithURL(t *testing.T) {
	// 测试带URL的按钮
	button := KeyboardButton{
		Text: "打开网站",
		URL:  "https://example.com",
	}

	data, err := json.Marshal(button)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded KeyboardButton
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.URL != "https://example.com" {
		t.Errorf("Expected URL 'https://example.com', got %s", decoded.URL)
	}
}

func TestKeyboardButtonWithRequest(t *testing.T) {
	// 测试请求联系人/位置的按钮
	button := KeyboardButton{
		Text:            "分享联系人",
		RequestContact:  true,
	}

	data, err := json.Marshal(button)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded KeyboardButton
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if !decoded.RequestContact {
		t.Error("Expected RequestContact to be true")
	}
}

func TestMessageForwardInfo(t *testing.T) {
	// 测试转发信息
	forwardInfo := &MessageForwardInfo{
		FromChatID:      "original_chat",
		FromMessageID:   "original_msg",
		FromSenderName:  "李四",
		FromSenderID:    "user999",
	}

	data, err := json.Marshal(forwardInfo)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded MessageForwardInfo
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.FromSenderName != "李四" {
		t.Errorf("Expected FromSenderName '李四', got %s", decoded.FromSenderName)
	}
}

func TestMessageMetadata(t *testing.T) {
	// 测试消息元数据
	now := time.Now()
	editTime := now.Add(1 * time.Minute)

	metadata := MessageMetadata{
		MessageID:     "msg_123",
		SenderID:      "user_456",
		SenderName:    "王五",
		ChatID:        "chat_789",
		Timestamp:     now,
		EditTimestamp: &editTime,
		ReplyToID:     "msg_prev",
		Pinned:        true,
		Views:         100,
		Forwards:      5,
		Channel:       "feishu",
	}

	data, err := json.Marshal(metadata)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded MessageMetadata
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.MessageID != "msg_123" {
		t.Errorf("Expected MessageID 'msg_123', got %s", decoded.MessageID)
	}

	if !decoded.Pinned {
		t.Error("Expected Pinned to be true")
	}

	if decoded.Views != 100 {
		t.Errorf("Expected Views 100, got %d", decoded.Views)
	}
}

func TestMessageText(t *testing.T) {
	// 测试文本消息
	msg := &Message{
		Type:    MessageTypeText,
		Content: "这是一条测试消息",
		Metadata: MessageMetadata{
			MessageID: "msg_001",
			SenderID:  "user_001",
			ChatID:    "chat_001",
			Timestamp: time.Now(),
			Channel:   "test",
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Type != MessageTypeText {
		t.Errorf("Expected type %s, got %s", MessageTypeText, decoded.Type)
	}

	if decoded.Content != "这是一条测试消息" {
		t.Errorf("Expected content '这是一条测试消息', got %s", decoded.Content)
	}
}

func TestMessageWithEntities(t *testing.T) {
	// 测试带实体的消息（@用户、链接等）
	msg := &Message{
		Type:    MessageTypeText,
		Content: "@张三 你好，请访问 https://example.com",
		Entities: []MessageEntity{
			{
				Type:   EntityTypeMention,
				Offset: 0,
				Length: 7,
				UserID: "user_zhangsan",
			},
			{
				Type:   EntityTypeLink,
				Offset: 14,
				Length: 18,
				URL:    "https://example.com",
			},
		},
		Metadata: MessageMetadata{
			MessageID: "msg_002",
			SenderID:  "user_002",
			ChatID:    "chat_002",
			Timestamp: time.Now(),
			Channel:   "test",
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(decoded.Entities) != 2 {
		t.Errorf("Expected 2 entities, got %d", len(decoded.Entities))
	}

	if decoded.Entities[0].Type != EntityTypeMention {
		t.Errorf("Expected first entity type %s, got %s", EntityTypeMention, decoded.Entities[0].Type)
	}
}

func TestMessageWithMedia(t *testing.T) {
	// 测试带媒体的消息
	msg := &Message{
		Type:    MessageTypeImage,
		Content: "这是一张图片",
		Media: &MediaContent{
			Type:      MediaTypeImage,
			URL:       "https://example.com/photo.jpg",
			Width:     1920,
			Height:    1080,
			FileName:  "photo.jpg",
			MimeType:  "image/jpeg",
		},
		Metadata: MessageMetadata{
			MessageID: "msg_003",
			SenderID:  "user_003",
			ChatID:    "chat_003",
			Timestamp: time.Now(),
			Channel:   "test",
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Media == nil {
		t.Fatal("Media should not be nil")
	}

	if decoded.Media.Type != MediaTypeImage {
		t.Errorf("Expected media type %s, got %s", MediaTypeImage, decoded.Media.Type)
	}
}

func TestMessageWithReference(t *testing.T) {
	// 测试带引用的消息（回复）
	msg := &Message{
		Type:    MessageTypeText,
		Content: "我同意你的观点",
		Reference: &MessageReference{
			MessageID:     "msg_original",
			ChatID:        "chat_001",
			SenderID:      "user_original",
			SenderName:    "李四",
			Content:       "原消息内容",
			ReferenceType: ReferenceTypeReply,
		},
		Metadata: MessageMetadata{
			MessageID: "msg_004",
			SenderID:  "user_004",
			ChatID:    "chat_001",
			Timestamp: time.Now(),
			Channel:   "test",
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Reference == nil {
		t.Fatal("Reference should not be nil")
	}

	if decoded.Reference.ReferenceType != ReferenceTypeReply {
		t.Errorf("Expected reference type %s, got %s", ReferenceTypeReply, decoded.Reference.ReferenceType)
	}
}

func TestMessageWithKeyboard(t *testing.T) {
	// 测试带键盘的消息
	msg := &Message{
		Type:    MessageTypeText,
		Content: "请选择一个选项",
		Keyboard: &Keyboard{
			Inline: true,
			Rows: []KeyboardRow{
				{
					Buttons: []KeyboardButton{
						{Text: "选项A", Data: "A"},
						{Text: "选项B", Data: "B"},
					},
				},
			},
		},
		Metadata: MessageMetadata{
			MessageID: "msg_005",
			SenderID:  "user_005",
			ChatID:    "chat_005",
			Timestamp: time.Now(),
			Channel:   "test",
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Keyboard == nil {
		t.Fatal("Keyboard should not be nil")
	}

	if len(decoded.Keyboard.Rows) != 1 {
		t.Errorf("Expected 1 keyboard row, got %d", len(decoded.Keyboard.Rows))
	}
}

func TestMessageLocation(t *testing.T) {
	// 测试位置消息
	msg := &Message{
		Type:     MessageTypeLocation,
		Location: &LocationInfo{
			Latitude:  31.2304,
			Longitude: 121.4737,
			Title:     "上海外滩",
			Address:   "上海市黄浦区中山东一路",
		},
		Metadata: MessageMetadata{
			MessageID: "msg_006",
			SenderID:  "user_006",
			ChatID:    "chat_006",
			Timestamp: time.Now(),
			Channel:   "test",
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Location == nil {
		t.Fatal("Location should not be nil")
	}

	if decoded.Location.Title != "上海外滩" {
		t.Errorf("Expected location title '上海外滩', got %s", decoded.Location.Title)
	}
}

func TestNewOutboundMessage(t *testing.T) {
	// 测试辅助函数
	msg := NewOutboundMessage("chat_123", "测试消息")

	if msg.Type != MessageTypeText {
		t.Errorf("Expected type %s, got %s", MessageTypeText, msg.Type)
	}

	if msg.Content != "测试消息" {
		t.Errorf("Expected content '测试消息', got %s", msg.Content)
	}

	if msg.Metadata.ChatID != "chat_123" {
		t.Errorf("Expected ChatID 'chat_123', got %s", msg.Metadata.ChatID)
	}
}

func TestComplexMessage(t *testing.T) {
	// 测试复杂的消息（包含多个特性）
	now := time.Now()
	msg := &Message{
		Type:    MessageTypeText,
		Content: "@全体成员 请查看这条消息，包含图片和投票",
		Entities: []MessageEntity{
			{
				Type:    EntityTypeMentionAll,
				Offset:  0,
				Length:  5,
			},
		},
		Media: &MediaContent{
			Type:      MediaTypeImage,
			URL:       "https://example.com/image.png",
			Thumbnail: "https://example.com/thumb.png",
		},
		Keyboard: &Keyboard{
			Inline: true,
			Rows: []KeyboardRow{
				{
					Buttons: []KeyboardButton{
						{Text: "同意", Data: "agree"},
						{Text: "反对", Data: "disagree"},
					},
				},
			},
		},
		Metadata: MessageMetadata{
			MessageID:   "complex_msg",
			SenderID:    "bot",
			SenderName:  "助手",
			ChatID:      "group_001",
			Timestamp:   now,
			Pinned:      false,
			Views:       0,
			Forwards:    0,
			Channel:     "test",
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal complex message: %v", err)
	}

	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal complex message: %v", err)
	}

	// 验证所有字段
	if decoded.Type != MessageTypeText {
		t.Errorf("Type mismatch")
	}

	if len(decoded.Entities) != 1 {
		t.Errorf("Expected 1 entity, got %d", len(decoded.Entities))
	}

	if decoded.Media == nil {
		t.Error("Media should not be nil")
	}

	if decoded.Keyboard == nil {
		t.Error("Keyboard should not be nil")
	}

	if decoded.Metadata.SenderName != "助手" {
		t.Errorf("Expected sender name '助手', got %s", decoded.Metadata.SenderName)
	}
}
