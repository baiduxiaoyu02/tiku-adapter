---
depth: quick
signals:
  changed_files: 7
  cross_module: false
  interface_or_model_change: false
  irreversible: false
  new_dependency: true
  unfamiliar_stack: false
  time_sensitive_fact: false
  multiple_candidates: false
  user_requested: true
sources: 7
skipped: false
skip_reason: ""
---

# 调研：题库服务 Windows 桌面打包

## 结论

- tiku-adapter 是纯 Go 静态编译单文件（CGO_ENABLED=0 + glebarez/sqlite 纯驱动 + go:embed 前端），Windows 原生构建可行，Docker 只是外壳。 来源: Dockerfile:3
- OCS「默认」解析器硬拦截含 `adapter-service/search` 的配置（保存流程硬判断 + return），必须用别名路由或内置 TikuAdapter 模式。 来源: https://github.com/ocsjs/ocsjs/blob/master/packages/scripts/src/projects/common.ts
- OCS 题库状态探测 = HEAD（405/501 回退 GET），HTTP 2xx-3xx 算可达、404 算失败。 来源: https://github.com/ocsjs/ocsjs/blob/master/packages/scripts/src/utils/ui.ts
- golang.org/x/tools@v0.13.0 与 Go 1.22+ 编译不兼容（负数组长度技巧失效），本机构建必须锁 go1.21.13 工具链。 来源: 实测：Go 1.27 构建报 invalid array length -delta * delta
- go1.21.13 工具链可经 goproxy.cn 下载，但要求校验数据库开启（GOSUMDB=off 会拒绝工具链下载）。 来源: 实测：GOSUMDB=off 时报 verifying module: checksum database disabled
- Inno Setup 6 与 Go 1.27 均可经 winget 静默安装。 来源: 命令: winget install GoLang.Go / JRSoftware.InnoSetup（已装 go1.27.0 + ISCC 6.x）

## 方案对比

| 方案 | 组成 | 优点 | 缺点 | 来源 |
|------|------|------|------|------|
| A 轻量安装包 | Windows exe + Go 内置别名路由 + PS 托盘 + Inno Setup | 工作量小、去 Docker/Python、OCS 配置直连 | 托盘是脚本、无主窗口 | 实测 |
| B Electron 桌面 | Electron 壳 + 托盘 + 主界面 + Go 子进程 | 最像 ZCode/OCS Desktop、可内嵌超星登录页 | 工作量半天、包体 150MB+ | 实测 |
| C Go 单文件 | 代理+托盘全并进 Go 二进制（systray） | 单 exe 零依赖 | 需引 systray 库、改 Go 代码最多 | https://github.com/getlantern/systray |

选择：方案 A（用户已拍板，"1"）。

## 未决问题

- 安装包是否要内置默认 AI key 模板（当前 config.yaml 模板不含 key，首启后用户填或复用现有配置）。
- OCS Desktop 与打包软件的 8060 端口互斥：同时装两套时需要一个改端口，文档写明即可。

## 假设

- 假设: 构建产物 dist/ 不入 git（.gitignore 追加），安装包是唯一交付物。
- 假设: 用户数据（SQLite 题库）在卸载时保留，路径文档写明。
