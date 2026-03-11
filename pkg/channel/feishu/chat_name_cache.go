package feishu

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	"github.com/yockii/wangshu/pkg/constant"
)

func (ch *FeishuChannel) loadChatNames() error {
	chatNameMapFile := filepath.Join(ch.workspace, constant.DirSessions, ch.name, constant.FileCachedChats)
	if _, err := os.Stat(chatNameMapFile); err != nil {
		if os.IsNotExist(err) {
			// 不存在，则创建
			if err = os.MkdirAll(filepath.Dir(chatNameMapFile), 0755); err != nil {
				return err
			}
			if err = os.WriteFile(chatNameMapFile, []byte("{}"), 0644); err != nil {
				return err
			}
		}
		return err
	}
	// 存在，则读取
	chatNames := make(map[string]string)
	data, err := os.ReadFile(chatNameMapFile)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(data, &chatNames); err != nil {
		return err
	}
	for chatID, name := range chatNames {
		ch.cachedChats.Store(chatID, name)
	}
	return nil
}

func (ch *FeishuChannel) getGroupChatName(chatID string) (string, error) {
	ch.cacheFileMu.Lock()
	defer ch.cacheFileMu.Unlock()
	if name, ok := ch.cachedChats.Load(chatID); ok {
		return name.(string), nil
	}
	// 不存在，则从飞书获取
	name, err := ch.getChatNameFromFeishu(chatID)
	if err != nil {
		return "", err
	}
	ch.cachedChats.Store(chatID, name)
	// 保存到文件
	m := make(map[string]string)
	ch.cachedChats.Range(func(key, value any) bool {
		m[key.(string)] = value.(string)
		return true
	})
	chatNameMapFile := filepath.Join(ch.workspace, constant.DirSessions, ch.name, constant.FileCachedChats)
	data, err := json.Marshal(m)
	if err != nil {
		return "", err
	}

	if err = os.WriteFile(chatNameMapFile, data, 0644); err != nil {
		return "", err
	}
	return name, nil
}

func (ch *FeishuChannel) getChatNameFromFeishu(chatID string) (string, error) {
	req := larkim.NewGetChatReqBuilder().ChatId(chatID).UserIdType(larkim.MemberIdTypeOpenId).Build()
	resp, err := ch.restClient.Im.V1.Chat.Get(context.Background(), req)
	if err != nil {
		return "", err
	}
	if !resp.Success() {
		return "", resp.CodeError
	}

	return *resp.Data.Name, nil
}

func (ch *FeishuChannel) saveP2pChatName(chatID, name string) error {
	ch.cacheFileMu.Lock()
	defer ch.cacheFileMu.Unlock()
	ch.cachedChats.Store(chatID, name)

	// 保存到文件
	m := make(map[string]string)
	ch.cachedChats.Range(func(key, value any) bool {
		m[key.(string)] = value.(string)
		return true
	})
	chatNameMapFile := filepath.Join(ch.workspace, constant.DirSessions, ch.name, constant.FileCachedChats)
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}

	if err = os.WriteFile(chatNameMapFile, data, 0644); err != nil {
		return err
	}
	return nil
}
