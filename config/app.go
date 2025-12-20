package config

type App struct {
	Server     *Server     `yaml:"server"`
	Log        *Log        `yaml:"log"`
	DataSource *DataSource `yaml:"datasource"`
}

var app *App

// Get 获取全局配置信息
func Get() *App {
	if app == nil {
	}
	return app
}
