package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	larkauth "github.com/larksuite/oapi-sdk-go/v3/service/auth/v3"
	larkcontact "github.com/larksuite/oapi-sdk-go/v3/service/contact/v3"
)

// getBotOpenID 获取机器人的OpenID
func (c *FeishuChannel) getBotOpenID() error {
	req := larkauth.NewInternalTenantAccessTokenReqBuilder().
		Body(larkauth.NewInternalTenantAccessTokenReqBodyBuilder().
			AppId(c.appID).
			AppSecret(c.appSecret).
			Build()).
		Build()
	resp, err := c.restClient.Auth.V3.TenantAccessToken.Internal(context.Background(), req)
	if err != nil {
		return fmt.Errorf("failed to get token: %v", err)
	}
	if !resp.Success() {
		return fmt.Errorf("failed to get token: %v", resp.CodeError)
	}
	tat := struct {
		Code              int    `json:"code"`
		Msg               string `json:"msg"`
		Expire            int    `json:"expire"`
		TenantAccessToken string `json:"tenant_access_token"`
	}{}
	if err := json.Unmarshal(resp.RawBody, &tat); err != nil {
		return fmt.Errorf("failed to unmarshal token: %v", err)
	}
	token := tat.TenantAccessToken
	if token == "" {
		return fmt.Errorf("tenant_access_token is empty")
	}
	// 获取机器人信息
	httpReq, err := http.NewRequest(http.MethodGet, "https://open.feishu.cn/open-apis/bot/v3/info", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	httpReq.Header.Set("Content-Type", "application/json")
	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer httpResp.Body.Close()
	if httpResp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to get bot info: %v", httpResp.Status)
	}
	bodyBytes, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}
	botInfo := struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Bot  struct {
			ActivateStatus int    `json:"activate_status"`
			AppName        string `json:"app_name"`
			OpenID         string `json:"open_id"`
		} `json:"bot"`
	}{}
	if err := json.Unmarshal(bodyBytes, &botInfo); err != nil {
		return fmt.Errorf("failed to unmarshal bot info: %v", err)
	}
	c.openID = botInfo.Bot.OpenID
	if c.openID == "" {
		return fmt.Errorf("open_id is empty")
	}
	c.channelStatus = botInfo.Bot.ActivateStatus
	if c.channelStatus != 2 {
		return fmt.Errorf("bot is not activated")
	}
	return nil
}

// getSenderName 获取发送者的名称
func (c *FeishuChannel) getSenderName(openID string) string {
	if val, ok := c.cachedUsers.Load(openID); ok {
		return val.(string)
	}

	// 如果没有，调用sdk查询
	if name, err := c.getMemberName(openID); err != nil {
		slog.Error("Feishu Channel getSenderName error", "err", err)
		return ""
	} else {
		c.cachedUsers.Store(openID, name)
		go func() {
			// 保存到文件
			if err := c.saveUsersInfoToCacheFile(); err != nil {
				slog.Warn("Failed to save group users to file", "error", err)
			}
		}()
		return name
	}
}

func (c *FeishuChannel) getMemberName(openID string) (string, error) {
	req := larkcontact.NewGetUserReqBuilder().UserId(openID).UserIdType(larkcontact.UserIdTypeOpenId).Build()
	resp, err := c.restClient.Contact.V3.User.Get(context.Background(), req)
	if err != nil {
		return "", fmt.Errorf("failed to get user: %v", err)
	}
	if !resp.Success() {
		return "", fmt.Errorf("failed to get user: %v", resp.CodeError)
	}
	if resp.Data.User != nil && resp.Data.User.Name != nil {
		return *resp.Data.User.Name, nil
	}
	return "", fmt.Errorf("user name is empty")
}

// // getAllGroupMembers 获取群聊的所有成员
// func (c *FeishuChannel) getAllGroupMembers(chatID string, pageToken string, result map[string]string) error {
// 	req := larkim.NewGetChatMembersReqBuilder().
// 		ChatId(chatID).
// 		MemberIdType("open_id").
// 		PageSize(100).
// 		PageToken(pageToken).
// 		Build()
// 	resp, err := c.restClient.Im.V1.ChatMembers.Get(context.Background(), req)
// 	if err != nil {
// 		slog.Error("Fetch Feishu Group Member Failed", "error", err)
// 		return err
// 	}

// 	if !resp.Success() {
// 		slog.Error("Feishu Channel getSenderName error", "requestId", resp.RequestId(), "response", larkcore.Prettify(resp.CodeError))
// 		return resp.CodeError
// 	}

// 	// 遍历成员列表
// 	for _, member := range resp.Data.Items {
// 		if member.MemberId != nil && member.Name != nil {
// 			openID := *member.MemberId
// 			result[openID] = *member.Name
// 		}
// 	}

// 	if resp.Data.HasMore != nil && *resp.Data.HasMore && resp.Data.PageToken != nil && *resp.Data.PageToken != "" {
// 		return c.getAllGroupMembers(chatID, *resp.Data.PageToken, result)
// 	}
// 	return nil
// }
