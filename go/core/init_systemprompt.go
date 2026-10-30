package core

import (
	"bufio"
	"os"

	"github.com/sirupsen/logrus"
)

// Readsystemprompt 加载 AI 提示词模板。
//
// 提示词含 few-shot 示例，属业务资产，不打印正文以免示例内容进入 systemd journal。
// 读取循环先累加再判错：文件末尾无换行时 ReadString 会返回 (尾行, io.EOF)，
// 若先判错后累加会静默丢掉最后一行示例。
func Readsystemprompt() (str string) {
	file, err := os.Open("./go/config/systemPrompt.txt")
	if err != nil {
		logrus.Errorf("读取 systemPrompt 文件失败: %v", err)
		return
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	for {
		line, readErr := reader.ReadString('\n')
		str += line
		if readErr != nil {
			break
		}
	}

	logrus.Infof("systemPrompt 加载完成，共 %d 字符", len(str))
	return
}

// ReadsystempromptWuyin 加载吴饮识别提示词（systemPromptWuyin.txt）。
// 文件缺失只告警不阻断：吴饮端点在提示词为空时调用 AI 会得到差质量结果，
// 但不影响一点点链路；读取循环与 Readsystemprompt 同款（防尾行丢失）。
func ReadsystempromptWuyin() (str string) {
	file, err := os.Open("./go/config/systemPromptWuyin.txt")
	if err != nil {
		logrus.Errorf("读取 systemPromptWuyin 文件失败: %v", err)
		return
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	for {
		line, readErr := reader.ReadString('\n')
		str += line
		if readErr != nil {
			break
		}
	}

	logrus.Infof("systemPromptWuyin 加载完成，共 %d 字符", len(str))
	return
}
