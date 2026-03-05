package feishu

import (
	"context"
	"os"

	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

// uploadImage 上传图片到飞书
func (c *FeishuChannel) uploadImage(p string) (string, error) {
	file, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer file.Close()
	req := larkim.NewCreateImageReqBuilder().
		Body(larkim.NewCreateImageReqBodyBuilder().
			ImageType(`message`).
			Image(file).
			Build()).
		Build()
	resp, err := c.restClient.Im.V1.Image.Create(context.Background(), req)
	if err != nil {
		return "", err
	}
	if !resp.Success() {
		return "", resp.CodeError
	}
	return *resp.Data.ImageKey, nil
}

// uploadFile 上传文件到飞书
func (c *FeishuChannel) uploadFile(p string) (string, error) {
	file, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer file.Close()
	req := larkim.NewCreateFileReqBuilder().
		Body(larkim.NewCreateFileReqBodyBuilder().
			FileName(file.Name()).
			File(file).
			Build()).
		Build()
	resp, err := c.restClient.Im.V1.File.Create(context.Background(), req)
	if err != nil {
		return "", err
	}
	if !resp.Success() {
		return "", resp.CodeError
	}
	return *resp.Data.FileKey, nil
}
