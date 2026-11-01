package util

import "golang.org/x/crypto/bcrypt"

// HashPassword 使用 bcrypt 哈希密码，返回哈希后的字符串。
// 用于创建用户时存储密码哈希，替代明文存储。
func HashPassword(pwd string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// VerifyPassword 使用 bcrypt 验证明文密码与哈希是否匹配。
// 用于登录时验证用户输入的密码。
func VerifyPassword(pwd, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(pwd))
	return err == nil
}
