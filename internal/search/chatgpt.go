package search

import (
	"encoding/json"
	"github.com/go-resty/resty/v2"
	"github.com/itihey/tikuAdapter/pkg/errors"
	"github.com/itihey/tikuAdapter/pkg/model"
	"regexp"
	"strings"
	"time"
)

// ChatGptClient ChatGPT AI 搜题客户端（题库无答案时 fallback 到大模型）
type ChatGptClient struct {
	Enable      bool   // 是否启用
	BaseURL     string // OpenAI 兼容 API 地址（如 http://127.0.0.1:8317）
	APIKey      string // API Key
	Model       string // 模型名称（如 glm-5.2）
	VisionModel string // 图片题用的视觉模型（如 sensenova-6.8-flash-lite），为空则用 Model
}

// getHTTPClient 获取HTTP客户端（含重试退避，429/5xx 自动重试）
func (in *ChatGptClient) getHTTPClient() *resty.Client {
	return resty.New().
		SetTimeout(30 * time.Second).
		SetRetryCount(2).
		SetRetryWaitTime(1 * time.Second).
		SetRetryMaxWaitTime(10 * time.Second).
		AddRetryCondition(func(r *resty.Response, err error) bool {
			return err != nil || r.StatusCode() == 429 || r.StatusCode() >= 500
		})
}

// chatRequest OpenAI 兼容请求体
type chatRequest struct {
	Model     string    `json:"model"`
	Messages  []chatMsg `json:"messages"`
	MaxTokens int       `json:"max_tokens,omitempty"` // 推理模型（如 glm-5.2）需要足够预算，否则返回空正文
}

type chatMsg struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"` // 纯文字时为 string，图片题时为 []interface{}（OpenAI 多模态格式）
}

// chatResponse OpenAI 兼容响应体
type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// thinkTagRegex 匹配 <think>...</think> 等推理标签
var thinkTagRegex = regexp.MustCompile(`(?s)<think>.*?</think>`)

// buildPrompt 按题型定制 Prompt
func buildPrompt(req model.SearchRequest) string {
	var sb strings.Builder
	sb.WriteString("你是一个答题助手，请根据题目给出正确答案。\n")
	sb.WriteString("题目：" + req.Question + "\n")

	if len(req.Options) > 0 {
		sb.WriteString("选项：\n")
		for i, opt := range req.Options {
			sb.WriteString(string(rune('A'+i)) + ". " + opt + "\n")
		}
	}

	switch req.Type {
	case 0: // 单选
		sb.WriteString("请只输出正确选项的字母（如A），不要输出其他内容。")
	case 1: // 多选
		sb.WriteString("请只输出正确选项的字母，多个选项用#分隔（如A#B#C），不要输出其他内容。")
	case 2: // 填空
		sb.WriteString("请直接输出填空的答案文本，不要输出其他内容。")
	case 3: // 判断
		sb.WriteString("请判断对错，只输出\"对\"或\"错\"，不要输出其他内容。")
	default: // 问答等
		sb.WriteString("请直接输出答案，不要输出其他内容。")
	}
	return sb.String()
}

