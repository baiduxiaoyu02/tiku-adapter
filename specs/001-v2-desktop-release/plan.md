# 计划：题库服务 v2 — GitHub 发布与模型供应商管理

## 架构决策

- **供应商配置存储**：configs.Config 新增 `Providers []Provider`（名称/接口地址/API密钥/模型/视觉模型/当前标记），沿用 viper YAML 体系持久化到 config.yaml；运行时改后提示一键重启生效（Windows 无 SIGHUP，重启 = 面板按钮调停启接口）。
- **供应商 API**：`GET /adapter-service/providers`（列表，密钥掩码）、`POST /adapter-service/providers`（新增/更新）、`DELETE /adapter-service/providers/:name`（删除）、`POST /adapter-service/providers/test`（连通性测试）、`POST /adapter-service/server/restart`（重启生效）。全部走 Auth 中间件。
- **CreateUser 修复**：控制器改绑 userCreateRequest DTO（内嵌 entity.User + 外层 Password json:"password"），创建时对 DTO.Password 做 bcrypt；entity.User 的 json:"-" 不变（列表不泄露哈希）。
- **OCS 配置生成页**：`GET /ocs-config` 返回内嵌 HTML（与 search.html 同 embed 风格），页面 JS 用 `location.origin + '/api/search'` 动态生成 AnswererWrapper JSON + 复制按钮 + 四步导入说明（含图片题 images handler）。
- **发布流水线**：仓库自带 go-releaser.yml（tag 推送触发）沿用；补 .goreleaser.yaml 的 archives 额外文件（config 模板/托盘/ocs-config 页资源）；Inno Setup 安装包在本机构建后用 gh release upload 追加。

## devflow 映射（Requirement Mapping）

| FR | 任务 | 目标文件 | 验收 |
|----|------|---------|------|
| FR-1 | T1 CreateUser DTO 修复（TDD） | internal/controller/user.go | AC-1 |
| FR-2 | T2 供应商配置 API（增删改查/测试/切换） | internal/api/router.go + internal/controller/providers.go | AC-2 |
| FR-3 | T2 配置持久化（config.yaml 写回） | internal/registry/config.go | AC-3 |
| FR-4 | T3 供应商管理页面（Vue） | web/src/views/ProviderManage.vue + 路由注册 | AC-3 |
| FR-5 | T3 重启生效入口（面板按钮） | web/src/views/ProviderManage.vue | AC-3 |
| FR-6 | T4 /ocs-config 页面（内嵌 HTML） | internal/controller/ocs_config.go + web/ocs-config-template.html | AC-4 |
| FR-7 | T5 发布流水线（.goreleaser.yaml 补打包清单 + 首个 tag） | .goreleaser.yaml + .github/workflows/go-releaser.yml | AC-4 |
| FR-8 | T5 历史压缩单提交 | scripts/squash-history.ps1 | AC-5 |
| FR-9 | T6 面板版本号显示 | web/src（沿用现有布局） | — |

## 任务依赖与分层

- T1（CreateUser 修复）与 T2（供应商 API + 持久化）不同文件可并行。
- T3（Vue 页面 + 路由）依赖 T2（API 就绪后对接）。
- T4（/ocs-config 页）独立，可与 T2/T3 并行。
- T5（发布流水线）依赖 T1-T4 全部完成。
- T6（版本号）依赖 T2。

## 风险与对策

- Vue 构建链回归 → builder1 在 CI（ubuntu node）与本地 Docker 均已验证；改 Vue 源后先本地构建再提交。
- goreleaser archives 缺文件 → .goreleaser.yaml 的 archives.files 显式列出 config 模板与托盘脚本，构建后校验压缩包内容。
- 历史压缩丢内容 → 压缩前 git bundle 全量备份到 dist/ 外部路径；压缩后 diff 校验工作树与 HEAD 一致。
- 朋友 key 泄露面板 → 密钥字段 API 响应掩码（首尾 4 字符），面板编辑时空值表示不变更。
