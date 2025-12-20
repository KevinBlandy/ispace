package config

type Log struct {
	Level  string  `yaml:"level"` // 日志级别
	Dir    string  `yaml:"dir"`   // 目录
	OutPut *OutPut `yaml:"out-put"`
}

type OutPut struct {
	Std  bool `yaml:"std"`
	File bool `yaml:"file"`
}
