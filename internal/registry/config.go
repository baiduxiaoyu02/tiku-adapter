package registry

import (
	"bytes"
	"fmt"
	"github.com/itihey/tikuAdapter/configs"
	"github.com/itihey/tikuAdapter/pkg/logger"
	"github.com/spf13/viper"
)

// AIConfig AI 大模型配置（题库无答案时 fallback 到大模型）
type AIConfig struct {
	Enable        bool   `yaml:"enable"`           // 是否启用 AI fallback
	BaseURL       string `yaml:"base_url"`         // API 地址（如 http://127.0.0.1:8317）
	APIKey        string `yaml:"api_key"`          // API Key
	Model         string `yaml:"model"`            // 模型名称（如 glm-5.2）
	VisionModel  string `yaml:"vision_model"`     // 图片题用的视觉模型（如 sensenova-6.8-flash-lite），为空则用 Model
	APIFormat     string `yaml:"api_format"`       // API 协议格式：openai（默认）/ anthropic / responses
	MaxDailyCalls int   `yaml:"max_daily_calls"`  // 每日最大调用次数，0 表示无限制
}

// GetAIConfig 从 viper 读取 AI 配置（config.yaml 的 ai 段）
func GetAIConfig() AIConfig {
	return AIConfig{
		Enable:         viper.GetBool("ai.enable"),
		BaseURL:        viper.GetString("ai.base_url"),
		APIKey:         viper.GetString("ai.api_key"),
		Model:          viper.GetString("ai.model"),
		VisionModel:    viper.GetString("ai.vision_model"),
		APIFormat:      viper.GetString("ai.api_format"),
		MaxDailyCalls:  viper.GetInt("ai.max_daily_calls"),
	}
}

var defaultConfig = []byte(`
limit:
  enable: false # 是否开启
  duration: 3  # 时间窗口为3秒
  requests: 1  # 允许用户在3秒内通过1个请求
`)

// Config get config instance
func Config() configs.Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		logger.SysLog(fmt.Sprintf("无法读取配置文件: %v,将以默认配置启动", err))
		_ = viper.ReadConfig(bytes.NewBuffer(defaultConfig))
	}

	var config configs.Config
	err := viper.Unmarshal(&config)
	if err != nil {
		logger.SysError(fmt.Sprintf("配置文件语法错误: %v", err))
	}
	return config
}
