package conf

import (
	"os"
	"strconv"
	"strings"

	"github.com/iceymoss/go-task/pkg/config"

	"github.com/spf13/viper"
)

type Config struct {
	Server ServerConfig `mapstructure:"server"`
	Jobs   []JobConfig  `mapstructure:"jobs"`
	Mysql  MysqlConfig  `mapstructure:"mysql"`
	Redis  RedisConfig  `mapstructure:"redis"`
}

type ServerConfig struct {
	Port string `mapstructure:"port"`
}

type MysqlConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
	LogLevel string `mapstructure:"log_level"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	Password string `mapstructure:"password"`
	Db       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"pool_size"`
}

type JobConfig struct {
	Name   string                 `mapstructure:"name"`
	Cron   string                 `mapstructure:"cron"`
	Enable bool                   `mapstructure:"enable"`
	Params map[string]interface{} `mapstructure:"params"`
}

// LoadConfig 加载配置
func LoadConfig(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	v.AutomaticEnv() // 自动读取环境变量

	// 允许环境变量替换 YAML 中的 ${VAR}
	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	// 显式展开环境变量
	for _, key := range v.AllKeys() {
		val := v.GetString(key)
		if strings.Contains(val, "${") {
			v.Set(key, os.ExpandEnv(val))
		}
	}

	var c Config
	if err := v.Unmarshal(&c); err != nil {
		return nil, err
	}

	protInt, err := strconv.Atoi(c.Mysql.Port)
	if err != nil {
		protInt = 3306
	}

	rdbPort, err := strconv.Atoi(c.Redis.Port)
	if err != nil {
		rdbPort = 6379
	}

	config.ServiceConf = &config.ServiceConfig{
		DB: config.MysqlConfig{
			Host:     c.Mysql.Host,
			Port:     protInt,
			Name:     c.Mysql.Database,
			User:     c.Mysql.User,
			Password: c.Mysql.Password,
			LogLevel: c.Mysql.LogLevel,
		},
		RedisDB: config.RedisConfig{
			Host:     c.Redis.Host,
			Port:     rdbPort,
			PassWord: c.Redis.Password,
		},
	}

	return &c, nil
}
