package conf

type Config struct {
	Log      Log           `yaml:"log"`
	DB       []DB          `yaml:"db"`
	Email    Email         `yaml:"email"`
	AiModel  AiModel       `yaml:"aimodel"`
	SavePath SavePath      `yaml:"savePath"`
	Version  VersionConfig `yaml:"version"`
	Server   Server        `yaml:"server"`
}

type VersionConfig struct {
	Repo string `yaml:"repo"`
}

type Server struct {
	Port         int      `yaml:"port"`
	AllowOrigins []string `yaml:"allowOrigins"`
}
