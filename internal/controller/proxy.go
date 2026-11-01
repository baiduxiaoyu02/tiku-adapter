package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/itihey/tikuAdapter/internal/registry"
	"net/http"
	"net/http/httputil"
	"net/url"
)

// Proxy 反向代理，目标 URL 从配置文件 proxy_target_url 读取（替代硬编码 base64）
func Proxy(c *gin.Context) {
	// 从配置读取代理目标 URL（t4 在 configs/config.go 添加 ProxyTargetURL 字段）
	targetURL := registry.Config().ProxyTargetURL
	if targetURL == "" {
		c.JSON(http.StatusBadGateway, gin.H{
			"message": "代理目标未配置，请在 config.yaml 中设置 proxy_target_url",
		})
		return
	}

	target, err := url.Parse(targetURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "代理目标 URL 解析失败",
		})
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	proxy.Director = func(req *http.Request) {
		req.Host = target.Host
		req.Body = c.Request.Body
		req.Method = c.Request.Method
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		// 不再注入硬编码管理员 Cookie 伪造身份；反向代理透传客户端自带的
		// Cookie/Header，目标服务的鉴权由客户端自行携带凭证完成。
	}
	proxy.ServeHTTP(c.Writer, c.Request)
}