// stripThinkTags 去除 think 等推理标签
func stripThinkTags(s string) string {
	s = thinkTagRegex.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

// parseMultiAnswer 多策略解析多选答案：先 # 分隔，再中英文逗号，再空格，最后正则提取连续字母
func parseMultiAnswer(content string) []string {
	// 策略 1：# 分隔
	if strings.Contains(content, "#") {
		return splitTrimFilter(content, "#")
	}
	// 策略 2：中英文逗号分隔
	if strings.Contains(content, ",") || strings.Contains(content, "，") {
		re := regexp.MustCompile(`[,，]`)
		return trimFilter(re.Split(content, -1))
	}
	// 策略 3：空格分隔（至少两个 token）
	parts := strings.Fields(content)
	if len(parts) > 1 {
		return parts
	}
	// 策略 4：正则提取连续字母 [A-Za-z]
	letters := regexp.MustCompile(`[A-Za-z]`).FindAllString(content, -1)
	if len(letters) > 1 {
		return letters
	}
	// 兜底：单值
	return []string{content}
}

// splitTrimFilter 按 sep 切分，trim 每个元素，过滤空串
func splitTrimFilter(s, sep string) []string {
	return trimFilter(strings.Split(s, sep))
}

// trimFilter trim 每个元素并过滤空串
func trimFilter(parts []string) []string {
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	if len(result) == 0 {
		return []string{}
	}
	return result
}

// mapLetterToOption 将字母答案映射为 req.Options 下标对应选项文本。
// 若答案不是字母（如判断题的"对"/"错"），原样返回。
func mapLetterToOption(answer string, options []string) string {
	answer = strings.TrimSpace(answer)
	upper := strings.ToUpper(answer)
	if len(upper) >= 1 && upper[0] >= 'A' && upper[0] <= 'Z' {
		idx := int(upper[0] - 'A')
		if idx < len(options) {
			return options[idx]
		}
	}
	return answer
}

// chatEndpoint 拼接 chat completions 端点。
// 兼容 base_url 两种写法：带 /v1 后缀（OpenAI 惯例，如 https://api.openai.com/v1）
// 或不带（如 https://token.sensenova.cn）——两者都得到正确的 /v1/chat/completions 路径。
func chatEndpoint(baseURL string) string {
	base := strings.TrimRight(baseURL, "/")
	if strings.HasSuffix(base, "/v1") {
		return base + "/chat/completions"
	}
	return base + "/v1/chat/completions"
}

// SearchAnswer 调用 OpenAI 兼容 API 搜索答案（题库 fallback）
func (in *ChatGptClient) SearchAnswer(req model.SearchRequest) (answer [][]string, err error) {
	answer = make([][]string, 0)
	if !in.Enable {
		return answer, nil
	}

	// 不使用 dummy 默认值：BaseURL/APIKey/Model 任一为空时直接返回空答案
	if in.BaseURL == "" || in.APIKey == "" || in.Model == "" {
		return answer, nil
	}
	baseURL := in.BaseURL
	apiKey := in.APIKey
	prompt := buildPrompt(req)

	// 图片题：切换视觉模型 + OpenAI 多模态消息格式；文字题：常规模型 + 纯文本
	var modelName string
	var reqContent interface{}

	if len(req.Images) > 0 && in.VisionModel != "" {
		modelName = in.VisionModel
		parts := []interface{}{
			map[string]string{"type": "text", "text": prompt + "\n（此题包含图片，请结合图片内容作答）"},
		}
		for _, imgURL := range req.Images {
			parts = append(parts, map[string]interface{}{
				"type":      "image_url",
				"image_url": map[string]string{"url": imgURL},
			})
		}
		reqContent = parts
	} else {
		modelName = in.Model
		reqContent = prompt
	}

	reqBody := chatRequest{
		Model: modelName,
		Messages: []chatMsg{
			{Role: "user", Content: reqContent},
		},
		MaxTokens: 4096,
	}

	resp, err := in.getHTTPClient().R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", "Bearer "+apiKey).
		SetBody(reqBody).
		Post(chatEndpoint(baseURL))
	if err != nil {
		return answer, errors.ErrTargetServerError
	}
	// HTTP 状态码检查：401/403/429/5xx 等不应静默返回空答案
	if resp.StatusCode() != 200 {
		return answer, errors.ErrTargetServerError
	}

	var res chatResponse
	err = json.Unmarshal(resp.Body(), &res)
	if err != nil {
		return answer, errors.ErrParserJSON
	}

	if len(res.Choices) == 0 {
		return answer, nil
	}

	content := stripThinkTags(res.Choices[0].Message.Content)
	content = strings.TrimSpace(content)
	if content == "" {
		return answer, nil
	}

	// 按题型解析答案
	var as []string
	switch req.Type {
	case 1: // 多选：多策略解析
		as = parseMultiAnswer(content)
	default: // 单选、判断、填空、问答：单值
		as = []string{content}
	}

	// 对选择题（type 0 单选 / type 1 多选），将字母答案映射为 req.Options 下标对应选项文本
	if (req.Type == 0 || req.Type == 1) && len(req.Options) > 0 {
		mapped := make([]string, 0, len(as))
		for _, a := range as {
			mapped = append(mapped, mapLetterToOption(a, req.Options))
		}
		as = mapped
	}

	if len(as) > 0 {
		answer = append(answer, as)
	}
	return answer, nil
}
