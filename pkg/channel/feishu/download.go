package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

func (c *FeishuChannel) donwloadFile(msgID, content string) (string, error) {
	fileStruct := struct {
		FileKey  string `json:"file_key"`
		FileName string `json:"file_name"`
	}{}
	if err := json.Unmarshal([]byte(content), &fileStruct); err != nil {
		return "", err
	}

	req := larkim.NewGetMessageResourceReqBuilder().
		MessageId(msgID).
		FileKey(fileStruct.FileKey).
		Type("file").
		Build()

	resp, err := c.restClient.Im.V1.MessageResource.Get(context.Background(), req)
	if err != nil {
		return "", err
	}
	if !resp.Success() {
		return "", fmt.Errorf("飞书渠道下载文件失败，code: %d, msg: %s", resp.Code, larkcore.Prettify(resp.CodeError))
	}

	path := filepath.Join(c.workspace, "download", fmt.Sprintf("%d-%s", time.Now().Unix(), fileStruct.FileName))

	os.MkdirAll(filepath.Dir(path), 0755)

	err = resp.WriteFile(path)
	if err != nil {
		return "", err
	}

	return path, nil
}

func (c *FeishuChannel) downloadImage(msgID, content string) (string, error) {
	imageStruct := struct {
		ImageKey string `json:"image_key"`
	}{}
	if err := json.Unmarshal([]byte(content), &imageStruct); err != nil {
		return "", err
	}

	req := larkim.NewGetMessageResourceReqBuilder().
		MessageId(msgID).
		FileKey(imageStruct.ImageKey).
		Type("image").
		Build()

	resp, err := c.restClient.Im.V1.MessageResource.Get(context.Background(), req)
	if err != nil {
		return "", err
	}
	if !resp.Success() {
		return "", fmt.Errorf("飞书渠道下载图片失败，code: %d, msg: %s", resp.Code, larkcore.Prettify(resp.CodeError))
	}

	suffix := ".jpg"
	contentType := resp.Header.Get("Content-Type")
	switch contentType {
	case "image/png":
		suffix = ".png"
	case "image/gif":
		suffix = ".gif"
	case "image/webp":
		suffix = ".webp"
	}

	path := filepath.Join(c.workspace, "download", fmt.Sprintf("%d%s", time.Now().Unix(), suffix))

	os.MkdirAll(filepath.Dir(path), 0755)

	err = resp.WriteFile(path)
	if err != nil {
		return "", err
	}

	return path, nil
}
