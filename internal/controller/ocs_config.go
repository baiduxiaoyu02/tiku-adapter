package controller

import (
	_ "embed"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed ocs-config.html
var ocsConfigHTML []byte

// OcsConfig 返回内嵌的 OCS 配置生成页：页面 JS 用 location.origin + '/api/search'
// 动态生成完整 AnswererWrapper JSON（含 options/type/images/suggestion 处理器），
// 带「复制配置」按钮与默认解析器导入步骤说明（AC-2）。
func OcsConfig(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", ocsConfigHTML)
}
