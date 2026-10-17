# 订单识别与结账系统 数据库迁移方案与 API 契约（2026-10-17）

> 上游依据：`docs/design/2026-10-17-iteration-architecture.md`（唯一架构依据）。
> 规则依据：`.trae/rules/api-contract.md`（契约格式）、`.trae/rules/backend-architecture.md`（B1–B7 分层与迁移规则）。
> 文档性质：数据库变更方案 + API 正式契约。本文档不改业务代码；实施 Agent 按本文档领取任务。
> 契约状态：**已登记** `.trae/rules/api-contract.md` 第 6/8 节（第 8 节已由「计划登记区」转为正式契约）。

## 0. 总负责人已裁定事项（2026-10-17，本文档全篇遵循）

| 编号 | 裁定 | 落点 |
| --- | --- | --- |
| RUL-1 | 价格持久化不保留历史，`PriceConfig` 表按 `productName+spec` 唯一 upsert（覆盖当前生效价） | §1.3、§3.6 |
| RUL-2 | 省份存量回填：`flags_db.go` AutoMigrate 成功后自动幂等执行（仅处理 province 为空的行），不新增手动触发端点 | §1.5 |
| RUL-3 | 关键字搜索采用全字段 LIKE（Likes 使用数据库实际列名 `product_specs` + 新增 `province`，不引入 MySQL FULLTEXT。2026-10-17 实测修正：实际列为 `product_specs` 蛇形命名，原"列名 bug"断言系对 GORM tag 语义的误判，见规则 B3.2） | §3.5 |

---

## 1. 数据库变更

### 1.1 变更总览

| 表 | 变更 | 类型 | 风险 |
| --- | --- | --- | --- |
| `address_infos` | 新增列 `province varchar(32) NULL` + 单列索引 `idx_province` | AutoMigrate 增列（只增不改） | 低（派生数据，可重跑回填重建） |
| `price_configs` | 新建表（含唯一索引 `uk_priceconfig_product_spec`） | AutoMigrate 新表 | 低（新表无存量数据） |
| `deleted_at` | **无 schema 变更**（`models.Model` 已内嵌 `gorm.Model`，软删列已存在） | 行为启用 | 见 §1.4 影响评估 |

表名按 GORM 默认复数策略：`AddressInfo` → `address_infos`，`PriceConfig` → `price_configs`（实施后以 `SHOW TABLES` 核实）。

### 1.2 AddressInfo 新增 province 列

模型改动（`go/models/addressInfo_models.go`，插在 `Address` 与 `ProductSpecs` 之间）：

```go
type AddressInfo struct {
	Model
	PreviewID    uint             `gorm:"-" json:"VID"`
	Recipient    string           `gorm:"recipient" json:"收件人"`
	Phone        string           `gorm:"phone" json:"手机号"`
	Address      string           `gorm:"address" json:"收货地址"`
	Province     string           `gorm:"column:province;size:32;index:idx_province" json:"省份"` // 新增
	ProductSpecs ProductSpecsJSON `gorm:"productSpecs;type:json" json:"商品规格"`
	Remarks      string           `gorm:"remarks" json:"备注"`
	Isstorage    bool             `gorm:"-" json:"Isstorage"`
}
```

**类型与 tag 说明**：

| 项 | 决策 | 理由 |
| --- | --- | --- |
| 列类型 `varchar(32)` | 必须显式 `size:32` | GORM MySQL 驱动对无 `size` 的 `string` 生成 `longtext`（现有 recipient/phone 等列即如此）；`longtext` 无法直接建二级索引（MySQL 报 BLOB/TEXT column used in key specification）。省份是固定 34 值枚举，最长全称「新疆维吾尔自治区」8 字符，32 字符富余。 |
| 列名 `province` | 单词列名，蛇形与驼形同形；显式用 `column:province`（B3.2.1） | `query.go` 的 Likes 与统计 SQL 直接引用**数据库实际列名**，逐字符一致（B3.2，注意实际列为蛇形如 `product_specs`）。 |
| 索引 `idx_province` | **采纳**，单列普通索引 | 见下方论证。 |
| 允许 NULL/空串 | 是 | 存量行迁移后为 NULL；AI 提取失败/地址无法识别的行保持空，统计与地图排除空值（`province <> ''`）。 |

**索引论证（省份统计 `GROUP BY province`）**：

- 方案 A（不建索引）：A1 量级 ≤10 万行，全表扫描 + 内存分组约 50–100ms，功能上可行。
- 方案 B（单列索引，**采纳**）：统计端点在详情页每次加载触发（高频）；province 基数仅 34，索引扫描避免临时表；回填预检 `WHERE province=''` 走索引；写入放大在单店日百级量级可忽略；未来若前端按省份筛选订单可直接复用。
- 不建复合索引（province, created_at）：统计带 created_at 范围过滤时单列索引已够用，复合索引属过度设计；量级达百万级时另行评估（对应架构文档 §9 演进项）。

**迁移方式**：`AddressInfo` 已在 `flags_db.go` 注册，模型加字段后执行 `go run main.go -db` 即自动 `ADD COLUMN` + `CREATE INDEX`（MySQL 8.0 instant/inplace DDL，10 万行秒级，无锁表风险）。**唯一迁移入口是 `go/flags/flags_db.go`，禁止手写 SQL 改表**（B3.1）。

### 1.3 新建 PriceConfig 表

新模型（`go/models/checkout_models.go`，新文件）：

```go
package models

// PriceConfig 商品当前生效价格配置。
// 裁定（2026-10-17）：只保留当前生效价，不保留历史；按 productName+spec 唯一 upsert。
// UpdatedAt（继承自 Model）即「最近生效时间」，不单设生效时间列。
type PriceConfig struct {
	Model
	ProductName  string  `gorm:"productName;size:191;uniqueIndex:uk_priceconfig_product_spec,priority:1" json:"productName"`
	Spec         string  `gorm:"spec;size:64;uniqueIndex:uk_priceconfig_product_spec,priority:2" json:"spec"` // 如 "5斤"
	PricePerKg   float64 `gorm:"pricePerKg" json:"pricePerKg"`                                             // 元/斤
	PricePerItem float64 `gorm:"pricePerItem" json:"pricePerItem"`                                         // 元/件
}
```

**字段与约束说明**：

