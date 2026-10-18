package conf

type Config struct {
	Log      Log      `yaml:"log"`
	DB       []DB     `yaml:"db"`
	Email    Email    `yaml:"email"`
	AiModel  AiModel  `yaml:"aimodel"`
	SavePath SavePath `yaml:"savePath"`
	Server   Server   `yaml:"server"`
	Auth     Auth     `yaml:"auth"`
}

type Server struct {
	Port         int      `yaml:"port"`
	AllowOrigins []string `yaml:"allowOrigins"`
}
