package application

import (
	"net/url"
	"order_server/go/conf"
	"order_server/go/global"
	"os"
	"path/filepath"
	"testing"
)

func setupConfig(t *testing.T, exportDir string) {
	t.Helper()
	global.Config = &conf.Config{}
	global.Config.SavePath.Excel = exportDir
}

func TestUploadURLEscapesChineseFilename(t *testing.T) {
	name := "2026年9月13日_2026年9月20日结账.xlsx"
	got := uploadURL(filepath.Join(`C:\data\uploads`, name))
	want := "/uploads/" + url.PathEscape(name)
	if got != want {
		t.Fatalf("uploadURL = %q, want %q", got, want)
	}
}

func TestEnsureExportDirCreatesMissingDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "uploads")
	setupConfig(t, dir)

	if err := ensureExportDir(); err != nil {
		t.Fatalf("ensureExportDir: %v", err)
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Fatalf("目录未创建: %v", err)
	}
}

func TestEnsureExportDirRejectsEmptyConfig(t *testing.T) {
	setupConfig(t, "")

	if err := ensureExportDir(); err == nil {
		t.Fatal("空目录配置应返回错误")
	}
}