| 项 | 决策 | 理由 |
| --- | --- | --- |
| 唯一索引 `uk_priceconfig_product_spec (productName, spec)` | 必须显式 `size` | ① `productName` 无 size 会生成 longtext，无法进唯一索引；② 唯一索引是并发 upsert 的数据库级兜底（两个并发保存请求不产生重复行）；③ 索引长度 191×4 + 64×4 = 1020 字节 < InnoDB 3072 字节上限（utf8mb4）。 |
| json 键英文 | 与结账域现状一致 | 结账域现有 `ProPriceRequest`/`Spec` 均为英文键（productName/spec/pricePerKg）；中文键是订单域 `AddressInfo` 的历史决定，不跨域复制。 |
| 不建独立「生效时间」列 | 裁定 RUL-1 | 不保留历史；`UpdatedAt` 随 upsert 自动刷新，即当前价的生效时间。 |
| 软删 + 唯一索引的冲突 | 登记为已知限制 | 本轮**没有**删除价格配置的端点（只 upsert）。若未来新增删除且走软删，同键新建会被唯一索引阻挡（软删行物理仍在）；届时需对价格配置使用物理删除（非订单类数据，不违反 B3.5）或改造唯一索引。见 §5 待确认。 |
| `Order` 模型不落库 | 维持现状 | `models.Order` 仍是内存统计中间产物（B3.6 登记），价格配置独立成表，不混入统计对象。 |

**AutoMigrate 注册（`go/flags/flags_db.go`，唯一迁移入口）**：

```go
func FlagDB() {
	err := global.Db.AutoMigrate(
		models.AddressInfo{},
		models.LogModels{},
		models.PriceConfig{}, // 2026-10-17 新增：价格配置表
	)
	if err != nil {
		logrus.Errorf("数据库迁移失败 %s", err)
		return // 顺手修正：现状迁移失败仍打印「成功」日志，违反 B2 不吞错
	}
	logrus.Info("数据库迁移成功")

	// RUL-2：迁移成功后自动幂等回填省份（仅 province 为空的行）
	updated, remaining, err := order.BackfillProvince()
	if err != nil {
		logrus.Errorf("省份存量回填失败 %s", err)
		return
	}
	logrus.Infof("省份存量回填完成：回填 %d 行，仍为空 %d 行", updated, remaining)
}
```

> 依赖方向说明：`flags` → `server/order`（BackfillProvince）为单向依赖、无环（server/order 不引用 flags）。B1 四层表未列 flags，B5 明确允许 flags 使用 `global.Db`，该调用不违反规则；回填与迁移收口在同一入口，运维一条命令完成。

**数据生命周期**：价格配置永久保留，upsert 覆盖当前价（RUL-1）；订单数据软删后永久保留，不做物理清理。

### 1.4 软删除设计确认与现有查询影响评估

**现状确认（无 schema 变更）**：`models.Model` 内嵌 `gorm.Model`（ID/CreatedAt/UpdatedAt/`DeletedAt gorm.DeletedAt`），`address_infos`、`log_models` 两表的 `deleted_at` 列及索引**已存在**。本轮 `DELETE /api/order/:id` 直接使用 GORM `Delete()`（生成 `UPDATE address_infos SET deleted_at=NOW() WHERE id=?`），无需任何迁移。

**对现有查询的影响（GORM 对含 `gorm.DeletedAt` 的模型自动追加 `deleted_at IS NULL`，全部查询无需改代码即获得过滤）**：

| 查询场景 | 影响 | 结论 |
| --- | --- | --- |
| `POST /api/order/query`（`common.ListQuery`） | 已删行从 list 与 count 同时消失 | 符合预期 |
| `POST /api/checkout/daydata` / `GetMonthdata` | 已删订单不再计入账单统计与件数 | 符合预期；**对账口径变化**：本轮之前订单不可删除，上线后删除会导致历史账单数字变化。删除操作有 alog 留痕可查，需在交付说明中明确告知用户 |
| 省份统计 `ProvinceStats` | 同上自动过滤 | 符合预期 |
| `BackfillProvince` | 已删行不被扫描回填 | 符合预期（已删行无需省份） |
| 预览态 `global.AddressList`、Excel 导出预览 | 内存态，与 DB 无关 | 无影响 |
| `ExceladdressInfo` | 不含 deleted_at 字段 | 前端无感知，符合预期 |

**风险与限制**：

1. 已软删数据本轮**无查看/恢复端点**，误删需 DBA 手工 `UPDATE address_infos SET deleted_at=NULL WHERE id=?` 恢复（见 §5 待确认）。
2. 禁止 `Unscoped()` 物理删除、禁止无 WHERE 批量删除（B3.5、T4）。
3. `PriceConfig` 同样继承软删列，但本轮无删除端点（见 §1.3 已知限制）。

### 1.5 存量数据回填方案（RUL-2）

#### 1.5.1 触发方式与幂等性

- 触发：`flags_db.go` 的 `FlagDB()` 内，AutoMigrate 成功后自动调用 `order.BackfillProvince()`（§1.3 代码）。运维执行 `go run main.go -db` 一条命令完成「加列 + 回填」。
- 幂等性证明：只查询 `province IS NULL OR province = ''` 的行；已回填行第二次不再命中。无法识别的行保持空，下次执行（如地址被修正后重跑 `-db`）可再尝试。可安全重复执行。
- 事务边界：逐行独立 UPDATE，**不包整体事务**（允许部分成功，失败行记日志继续；整体回滚反而不利于长尾修复）。每行失败记 `logrus.Errorf` + 继续。
- 性能：≤10 万行一次性 `Find` 空省份行 + 逐行 UPDATE，秒级完成；仅执行 `-db` 时触发，不影响正常启动路径。

#### 1.5.2 Go 实现（`go/server/order/loaddb.go` 新增）

```go
// BackfillProvince 扫描 province 为空的存量行，从 address 提取省份回填。幂等，可重复执行。
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
			remaining++ // 无法识别，保持空，留待人工修正（改单可补省份）
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
```

#### 1.5.3 省份提取工具（`go/utils/order/enter.go` 新增）

**34 个省级行政区完整映射表（全称 = 存储值，简称 = 匹配兜底）**：

