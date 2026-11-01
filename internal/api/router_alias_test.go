package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/itihey/tikuAdapter/internal/registry/manager"
)

// TestMain 把工作目录切到临时目录（避免在源码树内生成 sqlite 数据文件），
// 并初始化 Manager：限流中间件与 Search 控制器都依赖 manager 非 nil。
func TestMain(m *testing.M) {
	oldWd, _ := os.Getwd()
	tmp, err := os.MkdirTemp("", "tiku-api-router-test")
	if err != nil {
		fmt.Println("创建临时目录失败:", err)
		os.Exit(1)
	}
	if err = os.Chdir(tmp); err != nil {
		fmt.Println("切换工作目录失败:", err)
		os.Exit(1)
	}
	if _, err = manager.CreateManager(); err != nil {
		fmt.Println("初始化 manager 失败:", err)
		os.Exit(1)
	}
	code := m.Run()
	_ = os.Chdir(oldWd)
	_ = os.RemoveAll(tmp) // 只清理本轮自建的临时目录
	os.Exit(code)
}

func newAliasTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	SetAPIRouter(r)
	return r
}

// TestAPISearchAliasProbe OCS 探活：GET 与 HEAD /api/search 都必须返回 200。
func TestAPISearchAliasProbe(t *testing.T) {
	r := newAliasTestRouter()

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/search", nil))
	if w.Code != http.StatusOK {
		t.Errorf("GET /api/search 探活期望 200，实际 %d", w.Code)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodHead, "/api/search?t=1790000000000", nil))
	if w.Code != http.StatusOK {
		t.Errorf("HEAD /api/search 探活期望 200，实际 %d", w.Code)
	}
}

// TestAPISearchAliasIsomorphic /api/search 必须与 /adapter-service/search 同构：
// 同一请求经两条路由返回相同状态码与相同 JSON 结构。
func TestAPISearchAliasIsomorphic(t *testing.T) {
	r := newAliasTestRouter()

	doPost := func(url, body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, url, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		return w
	}
	assertSameResponse := func(t *testing.T, wAlias, wOrigin *httptest.ResponseRecorder) {
		t.Helper()
		if wAlias.Code != wOrigin.Code {
			t.Errorf("两条路由状态码不同构：/api/search=%d，/adapter-service/search=%d", wAlias.Code, wOrigin.Code)
		}
		var aliasResp, originResp map[string]any
		if err := json.Unmarshal(wAlias.Body.Bytes(), &aliasResp); err != nil {
			t.Fatalf("/api/search 响应不是 JSON：%v，body=%s", err, wAlias.Body.String())
		}
		if err := json.Unmarshal(wOrigin.Body.Bytes(), &originResp); err != nil {
			t.Fatalf("/adapter-service/search 响应不是 JSON：%v，body=%s", err, wOrigin.Body.String())
		}
		if !reflect.DeepEqual(aliasResp, originResp) {
			t.Errorf("两条路由响应不同构：/api/search=%s，/adapter-service/search=%s", wAlias.Body.String(), wOrigin.Body.String())
		}
	}

	t.Run("正常请求同构", func(t *testing.T) {
		// use=none 跳过本地题库与全部第三方源，noRecord=1 跳过答案收集，路径无外部依赖
		body := `{"question":"1+1等于几?","options":["1","2"],"type":0}`
		assertSameResponse(t,
			doPost("/api/search?use=none&noRecord=1", body),
			doPost("/adapter-service/search?use=none&noRecord=1", body))
	})

	t.Run("非法请求同构", func(t *testing.T) {
		body := `{invalid json`
		assertSameResponse(t,
			doPost("/api/search", body),
			doPost("/adapter-service/search", body))
	})
}
