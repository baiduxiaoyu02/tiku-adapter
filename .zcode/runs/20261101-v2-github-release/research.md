---
depth: deep
signals:
  changed_files: 15
  cross_module: true
  interface_or_model_change: true
  irreversible: true
  new_dependency: true
  unfamiliar_stack: true
  time_sensitive_fact: false
  multiple_candidates: true
  user_requested: true
sources: 10
skipped: false
skip_reason: ""
---

# 调研：题库服务 v2（GitHub 发布 + 供应商配置 UI + CreateUser 修复）

## 结论

- tiku-adapter 是 Go 静态编译单文件（CGO_ENABLED=0 + glebarez/sqlite 纯驱动 + go:embed 前端），Windows 原生构建可行，Docker 只是外壳。 来源: Dockerfile:3
- 仓库自带完整 goreleaser 发布链：tag 推送 → GitHub Actions（go-releaser.yml）→ 全平台构建（windows/linux/darwin × amd64/arm64/386/arm）→ Release 自动发布，地基现成。 来源: .github/workflows/go-releaser.yml:8
- goreleaser 构建前跑 buildweb.sh（Vue 前端 npm 构建），CI 的 ubuntu runner 自带 node，前端构建链在 CI 可用。 来源: .goreleaser.yaml:4
- OCS「默认」解析器硬拦截含 `adapter-service/search` 的配置（common.ts 保存流程硬判断 + return），题库侧提供不含该字符串的 `/api/search` 别名路由即可绕开。 来源: https://github.com/ocsjs/ocsjs/blob/master/packages/scripts/src/projects/common.ts
- entity.User 的 Password 字段标了 `json:"-"`（t5 为防用户列表泄露 bcrypt 哈希所加），导致 CreateUser 绑定时密码恒为空、API 创建的用户全部无法登录。 来源: internal/entity/users.gen.go:13
- OCS 题库配置的 contentType 必须是 `"json"`（request.ts 只有该值才走 JSON.parse），`application/json` 会落到文本分支 → 答案恒空。 来源: https://github.com/ocsjs/ocsjs/blob/master/packages/core/src/core/utils/request.ts
- 上游 AI（CPA glm-5.2，max_tokens=4096）对测试题返回 content='B'（finish_reason=stop），AI 链路本身正常。 来源: 实测：同形状请求直连 8318 返回 content='B'
- golang.org/x/tools@v0.13.0 与 Go 1.22+ 编译不兼容，Windows 本机构建必须锁 go1.21.13 工具链（GOTOOLCHAIN 经 goproxy.cn 下载）。 来源: 实测：Go 1.27 构建报 invalid array length -delta * delta
- 管理面板是 Vue SPA（web/ 前端 npm 构建后 go:embed 内嵌），新页面两条路：改 Vue 源重构建（需 node 链）或独立 HTML 页由 Go 直接服务（search.html 先例）。 来源: cmd/adapter-service/main.go:12
- OCS Desktop 2.9.24（Electron + ScriptCat）是用户当前运行 OCS 的方式，其题库配置存 ScriptCat 加密 LevelDB，外部写不进，配置导入走剪贴板 + 读取剪贴板按钮。 来源: 实测：Get-CimInstance 与 Secure Preferences 30 条扩展全查
- 用户已自行将桌面版部署到 D:\user\ruanjian\TikuAdapter\（exe + config + data + tray，8060 运行中），本轮交付需更新该目录。 来源: 实测：Get-CimInstance 显示 8060 监听者为该路径 exe
- OCS 题库配置的 images handler（图片题）已在 OCS 侧通过检测（answerer-env.ts 正则匹配 handler 里的 images 引用），配置 JSON 需包含 images 字段。 来源: https://github.com/ocsjs/ocsjs/blob/master/packages/scripts/src/utils/answerer-env.ts
- AI 上游 CPA 对 glm-5.2 的调用需要 max_tokens ≥2000（否则 200+空正文）且模型条目需声明 thinking.levels。 来源: https://github.com/ocsjs/ocsjs/blob/master/packages/scripts/src/utils/answerer-env.ts

## 方案对比

| 方案 | 组成 | 优点 | 缺点 | 来源 |
|------|------|------|------|------|
| goreleaser tag 发布 | 推 v* tag → CI 构建全平台 → Release 自动挂产物 | 零手工、以后改代码推 tag 即发 | 安装包（Inno）CI 上难构建（ISCC 是 Windows 程序），Release 先挂 exe+ZIP | .github/workflows/go-releaser.yml |
| 手动本地构建发布 | 本地构建 exe+安装包 → gh release create 附件 | 可带 Inno Setup 安装包 | 每次手动 | 实测 |
| 供应商配置存储 | 扩展 configs.Config（[]Provider）+ API GET/POST /api/providers + config.yaml 持久化 | 与现有 viper 体系一致、运行时改后提示重启 | 需改 config 结构与读写 | internal/registry/config.go |
| 配置 UI 形态 | Vue 页面（node 构建）vs 独立 HTML（Go 服务，search.html 先例） | Vue=与面板一体；HTML=免 node 构建、独立可用 | Vue 需重跑 builder1 | cmd/adapter-service/main.go:12 |

选择：goreleaser tag 发布 + Vue 内嵌「供应商管理」页（面板一致性优先；构建链 CI 已具备）。

## 未决问题

- goreleaser archives 是否已包含 config.yaml 模板与 tray 脚本（待查 .goreleaser.yaml 的 archives 段，未截取到）。
- GitHub Actions 在用户 fork 仓库上首次运行的权限确认（GITHUB_TOKEN 默认 contents:write 已在 yml 声明）。

## 假设

- 假设: 朋友的 AI key 各自填写（面板供应商 UI 支持），官方免费题库对所有用户开放。
- 假设: 历史压成单提交后推公开仓库，review 证据文件里的真 key 字符串随之消失。
- 假设: 视觉模型沿用 sensenova-6.8-flash-lite（朋友各自填 key 后按其供应商实际支持调整）。