| # | 全称（存储值） | 简称（匹配用） | 行政类型 |
| --- | --- | --- | --- |
| 1 | 北京市 | 北京 | 直辖市 |
| 2 | 天津市 | 天津 | 直辖市 |
| 3 | 上海市 | 上海 | 直辖市 |
| 4 | 重庆市 | 重庆 | 直辖市 |
| 5 | 河北省 | 河北 | 省 |
| 6 | 山西省 | 山西 | 省 |
| 7 | 辽宁省 | 辽宁 | 省 |
| 8 | 吉林省 | 吉林 | 省 |
| 9 | 黑龙江省 | 黑龙江 | 省 |
| 10 | 江苏省 | 江苏 | 省 |
| 11 | 浙江省 | 浙江 | 省 |
| 12 | 安徽省 | 安徽 | 省 |
| 13 | 福建省 | 福建 | 省 |
| 14 | 江西省 | 江西 | 省 |
| 15 | 山东省 | 山东 | 省 |
| 16 | 河南省 | 河南 | 省 |
| 17 | 湖北省 | 湖北 | 省 |
| 18 | 湖南省 | 湖南 | 省 |
| 19 | 广东省 | 广东 | 省 |
| 20 | 海南省 | 海南 | 省 |
| 21 | 四川省 | 四川 | 省 |
| 22 | 贵州省 | 贵州 | 省 |
| 23 | 云南省 | 云南 | 省 |
| 24 | 陕西省 | 陕西 | 省 |
| 25 | 甘肃省 | 甘肃 | 省 |
| 26 | 青海省 | 青海 | 省 |
| 27 | 台湾省 | 台湾 | 省 |
| 28 | 内蒙古自治区 | 内蒙古 | 自治区 |
| 29 | 广西壮族自治区 | 广西 | 自治区 |
| 30 | 西藏自治区 | 西藏 | 自治区 |
| 31 | 宁夏回族自治区 | 宁夏 | 自治区 |
| 32 | 新疆维吾尔自治区 | 新疆 | 自治区 |
| 33 | 香港特别行政区 | 香港 | 特别行政区 |
| 34 | 澳门特别行政区 | 澳门 | 特别行政区 |

```go
// provinceTable 中国 34 个省级行政区（全称+简称）。有序切片而非 map，保证匹配结果确定。
var provinceTable = []struct{ Full, Short string }{
	{"北京市", "北京"}, {"天津市", "天津"}, {"上海市", "上海"}, {"重庆市", "重庆"},
	{"河北省", "河北"}, {"山西省", "山西"}, {"辽宁省", "辽宁"}, {"吉林省", "吉林"},
	{"黑龙江省", "黑龙江"}, {"江苏省", "江苏"}, {"浙江省", "浙江"}, {"安徽省", "安徽"},
	{"福建省", "福建"}, {"江西省", "江西"}, {"山东省", "山东"}, {"河南省", "河南"},
	{"湖北省", "湖北"}, {"湖南省", "湖南"}, {"广东省", "广东"}, {"海南省", "海南"},
	{"四川省", "四川"}, {"贵州省", "贵州"}, {"云南省", "云南"}, {"陕西省", "陕西"},
	{"甘肃省", "甘肃"}, {"青海省", "青海"}, {"台湾省", "台湾"},
	{"内蒙古自治区", "内蒙古"}, {"广西壮族自治区", "广西"}, {"西藏自治区", "西藏"},
	{"宁夏回族自治区", "宁夏"}, {"新疆维吾尔自治区", "新疆"},
	{"香港特别行政区", "香港"}, {"澳门特别行政区", "澳门"},
}

// ExtractProvince 从地址字符串提取省级行政区全称；未识别返回空串。
// 两轮匹配：① 全称（全称均以 市/省/自治区/特别行政区 结尾，互不包含，无歧义，不限位置）；
// ② 简称兜底（34 个简称互不为子串），且简称须出现在地址前 6 个字符内——
//    省份惯例位于中文地址开头，可规避「南京市上海路」「青岛市广西路」类路名误判。
func ExtractProvince(address string) string {
	for _, p := range provinceTable {
		if strings.Contains(address, p.Full) {
			return p.Full
		}
	}
	head := address
	if len(head) > 18 { // UTF-8 中文每字 3 字节，前 6 个字符 ≈ 前 18 字节，粗截断即可
		head = head[:18]
	}
	for _, p := range provinceTable {
		if strings.Contains(head, p.Short) {
			return p.Full
		}
	}
	return ""
}
```

**已知限制与兜底（对应架构文档 R4）**：

1. 地址完全不含省级行政区信息（如仅「杭州市西湖区…」无省名、无城市→省份映射）→ 返回空串，不回填。城市→省份映射（300+ 条）本轮不做，属 scope 外。
2. 简称前 6 字符限定后仍可能误判（如订单文本开头即出现他省简称），残余误判由人工修正兜底：需求 3 的改单端点（§3.3）支持修改省份；回填幂等，修正地址后重跑 `-db` 即可修正。
3. 地址含多个省份字样时，按表序取第一个命中（确定性，可测试）。

#### 1.5.4 运维预检/验收 SQL（只读，不作为迁移手段）

```sql
-- 迁移前预检：待回填量
SELECT COUNT(*) FROM address_infos WHERE province IS NULL OR province = '';
-- 抽样人工核对提取质量
SELECT id, address FROM address_infos WHERE province IS NULL OR province = '' LIMIT 20;
-- 回填后验收：省份分布（口径与 ProvinceStats API 一致：未软删且非空）
SELECT province, COUNT(*) AS cnt
FROM address_infos
WHERE deleted_at IS NULL AND province <> ''
GROUP BY province ORDER BY cnt DESC;
-- 回填后验收：仍为空的行数（应等于日志中的 remaining）
SELECT COUNT(*) FROM address_infos WHERE province IS NULL OR province = '';
```

### 1.6 迁移执行步骤、风险与回滚

**执行步骤（运维）**：

1. 备份：`mysqldump <db> address_infos > address_infos_bak_20261017.sql`（含新表前无 price_configs，无需备份）。
2. 部署含新模型的代码（`addressInfo_models.go` 加 Province、新增 `checkout_models.go`、`flags_db.go` 注册）。
3. 仓库根执行 `go run main.go -db`，确认日志依次出现：`数据库迁移成功`、`省份存量回填完成：回填 N 行，仍为空 M 行`。
4. 验证（§1.5.4 验收 SQL + 以下结构检查）：

