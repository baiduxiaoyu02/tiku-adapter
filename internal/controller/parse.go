package controller

import (
	"code.sajari.com/docconv/v2"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/gookit/goutil/strutil"
	"github.com/itihey/tikuAdapter/internal/dao"
	"github.com/itihey/tikuAdapter/internal/entity"
	"github.com/itihey/tikuAdapter/internal/middleware"
	"github.com/itihey/tikuAdapter/internal/service"
	"github.com/itihey/tikuAdapter/pkg/global"
	"github.com/itihey/tikuAdapter/pkg/model"
	"github.com/itihey/tikuAdapter/pkg/util"
	"github.com/xuri/excelize/v2"
	"mime/multipart"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

// Parser 解析文件接口 支持解析docx和xlsx
func Parser(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, global.ErrorParam)
		return
	}
	uploadedFile, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, global.ErrorParam)
		return
	}
	defer func(uploadedFile multipart.File) {
		err := uploadedFile.Close()
		if err != nil {
			c.JSON(http.StatusBadRequest, global.ErrorParam)
			return
		}
	}(uploadedFile)
	contentType := file.Header.Get("Content-Type")
	if contentType == "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		f, err := excelize.OpenReader(uploadedFile)
		if err != nil {
			c.JSON(http.StatusBadRequest, global.ErrorParseFile)
			return
		}
		var options = make([]string, 0)
		err = json.Unmarshal([]byte(c.PostForm("options")), &options)
		if err != nil {
			c.JSON(http.StatusBadRequest, global.ErrorParam)
			return
		}
		opt := service.XLSXOptions{
			SheetName: c.PostForm("sheetName"),
			Question:  c.PostForm("question"),
			Answer:    c.PostForm("answer"),
			Option:    options,
		}

		c.JSON(http.StatusOK, service.ParseXls(f, opt))
	} else {
		convert, err := docconv.Convert(uploadedFile, contentType, true)
		if err != nil {
			c.JSON(http.StatusBadRequest, global.ErrorParseFile)
			return
		}
		// 结构化解析 docx 文本，识别题号、题型、选项、答案，输出 []model.Question
		questions := parseDocxText(convert.Body)

		// 将解析结果转换为 entity.Tiku 并批量入库
		plat, _ := strconv.Atoi(c.PostForm("plat"))
		courseName := c.PostForm("courseName")
		tikus := make([]*entity.Tiku, 0, len(questions))
		for _, q := range questions {
			opts, _ := json.Marshal(q.Options)
			ansBytes, _ := json.Marshal(q.Answer)
			t := &entity.Tiku{
				Question:   q.Question,
				Type:       int32(q.Type),
				Options:    string(opts),
				Answer:     string(ansBytes),
				Plat:       int32(plat),
				CourseName: courseName,
			}
			middleware.FillHash(t)
			t.QuestionText = util.GetQuestionText(t.Question)
			t.QuestionHash = strutil.Md5(t.Question)[:16]
			tikus = append(tikus, t)
		}
		if len(tikus) > 0 {
			if err := dao.Tiku.CreateInBatches(tikus, 100); err != nil {
				// 批量插入失败（可能存在重复 hash），回退逐条插入
				for _, tk := range tikus {
					_ = dao.Tiku.Create(tk)
				}
			}
		}

		c.JSON(http.StatusOK, questions)
	}
}

// 题号前缀正则：匹配 "1." "2、" "3．" "4:" 等
var questionNumRe = regexp.MustCompile(`^\s*(\d+)\s*[.．、：:]\s*(.*)`)

// 题型标记正则：匹配【单选题】【多选题】【判断题】【填空题】【问答题】等
var typeMarkerRe = regexp.MustCompile(`【(单选题|多选题|判断题|填空题|问答题|简答题|单选|多选|判断|填空|问答|简答)】\s*(.*)`)

// 选项行正则：匹配 "A." "B．" "C:" "D、" 等
var optionRe = regexp.MustCompile(`^\s*([A-Z])\s*[.．：:、]\s*(.*)`)

