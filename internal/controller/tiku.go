package controller

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gookit/goutil/strutil"
	"github.com/itihey/tikuAdapter/internal/dao"
	"github.com/itihey/tikuAdapter/internal/entity"
	"github.com/itihey/tikuAdapter/internal/middleware"
	"github.com/itihey/tikuAdapter/internal/registry/manager"
	"github.com/itihey/tikuAdapter/pkg/logger"
	"github.com/itihey/tikuAdapter/pkg/util"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Page 分页
type Page struct {
	PageNo   int            `json:"pageNo" form:"pageNo"`
	PageSize int            `json:"pageSize" form:"pageSize"`
	Total    int64          `json:"total" form:"total"`
	Items    []*entity.Tiku `json:"items" form:"items"`
}

// SearchValue 搜索参数
type SearchValue struct {
	PageNo              int    `json:"pageNo" form:"pageNo"`
	PageSize            int    `json:"pageSize" form:"pageSize"`
	Source              int32  `json:"source" form:"source"`
	Extra               string `json:"extra" form:"extra"`
	OnlyShowEmptyAnswer bool   `json:"onlyShowEmptyAnswer" form:"onlyShowEmptyAnswer"`
	Question            string `json:"question" form:"question"`
	CourseName          string `json:"courseName" form:"courseName"`

	Type int32 `json:"type" form:"type"`
	Plat int32 `json:"plat" form:"plat"`
}

// GetQuestions 获取题库
func GetQuestions(c *gin.Context) {
	var searchValue SearchValue
	err := c.ShouldBindJSON(&searchValue)
	if err != nil {
		c.JSON(400, gin.H{
			"message": "参数错误",
		})
		return
	}
	user, _ := c.Get("user")
	if searchValue.PageSize > 10 {
		r, _ := json.Marshal(searchValue)
		l := entity.Log{
			Qid:        0,
			Action:     3,
			UserID:     user.(*entity.User).ID,
			CreateTime: time.Now(),
			OldAnswer:  string(r),
		}
		dao.Log.Create(&l)
	}

	if searchValue.PageSize > 100 {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "服务器错误",
		})
		return
	}

	tx := dao.Tiku.Order(dao.Tiku.ID.Desc())
	if searchValue.Question != "" {
		tx.Where(dao.Tiku.Question.Like("%" + searchValue.Question + "%"))
	}
	if searchValue.Extra != "" {
		tx.Where(dao.Tiku.Extra.Like(searchValue.Extra))
	}

	if searchValue.CourseName != "" {
		tx.Where(dao.Tiku.CourseName.Eq(searchValue.CourseName))
	}

	if searchValue.Plat != -1 {
		tx.Where(dao.Tiku.Plat.Eq(searchValue.Plat))
	}

	if searchValue.Type != -1 {
		tx.Where(dao.Tiku.Type.Eq(searchValue.Type))
	}

	if searchValue.OnlyShowEmptyAnswer {
		tx.Where(dao.Tiku.Answer.Eq("[]"))
	}

	items, total, err := tx.FindByPage(searchValue.PageNo*searchValue.PageSize, searchValue.PageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "服务器错误",
		})
		return
	}

	c.JSON(http.StatusOK, Page{
		PageNo:   searchValue.PageNo,
		PageSize: searchValue.PageSize,
		Total:    total,
		Items:    items,
	})
}

// UpdateQuestions 更新题库
func UpdateQuestions(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{
			"message": "参数错误",
		})
		return
	}

	var tiku *entity.Tiku
	err = c.ShouldBindJSON(&tiku)
	if err != nil {
		c.JSON(400, gin.H{
			"message": "参数错误",
		})
		return
	}
	// 找到旧答案，加 err 守卫防止 nil 指针
	oldTiku, err := dao.Tiku.Where(dao.Tiku.ID.Eq(int32(id))).First()
	if err != nil {
		c.JSON(404, gin.H{
			"message": "题目不存在",
		})
		return
	}

	// IDOR 修复：仅允许管理员或题目创建者更新
	user, _ := c.Get("user")
	currentUser := user.(*entity.User)
	if !canModifyQuestion(currentUser, int32(id)) {
		c.JSON(403, gin.H{
			"message": "无权限修改此题目",
		})
		return
	}

	middleware.FillHash(tiku) // 更新题库的时候重新计算hash值
	updates, err := dao.Tiku.Where(dao.Tiku.ID.Eq(int32(id))).Updates(tiku)
	if err != nil {
		c.JSON(500, gin.H{
			"message": "服务器错误",
		})
		return
	}
	if updates.RowsAffected == 1 {
		l := entity.Log{
			Qid:        oldTiku.ID,
			Action:     1,
			UserID:     currentUser.ID,
			CreateTime: time.Now(),
			OldAnswer:  oldTiku.Answer,
			NewAnswer:  tiku.Answer,
		}
		dao.Log.Create(&l)
	}
	c.JSON(200, gin.H{
		"data": updates,
	})
}