```sql
SHOW COLUMNS FROM address_infos LIKE 'province';            -- varchar(32)
SHOW INDEX FROM address_infos WHERE Key_name = 'idx_province';
SHOW CREATE TABLE price_configs;                            -- UNIQUE KEY uk_priceconfig_product_spec (productName, spec)
```

5. `go test ./go/...` 通过。

**风险评估**：均为增量 DDL（加列/加索引/建新表），不动存量列，无数据丢失路径；10 万行量级 DDL 秒级。省份为派生数据，回填错误可修正后重跑。

**回滚方案（破坏性操作，需总负责人明确授权后方可执行）**：

| 对象 | 回滚 SQL | 数据影响 |
| --- | --- | --- |
| province 列 | `ALTER TABLE address_infos DROP INDEX idx_province; ALTER TABLE address_infos DROP COLUMN province;` | 丢失已回填省份（派生数据，可重跑回填重建，损失可控） |
| price_configs 表 | `DROP TABLE price_configs;` | 丢失全部价格配置（需重新录入或从备份恢复，**回滚前必须先备份**） |

代码回滚：git revert 后模型无 Province 字段，GORM AutoMigrate 只增不删，残留列/表对旧代码无害。

---

## 2. API 契约通用约定

| 项 | 约定 |
| --- | --- |
| 响应结构 | HTTP 恒 200 + `{code, msg, data}`；`code`：0 成功 / 40000 参数错误 / 50000 业务失败（api-contract §2） |
| 空集合 | Go nil slice/map 序列化为 `null`；前端类型一律 `T[] \| null` |
| 时间格式 | RFC3339/ISO8601 带时区。前端统一 `toISOString()`（UTC，`2026-10-17T00:00:00.000Z`）；服务端按本地时区（Asia/Shanghai）归一化日界，日期范围一律**闭区间**：`[startTime 当日 00:00:00.000, endTime 当日 23:59:59.999]` |
| 金额格式 | `float64`，单位元，存储与传输原值，展示层保留两位小数；`totalAmount` 一律以服务端计算为准（请求中的值仅回显，服务端重算覆盖）。单店量级不引入 decimal，登记为已知精度取舍 |
| 键风格 | 订单域业务字段中文键 + 技术字段英文键（`收件人`/`手机号`/`createdAt`/`VID`，与 `ExceladdressInfo` 一致）；结账域英文键（`productName`/`pricePerKg`，存量风格冻结）。新增结构遵循所属域 |
| 分页 | 仅 `POST /api/order/query` 使用 `common.PageInfo`（limit/page/key/order） |
| 幂等性总览 | `province-stats`：只读幂等；`audit`：只读幂等（AI 复核输出非确定性，契约标注）；`PUT /order/:id`：重复提交同值幂等；`DELETE /order/:id`：**非幂等**（第二次返回 50000「订单不存在」），前端删除成功后移除行避免重放；`POST /checkout/price`：upsert 幂等 |

---

## 3. 端点契约

### 3.1 GET /api/order/province-stats（省份统计，新端点）

**日期范围过滤建议（已采纳）**：支持可选 `startTime`/`endTime`（RFC3339），与 `/order/query` 的时间语义一致，供地图随详情页查询条件联动。**两参数必须成对出现**：只传其一 → 40000（避免「半开范围」歧义）；都不传 → 全量统计。

**归属链**：`go/router/enter.go`（`orderGroup.GET("/province-stats", handler.ProvinceStats)`）→ `go/handler/order.go` `ProvinceStats` → `go/application/order.go` 透传 + 类型别名（`ProvinceStat`/`ProvinceStatsResponse`）→ `go/server/order/stats.go`（新文件）。handler 从 query string 手工解析时间（`time.Parse(time.RFC3339, ...)`），禁止 handler 直连 server（B1）。

**请求示例**：

```
GET /api/order/province-stats?startTime=2026-10-01T00:00:00.000Z&endTime=2026-10-17T23:59:59.999Z
GET /api/order/province-stats   （全量）
```

**响应示例**：

```json
{
  "code": 0,
  "msg": "ok",
  "data": {
    "list": [
      { "name": "福建省", "shortName": "福建", "value": 46 },
      { "name": "广东省", "shortName": "广东", "value": 31 },
      { "name": "广西壮族自治区", "shortName": "广西", "value": 8 }
    ],
    "total": 85
  }
}
```

**出参结构**（定义在 `server/order/stats.go`，application 别名透传）：

```go
type ProvinceStat struct {
	Name      string `json:"name"`      // 省份全称（存储值，34 行政区枚举）
	ShortName string `json:"shortName"` // 省份简称（见 §1.5.3 映射表）
	Value     int    `json:"value"`     // 订单数
}
type ProvinceStatsResponse struct {
	List  []ProvinceStat `json:"list"`  // 按 value 降序、同值按 name 升序；空为 null
	Total int            `json:"total"` // Σvalue（province 非空订单总数）
}
```

**`shortName` 字段说明（联调风险消解）**：主流 china.json 有两版命名——传统 echarts 旧仓库版用简称（「福建」「内蒙古」），阿里 DataV 版用全称（「福建省」「内蒙古自治区」）。后端统一输出全称 + 简称双字段，前端按所采用 GeoJSON 的 `properties.name` 任选匹配，消除跨 Agent 联调失败点。**前端须在联调文档登记最终选用的 GeoJSON 版本**。

**统计口径**：`address_infos` 行数（非商品件数）；条件 `deleted_at IS NULL`（GORM 自动）`AND province <> '' AND province IS NOT NULL` + 可选 `created_at BETWEEN`；SQL 形态 `SELECT province AS name, count(*) AS value ... GROUP BY province`（shortName 由服务端按 §1.5.3 映射表填充，不入库）。

**错误码**：

| code | 场景 | msg 示例 |
| --- | --- | --- |
| 40000 | startTime/endTime 非 RFC3339 | `startTime 时间格式错误` |
| 40000 | 只传其一 | `startTime 与 endTime 必须成对传入` |
| 50000 | DB 查询失败 | `查询失败: <err>` |

**前端封装**（`web/src/api/order.ts`）：`getProvinceStats(startTime?: string, endTime?: string)`；`request.ts` 的 `get(path)` 不支持 params，在封装内拼接 query string（`encodeURIComponent`），不改共享 request.ts。

### 3.2 POST /api/order/preview/audit（预览审核，新端点）

