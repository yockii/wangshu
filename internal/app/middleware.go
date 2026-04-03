package app

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/yockii/wangshu/internal/config"
)

func localModelMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/live2d/") {
			next.ServeHTTP(w, r)
			return
		}

		// 提取相对路径，如 /live2d/hiyori/model.model3.json -> hiyori/model.model3.json
		relPath := strings.TrimPrefix(r.URL.Path, "/live2d/")
		relPath = filepath.Clean(relPath) // 防止 ../ 目录穿透攻击

		fullPath := filepath.Join(config.DefaultCfg.Live2D.ModelDir, relPath)

		if config.DefaultCfg.Live2D.ModelDir == "" || !strings.HasPrefix(fullPath, filepath.Clean(config.DefaultCfg.Live2D.ModelDir)+string(os.PathSeparator)) {
			http.Error(w, "Model directory not set or access denied", http.StatusForbidden)
			return
		}
		data, err := os.ReadFile(fullPath)
		if err != nil {
			if os.IsNotExist(err) {
				http.NotFound(w, r)
			} else {
				http.Error(w, "Failed to read file", http.StatusInternalServerError)
			}
			return
		}
		// 设置Content-Type
		switch filepath.Ext(fullPath) {
		case ".json":
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
		case ".moc3", ".mtn":
			w.Header().Set("Content-Type", "application/octet-stream")
		case ".png", ".jpg", ".jpeg", ".webp":
			// 图片可以不手动设置，让浏览器自动判断
		default:
			w.Header().Set("Content-Type", "application/octet-stream")
		}
		w.Write(data)
	})
}
