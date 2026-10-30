package conf

type Config struct {
	Log      Log      `yaml:"log"`
	DB       []DB     `yaml:"db"`
	Email    Email    `yaml:"email"`
	AiModel  AiModel  `yaml:"aimodel"`
	SavePath SavePath `yaml:"savePath"`
	Server   Server   `yaml:"server"`
	Auth     Auth     `yaml:"auth"`
	// DbWuyin 吴饮（吾饮良品）独立数据库，单连接（无读写分离需求）；
	// 未配置（Empty）时吴饮端点统一报错，一点点链路不受影响。
	DbWuyin DB `yaml:"dbWuyin"`
}

type Server struct {
	Port         int      `yaml:"port"`
	AllowOrigins []string `yaml:"allowOrigins"`
}
