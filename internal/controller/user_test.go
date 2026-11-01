package controller

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/itihey/tikuAdapter/internal/entity"
)

// TestUserCreateRequest 验证创建用户的请求 DTO 能绑定 JSON 中的 password 字段
func TestUserCreateRequest(t *testing.T) {
	var req userCreateRequest
	body := `{"username":"tester","password":"secret123","nickname":"nick","perms":"all"}`
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatalf("绑定 JSON 失败: %v", err)
	}
	if req.Password != "secret123" {
		t.Fatalf("DTO.Password 未收到值, got %q", req.Password)
	}
	if req.Username != "tester" {
		t.Fatalf("内嵌 entity.User 字段未绑定, got %q", req.Username)
	}
}

// TestUserEntityNoPasswordLeak 验证 entity.User 序列化不含 password 字段（用户列表不泄露哈希）
func TestUserEntityNoPasswordLeak(t *testing.T) {
	user := entity.User{
		ID:       1,
		Username: "tester",
		Password: "$2a$10$abcdefghijklmnopqrstuv",
		Perms:    "all",
		ParentID: 2,
		Nickname: "nick",
	}
	data, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("反序列化失败: %v", err)
	}
	if _, ok := m["password"]; ok {
		t.Fatalf("entity.User 序列化包含 password 字段: %s", data)
	}
	if bytes.Contains(data, []byte("$2a$10$abcdefghijklmnopqrstuv")) {
		t.Fatalf("密码哈希泄露到 JSON: %s", data)
	}
}