// canModifyQuestion 检查用户是否有权修改/删除题目（IDOR 防护）
// 管理员（ID=1 或 perms 含 admin）可操作所有题目；其余用户仅可操作自己创建的题目
func canModifyQuestion(user *entity.User, qid int32) bool {
	if user.ID == 1 || strings.Contains(user.Perms, "admin") {
		return true
	}
	// 通过 Log 表查该用户是否为题目创建者（Action=0 表示创建）
	_, err := dao.Log.Where(dao.Log.Qid.Eq(qid), dao.Log.Action.Eq(0), dao.Log.UserID.Eq(user.ID)).First()
	return err == nil
}

// DeleteQuestion 删除题目
func DeleteQuestion(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(400, gin.H{
			"message": "参数错误",
		})
		return
	}

	// 找到题目，加 err 守卫防止 nil 指针
	oldTiku, err := dao.Tiku.Where(dao.Tiku.ID.Eq(int32(id))).First()
	if err != nil {
		c.JSON(404, gin.H{
			"message": "题目不存在",
		})
		return
	}

	// IDOR 修复：仅允许管理员或题目创建者删除
	user, _ := c.Get("user")
	currentUser := user.(*entity.User)
	if !canModifyQuestion(currentUser, int32(id)) {
		c.JSON(403, gin.H{
			"message": "无权限删除此题目",
		})
		return
	}

	// 记录删除日志
	marshal, _ := json.Marshal(oldTiku)
	l := entity.Log{
		Qid:        oldTiku.ID,
		Action:     2,
		UserID:     currentUser.ID,
		CreateTime: time.Now(),
		OldAnswer:  string(marshal),
	}
	dao.Log.Create(&l)

	dao.Tiku.Where(dao.Tiku.ID.Eq(int32(id))).Delete()
	c.JSON(200, gin.H{
		"message": "删除成功",
	})
}

// CreateQuestion 创建题目
func CreateQuestion(c *gin.Context) {
	var tikus []*entity.Tiku
	err := c.ShouldBindJSON(&tikus)
	if err != nil {
		c.JSON(400, gin.H{
			"message": "参数错误",
		})
		return
	}
	var count = 0
	var dropped = 0
	for _, tiku := range tikus {
		t := tiku
		middleware.FillHash(t)
		// 填充 question_text 和 question_hash 用于模糊搜索
		t.QuestionText = util.GetQuestionText(t.Question)
		t.QuestionHash = strutil.Md5(t.Question)[:16]
		err := dao.Tiku.Create(t)
		if err == nil {
			// ES 双写：题目创建成功后同步到 Elasticsearch
			es := manager.GetManager().GetEs()
			if es != nil {
				if esErr := es.Create("tiku", strconv.Itoa(int(t.ID)), t); esErr != nil {
					logger.SysError(fmt.Sprintf("ES create failed for question %d: %s", t.ID, esErr.Error()))
				}
			}
			marshal, _ := json.Marshal(t)
			user, _ := c.Get("user")
			l := entity.Log{
				Qid:        t.ID,
				Action:     0,
				UserID:     user.(*entity.User).ID,
				CreateTime: time.Now(),
				OldAnswer:  string(marshal),
			}
			err := dao.Log.Create(&l)
			if err != nil {
				logger.SysError(err.Error())
			}
			count++
		} else {
			dropped++
			logger.SysError(fmt.Sprintf("question dropped (hash: %s): %s", t.Hash, err.Error()))
		}
	}
	c.JSON(200, gin.H{
		"message": "成功创建" + strconv.Itoa(count) + "条数据，丢弃" + strconv.Itoa(dropped) + "条",
		"count":   count,
		"dropped": dropped,
	})
}

// Courses 获取这个平台的所有课程名称
func Courses(c *gin.Context) {
	tx := dao.Tiku.Select(dao.Tiku.CourseName)

	if c.Query("plat") != "" {
		i, err := strconv.Atoi(c.Query("plat"))
		if err != nil {
			c.String(400, "平台类型有错误")
			return
		}
		tx.Where(dao.Tiku.Plat.Eq(int32(i)))
	}
	find, err := tx.Group(dao.Tiku.CourseName).Find()
	if err != nil {
		return
	}
	courses := make([]string, 0)
	for i := range find {
		if find[i].CourseName != "" {
			courses = append(courses, find[i].CourseName)
		}
	}
	c.JSON(200, courses)
}
