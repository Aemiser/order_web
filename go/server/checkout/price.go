package checkout

import (
	"errors"
	"fmt"

	"order_server/go/global"
	"order_server/go/models"
	"order_server/go/models/enum"
	alog "order_server/go/server/log"
	ordertutil "order_server/go/utils/order"

	"github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// 价格持久化（契约 §3.6/§3.7，裁定 RUL-1）：price_configs 表按 productName+spec
// 唯一 upsert，只保留当前生效价，不保留历史；UpdatedAt（GORM 自动维护）即最近生效时间。

// isDuplicateKeyErr 判断是否 MySQL 唯一键冲突（1062）。
// 用于 Create 撞 uk_priceconfig_product_spec（并发保存兜底）时降级为 Updates 重试。
func isDuplicateKeyErr(err error) bool {
	var myErr *mysql.MySQLError
	return errors.As(err, &myErr) && myErr.Number == 1062
}

// SavePriceConfig 单事务保存价格配置：逐项按 (productName, spec) upsert——
// 查到则 Updates 覆盖 pricePerKg/pricePerItem（RUL-1 不保留历史），查不到则 Create；
// Create 撞唯一键（1062，并发兜底）降级重试一次 Updates；全部成功才提交，任一失败整体回滚
// （契约 §3.6 行为变更 1/4，失败由调用方转 50000）。
// pricePerItem==0 时先按 pricePerKg×斤数 推导再持久化，保证库中回显与推导值一致（行为变更 2）。
func SavePriceConfig(list []ProPriceRequest) error {
	if len(list) == 0 {
		return errors.New("价格列表不能为空")
	}

	err := global.Db.Transaction(func(tx *gorm.DB) error {
		for i := range list {
			item := list[i]
			pricePerItem := item.PricePerItem
			if pricePerItem == 0 {
				// 沿用既有推导规则：元/斤 × 斤数 = 元/件
				pricePerItem = item.PricePerKg * float64(ordertutil.GetNumberInSpec(item.Spec))
			}

			var pc models.PriceConfig
			err := tx.Where("product_name = ? AND spec = ?", item.ProductName, item.Spec).
				First(&pc).Error
			if err == nil {
				// 更新键为数据库实际列名（B3.2）；GORM 对 Updates 自动刷新 UpdatedAt（=最近生效时间）
				if e := tx.Model(&pc).Updates(map[string]interface{}{
					"price_per_kg":   item.PricePerKg,
					"price_per_item": pricePerItem,
				}).Error; e != nil {
					return fmt.Errorf("更新 %s %s 价格失败: %w", item.ProductName, item.Spec, e)
				}
				continue
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("查询 %s %s 价格失败: %w", item.ProductName, item.Spec, err)
			}

			// 新建；撞唯一索引（并发兜底）降级重试一次 Updates
			pc = models.PriceConfig{
				ProductName:  item.ProductName,
				Spec:         item.Spec,
				PricePerKg:   item.PricePerKg,
				PricePerItem: pricePerItem,
			}
			if e := tx.Create(&pc).Error; e != nil {
				if !isDuplicateKeyErr(e) {
					return fmt.Errorf("新建 %s %s 价格失败: %w", item.ProductName, item.Spec, e)
				}
				if e2 := tx.Model(&models.PriceConfig{}).
					Where("product_name = ? AND spec = ?", item.ProductName, item.Spec).
					Updates(map[string]interface{}{
						"price_per_kg":   item.PricePerKg,
						"price_per_item": pricePerItem,
					}).Error; e2 != nil {
					return fmt.Errorf("唯一键冲突降级更新 %s %s 价格失败: %w", item.ProductName, item.Spec, e2)
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	// T4：金额类写库留痕（契约 §3.6 行为变更 5）
	alog.Logaction(enum.LogInfoLevel, "保存价格配置", fmt.Sprintf("共 %d 条", len(list)))
	return nil
}

// LoadPriceMap 加载全部当前生效价格，返回 map[商品名]map[规格]PriceConfig。
// 读取失败时记日志并返回空 map：价格回显降级为 0（未配置态），不阻塞结账查询
// （契约 §3.7 行为变更 3——daydata 现状不返回 50000，Q6 登记本轮不新增）。
func LoadPriceMap() map[string]map[string]models.PriceConfig {
	m := make(map[string]map[string]models.PriceConfig)
	var list []models.PriceConfig
	if err := global.Db.Find(&list).Error; err != nil {
		logrus.Errorf("加载价格配置失败: %v", err)
		return m
	}
	for _, p := range list {
		if m[p.ProductName] == nil {
			m[p.ProductName] = make(map[string]models.PriceConfig)
		}
		m[p.ProductName][p.Spec] = p
	}
	return m
}

// applyPrices 按 商品名+规格 从 priceMap 填充 resp.ProSPecs 的价格（匹配不到保持 0，
// 契约 §3.7 行为变更 3），并直接计算三个价格行——与 TotalRow 同序同长
// （按 ProSPecs 展开顺序：外层商品序、内层规格序）；pricePerItem 为 0 时先按
// pricePerKg×斤数 推导；TotalAmountRow[i] = TotalRow[i] × PricePerItemRow[i]（契约 §3.6/§3.7）。
func applyPrices(resp *CheckoutResponse, priceMap map[string]map[string]models.PriceConfig) {
	if resp == nil {
		return
	}
	var pricePerKgRow, pricePerItemRow, totalAmountRow []float64
	for i := range resp.ProSPecs {
		for j := range resp.ProSPecs[i].Specs {
			spec := &resp.ProSPecs[i].Specs[j]
			if pc, ok := priceMap[resp.ProSPecs[i].Name][spec.Name]; ok {
				spec.PricePerKg = pc.PricePerKg
				spec.PricePerItem = pc.PricePerItem
			}
			pricePerItem := spec.PricePerItem
			if pricePerItem == 0 {
				pricePerItem = spec.PricePerKg * float64(ordertutil.GetNumberInSpec(spec.Name))
			}
			idx := len(pricePerKgRow)
			pricePerKgRow = append(pricePerKgRow, spec.PricePerKg)
			pricePerItemRow = append(pricePerItemRow, pricePerItem)
			var total int
			if idx < len(resp.TotalRow) { // 防御：行序不一致时按 0 计，不越界
				total = resp.TotalRow[idx]
			}
			totalAmountRow = append(totalAmountRow, float64(total)*pricePerItem)
		}
	}
	resp.PricePerKgRow = pricePerKgRow
	resp.PricePerItemRow = pricePerItemRow
	resp.TotalAmountRow = totalAmountRow
}
