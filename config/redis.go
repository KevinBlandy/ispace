package config

import "time"

type Redis struct {
	Network        string        `yaml:"network"`
	Address        string        `yaml:"address"`
	Password       string        `yaml:"password"`
	Db             int           `yaml:"db"`
	ConnectTimeout time.Duration `yaml:"connect-timeout"`
	ReadTimeout    time.Duration `yaml:"read-timeout"`
	WriteTimeout   time.Duration `yaml:"write-timeout"`
	Pool           *RedisPool    `yaml:"pool"`
}

type RedisPool struct {
	MaxOpenConn int           `yaml:"max-open-conn"`
	MinIdleConn int           `yaml:"min-idle-conn"`
	MaxIdleConn int           `yaml:"max-idle-conn"`
	MaxIdleTime time.Duration `yaml:"max-idle-time"`
	MaxLifetime time.Duration `yaml:"max-life-time"`
	Timeout     time.Duration `yaml:"timeout"`
}
