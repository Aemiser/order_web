package global

import (
	"order_server/go/conf"
	"order_server/go/models"

	"gorm.io/gorm"
)

const (
	Version = "10.0.1"
)

var (
	Config           *conf.Config
	Db               *gorm.DB
	AddressList      []models.AddressInfo
	ProductBelongMap = map[string][]string{}
	AiMap            = map[string]int{}
	SystemPrompt     string
	// LastInputText 最近一次识别的原始输入文本（审核对比基准，契约 §3.2/架构 §5.4）。
	// B6 内存态：重启即失、审核只读；由 server/order 识别统一入口按 previewType
	// 与 AddressList 同步维护（RE 覆盖 / ADD 追加），单请求流假设下无并发写（R8）。
	LastInputText string
	// RecognitionInFlight 识别在途计数（2026-10-17 缺陷2修复，sync/atomic 访问）：
	// server/order 识别入口（OrderRec/StreamOrderRec）进入 +1、退出 -1；
	// audit 入口（server/order/audit.go AuditPreviewData）Load 判断，>0 时拒绝审核。
	// 动机：识别期间 recordLastInputText 先行更新 LastInputText，而 AddressList
	// 在 AI 返回后才写入/写完——在途窗口内两者不一致，此时审核会把未落定的
	// 新增批判为 missing 误报。计数器而非布尔：并发识别（连点多次）时不会提前放行。
	RecognitionInFlight int32
)
