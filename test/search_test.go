package test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/itihey/tikuAdapter/internal/search"
	"github.com/itihey/tikuAdapter/pkg/model"
	"github.com/itihey/tikuAdapter/pkg/util"
)

func TestSearchIcodefClient_SearchAnswer(t *testing.T) {
	var client = search.IcodefClient{}
	testRequest := model.SearchRequest{
		Question: "下面选项中,属于男性在青春期生理变化的内容有?()",
	}

	// 调用被测试的方法
	response, err := client.SearchAnswer(testRequest)

	if err != nil {
		t.Errorf("请求icodef题库异常: %v", err)
		return
	}
	marshal, _ := json.Marshal(response)

	fmt.Println("测试icodef题库 成功搜题", string(marshal))
}

func TestSearchWannengClient_SearchAnswer(t *testing.T) {
	var client = search.WannengClient{
		Enable: true,
	}
	testRequest := model.SearchRequest{
		Question: "大学生缓解性欲的常见途径是()",
		Options:  []string{"自然缓解", "升华", "转移", "克制"},
		Type:     0,
	}

	// 调用被测试的方法
	response, err := client.SearchAnswer(testRequest)

	if err != nil {
		fmt.Printf("请求万能题库异常: %v", err)
		return
	}
	marshal, _ := json.Marshal(response)

	fmt.Println("测试万能题库 成功搜题", string(marshal))
}

func TestSearchEnncyClient_SearchAnswer(t *testing.T) {
	var client = search.EnncyClient{
		Enable: true,
		Token:  os.Getenv("ENNCY_TOKEN"),
	}

	testRequest := model.SearchRequest{
		Question: "《红楼梦》传世抄本中，惟一一个可以确定抄录年代的，原为吴晓玲先生旧藏、今藏于首都图书馆的是？",
	}

	// 调用被测试的方法
	response, err := client.SearchAnswer(testRequest)

	if err != nil {
		fmt.Printf("请求enncy题库异常: %v", err)
		return
	}
	marshal, _ := json.Marshal(response)

	fmt.Println("测试enncy题库 成功搜题", string(marshal))
}

func TestBuguake_SearchAnswer(t *testing.T) {
	var client = search.BuguakeClient{
		Enable: true,
	}

	testRequest := model.SearchRequest{
		Question: "苏轼的《西江月》是其在哪里所创作的?( )",
	}

	// 调用被测试的方法
	response, err := client.SearchAnswer(testRequest)

	if err != nil {
		t.Errorf("请求不挂科题库异常: %v", err)
		return
	}
	marshal, _ := json.Marshal(response)

	fmt.Println("测试不挂科题库 成功搜题", string(marshal))
}

func TestAidian_SearchAnswer(t *testing.T) {

	var client = search.AidianClient{
		Enable: true,
	}

	testRequest := model.SearchRequest{
		Question: "9.十进制数(9)10比十六进制数(9)16小.( )",
	}

	// 调用被测试的方法
	response, err := client.SearchAnswer(testRequest)

	if err != nil {
		fmt.Printf("请求爱点题库异常: %v", err)
		return
	}
	marshal, _ := json.Marshal(response)

	fmt.Println("测试爱点题库 成功搜题", string(marshal))
}

// TestSearchLemonClient_SearchAnswer 测试柠檬题库接口
func TestSearchLemonClient_SearchAnswer(t *testing.T) {

	var client = search.LemonClient{
		Enable: true,
		Token:  "8a3debe92e2ba83d6786e186bef2a424",
	}

	testRequest := model.SearchRequest{
		Question: "大学生缓解性欲的常见途径是()",
	}

	response, err := client.SearchAnswer(testRequest)

	if err != nil {
		fmt.Printf("请求柠檬题库异常: %v", err)
		return
	}
	marshal, _ := json.Marshal(response)

	fmt.Println("测试柠檬题库 成功搜题", string(marshal))
}

// TestSearchTikuhaiClient_SearchAnswer 测试题库海题库接口
func TestSearchTikuhaiClient_SearchAnswer(t *testing.T) {
	var client = search.TikuhaiClient{
		Enable: true,
		Token:  "",
	}
	testRequest := model.SearchRequest{
		Question: "毛泽东思想形成的时代背景是( )",
		Options:  []string{"帝国主义战争与无产阶级革命成为时代主题", "和平与发展成为时代主题", "世界多极化成为时代主题", "经济全球化成为时代主题"},
		Type:     0,
	}
	//这道题付费题库里才有答案
	// 调用被测试的方法
	response, err := client.SearchAnswer(testRequest)

	if err != nil {
		fmt.Printf("请求题库海题库异常: %v", err)
		return
	}
	marshal, _ := json.Marshal(response)

	fmt.Println("测试题库海题库 成功搜题", string(marshal))
}

