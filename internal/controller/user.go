package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/itihey/tikuAdapter/internal/dao"
	"github.com/itihey/tikuAdapter/internal/entity"
	"github.com/itihey/tikuAdapter/pkg/logger"
	"github.com/itihey/tikuAdapter/pkg/util"
	"strconv"
)

// Auth 鉴权
func Auth(c *gin.Context) {
	claims, err := util.ParseJwtWithClaims(c.Request.Header.Get("Authorization"))

	if err != nil {
		logger.SysLog(err.Error())
		c.AbortWithStatusJSON(401, gin.H{
			"message": "请重新登录",
		})
		return
	}

	subject, err := claims.GetSubject()
	if err != nil {
		logger.SysLog(err.Error())
		c.AbortWithStatusJSON(401, gin.H{
			"message": "请重新登录",
		})
		return
	}
	userID, _ := strconv.Atoi(subject)

	user, err := dao.User.Where(dao.User.ID.Eq(int32(userID))).First()
	if err != nil {
		logger.SysLog(err.Error())
		c.AbortWithStatusJSON(401, gin.H{
			"message": "请重新登录",
		})
		return
	}
	c.Set("user", user)
}

// UserInfo 获取用户信息
func UserInfo(c *gin.Context) {
	v, _ := c.Get(`user`)
	c.JSON(200, v.(*entity.User))
}

// loginRequest 登录请求参数
type loginRequest struct {
	Username string `json:"username" form:"username" binding:"required"`
	Password string `json:"password" form:"password" binding:"required"`
}

// UserLogin 用户登录 - 使用 bcrypt 验证密码，从 POST body 读取凭据
func UserLogin(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(400, gin.H{
			"message": "参数错误",
		})
		return
	}
	us := dao.User
	user, err := us.Where(us.Username.Eq(req.Username)).First()
	if err != nil {
		c.JSON(400, gin.H{
			"message": "用户名或密码错误",
		})
		return
	}
	// 使用 bcrypt 比对密码哈希，不再明文比对
	if !util.VerifyPassword(req.Password, user.Password) {
		c.JSON(400, gin.H{
			"message": "用户名或密码错误",
		})
		return
	}
	jwt := util.GenerateJwt(user.ID)

	if jwt == "" {
		c.JSON(400, gin.H{
			"message": "jwt生成失败",
		})
	} else {
		c.JSON(200, gin.H{
			"message": "user login",
			"jwt":     jwt,
		})
	}
}

// UserList 自己开通的用户的列表
func UserList(c *gin.Context) {
	parent, _ := c.Get(`user`)
	users, err := dao.User.Where(dao.User.ParentID.Eq(parent.(*entity.User).ID)).Find()
	if err != nil {
		c.JSON(400, gin.H{
			"message": "获取用户列表失败",
		})
		return
	}
	c.JSON(200, users)
}

// userCreateRequest 创建用户请求参数：entity.User 的 Password 是 json:"-"，
// 直接绑定 entity 会导致 JSON 中的密码绑不进来，故外层单独绑定 Password
type userCreateRequest struct {
	entity.User
	Password string `json:"password"`
}

// CreateUser 创建用户
func CreateUser(c *gin.Context) {
	parent, _ := c.Get(`user`)

	var req userCreateRequest

	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(400, gin.H{
			"message": "参数错误",
		})
		return
	}

	user := req.User
	user.Password = req.Password

	// 检测用户名是否被使用
	_, err = dao.User.Where(dao.User.Username.Eq(user.Username)).First()
	if err == nil {
		c.JSON(400, gin.H{
			"message": "用户名已经被使用",
		})
		return
	}

	user.ParentID = parent.(*entity.User).ID

	// 使用 bcrypt 哈希密码，避免明文存储
	hashedPwd, err := util.HashPassword(user.Password)
	if err != nil {
		c.JSON(400, gin.H{
			"message": "密码加密失败",
		})
		return
	}
	user.Password = hashedPwd

	err = dao.User.Create(&user)
	if err != nil {
		c.JSON(400, gin.H{
			"message": "创建用户失败",
		})
		return
	}
	c.JSON(200, user)
}

// DeleteUser 删除用户
func DeleteUser(c *gin.Context) {
	parent, _ := c.Get(`user`)
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{
			"message": "参数错误",
		})
		return
	}

	user, err := dao.User.Where(dao.User.ID.Eq(int32(userID))).First()
	if err != nil {
		c.JSON(400, gin.H{
			"message": "用户不存在",
		})
		return
	}

	if user.ParentID == parent.(*entity.User).ID {
		dao.User.Delete(user)
		c.JSON(200, gin.H{
			"message": "删除用户成功",
		})
	} else {
		c.JSON(400, gin.H{
			"message": "无权限删除",
		})
	}
}
