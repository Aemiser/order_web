package order

import (
	"errors"
	"fmt"
	"order_server/go/global"
	"order_server/go/models"
	"order_server/go/models/enum"
	alog "order_server/go/server/log"
	ordertutil "order_server/go/utils/order"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
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
func CreateOrder(date time.Time) error {
	if global.AddressList == nil {
		return errors.New("请先识别订单信息")
	}

	// 不为今天的订单修改其日期
	if date.Format("2006-01-02") != time.Now().Format("2006-01-02") {
		for i := range global.AddressList {
			global.AddressList[i].CreatedAt = date
			global.AddressList[i].UpdatedAt = date
		}
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

// BackfillProvince 扫描 province 为空的存量行，从收货地址提取省份回填（契约 §1.5，RUL-2）。
// 幂等：仅命中 province IS NULL OR province='' 的行（GORM 对含 DeletedAt 的模型自动追加
// deleted_at IS NULL，已软删行不被扫描回填），已回填行重复执行不再命中；无法识别的行保持空
// （正常态，留待人工修正——地址被修正后重跑 -db 可再尝试）。
// 事务边界：逐行独立 UPDATE，不包整体事务（允许部分成功，失败行记日志继续，契约 §1.5.1）。
func BackfillProvince() (updated int, remaining int, err error) {
	var list []models.AddressInfo
	err = global.Db.Select("id", "address").
		Where("province IS NULL OR province = ''").
		Find(&list).Error
	if err != nil {
		return 0, 0, err
	}
	for _, row := range list {
		p := ordertutil.ExtractProvince(row.Address)
		if p == "" {
			remaining++ // 无法识别，保持空，留待人工修正（改单端点可补省份）
			continue
		}
		if e := global.Db.Model(&models.AddressInfo{}).
			Where("id = ?", row.ID).
			Update("province", p).Error; e != nil {
			logrus.Errorf("回填省份失败 id=%d: %v", row.ID, e)
			remaining++
			continue
		}
		updated++
	}
	alog.Logaction(enum.LogInfoLevel, "省份存量回填",
		fmt.Sprintf("扫描 %d 行，回填 %d 行，仍为空 %d 行", len(list), updated, remaining))
	return updated, remaining, nil
}

// UpdateOrderRequest 修改历史订单入参（契约 §3.3）。中文键与 ExceladdressInfo 一致；
// 指针字段区分「未传」（字段缺失/null，不更新）与「空串」（清空该字段）两种语义；
// 商品规格为字符串形态（与 SetPreview 同解析器同格式），服务端 Specprocessing 解析后整体替换。
type UpdateOrderRequest struct {
	Recipient    *string `json:"收件人"`
	Phone        *string `json:"手机号"`
	Address      *string `json:"收货地址"`
	Province     *string `json:"省份"`
	ProductSpecs *string `json:"商品规格"`
	Remarks      *string `json:"备注"`
}

// UpdateOrder 修改一条历史订单（契约 §3.3）。T4：First 存在性校验 + 留痕（含操作来源 IP，M3）。
// 逐字段指针非空才更新（空串=清空）；收货地址变更自动 ExtractProvince 重提取省份（先派生），
// 同请求显式传入省份则优先覆盖派生值；商品规格经 Specprocessing 解析为 ProductSpecsJSON
// 后整体替换，空串/解析失败返回错误（规格不可清空只能替换）；
// created_at 不可通过本端点修改（契约 §3.3 其他约定，影响账单归属）。
// clientIP 由 handler 经 c.ClientIP() → application 透传，仅用于留痕。
func UpdateOrder(id uint, req UpdateOrderRequest, clientIP string) error {
	var addr models.AddressInfo
	if err := global.Db.First(&addr, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("订单不存在")
		}
		return fmt.Errorf("查询失败: %w", err)
	}

	// updates 键为数据库实际列名（B3.2：product_specs 为蛇形实际列名，见 backend-architecture B3.2.1 存量登记）
	updates := map[string]interface{}{}
	if req.Recipient != nil {
		updates["recipient"] = *req.Recipient
	}
	if req.Phone != nil {
		updates["phone"] = *req.Phone
	}
	if req.Address != nil {
		updates["address"] = *req.Address
		updates["province"] = ordertutil.ExtractProvince(*req.Address) // 地址变更先派生省份
	}
	if req.Province != nil {
		updates["province"] = *req.Province // 显式省份优先，覆盖派生值（契约 §3.3 处理顺序：先派生后覆盖）
	}
	if req.Remarks != nil {
		updates["remarks"] = *req.Remarks
	}
	if req.ProductSpecs != nil {
		specs, err := ordertutil.Specprocessing(*req.ProductSpecs)
		if err != nil {
			// 含空串场景：Specprocessing 对空串返回「没有找到匹配的内容」，即契约
			// 错误码表的「商品规格解析失败: 没有找到匹配的内容」（规格不可清空只能替换）
			return fmt.Errorf("商品规格解析失败: %w", err)
		}
		updates["product_specs"] = specs
	}

	if err := global.Db.Model(&models.AddressInfo{}).
		Where("id = ?", id).
		Updates(updates).Error; err != nil {
		alog.Logaction(enum.LogErrLevel, "修改订单失败", fmt.Sprintf("id=%d: %s", id, err.Error()))
		return fmt.Errorf("修改失败: %w", err)
	}

	// 留痕记录修改后的收件人与手机号（未传字段保留库中原值，契约 §3.3 T4），
	// 含操作来源 IP（M3：定位操作发起方，防抵赖）
	recipient, phone := addr.Recipient, addr.Phone
	if req.Recipient != nil {
		recipient = *req.Recipient
	}
	if req.Phone != nil {
		phone = *req.Phone
	}
	alog.Logaction(enum.LogInfoLevel, "修改订单",
		fmt.Sprintf("id=%d 收件人=%s 手机号=%s 操作IP=%s", id, recipient, phone, clientIP))
	return nil
}

// DeleteOrder 软删除一条历史订单（契约 §3.4）。GORM Delete 生成
// UPDATE address_infos SET deleted_at=NOW() WHERE id=?，禁止 Unscoped 物理删除（B3.5）。
// T4：First 存在性校验 + 留痕（含操作来源 IP，M3）。重复删除时 First 自动过滤已软删行 →
// 返回「订单不存在」（非幂等，契约 §2 幂等性总览）；删除后该行自动从
// query/daydata/province-stats 消失（GORM 软删作用域）。
// clientIP 由 handler 经 c.ClientIP() → application 透传，仅用于留痕。
func DeleteOrder(id uint, clientIP string) error {
	var addr models.AddressInfo
	if err := global.Db.First(&addr, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("订单不存在")
		}
		return fmt.Errorf("查询失败: %w", err)
	}

	if err := global.Db.Delete(&addr).Error; err != nil {
		alog.Logaction(enum.LogErrLevel, "删除订单失败", fmt.Sprintf("id=%d: %s", id, err.Error()))
		return fmt.Errorf("删除失败: %w", err)
	}

	alog.Logaction(enum.LogInfoLevel, "删除订单",
		fmt.Sprintf("id=%d 收件人=%s 手机号=%s 地址=%s 操作IP=%s", id, addr.Recipient, addr.Phone, addr.Address, clientIP))
	return nil
}