// TestChatGptClient_SearchAnswer 使用 httptest 模拟 OpenAI 兼容 API，覆盖字母答案映射、stripThinkTags、错误状态码三路径
func TestChatGptClient_SearchAnswer(t *testing.T) {
	t.Run("letter mapping for single choice", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"choices": []map[string]interface{}{
					{
						"message": map[string]interface{}{
							"content": "A",
						},
					},
				},
			})
		}))
		defer ts.Close()

		client := search.ChatGptClient{
			Enable:  true,
			BaseURL: ts.URL,
			APIKey:  "test-key",
			Model:   "test-model",
		}

		testRequest := model.SearchRequest{
			Question: "测试题目",
			Options:  []string{"选项A内容", "选项B内容", "选项C内容", "选项D内容"},
			Type:     0,
		}

		response, err := client.SearchAnswer(testRequest)
		if err != nil {
			t.Errorf("ChatGpt请求异常: %v", err)
			return
		}

		if len(response) == 0 || len(response[0]) == 0 {
			t.Error("期望返回答案，实际为空")
			return
		}

		// 字母 A 应映射为 options[0] 即 "选项A内容"
		if response[0][0] != "选项A内容" {
			t.Errorf("期望答案为 选项A内容，实际为 %s", response[0][0])
		}
	})

	t.Run("strip think tags and map letter", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"choices": []map[string]interface{}{
					{
						"message": map[string]interface{}{
							"content": "<think>这是推理过程</think>B",
						},
					},
				},
			})
		}))
		defer ts.Close()

		client := search.ChatGptClient{
			Enable:  true,
			BaseURL: ts.URL,
			APIKey:  "test-key",
			Model:   "test-model",
		}

		testRequest := model.SearchRequest{
			Question: "测试题目",
			Options:  []string{"选项A", "选项B", "选项C", "选项D"},
			Type:     0,
		}

		response, err := client.SearchAnswer(testRequest)
		if err != nil {
			t.Errorf("ChatGpt请求异常: %v", err)
			return
		}

		if len(response) == 0 || len(response[0]) == 0 {
			t.Error("期望返回答案，实际为空")
			return
		}

		// think 标签应被剥离，字母 B 应映射为 options[1] 即 "选项B"
		if response[0][0] != "选项B" {
			t.Errorf("期望答案为 选项B，实际为 %s", response[0][0])
		}
	})

	t.Run("error status code returns error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer ts.Close()

		client := search.ChatGptClient{
			Enable:  true,
			BaseURL: ts.URL,
			APIKey:  "test-key",
			Model:   "test-model",
		}

		testRequest := model.SearchRequest{
			Question: "测试题目",
			Options:  []string{"选项A", "选项B"},
			Type:     0,
		}

		_, err := client.SearchAnswer(testRequest)
		if err == nil {
			t.Error("期望返回错误（401状态码），实际返回 nil")
		}
	})
}

// TestFillAnswerResponse_BestAnswerInOriginalOptions 验证 BestAnswer 返回原始选项文本而非格式化后的值。
// 场景：type=0，options=['A.选项一','B.选项二']（FormatOptions 去前缀后变为 ['选项一','选项二']），
// 答案 '选项一' 匹配格式化选项 [0]，修复前 BestAnswer 得到 '选项一'（格式化），不在原始 options 中；
// 修复后 BestAnswer 得到 'A.选项一'（原始），能在 resp.Options 中找到。
func TestFillAnswerResponse_BestAnswerInOriginalOptions(t *testing.T) {
	req := model.SearchRequest{
		Question: "测试题目",
		Options:  []string{"A.选项一", "B.选项二"},
		Type:     0,
	}
	answers := [][]string{{"选项一"}}

	resp := util.FillAnswerResponse(answers, &req)

	if len(resp.Answer.BestAnswer) == 0 {
		t.Fatal("期望 BestAnswer 非空，实际为空")
	}

	// BestAnswer[0] 必须能在 resp.Options（原始选项）中找到
	found := false
	for _, opt := range resp.Options {
		if opt == resp.Answer.BestAnswer[0] {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("BestAnswer[0]=%q 不在原始选项 %v 中", resp.Answer.BestAnswer[0], resp.Options)
	}

	// 验证 req 未被修改（原始选项保持不变）
	if len(req.Options) != 2 || req.Options[0] != "A.选项一" || req.Options[1] != "B.选项二" {
		t.Errorf("req.Options 被修改: %v", req.Options)
	}
}
