package conf

import (
	"os"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server ServerConfig `mapstructure:"server"`
	Jobs   []JobConfig  `mapstructure:"jobs"`
}

type ServerConfig struct {
	Port string `mapstructure:"port"`
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
	return &c, nil
}
