package order_recognition

import (
	"fmt"
	"order_server/go/models"
	"order_server/go/utils/save"

	"github.com/sirupsen/logrus"
)

func SaveToExcel(addressList []models.AddressInfo) {
	// 保存为Excel文件
	excelFile, err := save.SaveToExcel(addressList, "address_result.xlsx")
	if err != nil {
		logrus.Errorf("保存Excel文件失败: %v", err)
	}
	fmt.Printf("Excel文件已保存: %s\n", excelFile)
}

func SaveToJSON(addressList []models.AddressInfo) {
	// 保存为JSON文件
	jsonFile, err := save.SaveToJSON(addressList, "address_result.json")
	if err != nil {
		logrus.Errorf("保存JSON文件失败: %v", err)
	}
	fmt.Printf("JSON文件已保存: %s\n", jsonFile)
}
