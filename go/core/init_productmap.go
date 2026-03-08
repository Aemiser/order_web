package core

import (
	"encoding/json"
	"fmt"
	"os"
)

func ReadProductBelongMap() (mapps map[string][]string) {
	ptjsonpath := "./productbelong.json"
	f, err := os.Open(ptjsonpath)
	if err != nil {
		fmt.Println("open file err = ", err)
		return
	}

	defer f.Close()

	decoder := json.NewDecoder(f)
	err = decoder.Decode(&mapps)
	if err != nil {
		fmt.Printf("json decode has error:%v\n", err)
	}
	return
}

func WriteProductBelongMap(dataMap map[string][]string) {
	jsonBytes, err := json.MarshalIndent(dataMap, "", "    ")
	if err != nil {
		fmt.Printf("Map转换为JSON字符串失败，异常: %s\n", err)
		return
	}
	// 写进文件
	err = os.WriteFile("./productbelong.json", jsonBytes, 0644)
	if err != nil {
		fmt.Printf("写入JSON文件失败，异常: %s\n", err)
		return
	}
	fmt.Printf("Map转换为JSON字符串成功: %s\n", string(jsonBytes))
}
