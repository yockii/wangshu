package action

import (
	"context"
	"fmt"

	"github.com/yockii/wangshu/pkg/action/types"
	"github.com/yockii/wangshu/pkg/constant"
	"github.com/yockii/wangshu/pkg/tools"
)

type ToolFunc func(ctx context.Context, params map[string]any) (*types.ActionOutput, error)

var toolMapper = make(map[string]ToolFunc)

func commonParamsExtract(po map[string]any) map[string]string {
	pr := make(map[string]string)
	if _, ok := po[constant.ToolCallParamAgentName]; ok {
		pr[constant.ToolCallParamAgentName] = po[constant.ToolCallParamAgentName].(string)
	}
	if _, ok := po[constant.ToolCallParamWorkspace]; ok {
		pr[constant.ToolCallParamWorkspace] = po[constant.ToolCallParamWorkspace].(string)
	}
	if _, ok := po[constant.ToolCallParamChannel]; ok {
		pr[constant.ToolCallParamChannel] = po[constant.ToolCallParamChannel].(string)
	}
	if _, ok := po[constant.ToolCallParamChatID]; ok {
		pr[constant.ToolCallParamChatID] = po[constant.ToolCallParamChatID].(string)
	}
	if _, ok := po[constant.ToolCallParamSenderID]; ok {
		pr[constant.ToolCallParamSenderID] = po[constant.ToolCallParamSenderID].(string)
	}
	if _, ok := po[constant.ToolCallParamChatType]; ok {
		pr[constant.ToolCallParamChatType] = po[constant.ToolCallParamChatType].(string)
	}
	if _, ok := po[constant.ToolCallParamTaskID]; ok {
		pr[constant.ToolCallParamTaskID] = po[constant.ToolCallParamTaskID].(string)
	}
	return pr
}

