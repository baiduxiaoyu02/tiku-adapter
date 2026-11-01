package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestOcsConfigRoute 绑定测试：GET /ocs-config 已注册并返回内嵌 HTML 配置生成页
// （页面展示 AnswererWrapper JSON：url=location.origin+'/api/search'，含
// options/type/images/suggestion 处理器、「复制配置」按钮与导入步骤说明）。
func TestOcsConfigRoute(t *testing.T) {
	r := newAliasTestRouter()

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ocs-config", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("GET /ocs-config 期望 200，实际 %d，body=%s", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Errorf("Content-Type 期望含 text/html，实际 %s", ct)
	}
	body := w.Body.String()
	// images handler 字符串（OCS 图片支持检测依据，AC-2）
	if !strings.Contains(body, "env.images") {
		t.Errorf("页面缺少 images handler（env.images）")
	}
	if !strings.Contains(body, "env.options") {
		t.Errorf("页面缺少 options handler")
	}
	if !strings.Contains(body, "env.type === 'single'") {
		t.Errorf("页面缺少 type handler")
	}
	if !strings.Contains(body, "suggestion_title") {
		t.Errorf("页面缺少 suggestion 字段")
	}
	// url 按本机地址动态生成为 location.origin + '/api/search'
	if !strings.Contains(body, "location.origin") || !strings.Contains(body, "/api/search") {
		t.Errorf("页面缺少 location.origin + '/api/search' 的 url 生成")
	}
	// 「复制配置」按钮
	if !strings.Contains(body, "复制配置") {
		t.Errorf("页面缺少「复制配置」按钮")
	}
	// 默认解析器导入步骤说明（读取剪贴板流程）
	if !strings.Contains(body, "读取剪贴板") {
		t.Errorf("页面缺少导入步骤说明（读取剪贴板）")
	}
}
