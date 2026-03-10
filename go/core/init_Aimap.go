package core

import (
	"encoding/json"
	"fmt"
	"os"
)

func ReadAiMap() (mapps map[string]int) {
	ptjsonpath := "./aimap.json"
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
