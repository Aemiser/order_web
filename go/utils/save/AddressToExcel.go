package save

import (
	"order_server/go/models"
	"strconv"
)

type ExceladdressInfo struct {
	Recipient    string `json:"收件人"`
	Phone        string `json:"手机号"`
	Address      string `json:"收货地址"`
	ProductSpecs string `json:"商品规格"`
}

func AddressToExcelAddress(addresss []models.AddressInfo) (exceladdress []ExceladdressInfo) {
	for _, address := range addresss {
		// 拼接商品规格消息
		productSpecs := ""
		for i, spec := range address.ProductSpecs {
			productSpecs += "【"
			if i == len(address.ProductSpecs)-1 {
				productSpecs += spec.Name + strconv.Itoa(spec.Weight) + "斤】"
				productSpecs += address.Remarks
				break
			}
			productSpecs += spec.Name + strconv.Itoa(spec.Weight) + "斤】,"
		}

		exceladdress = append(exceladdress, ExceladdressInfo{
			Address:      address.Address,
			Phone:        address.Phone,
			ProductSpecs: productSpecs,
			Recipient:    address.Recipient,
		})
	}
	return
}
