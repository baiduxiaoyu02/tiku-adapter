# 计划：CreateUser 修复 + OCS 配置生成页

## 架构决策

- **CreateUser 修复**：控制器改绑 `userCreateRequest` DTO（内嵌 entity.User + 外层
  `Password string json:"password"`）——创建请求的密码能绑定，入库前 bcrypt；响应里
  Password 仍是 json:"-"（内嵌字段的标签不受外层影响），用户列表不泄露哈希。
- **OCS 配置生成页**：新增 `GET /ocs-config` 路由，返回内嵌 HTML（与 search.html 同
  embed 风格）。页面 JS 用 `location.origin + '/api/search'` 动态生成 AnswererWrapper
  JSON（含 options/type/images/suggestion 处理器），「复制配置」按钮 +
  默认解析器四步导入说明。图片题 images handler 已包含（警告会消失）。
- **交付**：重建 Windows exe + Inno Setup 安装包，部署到 D:\user\ruanjian\TikuAdapter\
  并重启实例（用户已自装该目录，8060 运行中）。

## devflow 映射（Requirement Mapping）

| FR | 任务 | 目标文件 | 验收 |
|----|------|---------|------|
| FR-1 | T1 CreateUser DTO 修复（TDD） | internal/controller/user.go | AC-1 |
| FR-2 | T2 OCS 配置生成页 | internal/controller/ocs_config.go | AC-2 |
| FR-3 | T3 重建 exe/安装包并部署 | dist/tiku-adapter-setup.exe | AC-3 |

## 任务依赖与分层

- T1 独立先行（Go 控制器 + 测试）。
- T2 独立（新路由 + 内嵌页，与 T1 不同文件可并行）。
- T3 依赖 T1+T2（构建包含两者，部署到用户目录）。

## 风险与对策

- 内嵌 HTML 的 handler 转义：`\n` 在 Go 字符串与页面 JS 双层转义，用
  `chr(92)`/String.raw 思路显式构造，生成后用 Node 验证 JSON.parse → Function 链路
  （上轮 \n 被 heredoc 吃掉的教训）。
- 部署覆盖运行中的 exe → 文件被锁：先 taskkill 再覆盖再启动（脚手架脚本处理）。
- 无代码签名 → SmartScreen 提示「仍要运行」，README 已写明。
