package wuyin

import (
	"errors"
	"fmt"
	"order_server/go/global"
	"order_server/go/models"
	"order_server/go/models/enum"
	alog "order_server/go/server/log"
	ordertutil "order_server/go/utils/order"
	wuyinutil "order_server/go/utils/wuyin"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// ---- 预览数据（内存态 global.WuyinAddressList），与一点点 loaddb.go 同构 ----

// GetWuyinPreview 获取预览数据
func GetWuyinPreview() []wuyinutil.WuyinExceladdressInfo {
	if global.WuyinAddressList == nil {
		return nil
	}
	return wuyinutil.WuyinToExcelAddress(global.WuyinAddressList)
}

// SetWuyinPreview 修改一条预览数据（规格字符串经 SpecItemsProcessing 解析为件计价）
func SetWuyinPreview(address wuyinutil.WuyinExceladdressInfo) (err error) {
	if global.WuyinAddressList == nil {
		return
	}
	for i := range global.WuyinAddressList {
		if global.WuyinAddressList[i].PreviewID == address.PreviewID {
			global.WuyinAddressList[i].ProductSpecs, err = wuyinutil.SpecItemsProcessing(address.ProductSpecs)
			if err != nil {
				logrus.Errorf("吴饮商品规格处理失败: %v", err)
				return fmt.Errorf("商品规格解析失败: %w", err)
			}
			global.WuyinAddressList[i].Recipient = address.Recipient
			global.WuyinAddressList[i].Phone = address.Phone
			global.WuyinAddressList[i].Address = address.Address
			global.WuyinAddressList[i].Remarks = address.Remarks
			global.WuyinAddressList[i].Tag = address.Tag
			global.WuyinAddressList[i].Sender = address.Sender
			return
		}
	}
	return
}

// DeleteWuyinPreview 删除一条预览数据
func DeleteWuyinPreview(id uint) error {
	if global.WuyinAddressList == nil {
		return errors.New("请先识别订单信息")
	}
	for i := range global.WuyinAddressList {
		if global.WuyinAddressList[i].PreviewID == id {
			logtent := "删除吴饮预览订单：\n" + global.WuyinAddressList[i].Recipient + " " +
				global.WuyinAddressList[i].Phone + " " + global.WuyinAddressList[i].ProductSpecsString(true)
			alog.Logaction(enum.LogInfoLevel, "删除吴饮预览订单", logtent)
			global.WuyinAddressList = append(global.WuyinAddressList[:i], global.WuyinAddressList[i+1:]...)
			return nil
		}
	}
	return errors.New("未找到该订单")
}

// CreateWuyinOrder 预览数据导入独立库（与一点点 CreateOrder 同构：Isstorage 防重复入库，
// 非今日订单改写 CreatedAt 归账）
func CreateWuyinOrder(date time.Time) error {
	if global.WuyinAddressList == nil {
		return errors.New("请先识别订单信息")
	}

	if date.Format("2006-01-02") != time.Now().Format("2006-01-02") {
		for i := range global.WuyinAddressList {
			global.WuyinAddressList[i].CreatedAt = date
			global.WuyinAddressList[i].UpdatedAt = date
		}
	}

	var orderlist []models.WuyinAddressInfo
	for i := range global.WuyinAddressList {
		if !global.WuyinAddressList[i].Isstorage {
			orderlist = append(orderlist, global.WuyinAddressList[i])
			global.WuyinAddressList[i].Isstorage = true
		}
	}
	if len(orderlist) == 0 {
		return errors.New("没有需要入库的订单")
	}

	if err := global.DbWuyin.Create(&orderlist).Error; err != nil {
		alog.Logaction(enum.LogErrLevel, "吴饮创建订单失败", err.Error())
		return err
	}
	alog.Logaction(enum.LogInfoLevel, "吴饮创建订单成功", fmt.Sprintf("共 %d 条", len(orderlist)))
	return nil
}

// UpdateWuyinOrderRequest 修改历史订单入参（指针字段区分未传/清空，同一点点 UpdateOrderRequest）
type UpdateWuyinOrderRequest struct {
	Recipient    *string `json:"收件人"`
	Phone        *string `json:"手机号"`
	Address      *string `json:"收货地址"`
	Province     *string `json:"省份"`
	ProductSpecs *string `json:"商品规格"`
	Remarks      *string `json:"备注"`
	Tag          *string `json:"标签"`
}

// UpdateWuyinOrder 修改一条吴饮历史订单（软删作用域内 First 校验存在性）
func UpdateWuyinOrder(id uint, req UpdateWuyinOrderRequest) error {
	var addr models.WuyinAddressInfo
	if err := global.DbWuyin.First(&addr, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("订单不存在")
		}
		return fmt.Errorf("查询失败: %w", err)
	}

	updates := map[string]interface{}{}
	if req.Recipient != nil {
		updates["recipient"] = *req.Recipient
	}
	if req.Phone != nil {
		updates["phone"] = *req.Phone
	}
	if req.Address != nil {
		updates["address"] = *req.Address
		updates["province"] = ordertutil.ExtractProvince(*req.Address)
	}
	if req.Province != nil {
		updates["province"] = *req.Province
	}
	if req.Remarks != nil {
		updates["remarks"] = *req.Remarks
	}
	if req.Tag != nil {
		updates["tag"] = *req.Tag
	}
	if req.ProductSpecs != nil {
		specs, err := wuyinutil.SpecItemsProcessing(*req.ProductSpecs)
		if err != nil {
			return fmt.Errorf("商品规格解析失败: %w", err)
		}
		updates["product_specs"] = specs
	}

	if len(updates) == 0 {
		return errors.New("至少提供一项需要修改的字段")
	}

	if err := global.DbWuyin.Model(&models.WuyinAddressInfo{}).
		Where("id = ?", id).
		Updates(updates).Error; err != nil {
		alog.Logaction(enum.LogErrLevel, "吴饮修改订单失败", fmt.Sprintf("id=%d: %s", id, err.Error()))
		return fmt.Errorf("修改失败: %w", err)
	}
	alog.Logaction(enum.LogInfoLevel, "吴饮修改订单", fmt.Sprintf("id=%d", id))
	return nil
}

// DeleteWuyinOrder 软删除一条吴饮历史订单
func DeleteWuyinOrder(id uint) error {
	var addr models.WuyinAddressInfo
	if err := global.DbWuyin.First(&addr, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("订单不存在")
		}
		return fmt.Errorf("查询失败: %w", err)
	}
	if err := global.DbWuyin.Delete(&addr).Error; err != nil {
		return fmt.Errorf("删除失败: %w", err)
	}
	alog.Logaction(enum.LogInfoLevel, "吴饮删除订单", fmt.Sprintf("id=%d 收件人=%s tag=%s", id, addr.Recipient, addr.Tag))
	return nil
}
