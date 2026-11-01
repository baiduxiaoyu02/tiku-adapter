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
	APIFormat   string // API 协议格式：openai（默认）/ anthropic / responses
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

// versionedEndpoint 按协议拼接端点：/v1 后缀可省略（anthropic: /v1/messages，responses: /v1/responses）。
func versionedEndpoint(baseURL, path string) string {
	base := strings.TrimRight(baseURL, "/")
	if strings.HasSuffix(base, "/v1") {
		return base + path
	}
	return base + "/v1" + path
}

// parseDataURL 解析 data URL（OCS 发送 base64 图片的形式）。
// "data:image/png;base64,QAAA" → ("image/png", "QAAA", true)
func parseDataURL(u string) (mediaType, data string, ok bool) {
	const prefix = "data:"
	if !strings.HasPrefix(u, prefix) {
		return "", "", false
	}
	rest := u[len(prefix):]
	semi := strings.Index(rest, ";")
	comma := strings.Index(rest, ",")
	if semi < 0 || comma < semi {
		return "", "", false
	}
	mediaType = rest[:semi]
	payload := rest[semi+1:]
	if !strings.HasPrefix(payload, "base64,") {
		return "", "", false
	}
	data = payload[len("base64,"):]
	if mediaType == "" || data == "" {
		return "", "", false
	}
	return mediaType, data, true
}

// askModel 按客户端 API 格式调用模型，返回剥离 think 标签后的纯文本。
// 文字题 images 为空；图片题 images 为 URL（data URL 或 http URL）。
func (in *ChatGptClient) askModel(modelName, prompt string, images []string) (string, error) {
	switch in.APIFormat {
	case "anthropic":
		return in.callAnthropic(modelName, prompt, images)
	case "responses":
		return in.callResponses(modelName, prompt, images)
	default:
		return in.callOpenAI(modelName, prompt, images)
	}
}

// TestConnect 以给定模型做一次真实连通调用（供供应商管理测试接口复用协议路径）。
func (in *ChatGptClient) TestConnect(modelName string) (string, error) {
	return in.askModel(modelName, "回复OK", nil)
}

// callOpenAI OpenAI Chat Completions 协议（/v1/chat/completions，Bearer 头）。
func (in *ChatGptClient) callOpenAI(modelName, prompt string, images []string) (string, error) {
	var reqContent interface{}
	if len(images) > 0 {
		parts := []interface{}{
			map[string]string{"type": "text", "text": prompt},
		}
		for _, imgURL := range images {
			parts = append(parts, map[string]interface{}{
				"type":      "image_url",
				"image_url": map[string]string{"url": imgURL},
			})
		}
		reqContent = parts
	} else {
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
		SetHeader("Authorization", "Bearer "+in.APIKey).
		SetBody(reqBody).
		Post(chatEndpoint(in.BaseURL))
	if err != nil || resp.StatusCode() != 200 {
		return "", errors.ErrTargetServerError
	}
	var res chatResponse
	if err = json.Unmarshal(resp.Body(), &res); err != nil {
		return "", errors.ErrParserJSON
	}
	if len(res.Choices) == 0 {
		return "", nil
	}
	return stripThinkTags(res.Choices[0].Message.Content), nil
}

// callAnthropic Anthropic Messages 协议（/v1/messages，x-api-key 头，max_tokens 必填）。
func (in *ChatGptClient) callAnthropic(modelName, prompt string, images []string) (string, error) {
	var content interface{}
	if len(images) > 0 {
		blocks := []interface{}{
			map[string]string{"type": "text", "text": prompt},
		}
		for _, imgURL := range images {
			if mediaType, data, ok := parseDataURL(imgURL); ok {
				blocks = append(blocks, map[string]interface{}{
					"type": "image",
					"source": map[string]string{
						"type": "base64", "media_type": mediaType, "data": data,
					},
				})
			} else {
				blocks = append(blocks, map[string]interface{}{
					"type": "image",
					"source": map[string]string{
						"type": "url", "url": imgURL,
					},
				})
			}
		}
		content = blocks
	} else {
		content = prompt
	}
	reqBody := map[string]interface{}{
		"model":      modelName,
		"max_tokens": 4096,
		"messages": []map[string]interface{}{
			{"role": "user", "content": content},
		},
	}
	resp, err := in.getHTTPClient().R().
		SetHeader("Content-Type", "application/json").
		SetHeader("x-api-key", in.APIKey).
		SetHeader("anthropic-version", "2023-06-01").
		SetBody(reqBody).
		Post(versionedEndpoint(in.BaseURL, "/messages"))
	if err != nil || resp.StatusCode() != 200 {
		return "", errors.ErrTargetServerError
	}
	var res struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err = json.Unmarshal(resp.Body(), &res); err != nil {
		return "", errors.ErrParserJSON
	}
	var sb strings.Builder
	for _, b := range res.Content {
		if b.Type == "text" {
			sb.WriteString(b.Text)
		}
	}
	return stripThinkTags(sb.String()), nil
}

// callResponses OpenAI Responses 协议（/v1/responses，Bearer 头，input 格式）。
func (in *ChatGptClient) callResponses(modelName, prompt string, images []string) (string, error) {
	var input interface{}
	if len(images) > 0 {
		parts := []interface{}{
			map[string]string{"type": "input_text", "text": prompt},
		}
		for _, imgURL := range images {
			parts = append(parts, map[string]interface{}{
				"type":      "input_image",
				"image_url": imgURL,
			})
		}
		input = parts
	} else {
		input = prompt
	}
	reqBody := map[string]interface{}{
		"model":             modelName,
		"input":             input,
		"max_output_tokens": 4096,
	}
	resp, err := in.getHTTPClient().R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", "Bearer "+in.APIKey).
		SetBody(reqBody).
		Post(versionedEndpoint(in.BaseURL, "/responses"))
	if err != nil || resp.StatusCode() != 200 {
		return "", errors.ErrTargetServerError
	}
	var res struct {
		OutputText string `json:"output_text"`
		Output     []struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}
	if err = json.Unmarshal(resp.Body(), &res); err != nil {
		return "", errors.ErrParserJSON
	}
	if res.OutputText != "" {
		return stripThinkTags(res.OutputText), nil
	}
	var sb strings.Builder
	for _, o := range res.Output {
		for _, b := range o.Content {
			if b.Type == "output_text" {
				sb.WriteString(b.Text)
			}
		}
	}
	return stripThinkTags(sb.String()), nil
}

// SearchAnswer 调用模型搜索答案（题库 fallback）
func (in *ChatGptClient) SearchAnswer(req model.SearchRequest) (answer [][]string, err error) {
	answer = make([][]string, 0)
	if !in.Enable {
		return answer, nil
	}

	// 不使用 dummy 默认值：BaseURL/APIKey/Model 任一为空时直接返回空答案
	if in.BaseURL == "" || in.APIKey == "" || in.Model == "" {
		return answer, nil
	}
	prompt := buildPrompt(req)

	// 图片题：切换视觉模型；文字题：常规模型
	modelName := in.Model
	images := req.Images
	if len(images) > 0 {
		if in.VisionModel != "" {
			modelName = in.VisionModel
		}
		prompt += "\n（此题包含图片，请结合图片内容作答）"
	}

	content, err := in.askModel(modelName, prompt, images)
	if err != nil {
		return answer, err
	}
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
