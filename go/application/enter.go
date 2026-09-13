package application

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"order_server/go/global"
)

// ensureExportDir 确保导出目录存在
func ensureExportDir() error {
	dir := global.Config.SavePath.Excel
	if dir == "" {
		return fmt.Errorf("未配置导出目录 savePath.excel")
	}
	return os.MkdirAll(dir, 0755)
}

// uploadURL 把本地导出文件路径转换为 /uploads/ 下的下载地址
func uploadURL(path string) string {
	return "/uploads/" + url.PathEscape(filepath.Base(path))
}