**原始文本来源（按架构文档 §5.4 裁定落地）**：后端暂存——识别入口把原始文本写入 `global.LastInputText`，审核请求**不回传文本**。

**对架构文档的必要细化（实现时必须遵守）**：`LastInputText` 按 `previewType` 与 `global.AddressList` 同步维护——`previewType=0`（清空重识别）→ `LastInputText = 本次 text`；`previewType=1`（追加）→ `LastInputText += "\n" + 本次 text`。否则追加识别后审核，首批订单会被误判为「多单」。

**归属链**：`go/router/enter.go`（`orderGroup.POST("/preview/audit", handler.AuditPreview)`）→ `go/handler/order.go` `AuditPreview` → `go/application/order.go` 透传（`AuditRequest`/`AuditResponse` 别名）→ `go/server/order/audit.go`（新文件，程序对比 + AI 复核编排）。

**请求示例**（空 JSON 对象，无必填字段）：

```json
{}
```

**响应示例**：

```json
{
  "code": 0,
  "msg": "ok",
  "data": {
    "items": [
      {
        "VID": 0,
        "收件人": "李四",
        "手机号": "13800000004",
        "收货地址": "广东省广州市天河区YY路2号",
        "商品规格": "【香水柠檬5斤】",
        "diffType": "missing",
        "diffDesc": "原始文本含此订单，预览表无对应行（手机号 13800000004）",
        "aiVerdict": "误报",
        "aiSuggest": ""
      },
      {
        "VID": 3,
        "收件人": "张三",
        "手机号": "13800000003",
        "收货地址": "福建省厦门市思明区XX路1号",
        "商品规格": "【香水柠檬5斤】",
        "diffType": "wrong-spec",
        "diffDesc": "原始文本规格为【香水柠檬3斤】，预览为【香水柠檬5斤】",
        "aiVerdict": "真差异",
        "aiSuggest": "建议将规格改回【香水柠檬3斤】"
      },
      {
        "VID": 5,
        "收件人": "王五",
        "手机号": "13800000005",
        "收货地址": "浙江省杭州市西湖区ZZ路3号",
        "商品规格": "【香水柠檬5斤】",
        "diffType": "normal",
        "diffDesc": "",
        "aiVerdict": "",
        "aiSuggest": ""
      }
    ],
    "summary": { "missing": 1, "wrong": 1, "extra": 0, "normal": 1 },
    "aiFailed": false
  }
}
```

**出参结构**（定义在 `server/order/audit.go`，application 别名透传）：

```go
type AuditItem struct {
	VID          uint   `json:"VID"`        // 预览行标识（与 ExceladdressInfo.VID 一致；缺单项为 0）
	Recipient    string `json:"收件人"`      // 缺单/错单时为原始文本侧值，其余为预览侧值
	Phone        string `json:"手机号"`
	Address      string `json:"收货地址"`
	ProductSpecs string `json:"商品规格"`    // 展示串（ProductSpecsString 格式）
	DiffType     string `json:"diffType"`   // 枚举见下
	DiffDesc     string `json:"diffDesc"`   // 程序对比差异描述（中文，normal 为空串）
	AiVerdict    string `json:"aiVerdict"`  // "真差异" | "误报" | ""（未调 AI）
	AiSuggest    string `json:"aiSuggest"`  // AI 修正建议，无则 ""
}
type AuditSummary struct {
	Missing int `json:"missing"` // 缺单
	Wrong   int `json:"wrong"`   // 错单（wrong-key + wrong-spec 合计）
	Extra   int `json:"extra"`   // 多单
	Normal  int `json:"normal"`  // 正常
}
type AuditResponse struct {
	Items    []AuditItem  `json:"items"`    // 按 missing→wrong-key→wrong-spec→extra→normal 排序，同级按 VID 升序
	Summary  AuditSummary `json:"summary"`
	AiFailed bool         `json:"aiFailed"` // AI 复核环节异常时 true（此时 aiVerdict/aiSuggest 全空，程序对比结果仍有效）
}
```

**键风格决策**：订单业务字段用中文键（`收件人`/`手机号`/`收货地址`/`商品规格`，与 `ExceladdressInfo` 逐字一致，前端可复用行渲染与 F4 双写）；审核域字段用英文小驼峰（`VID`/`diffType`/`diffDesc`/`aiVerdict`/`aiSuggest`/`summary`/`aiFailed`，与 `createdAt`/`VID` 技术键风格一致）。

**diffType 枚举（值冻结，新增只能尾加；为字符串枚举，不入 models/enum 的 iota 体系）**：

| 值 | 含义 | 严重级 |
| --- | --- | --- |
| `missing` | 缺单：原始文本有、预览表无 | 高 |
| `wrong-key` | 错单-关键字段：收件人/手机号/地址不一致 | 高 |
| `wrong-spec` | 错单-规格：规格数量或内容不一致 | 中 |
| `extra` | 多单：预览表有、原始文本无 | 中 |
| `normal` | 一致 | - |

**行为约定**：

1. 程序对比基准 = 对 `global.LastInputText` 做确定性解析（架构文档 §5.2 方案 C）；匹配键 = 手机号（主）+ 收件人 + 地址前 8 字符；规格不作匹配键。
2. AI 复核仅针对 `missing/wrong-key/wrong-spec/extra` 项批量调用一次；失败**不阻塞**响应——`aiFailed=true` + 全部 `aiVerdict` 置空返回程序对比结果，同时 `alog.Logaction(ErrLevel, ...)` 记日志（满足 B2 不吞错：显式标记 + 留痕）。
3. 只读端点：不修改 `global.AddressList` 与 `LastInputText`，可重复调用；AI 复核建议具有非确定性，仅作参考（R3）。
4. 「应用修正」不新增端点：前端按 `aiSuggest` 引导用户编辑，落库走既有 `PUT /api/order/preview`。

**错误码**：

| code | 场景 | msg 示例 |
| --- | --- | --- |
| 40000 | body 缺失或非 JSON 对象 | `请求参数错误: ...` |
| 50000 | 预览态为空（`global.AddressList` 为空） | `请先识别订单信息` |
| 50000 | 原始文本为空（`LastInputText` 为空，如服务重启后） | `无可审核的原始文本，请重新识别` |
| 50000 | 程序对比/AI 编排内部错误 | `审核失败: <err>` |

### 3.3 PUT /api/order/:id（修改历史订单，新端点）

