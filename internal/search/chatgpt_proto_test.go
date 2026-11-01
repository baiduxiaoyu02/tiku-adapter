package search

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/itihey/tikuAdapter/pkg/model"
)

// TestAskModelOpenAIRegression OpenAI 格式回归：端点、Bearer 头、消息体、响应解析
func TestAskModelOpenAIRegression(t *testing.T) {
	var gotPath, gotAuth string
	var gotBody chatRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"A"}}]}`))
	}))
	defer srv.Close()

	client := &ChatGptClient{Enable: true, BaseURL: srv.URL, APIKey: "sk-test", Model: "m1"}
	text, err := client.askModel("m1", "题目", nil)
	if err != nil {
		t.Fatalf("askModel err: %v", err)
	}
	if text != "A" {
		t.Fatalf("text = %q, want A", text)
	}
	if gotPath != "/v1/chat/completions" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotAuth != "Bearer sk-test" {
		t.Fatalf("auth = %q", gotAuth)
	}
	if gotBody.Model != "m1" || len(gotBody.Messages) != 1 {
		t.Fatalf("body = %+v", gotBody)
	}
}

// TestAskModelAnthropic Anthropic Messages 格式：x-api-key 头、消息结构、content[].text 解析
func TestAskModelAnthropic(t *testing.T) {
	var gotPath, gotKey, gotVer string
	var raw map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotKey = r.Header.Get("x-api-key")
		gotVer = r.Header.Get("anthropic-version")
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &raw)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"答案"},{"type":"text","text":"B"}]}`))
	}))
	defer srv.Close()

	client := &ChatGptClient{Enable: true, BaseURL: srv.URL, APIKey: "sk-ant", Model: "m1", APIFormat: "anthropic"}
	text, err := client.askModel("m2", "题目", nil)
	if err != nil {
		t.Fatalf("askModel err: %v", err)
	}
	if text != "答案B" {
		t.Fatalf("text = %q, want 答案B", text)
	}
	if gotPath != "/v1/messages" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotKey != "sk-ant" {
		t.Fatalf("x-api-key = %q", gotKey)
	}
	if gotVer == "" {
		t.Fatalf("missing anthropic-version header")
	}
	if raw["model"] != "m2" {
		t.Fatalf("model = %v", raw["model"])
	}
	if _, ok := raw["max_tokens"]; !ok {
		t.Fatalf("max_tokens required by anthropic")
	}
	msgs, _ := raw["messages"].([]interface{})
	if len(msgs) != 1 {
		t.Fatalf("messages = %v", msgs)
	}
}

// TestAskModelAnthropicImage Anthropic 图片题：data URL 转 base64 source
func TestAskModelAnthropicImage(t *testing.T) {
	var contentArr []interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		_ = json.Unmarshal(b, &req)
		msgs, _ := req["messages"].([]interface{})
		if len(msgs) > 0 {
			m, _ := msgs[0].(map[string]interface{})
			contentArr, _ = m["content"].([]interface{})
		}
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"图上是中国国旗"}]}`))
	}))
	defer srv.Close()

	client := &ChatGptClient{Enable: true, BaseURL: srv.URL, APIKey: "k", Model: "m", APIFormat: "anthropic"}
	text, err := client.askModel("m", "这是什么", []string{"data:image/png;base64,QAAA"})
	if err != nil {
		t.Fatalf("askModel err: %v", err)
	}
	if text != "图上是中国国旗" {
		t.Fatalf("text = %q", text)
	}
	if len(contentArr) != 2 {
		t.Fatalf("content blocks = %d, want 2", len(contentArr))
	}
	img, _ := contentArr[1].(map[string]interface{})
	if img["type"] != "image" {
		t.Fatalf("block type = %v", img["type"])
	}
	src, _ := img["source"].(map[string]interface{})
	if src["type"] != "base64" || src["media_type"] != "image/png" || src["data"] != "QAAA" {
		t.Fatalf("source = %v", src)
	}
}

// TestAskModelResponses OpenAI Responses 格式：input 结构、output_text 解析
func TestAskModelResponses(t *testing.T) {
	var gotPath, gotAuth string
	var raw map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &raw)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"output":[{"type":"message","content":[{"type":"output_text","text":"B"}]}]}`))
	}))
	defer srv.Close()

	client := &ChatGptClient{Enable: true, BaseURL: srv.URL, APIKey: "sk", Model: "m1", APIFormat: "responses"}
	text, err := client.askModel("m3", "题目", nil)
	if err != nil {
		t.Fatalf("askModel err: %v", err)
	}
	if text != "B" {
		t.Fatalf("text = %q", text)
	}
	if gotPath != "/v1/responses" {
		t.Fatalf("path = %q", gotPath)
	}
	if gotAuth != "Bearer sk" {
		t.Fatalf("auth = %q", gotAuth)
	}
	if raw["model"] != "m3" {
		t.Fatalf("model = %v", raw["model"])
	}
	if raw["input"] != "题目" {
		t.Fatalf("input = %v", raw["input"])
	}
}

// TestSearchAnswerWithFormat 端到端：anthropic 格式的完整搜题（多选解析仍走既有逻辑）
func TestSearchAnswerWithFormat(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"A"}]}`))
	}))
	defer srv.Close()

	client := &ChatGptClient{Enable: true, BaseURL: srv.URL, APIKey: "k", Model: "m", APIFormat: "anthropic"}
	ans, err := client.SearchAnswer(model.SearchRequest{Question: "中国的国土面积?", Options: []string{"960万", "100万"}, Type: 0})
	if err != nil {
		t.Fatalf("SearchAnswer err: %v", err)
	}
	if len(ans) == 0 || ans[0][0] != "960万" {
		t.Fatalf("ans = %v", ans)
	}
}
