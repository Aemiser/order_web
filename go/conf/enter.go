package conf

type Config struct {
	Log      Log      `yaml:"log"`
	DB       []DB     `yaml:"db"`
	Email    Email    `yaml:"email"`
	AiModel  AiModel  `yaml:"aimodel"`
	SavePath SavePath `yaml:"savePath"`
}
