package main

import (
	"embed"
	"github.com/gin-gonic/gin"
	"github.com/itihey/tikuAdapter/internal/api"
	"github.com/itihey/tikuAdapter/internal/registry/manager"
	"github.com/itihey/tikuAdapter/internal/service/timer"
	"github.com/itihey/tikuAdapter/pkg/logger"
)

//go:embed dist
var buildFS embed.FS

//go:embed dist/index.html
var indexPage []byte

func main() {
	mg, err := manager.CreateManager()
	if err != nil {
		logger.FatalLog(err)
	}
	defer func(mg *manager.Manager) {
		err := mg.CloseManager()
		if err != nil {
			logger.FatalLog(err)
		}
	}(mg)

	server := gin.Default()
	api.SetAPIRouter(server)
	api.SetWebRouter(buildFS, indexPage, server)
	timer.StartTimer()
	// 从配置读取端口，默认 0.0.0.0:8060（t4 在 configs/config.go 添加 ServerPort 字段）
	port := mg.GetConfig().ServerPort
	if port == "" {
		port = "0.0.0.0:8060"
	}
	err = server.Run(port)
	if err != nil {
		logger.FatalLog(err)
	}
}
