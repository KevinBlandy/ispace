package config

type Server struct {
	Address       string `yaml:"address"`
	MaxHeaderSize int    `yaml:"max-header-size"`
}
