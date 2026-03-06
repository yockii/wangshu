package feishu

import (
	"context"
	"os"
	"strings"

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
			FileType(fileType(file.Name())).
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

// fileType 根据文件名返回文件类型，飞书的文件类型可用值：opus音频（非opus的需要转为该格式）、mp4、pdf、doc、xls、ppt、stream（不属于以上类型）
func fileType(name string) string {
	switch {
	case strings.HasSuffix(name, ".opus"):
		return "opus"
	case strings.HasSuffix(name, ".mp4"):
		return "mp4"
	case strings.HasSuffix(name, ".pdf"):
		return "pdf"
	case strings.HasSuffix(name, ".doc"):
		return "doc"
	case strings.HasSuffix(name, ".xls"):
		return "xls"
	case strings.HasSuffix(name, ".ppt"):
		return "ppt"
	default:
		return "stream"
	}
}
