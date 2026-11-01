package util

import (
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/itihey/tikuAdapter/pkg/logger"
	"github.com/spf13/viper"
	"strconv"
	"sync"
)

var (
	jwtKey []byte
	once   sync.Once
)

// initKey 延迟初始化 JWT 密钥，从配置文件 jwt_secret 读取。
// 使用 sync.Once 确保只初始化一次，且在 viper.ReadInConfig() 之后执行（首次 JWT 调用发生在请求处理时）。
func initKey() {
	once.Do(func() {
		// 从 viper 读取 jwt_secret（t4 在 configs/config.go 添加 JWTSecret 字段，config.yaml 配置 jwt_secret 键）
		secret := viper.GetString("jwt_secret")
		if secret == "" {
			// 未配置时使用默认密钥并告警，生产环境必须配置 jwt_secret
			secret = "tiku-adapter-default-jwt-secret-change-me"
			logger.SysLog("警告：未配置 jwt_secret，使用默认密钥，生产环境请务必在 config.yaml 中设置 jwt_secret")
		}
		jwtKey = []byte(secret)
	})
}

// GenerateJwt 生成一个jwt
func GenerateJwt(id int32) string {
	initKey()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": strconv.Itoa(int(id)),
	})
	jwtStr, err := token.SignedString(jwtKey)
	if err != nil {
		logger.SysLog(err.Error())
		return ""
	}
	return jwtStr
}

// ParseJwtWithClaims 解析 jwt 字符串，返回 Claims 对象
func ParseJwtWithClaims(jwtStr string, options ...jwt.ParserOption) (jwt.Claims, error) {
	initKey()
	mc := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(jwtStr, mc, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	}, options...)
	if err != nil {
		return nil, err
	}
	// 校验 Claims 对象是否有效，基于 exp（过期时间），nbf（不早于），iat（签发时间）等进行判断（如果有这些声明的话）。
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	return token.Claims, nil
}
