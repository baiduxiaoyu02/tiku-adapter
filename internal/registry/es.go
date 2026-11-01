package registry

import (
	"github.com/itihey/tikuAdapter/configs"
	"github.com/itihey/tikuAdapter/internal/service"
)

// RegisterEs 注册配置文件
func RegisterEs(cfg configs.Config) service.Elasticsearch {
	addrs := cfg.Elasticsearch.Addresses()
	if len(addrs) == 0 {
		return nil
	}
	client, err := service.NewElasticsearchClient(addrs)
	if err != nil {
		return nil
	}
	return client
}
