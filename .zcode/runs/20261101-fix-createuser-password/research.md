---
depth: quick
signals:
  changed_files: 4
  cross_module: false
  interface_or_model_change: false
  irreversible: false
  new_dependency: false
  unfamiliar_stack: false
  time_sensitive_fact: false
  multiple_candidates: false
  user_requested: true
sources: 6
skipped: false
skip_reason: ""
---

# 调研：CreateUser 修复 + OCS 配置生成

## 结论

- entity.User 的 Password 字段标了 `json:"-"`（t5 为防用户列表泄露 bcrypt 哈希所加），导致 CreateUser 绑定时密码恒为空、入库的是空密码哈希，API 创建的用户全部无法登录。 来源: internal/entity/users.gen.go:13
- OCS「默认」解析器硬拦截含 `adapter-service/search` 的配置（common.ts 保存流程硬判断 + return），题库侧提供不含该字符串的 `/api/search` 别名路由即可绕开。 来源: https://github.com/ocsjs/ocsjs/blob/master/packages/scripts/src/projects/common.ts
- OCS 题库配置的 images handler（图片题支持）已在 OCS 侧通过检测（answerer-env.ts 正则匹配 handler 里的 images 引用），配置 JSON 需包含 images 字段。 来源: https://github.com/ocsjs/ocsjs/blob/master/packages/scripts/src/utils/answerer-env.ts
- 上游 AI（CPA glm-5.2，max_tokens=4096）对测试题返回 content='B'（finish_reason=stop），AI 链路本身正常。 来源: 实测：同形状请求直连 8318 返回 content='B'
- 用户已自行将桌面版部署到 D:\user\ruanjian\TikuAdapter\（exe+config+data+tray 齐全，8060 运行中），本轮交付物需更新到该目录。 来源: 实测：Get-CimInstance 显示 8060 监听者为该路径 exe

## 方案对比

| 方案 | 组成 | 优点 | 缺点 | 来源 |
|------|------|------|------|------|
| DTO 绑定修复 | CreateUser 改绑 userCreateRequest（内嵌 User + 外层 Password） | 密码能入库、列表仍不泄露哈希、改动最小 | 需重建 exe | internal/controller/user.go |
| 直接改 SQLite | 外部 bcrypt 哈希后 UPDATE users 表 | 立即生效 | 只救现有账号，新创建的用户还是会坏 | 密码哈希实测（Go 工具链生成 $2a$10$ 哈希直写 DB 后登录成功） |
| 配置生成落点 | Go 内嵌 HTML 页（/ocs-config） vs 静态 JSON 文件 | HTML 页可按访问地址动态生成 url、带复制按钮与步骤 | 静态文件地址写死不便分享 | https://github.com/ocsjs/ocsjs/blob/master/packages/scripts/src/utils/answerer-connect.ts |

选择：DTO 绑定修复 + Go 内嵌 HTML 配置页（用户已拍板「加个生成配置文件的功能」）。

## 未决问题

- 面板「用户列表」的删除按钮能否删除自己（admin 自删）——未验证，不在本轮范围。
- 安装包静默安装的 UAC 实测（上轮遗留 R6 的一部分，待用户安装时确认）。

## 假设

- 假设: OCS 配置 JSON 的 url 用 location.origin 动态生成——同机访问即 localhost:8060，无需配置项。
- 假设: 旧 admin 账号（changeme）保留不删，用户可自行在面板处理。