func init() {
	toolMapper["time.now"] = func(ctx context.Context, params map[string]any) (*types.ActionOutput, error) {
		pr := commonParamsExtract(params)
		tr := tools.GetDefaultToolRegistry().Execute(ctx, constant.ToolNameCurrentTime, pr)
		if tr.Err != nil {
			return tr.Structured, tr.Err
		}
		return tr.Structured, nil
	}
	toolMapper["time.sleep"] = func(ctx context.Context, params map[string]any) (*types.ActionOutput, error) {
		pr := commonParamsExtract(params)
		if _, ok := params["seconds"]; !ok {
			return nil, fmt.Errorf("seconds is required")
		}
		pr["seconds"] = params["seconds"].(string)
		tr := tools.GetDefaultToolRegistry().Execute(ctx, constant.ToolNameSleep, pr)
		if tr.Err != nil {
			return tr.Structured, tr.Err
		}
		return tr.Structured, nil
	}
	// 文件系统相关
	{
		toolMapper["fs.read"] = func(ctx context.Context, params map[string]any) (*types.ActionOutput, error) {
			pr := commonParamsExtract(params)
			if _, ok := params["path"]; !ok {
				return nil, fmt.Errorf("path is required")
			}
			pr["path"] = params["path"].(string)
			tr := tools.GetDefaultToolRegistry().Execute(ctx, constant.ToolNameFSRead, pr)
			if tr.Err != nil {
				return tr.Structured, tr.Err
			}
			return tr.Structured, nil
		}
		toolMapper["fs.write"] = func(ctx context.Context, params map[string]any) (*types.ActionOutput, error) {
			pr := commonParamsExtract(params)
			if _, ok := params["path"]; !ok {
				return nil, fmt.Errorf("path is required")
			}
			pr["path"] = params["path"].(string)
			if _, ok := params["content"]; !ok {
				return nil, fmt.Errorf("content is required")
			}
			pr["content"] = params["content"].(string)
			if p, ok := params["append"]; ok {
				pr["append"] = p.(string)
			}
			tr := tools.GetDefaultToolRegistry().Execute(ctx, constant.ToolNameFSWrite, pr)
			if tr.Err != nil {
				return tr.Structured, tr.Err
			}
			return tr.Structured, nil
		}
		toolMapper["fs.list"] = func(ctx context.Context, params map[string]any) (*types.ActionOutput, error) {
			pr := commonParamsExtract(params)
			if _, ok := params["path"]; !ok {
				return nil, fmt.Errorf("path is required")
			}
			pr["path"] = params["path"].(string)
			tr := tools.GetDefaultToolRegistry().Execute(ctx, constant.ToolNameFSList, pr)
			if tr.Err != nil {
				return tr.Structured, tr.Err
			}
			return tr.Structured, nil
		}
		toolMapper["fs.copy"] = func(ctx context.Context, params map[string]any) (*types.ActionOutput, error) {
			pr := commonParamsExtract(params)
			if _, ok := params["src"]; !ok {
				return nil, fmt.Errorf("src is required")
			}
			pr["source_path"] = params["src"].(string)
			if _, ok := params["dest"]; !ok {
				return nil, fmt.Errorf("dest is required")
			}
			pr["target_path"] = params["dest"].(string)
			if _, ok := params["overwrite"]; ok {
				pr["overwrite"] = params["overwrite"].(string)
			}
			tr := tools.GetDefaultToolRegistry().Execute(ctx, constant.ToolNameFSList, pr)
			if tr.Err != nil {
				return tr.Structured, tr.Err
			}
			return tr.Structured, nil
		}
		toolMapper["fs.move"] = func(ctx context.Context, params map[string]any) (*types.ActionOutput, error) {
			pr := commonParamsExtract(params)
			if _, ok := params["old_path"]; !ok {
				return nil, fmt.Errorf("old_path is required")
			}
			pr["old_path"] = params["old_path"].(string)
			if _, ok := params["new_path"]; !ok {
				return nil, fmt.Errorf("new_path is required")
			}
			pr["new_path"] = params["new_path"].(string)
			tr := tools.GetDefaultToolRegistry().Execute(ctx, constant.ToolNameFSMove, pr)
			if tr.Err != nil {
				return tr.Structured, tr.Err
			}
			return tr.Structured, nil
		}
		toolMapper["fs.delete"] = func(ctx context.Context, params map[string]any) (*types.ActionOutput, error) {
			pr := commonParamsExtract(params)
			if _, ok := params["path"]; !ok {
				return nil, fmt.Errorf("path is required")
			}
			pr["path"] = params["path"].(string)
			tr := tools.GetDefaultToolRegistry().Execute(ctx, constant.ToolNameFSDelete, pr)
			if tr.Err != nil {
				return tr.Structured, tr.Err
			}
			return tr.Structured, nil
		}
		toolMapper["fs.edit"] = func(ctx context.Context, params map[string]any) (*types.ActionOutput, error) {
			pr := commonParamsExtract(params)
			if _, ok := params["file_path"]; !ok {
				return nil, fmt.Errorf("file_path is required")
			}
			pr["file_path"] = params["file_path"].(string)
			if _, ok := params["old_str"]; !ok {
				return nil, fmt.Errorf("old_str is required")
			}
			pr["old_str"] = params["old_str"].(string)
			if _, ok := params["new_str"]; !ok {
				return nil, fmt.Errorf("new_str is required")
			}
			pr["new_str"] = params["new_str"].(string)
			tr := tools.GetDefaultToolRegistry().Execute(ctx, constant.ToolNameFsEdit, pr)
			if tr.Err != nil {
				return tr.Structured, tr.Err
			}
			return tr.Structured, nil
		}
		toolMapper["fs.search"] = func(ctx context.Context, params map[string]any) (*types.ActionOutput, error) {
			pr := commonParamsExtract(params)
			if _, ok := params["pattern"]; !ok {
				return nil, fmt.Errorf("pattern is required")
			}
			pr["pattern"] = params["pattern"].(string)
			tr := tools.GetDefaultToolRegistry().Execute(ctx, constant.ToolNameFSSearch, pr)
			if tr.Err != nil {
				return tr.Structured, tr.Err
			}
			return tr.Structured, nil
		}
		toolMapper["text.search"] = func(ctx context.Context, params map[string]any) (*types.ActionOutput, error) {
			pr := commonParamsExtract(params)
			if _, ok := params["pattern"]; !ok {
				return nil, fmt.Errorf("pattern is required")
			}
			pr["pattern"] = params["pattern"].(string)
			if p, ok := params["path"]; ok {
				pr["path"] = p.(string)
			}
			if p, ok := params["include"]; ok {
				pr["include"] = p.(string)
			}
			tr := tools.GetDefaultToolRegistry().Execute(ctx, constant.ToolNameGrepFile, pr)
			if tr.Err != nil {
				return tr.Structured, tr.Err
			}
			return tr.Structured, nil
		}
	}
	// 网络相关
	{
		toolMapper["web.search"] = func(ctx context.Context, params map[string]any) (*types.ActionOutput, error) {
			pr := commonParamsExtract(params)
			if _, ok := params["query"]; !ok {
				return nil, fmt.Errorf("query is required")
			}
			pr["query"] = params["query"].(string)
			if p, ok := params["num_results"]; ok {
				pr["num_results"] = p.(string)
			}
			if p, ok := params["engine"]; ok {
				pr["engine"] = p.(string)
			}
			tr := tools.GetDefaultToolRegistry().Execute(ctx, constant.ToolNameWebSearch, pr)
			if tr.Err != nil {
				return tr.Structured, tr.Err
			}
			return tr.Structured, nil
		}
		toolMapper["web.fetch"] = func(ctx context.Context, params map[string]any) (*types.ActionOutput, error) {
			pr := commonParamsExtract(params)
			if _, ok := params["url"]; !ok {
				return nil, fmt.Errorf("url is required")
			}
			pr["url"] = params["url"].(string)

			// timeout
			if p, ok := params["timeout"]; ok {
				pr["timeout"] = p.(string)
			}
			// raw
			if p, ok := params["raw"]; ok {
				pr["raw"] = p.(string)
			}

			tr := tools.GetDefaultToolRegistry().Execute(ctx, constant.ToolNameWebFetch, pr)
			if tr.Err != nil {
				return tr.Structured, tr.Err
			}
			return tr.Structured, nil
		}
	}
	// 浏览器相关
	{
		toolMapper["browser.open"] = func(ctx context.Context, params map[string]any) (*types.ActionOutput, error) {
			pr := commonParamsExtract(params)
			if _, ok := params["url"]; !ok {
				return nil, fmt.Errorf("url is required")
			}
			pr["url"] = params["url"].(string)
			pr["action"] = "open"
			tr := tools.GetDefaultToolRegistry().Execute(ctx, constant.ToolNameBrowser, pr)
			if tr.Err != nil {
				return tr.Structured, tr.Err
			}
			return tr.Structured, nil
		}
		toolMapper["browser.click"] = func(ctx context.Context, params map[string]any) (*types.ActionOutput, error) {
			pr := commonParamsExtract(params)
			if _, ok := params["selector"]; !ok {
				return nil, fmt.Errorf("selector is required")
			}
			pr["selector"] = params["selector"].(string)
			if p, ok := params["timeout"]; ok {
				pr["timeout"] = p.(string)
			}

			pr["action"] = "click"
			tr := tools.GetDefaultToolRegistry().Execute(ctx, constant.ToolNameBrowser, pr)
			if tr.Err != nil {
				return tr.Structured, tr.Err
			}
			return tr.Structured, nil
		}
		toolMapper["browser.fill"] = func(ctx context.Context, params map[string]any) (*types.ActionOutput, error) {
			pr := commonParamsExtract(params)
			if _, ok := params["selector"]; !ok {
				return nil, fmt.Errorf("selector is required")
			}
			pr["selector"] = params["selector"].(string)
			if _, ok := params["text"]; !ok {
				return nil, fmt.Errorf("text is required")
			}
			pr["text"] = params["text"].(string)

			if p, ok := params["timeout"]; ok {
				pr["timeout"] = p.(string)
			}
			pr["action"] = "fill"
			tr := tools.GetDefaultToolRegistry().Execute(ctx, constant.ToolNameBrowser, pr)
			if tr.Err != nil {
				return tr.Structured, tr.Err
			}
			return tr.Structured, nil
		}
		toolMapper["browser.html"] = func(ctx context.Context, params map[string]any) (*types.ActionOutput, error) {
			pr := commonParamsExtract(params)
			if p, ok := params["format"]; ok {
				pr["format"] = p.(string)
			}
			if p, ok := params["start"]; ok {
				pr["start"] = p.(string)
			}
			if p, ok := params["max_length"]; ok {
				pr["max_length"] = p.(string)
			}
			pr["action"] = "html"
			tr := tools.GetDefaultToolRegistry().Execute(ctx, constant.ToolNameBrowser, pr)
			if tr.Err != nil {
				return tr.Structured, tr.Err
			}
			return tr.Structured, nil
		}
		toolMapper["browser.screenshot"] = func(ctx context.Context, params map[string]any) (*types.ActionOutput, error) {
			pr := commonParamsExtract(params)
			if p, ok := params["screenshot_path"]; ok {
				pr["screenshot_path"] = p.(string)
			}

			pr["action"] = "screenshot"
			tr := tools.GetDefaultToolRegistry().Execute(ctx, constant.ToolNameBrowser, pr)
			if tr.Err != nil {
				return tr.Structured, tr.Err
			}
			return tr.Structured, nil
		}
		toolMapper["browser.wait"] = func(ctx context.Context, params map[string]any) (*types.ActionOutput, error) {
			pr := commonParamsExtract(params)
			if _, ok := params["selector"]; !ok {
				return nil, fmt.Errorf("selector is required")
			}
			pr["selector"] = params["selector"].(string)

			if p, ok := params["timeout"]; ok {
				pr["timeout"] = p.(string)
			}

			pr["action"] = "wait"
			tr := tools.GetDefaultToolRegistry().Execute(ctx, constant.ToolNameBrowser, pr)
			if tr.Err != nil {
				return tr.Structured, tr.Err
			}
			return tr.Structured, nil
		}
		toolMapper["browser.close"] = func(ctx context.Context, params map[string]any) (*types.ActionOutput, error) {
			pr := commonParamsExtract(params)

			pr["action"] = "close"
			tr := tools.GetDefaultToolRegistry().Execute(ctx, constant.ToolNameBrowser, pr)
			if tr.Err != nil {
				return tr.Structured, tr.Err
			}
			return tr.Structured, nil
		}
	}
	toolMapper["message.send"] = func(ctx context.Context, params map[string]any) (*types.ActionOutput, error) {
		pr := commonParamsExtract(params)
		if _, ok := params["content"]; !ok {
			return nil, fmt.Errorf("content is required")
		}
		pr["content"] = params["content"].(string)
		if p, ok := params["fileType"]; ok {
			pr["fileType"] = p.(string)

			if p, ok := params["filePath"]; ok {
				pr["filePath"] = p.(string)
			} else {
				return nil, fmt.Errorf("filePath is required when fileType is set")
			}
		}

		tr := tools.GetDefaultToolRegistry().Execute(ctx, constant.ToolNameMessage, pr)
		if tr.Err != nil {
			return tr.Structured, tr.Err
		}
		return tr.Structured, nil
	}
}