// 答案行正则：匹配 "答案：A" "正确答案:AB" "答案：正确" 等
var answerRe = regexp.MustCompile(`^\s*(?:正确)?答案\s*[：:]\s*(.*)`)

// typeMap 将题型标记映射为 model.Question.Type 值
var typeMap = map[string]uint{
	"单选题": 0, "单选": 0,
	"多选题": 1, "多选": 1,
	"填空题": 2, "填空": 2,
	"判断题": 3, "判断": 3,
	"问答题": 4, "问答": 4, "简答题": 4, "简答": 4,
}

// parseDocxText 将 docx 提取的纯文本解析为结构化的题目列表。
// 识别题号（1./2.）、题型（【单选题】等）、选项（A. B.）、答案（答案：XX），
// 输出 []model.Question，与 xlsx 路径产出格式一致。
func parseDocxText(text string) []model.Question {
	lines := strings.Split(text, "\n")
	var questions []model.Question
	var current *model.Question
	var currentOptions []string

	flush := func() {
		if current != nil && current.Question != "" {
			// 未显式设置类型时根据答案和选项推断
			if current.Type == 0 {
				if len(currentOptions) == 0 && len(current.Answer) > 0 {
					current.Type = 4 // 无选项有答案 → 问答题
				} else if len(current.Answer) == 1 && (util.IsTrue(current.Answer[0]) || util.IsFalse(current.Answer[0])) {
					current.Type = 3 // 判断题
				} else if len(current.Answer) > 1 {
					current.Type = 1 // 多选
				}
			}
			questions = append(questions, *current)
		}
		current = nil
		currentOptions = nil
	}

	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}

		// 检测题号前缀（1. 2、 3． 等）
		if m := questionNumRe.FindStringSubmatch(line); m != nil {
			flush()
			current = &model.Question{}
			rest := strings.TrimSpace(m[2])
			// 检测题型标记【单选题】等
			if tm := typeMarkerRe.FindStringSubmatch(rest); tm != nil {
				if t, ok := typeMap[tm[1]]; ok {
					current.Type = t
				}
				rest = strings.TrimSpace(tm[2])
			}
			current.Question = util.FormatString(rest)
			continue
		}

		// 检测答案行
		if m := answerRe.FindStringSubmatch(line); m != nil {
			if current != nil {
				ans := strings.TrimSpace(m[1])
				current.Answer = parseAnswerValue(ans, currentOptions)
			}
			continue
		}

		// 检测选项行（A. B. C. 等）
		if m := optionRe.FindStringSubmatch(line); m != nil {
			if current != nil {
				opt := strings.TrimSpace(m[2])
				currentOptions = append(currentOptions, opt)
				current.Options = currentOptions
			}
			continue
		}

		// 未匹配任何模式的行，追加到当前题目文本
		if current != nil && current.Question != "" {
			current.Question += " " + line
		}
	}
	flush()
	return questions
}

// parseAnswerValue 将答案文本解析为答案列表。
// 若答案为纯字母（如 "AB"）且提供了选项，则映射到选项值；
// 若答案为纯字母但无选项，则拆分为单个字母；
// 否则按 # 分隔或作为单个答案返回。
func parseAnswerValue(ans string, options []string) []string {
	ans = strings.TrimSpace(ans)
	if ans == "" {
		return []string{}
	}

	// 去除答案中常见的括号和标点
	ans = strings.Trim(ans, "（）()【】[]。.")

	if util.IsAlpha(ans) && len(options) > 0 {
		// 字母答案且有选项 → 映射到选项值
		result := make([]string, 0, len(ans))
		for _, ch := range ans {
			idx := int(ch - 'A')
			if idx >= 0 && idx < len(options) {
				result = append(result, options[idx])
			}
		}
		if len(result) > 0 {
			return result
		}
	}

	if util.IsAlpha(ans) {
		// 纯字母答案无选项 → 拆分为单个字母
		return strings.Split(ans, "")
	}

	// 非字母答案 → 按 # 分隔或整体返回
	if strings.Contains(ans, "#") {
		return strings.Split(ans, "#")
	}
	return []string{ans}
}