**归属链**：`go/router/enter.go`（`orderGroup.PUT("/:id", handler.UpdateOrder)`）→ `go/handler/order.go` `UpdateOrder` → `go/application/order.go` 透传（`UpdateOrderRequest` 别名）→ `go/server/order/loaddb.go` `UpdateOrder`。

**前置契约依赖（关键）**：`:id` 为 `address_infos` 主键。现有 `/api/order/query` 出参 `ExceladdressInfo` **不含 DB 主键**（`PreviewID` 不落库，历史订单行 VID 恒为 0），前端无从获得 id。因此本轮 `ExceladdressInfo` **新增 `ID uint json:"id"` 字段**，`AddressToExcelAddress` 补 `ID: address.ID` 映射（预览行 id=0，历史订单行 id>0；前端以 `id>0` 判定可删改）。同步处：`ExceladdressInfo` 结构体 + `web/src/api/order.ts` 的 `ExcelAddressInfo`/`RecognitionResult` interface（加 `id?: number`），无需动提示词与 Likes。

**请求示例**（只传需要改的字段；键中文，与 `ExceladdressInfo` 一致）：

```
PUT /api/order/1024
```

```json
{
  "收货地址": "福建省厦门市思明区软件园二期望海路10号",
  "商品规格": "【香水柠檬3斤】,【黄金葡萄柚10斤】",
  "备注": "用户要求周一送达"
}
```

**响应示例**：

```json
{ "code": 0, "msg": "ok", "data": null }
```

**入参结构**（定义在 `server/order/loaddb.go`，application 别名透传；指针区分「未传」与「清空」）：

```go
type UpdateOrderRequest struct {
	Recipient    *string `json:"收件人"`
	Phone        *string `json:"手机号"`
	Address      *string `json:"收货地址"`
	Province     *string `json:"省份"`
	ProductSpecs *string `json:"商品规格"` // 字符串形态，服务端 Specprocessing 解析（与 SetPreview 同解析器同格式）
	Remarks      *string `json:"备注"`
}
```

**字段语义**：

| 传值 | 语义 |
| --- | --- |
| 字段缺失或 `null` | 不更新该字段 |
| `""`（空串） | 清空该字段（`商品规格` 例外：空串/无法解析 → 50000，规格不可清空只能替换） |
| `收货地址` 传值 | 服务端自动 `ExtractProvince(新地址)` 重提取省份写入 `province`；若同请求**显式**传了 `省份`，显式值优先覆盖派生值（处理顺序：先派生后覆盖） |
| `商品规格` 传值 | `Specprocessing` 解析为 `ProductSpecsJSON` 后整体替换；格式 `【名称数字斤】`，多项逗号分隔 |
| `UpdatedAt` | GORM 自动刷新 |

**校验与错误码**：

| code | 场景 | msg 示例 |
| --- | --- | --- |
| 40000 | id 非十进制正整数 | `id 参数错误` |
| 40000 | body 绑定失败 | `请求参数错误: ...` |
| 40000 | 六个字段全部未传 | `至少提供一项需要修改的字段` |
| 50000 | 订单不存在（含已软删） | `订单不存在` |
| 50000 | 规格解析失败 | `商品规格解析失败: 没有找到匹配的内容` |
| 50000 | DB 更新失败 | `修改失败: <err>` |

**其他约定**：不做手机号/地址格式强校验（与现有 SetPreview 宽松风格一致，见 §5）；无乐观锁，并发最后写入胜出（单用户场景，R7 关联）；重复提交同请求幂等（Updates 写同值）；T4 四项：handler 参数校验 + server `First` 存在性校验 + 前端二次确认（弹窗显示收件人+手机号）+ `alog.Logaction("修改订单", "id=.. 收件人=.. 手机号=..")` 留痕。**订单日期（created_at）不可通过本端点修改**（影响账单归属，本轮明确不做，见 §5）。

### 3.4 DELETE /api/order/:id（软删除历史订单，新端点）

**归属链**：同 §3.3（`DeleteOrder` handler/application/server，路由 `orderGroup.DELETE("/:id", handler.DeleteOrder)`）。

**请求示例**：

```
DELETE /api/order/1024
```

**响应示例**：

```json
{ "code": 0, "msg": "ok", "data": null }
```

**行为**：`global.Db.Delete(&addr)` → `UPDATE address_infos SET deleted_at = NOW() WHERE id = ?`（软删，禁止 Unscoped，B3.5）。删除后该行**立即**从 `/order/query`、`/checkout/daydata`、`/province-stats`、Excel 导出中消失（§1.4 评估）。

**错误码**：

| code | 场景 | msg 示例 |
| --- | --- | --- |
| 40000 | id 非法 | `id 参数错误` |
| 50000 | 订单不存在（含重复删除——GORM First 自动过滤已软删行） | `订单不存在` |
| 50000 | DB 删除失败 | `删除失败: <err>` |

**其他约定**：非幂等（重复调用同 id 返回 50000），前端删除成功后从列表移除该行；无恢复端点（误删恢复路径见 §1.4）；T4 四项：id 校验 + 存在性校验 + 前端二次确认（显示收件人+手机号+地址）+ `alog.Logaction("删除订单", "id=.. 收件人=.. 手机号=.. 地址=..")` 留痕；仅支持单条删除，无批量删除端点。

### 3.5 POST /api/order/query（既有端点变更）

**归属链**：不变（`handler/order.go` `OrderQuery` → `application/order.go` → `server/order/query.go`）。

**变更 1：Likes 新增 province（B3.2 / RUL-3，2026-10-17 实测修正）**：

```go
// 修正前（product_specs 为数据库实际列名，此前"列名 bug"断言系误判，保持不变；
// 真正的问题是 OrderQuery 吞错导致查询失败静默返回空结果）：
Likes: []string{"recipient", "address", "phone", "product_specs", "remarks"},
// 修正后（仅追加 province）：
Likes: []string{"recipient", "address", "phone", "product_specs", "remarks", "province"},
```

**全字段搜索说明（key 模糊列，`LIKE '%key%'` OR 拼接，Likes 写数据库实际列名）**：

