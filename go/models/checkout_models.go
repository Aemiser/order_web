package models

// PriceConfig 商品当前生效价格配置（表 price_configs，契约 §1.3）。
// 裁定（2026-10-17，RUL-1）：只保留当前生效价，不保留历史，按 productName+spec 唯一 upsert；
// UpdatedAt（继承自 Model）即「最近生效时间」，不单设生效时间列。
//
// 列名一律显式 column: 前缀且小写蛇形（B3.2.1/B3.3）。契约 §1.3 示例 tag 中的
// 无前缀段（如 gorm:"productName"）不指定列名，实际列按 GORM 默认蛇形命名生成，
// 故本模型实际列 product_name/spec/price_per_kg/price_per_item 与契约示例行为一致。
// productName 必须显式 size:191——无 size 的 string 列 GORM 生成 longtext，
// 无法进入唯一索引；索引长度 191×4+64×4=1020 字节 < InnoDB utf8mb4 3072 字节上限。
// 已知限制（契约 §1.3 登记）：软删行占用唯一索引，本轮无删除端点，仅 upsert。
type PriceConfig struct {
	Model
	ProductName  string  `gorm:"column:product_name;size:191;uniqueIndex:uk_priceconfig_product_spec,priority:1" json:"productName"`
	Spec         string  `gorm:"column:spec;size:64;uniqueIndex:uk_priceconfig_product_spec,priority:2" json:"spec"` // 如 "5斤"
	PricePerKg   float64 `gorm:"column:price_per_kg" json:"pricePerKg"`                                               // 元/斤
	PricePerItem float64 `gorm:"column:price_per_item" json:"pricePerItem"`                                           // 元/件
}
