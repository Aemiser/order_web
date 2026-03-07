package order

import (
	"errors"
	"fmt"
	"order_server/go/global"
	"order_server/go/models"
	"order_server/go/models/enum"
	alog "order_server/go/server/log"
	ordertutil "order_server/go/utils/order"

	"github.com/sirupsen/logrus"
)

// 获取预览数据
func GetpreviewData() []ordertutil.ExceladdressInfo {
	if global.AddressList == nil {
		return nil
	}

	return ordertutil.AddressToExcelAddress(global.AddressList)
}

// 修改预览数据
func SetpreviewData(address ordertutil.ExceladdressInfo) (err error) {
	if global.AddressList == nil {
		return
	}

	for i, _ := range global.AddressList {
		if global.AddressList[i].PreviewID == address.PreviewID {
			// 对商品规格的处理
			fmt.Println(address.ProductSpecs)
			global.AddressList[i].ProductSpecs, err = ordertutil.Specprocessing(address.ProductSpecs)
			if err != nil {
				alog.Logaction(enum.LogErrLevel, "商品规格处理失败", err.Error())
				logrus.Errorf("商品规格处理失败: %v", err)
				return
			}
			global.AddressList[i].Recipient = address.Recipient
			global.AddressList[i].Phone = address.Phone
			global.AddressList[i].Address = address.Address
			global.AddressList[i].Remarks = address.Remarks
			return
		}
	}
	return
}

// 删除预览数据
func DeletepreviewData(id uint) error {
	if global.AddressList == nil {
		return errors.New("请先识别订单信息")
	}

	for i, _ := range global.AddressList {
		if global.AddressList[i].PreviewID == id {
			// 记录日志
			logtent := "删除订单：\n" + global.AddressList[i].Recipient + " " + global.AddressList[i].Phone + " " + global.AddressList[i].Address + "" + global.AddressList[i].ProductSpecsString(true)
			alog.Logaction(enum.LogInfoLevel, "删除预览订单", logtent)
			logrus.Infof("成功删除id为 %d 的订单", id)
			// 修改
			global.AddressList = append(global.AddressList[:i], global.AddressList[i+1:]...)
			return nil
		}
	}

	logtent := fmt.Sprintf("未找到id为 %d 的订单", id)
	alog.Logaction(enum.LogInfoLevel, "删除预览订单", logtent)
	return errors.New("未找到该订单")
}

// 导入数据库
func CreateOrder() error {
	if global.AddressList == nil {
		return errors.New("请先识别订单信息")
	}

	var orderlist []models.AddressInfo
	// 找到未入库的订单
	for i := range global.AddressList {
		if !global.AddressList[i].Isstorage {
			// 添加到待保存列表（值拷贝）
			orderlist = append(orderlist, global.AddressList[i])
			// 修改原切片的 Isstorage 字段为 true
			global.AddressList[i].Isstorage = true
		}
	}

	if len(orderlist) == 0 {
		return errors.New("没有需要入库的订单")
	}

	fmt.Println(orderlist)
	err := global.Db.Create(&orderlist).Error
	if err != nil {
		alog.Logaction(enum.LogErrLevel, "创建订单失败", err.Error())
		return err
	}

	alog.Logaction(enum.LogInfoLevel, "创建订单成功", "")
	return nil
}