| 列 | 数据库实际列名 | 匹配说明 |
| --- | --- | --- |
| 收件人 | `recipient` | 子串匹配 |
| 收货地址 | `address` | 子串匹配 |
| 手机号 | `phone` | 子串匹配 |
| 商品规格 | `product_specs` | JSON 列按序列化字符串文本匹配：输入「柠檬」可命中 `[{"name":"香水柠檬","weight":5}]`；输入「5」可命中 weight:5。近似匹配，精确 JSON 查询（JSON_EXTRACT）本轮不做 |
| 备注 | `remarks` | 子串匹配 |
| 省份 | `province`（新增） | 子串匹配：输入「福建」命中「福建省」，输入全称亦可 |

**变更 2：出参行新增字段**（`ExceladdressInfo` 结构变更，`AddressToExcelAddress` 补映射）：

| 新增键 | 类型 | 说明 |
| --- | --- | --- |
| `id` | number | DB 主键（§3.3 前置依赖）；预览行恒为 0，历史订单行 >0 |
| `省份` | string | 省份全称；未识别为空串 |

**变更 3：入参新增等值过滤**：请求体 `AddressInfo` 部分可传 `"省份": "福建省"` 做精确等值（`Where(model)` 非零字段等值匹配），与 key 模糊搜索互补。`商品规格` 字段为 JSON 数组结构（`[{name,weight}]`），不作为等值过滤入参（现状亦然），规格检索请用 `key`。

**变更 4（B2 合规修复，建议随本轮实施）**：`OrderQuery` 现状吞掉 `ListQuery` 错误返回空列表（code 0）；修正为 err → `common.Fail`（50000）。前端对 50000 已有统一错误处理，无破坏。

**请求示例**：

```json
{
  "limit": 10,
  "page": 1,
  "key": "柠檬",
  "order": "created_at",
  "startTime": "2026-10-01T00:00:00.000Z",
  "endTime": "2026-10-17T23:59:59.999Z"
}
```

**响应示例**：

```json
{
  "code": 0,
  "msg": "ok",
  "data": {
    "list": [
      {
        "id": 1024,
        "createdAt": "2026-10-16T09:30:00+08:00",
        "VID": 0,
        "收件人": "张三",
        "手机号": "13800000001",
        "收货地址": "福建省厦门市思明区望海路10号",
        "省份": "福建省",
        "商品规格": "【香水柠檬5斤】",
        "备注": ""
      }
    ],
    "count": 1
  }
}
```

**错误码**：40000（body 绑定失败）；50000（查询失败，变更 4 起）。

### 3.6 POST /api/checkout/price（既有端点变更：价格持久化，RUL-1）

**归属链**：不变（`handler/checkout.go` `SetPrice` → `application/checkout.go` → `server/checkout`）。**不新增** `PUT /api/checkout/price-config` 端点（裁定：扩展现有端点，避免两个写价格的入口分裂；前端现有 `setPrice` 调用点零改动）。

**请求示例**（结构不变，新增校验）：

```json
[
  { "productName": "香水柠檬", "spec": "5斤", "total": 12, "pricePerKg": 8.5, "pricePerItem": 42.5, "totalAmount": 510 },
  { "productName": "香水柠檬", "spec": "3斤", "total": 8, "pricePerKg": 8.5, "pricePerItem": 0, "totalAmount": 0 }
]
```

**新增校验**（handler 层，三段式第 2 段）：

| code | 场景 | msg 示例 |
| --- | --- | --- |
| 40000 | body 非数组或为空数组 | `价格列表不能为空` |
| 40000 | 某项 productName 或 spec 为空 | `第 N 项 productName 不能为空` |

**行为变更（持久化）**：

1. `server/checkout/price.go`（新文件）`SavePriceConfig(list)`：**单个事务**（`global.Db.Transaction`）内逐项按 `(productName, spec)` upsert 至 `price_configs`——查到则 `Updates` 覆盖 pricePerKg/pricePerItem（RUL-1 不保留历史），查不到则 `Create`；全部成功或整体回滚（50000）。
2. `pricePerItem == 0` 时按 `pricePerKg × 斤数` 推导（沿用现有规则），**推导后的值持久化**，保证回显与库一致。
3. `total`/`totalAmount` 以服务端重算为准（请求值仅回显用途，服务端忽略并覆盖）。
4. 并发兜底：唯一索引 `uk_priceconfig_product_spec` 保证不产生重复行；`Create` 撞唯一键（MySQL 1062）时降级重试一次 `Updates`。
5. 事务提交后 `alog.Logaction(InfoLevel, "保存价格配置", "共 N 条")` 留痕（T4：金额类写库留痕）。

**响应示例**（结构不变，`CheckoutResponse` 线上键为首字母大写原名——存量字段无 json tag，拼写冻结）：

```json
{
  "code": 0,
  "msg": "ok",
  "data": {
    "FieldList": [],
    "PBMap": { "香水柠檬": ["香水柠檬"] },
    "Count": 20,
    "ProSPecs": [
      { "SID": 1, "Name": "香水柠檬",
        "Specs": [
          { "name": "3斤", "pricePerKg": 8.5, "pricePerItem": 25.5, "total": 8 },
          { "name": "5斤", "pricePerKg": 8.5, "pricePerItem": 42.5, "total": 12 }
        ] }
    ],
    "TableData": ["10月16日,8,12"],
    "TotalRow": [8, 12],
    "PricePerKgRow": [8.5, 8.5],
    "PricePerItemRow": [25.5, 42.5],
    "TotalAmountRow": [204, 510]
  }
}
```

**错误码**：40000（见校验表）；50000（事务失败/DB 错误）。**幂等**：同请求重复提交 upsert 同值。

### 3.7 POST /api/checkout/daydata（既有端点变更：bug 修复 + 价格从 DB 读取）

**归属链**：不变（`handler/checkout.go` `GetDayData` → `application/checkout.go` → `server/checkout/enter.go` `GetDaydata`）。

**请求**：不变，`{startTime, endTime}` 必填（缺失 → 40000 `startTime/endTime 不能为空`）。

**行为变更 1（起始日期 bug 修复，`enter.go` L227）**：`statime.Day()+1` 改为 `statime.Day()`，查询范围**包含 startTime 当日**（原少算第一天）。

**行为变更 2（Count bug 修复，`enter.go` L236）**：GORM 链式顺序修正为先 `Where` 后 `Count`，`Count` = 日期范围内条数（原为全表数）。

**行为变更 3（价格从 DB 回填）**：

