package api

import (
	"embed"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/itihey/tikuAdapter/internal/controller"
	"github.com/itihey/tikuAdapter/internal/middleware"
	"io/fs"
	"net/http"
)

// SetAPIRouter 设置API路由
func SetAPIRouter(router *gin.Engine) {

	router.Any("/sqp/*path", controller.Proxy)

	apiRouter := router.Group("/adapter-service")

	apiRouter.Use(middleware.GlobalAPIRateLimit) // 全局限流

	apiRouter.POST("/upload", controller.UploadFile)
	apiRouter.GET("/plat", controller.Plat)

	apiRouter.POST("/search", controller.Search)
	apiRouter.POST("/parser", controller.Parser)

	apiRouter.GET("/courses", controller.Auth, controller.Courses)
	apiRouter.POST("/questions/search", controller.Auth, controller.GetQuestions)
	apiRouter.POST("/questions", controller.Auth, controller.CreateQuestion)
	apiRouter.PUT("/questions/:id", controller.Auth, controller.UpdateQuestions)
	apiRouter.DELETE("/questions/:id", controller.Auth, controller.DeleteQuestion)

	apiRouter.POST("/user/login", controller.UserLogin)                   // 登录接口
	apiRouter.GET("/user", controller.Auth, controller.UserList)          // 自己创建的用户列表
	apiRouter.POST("/user", controller.Auth, controller.CreateUser)       // 创建用户
	apiRouter.DELETE("/user/:id", controller.Auth, controller.DeleteUser) // 删除用户

	apiRouter.GET("/logs", controller.Auth, controller.LogList) // 日志列表

	// /api/search 路由别名：与 /adapter-service/search 复用同一 Search 处理器。
	// OCS 默认解析器硬拦截含 adapter-service/search 字符串的配置，别名路由绕开该字符串；
	// GET/HEAD 探活返回 200，POST 搜题。
	router.GET("/api/search", apiSearchProbe)
	router.HEAD("/api/search", apiSearchProbe)
	router.POST("/api/search", controller.Search)

	// /ocs-config：内嵌 OCS 配置生成页，按本机地址展示可导入的题库配置 JSON
	router.GET("/ocs-config", controller.OcsConfig)

	// /providers：模型供应商管理页与配置 API（本机回环使用，与搜题接口一致不挂 Auth）
	router.GET("/providers", controller.ProvidersPage)
	router.GET("/adapter-service/providers", controller.ProvidersList)
	router.POST("/adapter-service/providers/save", controller.ProvidersSave)
	router.POST("/adapter-service/providers/switch", controller.ProvidersSwitch)
	router.POST("/adapter-service/providers/delete", controller.ProvidersDelete)
	router.POST("/adapter-service/providers/test", controller.ProvidersTest)
}

// apiSearchProbe /api/search 探活处理器
func apiSearchProbe(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok"})
}

type embedFileSystem struct {
	http.FileSystem
}

// Exists 判断文件是否存在
func (e embedFileSystem) Exists(_ string, path string) bool {
	_, err := e.Open(path)
	return err == nil
}

func embedFolder(fsEmbed embed.FS, targetPath string) static.ServeFileSystem {
	efs, err := fs.Sub(fsEmbed, targetPath)
	if err != nil {
		panic(err)
	}
	return embedFileSystem{
		FileSystem: http.FS(efs),
	}
}

// SetWebRouter 设置web路由
func SetWebRouter(buildFS embed.FS, indexPage []byte, router *gin.Engine) {
	router.Use(static.Serve("/", embedFolder(buildFS, "dist")))
	router.NoRoute(func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexPage)
	})
}
