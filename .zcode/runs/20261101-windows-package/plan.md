# 计划：题库服务 Windows 桌面打包

## 架构决策

- **运行形态**：tiku-adapter.exe 直接跑在 Windows（Go 静态编译，前端 + SQLite 内嵌），
  监听 8060。Docker 容器与 Python 代理不再属于启动链（现有容器部署保留但互不依赖）。
- **OCS 对接**：新增 `/api/search` 别名路由（GET/HEAD 探活 + POST 搜题，与
  `/adapter-service/search` 同一 Search 控制器）。OCS「默认」解析器硬拦截含
  `adapter-service/search` 字符串的配置（common.ts 保存流程），别名路由不含该字符串，
  保存不再被拦；同时 OCS 配置不再需要 Python 代理（8061）转发。
- **工具链**：Go 1.27（已装）+ `GOTOOLCHAIN=go1.21.13`（golang.org/x/tools@v0.13.0
  与 Go 1.22+ 不兼容，工具链经 goproxy.cn 下载并校验）；Inno Setup 6（已装）。
- **托盘**：沿用 PowerShell NotifyIcon 方案（紫色 ? 图标、左键开搜题页、右键菜单），
  打包版脚本改为启动本地 exe（不再触碰 Docker / Python）。

## devflow 映射（Requirement Mapping）

| FR | 任务 | 目标文件 | 验收 |
|----|------|---------|------|
| FR-1 | T1 路由别名（TDD） | internal/api/router.go | AC-1 |
| FR-2 | T2 Windows 构建脚本 | scripts/build-windows.ps1 | AC-2 |
| FR-3 | T3 打包目录与托盘 | packaging/tray/tiku-tray.ps1 | AC-3 |
| FR-4 | T4 Inno Setup 安装包 | packaging/tiku-setup.iss | AC-3 / AC-4 |
| FR-5 | T3 OCS 配置随包更新 | web/ocs-config.json | AC-1 |
| 全链 | T5 端到端冒烟与文档 | packaging/README.md | AC-1 / AC-2 / AC-3 |

## 任务依赖与分层

- T1 独立先行（Go 代码 + 测试）。
- T2 依赖 T1（新路由要进 exe）。
- T3 依赖 T2（托盘管理的 exe 由 T2 产出）。
- T4 依赖 T3（安装器打包 T3 的目录）。
- T5 依赖 T4（端到端在安装产物上验证）。

## 风险与对策

- Go 1.22+ 编译不过 → GOTOOLCHAIN 锁 go1.21.13（已验证工具链可经 goproxy.cn 下载）。
- 安装器签名 → 无代码签名证书，安装时 SmartScreen 会提示「仍要运行」，文档写明。
- 8060 端口冲突 → 配置模板 server_port 可改；安装文档写明与 Docker 版互斥运行。
- 卸载残留 → 用户数据目录（data/）卸载时保留，README 写明手动清理路径。
