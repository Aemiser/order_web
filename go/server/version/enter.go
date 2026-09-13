package version

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"order_server/go/global"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"time"

	"github.com/sirupsen/logrus"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type GitHubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

type GitHubRelease struct {
	TagName    string        `json:"tag_name"`
	Name       string        `json:"name"`
	Prerelease bool          `json:"prerelease"`
	Assets     []GitHubAsset `json:"assets"`
}

var downloadedFilePath string

func getLatestRelease(repo string) (*GitHubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	return &release, nil
}

func CheckUpdate(ctx context.Context) {
	repo := global.Config.Version.Repo
	if repo == "" || repo == "your-username/your-repo" {
		return
	}

	release, err := getLatestRelease(repo)
	if err != nil {
		logrus.Warnf("版本检查失败: %v", err)
		return
	}

	if release.TagName != global.Version {
		wailsruntime.EventsEmit(ctx, "update_available", release.TagName, release.Name)
	}
}

func DownloadUpdate(ctx context.Context) {
	repo := global.Config.Version.Repo
	if repo == "" || repo == "your-username/your-repo" {
		wailsruntime.EventsEmit(ctx, "download_error", "未配置版本仓库")
		return
	}

	release, err := getLatestRelease(repo)
	if err != nil {
		msg := fmt.Sprintf("获取发布信息失败: %v", err)
		logrus.Errorf("版本更新下载: %s", msg)
		wailsruntime.EventsEmit(ctx, "download_error", msg)
		return
	}

	asset := findAsset(release.Assets)
	if asset == nil {
		msg := "未找到适用于当前系统的更新包"
		logrus.Errorf("版本更新下载: %s", msg)
		wailsruntime.EventsEmit(ctx, "download_error", msg)
		return
	}

	downloadPath := filepath.Join(os.TempDir(), asset.Name)
	logrus.Infof("开始下载更新: %s -> %s (%.2f MB)", asset.BrowserDownloadURL, downloadPath, float64(asset.Size)/1024/1024)

	resp, err := http.Get(asset.BrowserDownloadURL)
	if err != nil {
		msg := fmt.Sprintf("下载请求失败: %v", err)
		logrus.Errorf("版本更新下载: %s", msg)
		wailsruntime.EventsEmit(ctx, "download_error", msg)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("下载请求返回状态码 %d", resp.StatusCode)
		logrus.Errorf("版本更新下载: %s", msg)
		wailsruntime.EventsEmit(ctx, "download_error", msg)
		return
	}

	out, err := os.Create(downloadPath)
	if err != nil {
		msg := fmt.Sprintf("创建下载文件失败: %v", err)
		logrus.Errorf("版本更新下载: %s", msg)
		wailsruntime.EventsEmit(ctx, "download_error", msg)
		return
	}
	defer out.Close()

	totalSize := resp.ContentLength
	if totalSize <= 0 {
		totalSize = asset.Size
	}

	reader := &progressReader{
		reader:    resp.Body,
		total:     totalSize,
		ctx:       ctx,
		startTime: time.Now(),
	}
	reader.lastEmit = time.Now()

	_, err = io.Copy(out, reader)
	if err != nil {
		msg := fmt.Sprintf("下载写入失败: %v", err)
		logrus.Errorf("版本更新下载: %s", msg)
		wailsruntime.EventsEmit(ctx, "download_error", msg)
		return
	}

	downloadedFilePath = downloadPath
	elapsed := time.Since(reader.startTime).Seconds()
	speedMB := float64(reader.current) / elapsed / 1024 / 1024
	logrus.Infof("更新下载完成: %s (耗时 %.1fs, 平均速度 %.2f MB/s)", downloadPath, elapsed, speedMB)
	wailsruntime.EventsEmit(ctx, "download_complete", downloadPath)
}

func RestartApplication(ctx context.Context) {
	if downloadedFilePath == "" {
		wailsruntime.EventsEmit(ctx, "download_error", "没有已下载的更新文件")
		return
	}

	exePath, err := os.Executable()
	if err != nil {
		msg := fmt.Sprintf("获取当前程序路径失败: %v", err)
		logrus.Errorf("应用重启: %s", msg)
		wailsruntime.EventsEmit(ctx, "download_error", msg)
		return
	}

	batContent := fmt.Sprintf(`@echo off
timeout /t 2 /nobreak >nul
move /Y "%s" "%s"
start "" "%s"
del "%%~f0"
`, downloadedFilePath, exePath, exePath)

	batPath := filepath.Join(os.TempDir(), "order_server_update.bat")
	err = os.WriteFile(batPath, []byte(batContent), 0666)
	if err != nil {
		msg := fmt.Sprintf("创建重启脚本失败: %v", err)
		logrus.Errorf("应用重启: %s", msg)
		wailsruntime.EventsEmit(ctx, "download_error", msg)
		return
	}

	logrus.Infof("开始重启应用, 更新文件: %s -> %s", downloadedFilePath, exePath)
	cmd := exec.Command("cmd", "/c", batPath)
	cmd.Start()
	os.Exit(0)
}

func findAsset(assets []GitHubAsset) *GitHubAsset {
	for i := range assets {
		name := assets[i].Name
		if goruntime.GOOS == "windows" && filepath.Ext(name) == ".exe" {
			return &assets[i]
		}
		if goruntime.GOOS == "darwin" && (filepath.Ext(name) == ".app" || filepath.Ext(name) == ".dmg") {
			return &assets[i]
		}
		if goruntime.GOOS == "linux" && filepath.Ext(name) == "" && !hasExt(name) {
			return &assets[i]
		}
	}
	return nil
}

func hasExt(name string) bool {
	ext := filepath.Ext(name)
	return ext != "" && ext != name
}

type progressReader struct {
	reader    io.Reader
	total     int64
	current   int64
	ctx       context.Context
	startTime time.Time
	lastEmit  time.Time
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.current += int64(n)

	if time.Since(pr.lastEmit) > 200*time.Millisecond || pr.current >= pr.total {
		pr.lastEmit = time.Now()
		percentage := 0.0
		if pr.total > 0 {
			percentage = float64(pr.current) / float64(pr.total) * 100
		}
		elapsed := time.Since(pr.startTime).Seconds()
		speed := 0.0
		if elapsed > 0 {
			speed = float64(pr.current) / elapsed / 1024 / 1024
		}
		wailsruntime.EventsEmit(pr.ctx, "download_progress", pr.current, pr.total, percentage, speed)
	}

	return n, err
}
