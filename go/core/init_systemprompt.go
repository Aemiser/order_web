package core

import (
	"bufio"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
)

func Readsystemprompt() (str string) {
	file, err := os.Open("./go/config/systemPrompt.txt")
	if err != nil {
		logrus.Errorf("读取 systemPrompt 文件失败: %v", err)
		return
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	for {
		line, err1 := reader.ReadString('\n')
		if err1 != nil {
			break
		}
		fmt.Print(line)
		str += line
	}

	fmt.Println(str)
	return
}
