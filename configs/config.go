package configs

import (
	"github.com/itihey/tikuAdapter/internal/search"
)

// Config 所有的配置文件
type Config struct {
	Limit             LimitConfig    `yaml:"limit"`
	API               []search.API   `yaml:"api"`
	Elasticsearch     ESConfig       `yaml:"elasticsearch"`
	RecordEmptyAnswer bool           `yaml:"recordEmptyAnswer"`
	Mysql             string         `yaml:"mysql"`
	Database          DatabaseConfig `yaml:"database"`
	OSS               OSSConfig      `yaml:"oss"`
	Plat              []PlatConfig   `yaml:"plat"`
	JWTSecret         string         `yaml:"jwt_secret"`
	ServerPort        string         `yaml:"server_port"`
	ProxyTargetURL    string         `yaml:"proxy_target_url"`
}

// ESConfig Elasticsearch 配置（自定义结构体，yaml 用 url 字段匹配 config.yaml）
type ESConfig struct {
	URL string `yaml:"url"`
}

// Addresses 返回 Elasticsearch 地址列表，URL 为空时返回 nil 表示未配置
func (c ESConfig) Addresses() []string {
	if c.URL == "" {
		return nil
	}
	return []string{c.URL}
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Path string `yaml:"path"` // SQLite数据库文件路径，默认为 tiku.db（向后兼容），Docker部署建议使用 data/tiku.db
}

// LimitConfig 限流配置
type LimitConfig struct {
	Enable   bool   `yaml:"enable"`
	Duration uint   `yaml:"duration"`
	Requests uint64 `yaml:"requests"`
}

// OSSConfig 阿里云oss配置
type OSSConfig struct {
	EndPoint        string `yaml:"endpoint"`
	AccessKeyID     string `yaml:"accessKeyID"`
	AccessKeySecret string `yaml:"accessKeySecret"`
	BucketName      string `yaml:"bucketName"`
}

// PlatConfig 自定义的平台类型
type PlatConfig struct {
	Label string `yaml:"label"`
	Value string `yaml:"value"`
}