1. `ProSPecs[].Specs[].pricePerKg/pricePerItem` 从 `LoadPriceMap()`（读 `price_configs`）按 商品名+规格 匹配填充；匹配不到保持 0（未配置，由用户在价格栏补录后走 §3.6 持久化）。替换现状的 `PricePerKg: 0` 硬编码。
2. `PricePerKgRow`/`PricePerItemRow`/`TotalAmountRow` 由 `GetDaydata` 直接计算填充（原来三行缺省 null，仅 SetPrice 后才有）：与 `TotalRow` **同序同长**（按响应 `ProSPecs` 展开顺序：外层商品序、内层规格序）；`TotalAmountRow[i] = TotalRow[i] × PricePerItemRow[i]`，`pricePerItem` 为 0 时先按 `pricePerKg × 斤数` 推导。
3. 效果：打开结账页即回显持久化价格，无需先保存一次；重启后价格不丢。

**响应**：结构不变（`CheckoutResponse`，线上键 `FieldList/PBMap/Count/ProSPecs/TableData/TotalRow/PricePerKgRow/PricePerItemRow/TotalAmountRow`）；差异仅在于三个价格行从缺省 null 变为有值（additive，TS 接口中已是可选字段）。响应示例见 §3.6。

**错误码**：40000（startTime/endTime 缺失或绑定失败）。业务查询现状不返回错误（空数据返回空结构），维持现状；DB 级故障由 GORM 日志暴露，本轮不为该端点新增 50000 路径（避免扩大改动面，登记 §5）。

---

## 4. 向后兼容评估

| # | 变更 | 兼容类型 | 对现有前端调用的影响 | 前端是否需同步改 |
| --- | --- | --- | --- | --- |
| 1 | `AddressInfo`/`ExceladdressInfo` 新增 `省份` | 纯新增 | 老调用忽略新键，无影响；AI 响应多一个键，`RecognitionResult` 索引签名兼容 | 是（本轮任务）：TS interface 双写 `省份/Province`、预览表与详情表加列 |
| 2 | `ExceladdressInfo` 新增 `id` | 纯新增 | 老调用忽略新键，无影响 | 是（本轮任务）：interface 加 `id?`；行操作按 `id>0` 启用编辑/删除 |
| 3 | query Likes 修正 + 加 province | 行为修复 | 搜索从「静默空结果」恢复为正常工作；等值过滤新增可选键 | 否（可选增强：搜索说明文案） |
| 4 | query 吞错修复（50000） | 行为修复 | 仅错误路径从 code 0 空列表变为 code 50000 报错提示 | 否（前端对 50000 已统一处理） |
| 5 | `GET /province-stats` | 新端点 | 无影响 | 是（新组件消费） |
| 6 | `POST /preview/audit` | 新端点 | 无影响 | 是（新按钮消费） |
| 7 | `PUT/DELETE /api/order/:id` | 新端点 | 无影响 | 是（启用被注释的操作列） |
| 8 | `/checkout/price` 持久化 | 行为增强 | 请求/响应结构不变；新增空数组/空字段校验（现状空数组静默返回原数据，新契约 40000——前端正常保存至少一行，实际不触发） | 否 |
| 9 | `/checkout/daydata` 日期/Count 修复 | 行为修复 | 统计数字更准（含首日、Count 正确）；结构不变 | 否（CheckTable 的 `totalAmountRow` 大小写 bug 是前端自身修复项，属前端任务） |
| 10 | `/checkout/daydata` 价格回填 | 纯新增（字段由 null→有值） | 打开页面即显示价格；TS 可选字段兼容 | 建议（顺手补全 CheckoutResponse interface，非阻塞） |
| 11 | 软删除启用 | 对读端点透明 | 已删行自动从全部查询/统计消失 | 否 |

**结论**：无任何破坏性契约变更（无字段删除/改名/类型变化/语义反转）；所有变更为新增、修复或行为增强。前端必须同步的仅有消费新能力的部分（#1、#2、#5、#6、#7），均为本轮前端任务清单内。

---

## 5. 待确认问题（需总负责人/相关角色确认）

| # | 问题 | 当前设计假设 | 影响 |
| --- | --- | --- | --- |
| Q1 | 前端 china.json 采用哪一版命名（传统 echarts 短名版 vs 阿里 DataV 全称版） | 已通过 `shortName` 字段解耦双版本风险 | 前端 Agent 选定后在联调文档登记；若两版都不满足再评估 |
| Q2 | 已软删订单是否需要「回收站/恢复」端点 | 本轮不做（DBA 手工恢复） | 误删恢复体验；若需要另起迭代 |
| Q3 | 修改订单是否允许改 `created_at`（订单归属日期） | 不允许（影响账单归属） | 若业务确需改单日期，需补充契约与对账说明 |
| Q4 | 手机号/收货地址是否需要格式强校验（如 11 位手机号） | 不校验（与现有宽松风格一致） | 脏数据风险由 AI 识别 + 人工审核兜底 |
| Q5 | 价格配置是否需要删除端点（如商品下架） | 本轮不做；若做须物理删除（软删行占用唯一索引，§1.3） | PriceConfig 生命周期管理 |
| Q6 | `/checkout/daydata` DB 故障是否返回 50000 | 维持现状不返回（避免扩大改动面） | B2 严格解读下属吞错，可另起小任务修复 |
| Q7 | `LastInputText` 按 previewType 覆盖/追加维护（§3.2 细化）是否认可 | 按此实现（否则追加模式下审核误报「多单」） | 后端实现 Agent 须知悉 |
| Q8 | 简称匹配「前 6 字符」限定的残余误判（如文本开头即他省简称） | 接受，人工改单兜底 + 回填幂等重跑 | 建议回填后按 §1.5.4 SQL 抽查 |

---

## 6. 契约变更影响范围与通知

本文档契约已同步登记至 `.trae/rules/api-contract.md`（第 6 节三行更新 + ExceladdressInfo 字段行更新；第 8 节由计划登记区转为正式契约区并附裁定记录）。后续任何契约变更须先改该规则文件再改代码（契约优先）。

需通知的角色：后端 Agent（全部端点与迁移）、前端 Agent（§4 兼容表 #1/#2/#5/#6/#7 及 CheckTable/Field 修复项）、测试 Agent（新端点正负例 + `ExtractProvince` 34 全称/简称/误判用例 + `BackfillProvince` 幂等二次运行用例）、运维 Agent（§1.6 迁移步骤与回滚授权边界）。
