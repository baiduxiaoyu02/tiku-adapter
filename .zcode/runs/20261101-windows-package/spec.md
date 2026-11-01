# 规格：题库服务 Windows 桌面打包（方案 A）

## 目标（Goal）

把 tiku-adapter 题库服务从「Docker 容器 + Python 代理 + PowerShell 托盘」三层组件打包成
独立 Windows 桌面软件：双击启动、托盘常驻、开机自启可选，去掉 Docker 与 Python 依赖。
安装包交付，OCS 题库配置指向本机 `/api/search`。

## 用户故事

1. 作为用户，我双击桌面「题库服务」图标后题库服务在本机启动并在托盘常驻，不再需要 Docker/WSL。
2. 作为用户，安装后 OCS 的题库配置指向本机 `http://localhost:8060/api/search` 即可搜题（含图片题）。
3. 作为用户，我可以通过开始菜单与控制面板卸载软件，卸载移除程序与快捷方式。

## 需求（Functional Requirements）

- FR-1 `/api/search` 路由别名：与 `/adapter-service/search` 复用同一搜题处理器；POST 搜题、
  GET/HEAD 探活返回 200。目的：OCS 在「默认」解析器下硬拦截含 `adapter-service/search` 的
  配置（common.ts 保存流程硬判断），别名路由绕开该字符串。
- FR-2 Windows 原生构建：产出 `dist/tiku-adapter.exe`（前端与 SQLite 内嵌，无外部运行时依赖）。
  构建工具链锁 go1.21.13（golang.org/x/tools@v0.13.0 与 Go 1.22+ 不兼容）。
- FR-3 打包目录结构：`tiku-adapter.exe` + `config.yaml`（本地路径与 AI 配置模板）+ `data/`
  （SQLite 数据库，安装后生成）+ `tray/`（托盘脚本，启动本地 exe 而非 Docker）。
- FR-4 Inno Setup 安装包：安装到 Program Files、开始菜单与桌面快捷方式、可选开机自启、
  控制面板卸载器（卸载保留用户数据目录）。
- FR-5 OCS 配置文件随包分发：`ocs-config.json` 的 url 指向 `http://localhost:8060/api/search`。

## 验收（Acceptance）

- AC-1（FR-1）`POST /api/search` 返回与 `/adapter-service/search` 相同 JSON 结构；
  `GET /api/search` 与 `HEAD /api/search?t=...` 返回 200。有自动化路由测试覆盖。
- AC-2（FR-2）`dist/tiku-adapter.exe` 在本机启动后 8060 端口可访问，前端页面 HTTP 200。
- AC-3（FR-3/FR-4）安装包编译成功；安装后桌面/开始菜单出现「题库服务」快捷方式，
  托盘图标出现，搜题请求经本机服务返回正确答案。
- AC-4（FR-4）卸载后程序目录与快捷方式消失，用户数据目录保留。

## 非目标（Out of Scope）

- 不改 Docker 镜像构建链（现有容器部署继续可用，二者并存但互不依赖）。
- 不做 Electron 图形主界面（方案 B 范畴，后续可选升级）。
- 不打包 OCS 本身（用户已装 OCS Desktop 2.9.24，其题库配置指向本机服务）。
- 不做自动更新器（版本升级走重新安装）。
- 不做代码签名（无证书，安装时 SmartScreen 提示「仍要运行」，文档写明）。

## 澄清（Clarification）

- 用户在「方案 A / 方案 C」中选择了方案 A（轻量安装包），明确不做 Electron 壳。
- 「像 ZCode 这样的软件」的落点 = 托盘常驻 + 安装包 + 开机自启，不含主窗口（当前无此需求，
  搜题入口是 OCS 面板与托盘菜单）。
- 打包后与现有 Docker 部署并存：8060 端口互斥使用，文档写明切换方式。

## 假设

- AI key 沿用现有 sensenova 配置（含视觉模型 sensenova-6.8-flash-lite）。
- OCS Desktop 的题库配置由用户通过剪贴板 + 「读取剪贴板」按钮粘贴一次（流程已就绪）。
- 构建工具链：Go 1.27（已装）+ GOTOOLCHAIN=go1.21.13（经 goproxy.cn 下载并校验）；
  Inno Setup 6（已装）。
