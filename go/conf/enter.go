package conf

type Config struct {
	Log      Log      `yaml:"log"`
	DB       []DB     `yaml:"db"`
	Email    Email    `yaml:"email"`
	Deepseek Deepseek `yaml:"deepseek"`
	SavePath SavePath `yaml:"savePath"`
}
