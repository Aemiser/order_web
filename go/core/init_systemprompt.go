package core

import (
	"bufio"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
)

func Readsystemprompt() (str string) {
	file, err := os.Open("systemPrompt.txt")
	if err != nil {
		logrus.Errorf("Error:", err)
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
