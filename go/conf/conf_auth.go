package conf

// Auth 登录鉴权配置，对应 settings.yaml 的 auth 块。
//
// 注意 settings.yaml 不随部署包下发（CI 只同步 aimap.json 与 systemPrompt.txt），
// 因此新增本块后必须在服务器的 settings.yaml 上手工补齐，否则服务会以
// 随机密钥启动——不会出安全漏洞，但每次重启所有人都会被登出。
type Auth struct {
	// Secret JWT HMAC-SHA256 签名密钥，建议 32 字节以上的随机串。
	// 留空时 core.InitAuth 会在启动期生成随机密钥并打警告：随机密钥不可预测，
	// 所以不存在「空密钥可被伪造令牌」的问题，代价仅是重启即失效。
	Secret string `yaml:"secret"`
	// ExpireHours 令牌有效期（小时）。<=0 时取 DefaultExpireHours。
	ExpireHours int `yaml:"expireHours"`
	// Issuer 签发者标识，解析时校验，用于确认令牌出自本服务。留空取 DefaultIssuer。
	Issuer string `yaml:"issuer"`
}
