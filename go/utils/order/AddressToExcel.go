package ordertutil

import (
	"order_server/go/models"
)

type ExceladdressInfo struct {
	PreviewID    uint   `json:"VID"`
	Recipient    string `json:"收件人"`
	Phone        string `json:"手机号"`
	Address      string `json:"收货地址"`
	ProductSpecs string `json:"商品规格"`
	Remarks      string `json:"备注"`
}

func AddressToExcelAddress(addresss []models.AddressInfo) (exceladdress []ExceladdressInfo) {
	for _, address := range addresss {
		exceladdress = append(exceladdress, ExceladdressInfo{
			PreviewID:    address.PreviewID,
			Address:      address.Address,
			Phone:        address.Phone,
			ProductSpecs: address.ProductSpecsString(false),
			Recipient:    address.Recipient,
			Remarks:      address.Remarks,
		})
	}
	return
}
