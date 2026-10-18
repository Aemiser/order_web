package models

import "time"

// 账号角色。当前只有两档：admin 可见全部功能，operator 为日常操作员。
// 角色只随令牌下发供前端渲染，后端未按角色做接口级鉴权——本系统是内部工具，
// 门禁的目标是「挡住未登录者」而非细分权限。真要分级时在这里加常量并补中间件。
const (
	RoleAdmin    = "admin"
	RoleOperator = "operator"
)

// User 登录账号（表 users）。
//
// 账号不开放自助注册：注册入口既不在前端也不在后端，由本地 tools/userctl
// 脚本直连数据库写入。因此本模型没有配套的 create/update HTTP 端点。
//
// Password 存 bcrypt 哈希（60 字符，size:100 留余量）。json:"-" 是双保险：
// 即便将来有人把整个 User 结构体直接塞进响应体，哈希也不会被序列化出去。
//
// 已知限制：Model 内嵌 gorm.Model 带来软删，被软删的行仍占用 username 唯一索引，
// 因此删除后无法用同一用户名重建——与 PriceConfig 登记的限制同源。
// userctl 脚本删除账号时走硬删（Unscoped）以规避此问题。
type User struct {
	Model
	Username string `gorm:"column:username;size:64;uniqueIndex:uk_user_username" json:"username"`
	Password string `gorm:"column:password;size:100" json:"-"`
	Nickname string `gorm:"column:nickname;size:64" json:"nickname"`
	Role     string `gorm:"column:role;size:32;default:operator" json:"role"`
	Enabled  bool   `gorm:"column:enabled;default:true" json:"enabled"`
	// 最近登录时间与来源 IP，仅用于运维排查，不下发给前端。
	LastLoginAt *time.Time `gorm:"column:last_login_at" json:"-"`
	LastLoginIP string     `gorm:"column:last_login_ip;size:64" json:"-"`
}
