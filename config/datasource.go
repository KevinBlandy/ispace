package config

import "time"

// DataSource 数据源配置
type DataSource struct {
	Type string          `yaml:"type"`
	Url  string          `yaml:"url"`
	Pool *DataSourcePool `yaml:"pool"`
}

// DataSourcePool 数据源连接池
type DataSourcePool struct {
	MaxIdleTime time.Duration `yaml:"max-idle-time"`
	MaxLifetime time.Duration `yaml:"max-lifetime"`
	MaxIdleConn int           `yaml:"max-idle-conn"`
	MaxOpenConn int           `yaml:"max-open-conn"`
}
