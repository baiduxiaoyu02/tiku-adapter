package controller

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/itihey/tikuAdapter/internal/middleware"
	"github.com/itihey/tikuAdapter/internal/registry"
	"github.com/itihey/tikuAdapter/internal/registry/manager"
	"github.com/itihey/tikuAdapter/internal/search"
	"github.com/itihey/tikuAdapter/pkg/global"
	"github.com/itihey/tikuAdapter/pkg/logger"
	"github.com/itihey/tikuAdapter/pkg/model"
	"github.com/itihey/tikuAdapter/pkg/util"
	"github.com/spf13/viper"
)

// tokenParams 从 POST body 读取的 token 参数（与 URL query 并存，向后兼容）
type tokenParams struct {
	IcodefToken  string `json:"icodefToken"`
	WannengToken string `json:"wannengToken"`
	TikuhaiToken string `json:"tikuhaiToken"`
	EnncyToken   string `json:"enncyToken"`
	AidianYToken string `json:"aidianYToken"`
	LemonToken   string `json:"lemonToken"`
}

// get 优先从 URL query 读取，为空时回退到 POST body（向后兼容）
func (t *tokenParams) get(c *gin.Context, key string) string {
	if v := c.Query(key); v != "" {
		return v
	}
	switch key {
	case "icodefToken":
		return t.IcodefToken
	case "wannengToken":
		return t.WannengToken
	case "tikuhaiToken":
		return t.TikuhaiToken
	case "enncyToken":
		return t.EnncyToken
	case "aidianYToken":
		return t.AidianYToken
	case "lemonToken":
		return t.LemonToken
	}
	return ""
}

// AI 调用计数与熔断（日维度）
var (
	aiCallMu      sync.Mutex
	aiCallCount   int
	aiCallDate    string
	aiCircuitOpen bool
)

// aiCircuitBreakerCheck 检查 AI 熔断器状态，返回是否允许调用
// maxDailyCalls 为 0 表示无限制（向后兼容）
func aiCircuitBreakerCheck(maxDailyCalls int) bool {
	aiCallMu.Lock()
	defer aiCallMu.Unlock()
	today := time.Now().Format("2006-01-02")
	if today != aiCallDate {
		// 跨日重置计数
		aiCallDate = today
		aiCallCount = 0
		aiCircuitOpen = false
	}
	if maxDailyCalls > 0 && aiCallCount >= maxDailyCalls {
		aiCircuitOpen = true
		return false // 熔断，不允许调用
	}
	return true
}

// aiRecordCall 记录一次 AI 调用
func aiRecordCall() {
	aiCallMu.Lock()
	defer aiCallMu.Unlock()
	today := time.Now().Format("2006-01-02")
	if today != aiCallDate {
		aiCallDate = today
		aiCallCount = 0
	}
	aiCallCount++
}

// Search 搜题接口
func Search(c *gin.Context) {
	var req model.SearchRequest
	if err := c.ShouldBindBodyWith(&req, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, global.ErrorParam)
		return
	}

	// Token 支持从 POST body 读取，与 URL query 并存（向后兼容）
	var tokens tokenParams
	_ = c.ShouldBindBodyWith(&tokens, binding.JSON)

	use := c.Query("use")

	// 本地题库默认参与搜索（use 为空或含 local 时启用）
	var result [][]string
	var localAnswer [][]string

	if use == "" || strings.Contains(use, "local") {
		var err error
		localAnswer, err = search.GetDBSearch().SearchAnswer(req)
		if err != nil {
			logger.SysError(fmt.Sprintf("查询本地答案出错：%s", err.Error()))
		}
		result = append(result, localAnswer...)
	}

	// questionTooLarge 阈值从 config 读取（默认 5000，向后兼容）
	questionTooLargeThreshold := viper.GetInt("question_too_large_threshold")
	if questionTooLargeThreshold == 0 {
		questionTooLargeThreshold = 5000
	}
	questionTooLarge := len(req.Question) > questionTooLargeThreshold

	aiCfg := registry.GetAIConfig()

	// Phase 1: 非 AI 源并发查询（questionTooLarge 时跳过传统第三方搜索）
	if len(result) == 0 && !questionTooLarge {
		var nonAIClients = []search.Search{
			&search.BuguakeClient{
				Enable: strings.Contains(use, "buguake") || use == "",
			},
			&search.IcodefClient{
				Token:  tokens.get(c, "icodefToken"),
				Enable: strings.Contains(use, "icodef") || use == "",
			},
			&search.WannengClient{
				Token:  tokens.get(c, "wannengToken"),
				Enable: strings.Contains(use, "wanneng") || use == "",
			},
			&search.TikuhaiClient{
				Enable: strings.Contains(use, "tikuhai") || use == "",
				Token:  tokens.get(c, "tikuhaiToken"),
			},
			&search.EnncyClient{
				Token:  tokens.get(c, "enncyToken"),
				Enable: strings.Contains(use, "enncy"),
			},
			&search.AidianClient{
				Enable: strings.Contains(use, "aidian"),
				YToken: tokens.get(c, "aidianYToken"),
			},
			&search.LemonClient{
				Enable: strings.Contains(use, "lemon"),
				Token:  tokens.get(c, "lemonToken"),
			},
		}
		cfg := manager.GetManager().GetConfig()
		for _, api := range cfg.API {
			if strings.Contains(use, api.Name) {
				nonAIClients = append(nonAIClients, api)
			}
		}

		var wg sync.WaitGroup
		var mu sync.Mutex

		for i := range nonAIClients {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						logger.SysError(fmt.Sprintf("搜索客户端 panic recovered: %v", r))
					}
				}()
				res, err := nonAIClients[idx].SearchAnswer(req)
				if err == nil && len(res) > 0 {
					mu.Lock()
					defer mu.Unlock()
					result = append(result, res...)
				}
			}(i)
		}
		wg.Wait()
	}

	// Phase 2: AI fallback（两阶段：非 AI 源全部无果后才调用 ChatGPT）
	// LLM 路径不受 questionTooLarge 阈值限制
	if len(result) == 0 && aiCfg.Enable {
		maxDailyCalls := aiCfg.MaxDailyCalls
		if !aiCircuitBreakerCheck(maxDailyCalls) {
			logger.SysError("AI 熔断器已打开，当日调用次数已达上限，跳过 AI fallback")
		} else {
			chatGpt := &search.ChatGptClient{
				Enable:      aiCfg.Enable,
				BaseURL:     aiCfg.BaseURL,
				APIKey:      aiCfg.APIKey,
				Model:       aiCfg.Model,
				VisionModel: aiCfg.VisionModel,
			}
			res, err := chatGpt.SearchAnswer(req)
			if err != nil {
				logger.SysError(fmt.Sprintf("AI fallback 查询出错：%s", err.Error()))
			}
			if err == nil && len(res) > 0 {
				result = append(result, res...)
			}
			aiRecordCall()
		}
	}

	resp := util.FillAnswerResponse(result, &req)

	// 设置 reason 字段标识为何返回空答案
	if len(result) == 0 {
		if questionTooLarge {
			resp.Reason = "question too large, skipped traditional third-party search"
		} else if aiCfg.Enable && aiCircuitOpen {
			resp.Reason = "AI circuit breaker open, daily call limit exceeded"
		}
	}

	// 本地题库没有答案时收集答案（CollectAnswer 签名冻结：resp, courseName, extra）
	if len(localAnswer) == 0 && c.Query("noRecord") == "" {
		middleware.CollectAnswer(resp, req.CourseName, req.Extra)
	}
	c.JSON(http.StatusOK, resp)
}
