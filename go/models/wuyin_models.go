package models

import (
	"database/sql/driver"
	"encoding/json"
	"strconv"
)

// 吴饮（吾饮良品）品牌链路的数据模型。与一点点（AddressInfo）完全隔离：
// 独立数据库（global.DbWuyin）、按件计价、多仓库标签维度。

// WuyinProductSpec 商品规格（斤存储）。口径修正（2026-10-30）：商品规格字段一律
// 以"斤"为单位（同一点点；吾饮标准件 9斤/件，"两件"记为两个 9 斤条目）；
// "件"只在结账计数时体现——每个规格条目即 1 件，结账按条目数计件、按 元/件 计价。
type WuyinProductSpec struct {
	Name   string `json:"name"`   // 商品名称（归一后，如"香水柠檬"）
	Weight int    `json:"weight"` // 斤数（每个条目 = 1 件，如 9）
}

// Value 实现 driver.Valuer，写入数据库
func (ps WuyinProductSpec) Value() (driver.Value, error) {
	return json.Marshal(ps)
}

// Scan 实现 sql.Scanner，从数据库读取
func (ps *WuyinProductSpec) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(b, ps)
}

// WuyinProductSpecsJSON 吴饮商品规格列表（GORM JSON 列）
type WuyinProductSpecsJSON []WuyinProductSpec

// Value 实现 driver.Valuer，写入数据库
func (p WuyinProductSpecsJSON) Value() (driver.Value, error) {
	return json.Marshal(p)
}

// Scan 实现 sql.Scanner，从数据库读取
func (p *WuyinProductSpecsJSON) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(b, p)
}

// WuyinAddressInfo 吴饮地址信息（表 wuyin_address_infos，独立库）。
// 在 AddressInfo 基础上：新增 Tag（仓库标签，结账分组维度）与 Sender（群聊发送人，
// 自动打标依据与人工复核线索）；商品规格以斤存储（件仅结账计数）。列名规则与
// AddressInfo 保持同款约定：无 column: 前缀的段按 GORM 默认蛇形命名生成列
// （productSpecs → product_specs）。
type WuyinAddressInfo struct {
	Model
	PreviewID uint   `gorm:"-" json:"VID"`
	Recipient string `gorm:"recipient" json:"收件人"`
	Phone     string `gorm:"phone" json:"手机号"`
	Address   string `gorm:"address" json:"收货地址"`
	Province  string `gorm:"column:province;size:32;index:idx_province" json:"省份"`
	// ProductSpecs 斤计规格（type:json 存 WuyinProductSpecsJSON），条目数即件数。
	ProductSpecs WuyinProductSpecsJSON `gorm:"productSpecs;type:json" json:"商品规格"`
	Remarks      string                `gorm:"remarks" json:"备注"`
	// Tag 仓库标签（合肥仓/广西仓/云南仓…），由发送人按 WuyinTagConfig 关键词匹配，
	// 预览与详情页均可人工改。size:64 显式声明以支持索引（同 AddressInfo.Province 的理由）。
	Tag string `gorm:"column:tag;size:64;index:idx_tag" json:"标签"`
	// Sender 下单消息的群聊发送人昵称（如"合肥仓工作微信"），最长 128 兜底长昵称+手机号。
	Sender    string `gorm:"column:sender;size:128" json:"发送人"`
	Isstorage bool   `gorm:"-" json:"Isstorage"`
}

// ProductSpecsString 拼接商品规格字符串（"【香水柠檬9斤】"，条目数即件数），
// 与 AddressInfo.ProductSpecsString 同构，供预览/导出展示；remarks 为 true 时附加备注。
func (address WuyinAddressInfo) ProductSpecsString(remarks bool) string {
	var specsString string
	for i, spec := range address.ProductSpecs {
		specsString += "【"
		if i == len(address.ProductSpecs)-1 {
			specsString += spec.Name + strconv.Itoa(spec.Weight) + "斤】"
			if remarks {
				specsString += address.Remarks
			}
			break
		}
		specsString += spec.Name + strconv.Itoa(spec.Weight) + "斤】,"
	}
	return specsString
}

// WuyinTagConfig 仓库标签配置（表 wuyin_tag_configs，独立库）。
// Keywords 为发送人匹配关键词数组（包含匹配，如"合肥"命中"合肥仓工作微信"）；
// TagsSetting 页维护。同一发送人可能命中多个标签时按 Sort 升序、ID 升序取先注册者。
type WuyinTagConfig struct {
	Model
	Name     string            `gorm:"column:name;size:64;uniqueIndex:uk_wuyintag_name" json:"name"`
	Keywords WuyinKeywordsJSON `gorm:"keywords;type:json" json:"keywords"`
	Sort     int               `gorm:"column:sort" json:"sort"`
}

// WuyinKeywordsJSON 标签关键词数组（GORM JSON 列）
type WuyinKeywordsJSON []string

// Value 实现 driver.Valuer，写入数据库
func (p WuyinKeywordsJSON) Value() (driver.Value, error) {
	return json.Marshal(p)
}

// Scan 实现 sql.Scanner，从数据库读取
func (p *WuyinKeywordsJSON) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(b, p)
}

// WuyinPriceConfig 吴饮商品价格（表 wuyin_price_configs，独立库）。
// 只按元/件计价（无斤价），productName+spec 唯一 upsert（spec 为规格串如 "9斤"），
// 同 PriceConfig 的"只保留当前生效价"裁定。
type WuyinPriceConfig struct {
	Model
	ProductName  string  `gorm:"column:product_name;size:191;uniqueIndex:uk_wuyinprice_product_spec,priority:1" json:"productName"`
	Spec         string  `gorm:"column:spec;size:64;uniqueIndex:uk_wuyinprice_product_spec,priority:2" json:"spec"` // 如 "9斤"
	PricePerItem float64 `gorm:"column:price_per_item" json:"pricePerItem"`                                         // 元/件
}
