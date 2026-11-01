# 题库服务 TikuAdapter — Windows 桌面版

题库服务（tikuAdapter）的 Windows 桌面打包：单个 `tiku-adapter.exe`（Go 静态编译，
管理前端与 SQLite 内嵌），**无需 Docker / WSL / Python**。启动后托盘常驻，OCS
题目助手把题库配置指向本机 `/api/search` 即可搜题（含图片题）。

安装包：`dist/tiku-adapter-setup.exe`（Inno Setup 6 编译，见 `packaging/tiku-setup.iss`）。

## 系统要求

- Windows 10 / 11 x64；安装到 `C:\Program Files\TikuAdapter` 需要管理员权限。
- 无外部运行时依赖（Go 静态编译，前端与 SQLite 内嵌）。
- 安装包未做代码签名：SmartScreen 首次运行会提示「Windows 已保护你的电脑」，
  点 **更多信息 → 仍要运行** 即可。

## 安装

### 图形安装

双击 `tiku-adapter-setup.exe`，按向导完成：

- 安装目录：`C:\Program Files\TikuAdapter`
- 附加图标：可选创建桌面快捷方式
- 自启动：可选「开机自动启动题库服务」（默认不勾选；写入当前用户
  `HKCU\...\Run` 的 `TikuAdapter` 值，登录后经托盘拉起服务）
- 安装完成页可勾选「启动题库服务」立即启动

### 静默安装（命令行）

```bat
tiku-adapter-setup.exe /VERYSILENT /SUPPRESSMSGBOXES /NORESTART
```

可选任务开关（多个用逗号分隔）：

```bat
tiku-adapter-setup.exe /VERYSILENT /SUPPRESSMSGBOXES /NORESTART /TASKS="desktopicon,autostart"
```

### 安装内容

| 位置 | 内容 |
|------|------|
| `C:\Program Files\TikuAdapter\tiku-adapter.exe` | 主程序（前端 + SQLite 内嵌） |
| `...\config.yaml` | 配置文件（已存在则保留用户配置） |
| `...\ocs-config.json` | OCS 题库配置模板（url 已指向本机 `/api/search`） |
| `...\data\` | SQLite 题库数据（首次启动生成；**卸载时保留**） |
| `...\tray\` | 托盘脚本与图标 |
| 开始菜单「题库服务」 | 「题库服务」与「卸载题库服务」快捷方式 |
| 桌面 | 「题库服务」快捷方式（可选） |
| 控制面板 → 应用和功能 | 卸载项「题库服务 TikuAdapter」 |

### 端口与监听地址

- 默认 `127.0.0.1:8060`：只监听本机回环，首次启动不会触发 Windows 防火墙放行弹窗。
- 生效配置键是 **`serverport`**（`server_port` 键不生效——应用按 mapstructure 规则匹配，
  已实测），例：`serverport: "127.0.0.1:8060"`。
- 如需局域网其他设备访问，改为 `serverport: "0.0.0.0:8060"` 并在防火墙放行 8060 端口。
- 修改配置后需重启服务（托盘右键 → 重新启动服务）。

## 与 Docker 版并存（8060 互斥）

桌面版与既有 Docker 部署并存但**互不依赖**；两者默认都用 8060，同一时刻只能运行一个：

- 继续用 Docker 版：不启动桌面版托盘即可。
- 想同时跑（例如迁移对比期）：把桌面版安装目录 `config.yaml` 的
  `serverport` 改成其他端口（如 `"127.0.0.1:18060"`），OCS 配置同步改端口；
  或停用 Docker 容器后桌面版直接用 8060。
- 托盘启动时若检测到端口被其他进程（含 Docker 版）占用会明确报错，不会强抢端口。

## 使用

### 启动与托盘

双击桌面/开始菜单「题库服务」→ 托盘出现紫色 **?** 图标，服务随托盘自动拉起：

- **左键**托盘图标：打开管理面板（`http://127.0.0.1:8060/`，可导入题库、查看记录）
- **右键**菜单：打开管理面板 / 打开数据目录 / 重新启动服务 / 停止服务 / 退出（停止服务）

### 探活

```bat
curl http://127.0.0.1:8060/api/search            :: GET  -> 200
curl -I --head http://127.0.0.1:8060/api/search  :: HEAD -> 200
```

### 搜题接口（OCS 同款）

`POST /api/search`，`Content-Type: application/json`：

```bat
curl -X POST http://127.0.0.1:8060/api/search ^
  -H "Content-Type: application/json" ^
  -d "{\"question\":\"题目文本\",\"options\":[\"选项A\",\"选项B\"],\"type\":0}"
```

- `type`：0 单选 / 1 多选 / 2 填空（completion）/ 3 判断 / 4 简答；`images` 传图片 URL
  数组即为图片题（走视觉模型）。
- 响应核心字段：`answer.bestAnswer`（最可能答案）、`answer.allAnswer`（聚合答案）、
  `answer.answerText`（`#` 分隔文本）。
- 搜题顺序：本地题库 → 已启用的第三方接口 → AI 大模型兜底（`config.yaml` 的 `ai` 段，
  题库无答案时才调用）。

### 路由别名说明

`/api/search` 与 `/adapter-service/search` 是**同一搜题处理器**的两个路由，返回结构完全一致。
OCS「默认」解析器在保存配置时会硬拦截 URL 含 `adapter-service/search` 字符串的题库，
因此随包配置统一使用别名 `/api/search`。

## OCS 对接（OCS Desktop 2.9.24 实测）

1. 打开安装目录的 `ocs-config.json`，全选复制其内容（url 已是
   `http://localhost:8060/api/search`，无需改动；若改过端口，同步改 url 端口）。
2. OCS 设置 → 题库 → 启用自定义题库 → **读取剪贴板** → 保存。
3. 随便打开一道题，OCS 面板即经本机服务搜题。

题库内容来源：管理面板（`http://127.0.0.1:8060/`）导入/记录，或直接向
`POST /api/search` 的请求会按「未收录题目」自动收集（可在 `config.yaml` 用
`recordEmptyAnswer` 控制是否记录空答案）。

## 卸载

- 入口：控制面板「应用和功能」→「题库服务 TikuAdapter」，或开始菜单「卸载题库服务」。
- 行为：删除程序文件、开始菜单/桌面快捷方式、自启注册表值与卸载注册表项；
  **保留 `data\` 用户数据目录**（题库数据库 `data\tiku.db` 不删）。
- 卸载器会在卸载前自动停止本安装的题库服务与托盘进程（只按安装路径精确匹配，
  不影响机器上其他副本）。
- 不再需要数据时手动清理：删除 `C:\Program Files\TikuAdapter\data\` 目录。

## 修复记录（2026-11-01 端到端冒烟发现，同日修复并复验）

- **D1 托盘 UI 路径启动崩溃（已修复）**：`packaging/tray/tiku-tray.ps1` 的
  `New-Object System.Threading.Mutex($true, "Local\tiku-adapter-tray", $true)` 第三参
  是 `out bool`，PowerShell 5.1 要求 `[ref]`；已改为 `[ref]$createdNew`。修复后
  快捷方式/自启 → `tiku-launcher.vbs` → 托盘 → 拉起服务全链路复验通过
  （托盘存活、服务运行、`/api/search` 探活 200）。
- **D2 卸载器未停止运行中的服务（已修复）**：`packaging/tiku-setup.iss [Code]`
  停进程过滤原为 `$_.ExecutablePath -eq '{app}'`（目录与 exe 全路径比较，永为
  false），已改为与 `'{app}\tiku-adapter.exe'` 精确比较；托盘停进程由命令行
  全匹配改为按本安装路径 `'*{app}\tray\tiku-tray.ps1*'` 限定（不误杀机器上其他
  副本）并排除枚举进程自身。模拟卸载实测：服务进程被停止、exe 文件解锁可删。

## 故障排查

| 现象 | 处理 |
|------|------|
| 托盘气泡提示「端口 8060 已被其他进程监听」 | 与 Docker 版冲突：改 `serverport` 或停容器（见上文并存章节） |
| OCS 搜题无响应 | 先确认探活 `curl http://127.0.0.1:8060/api/search` 返回 200；再核对 OCS 题库 url 端口 |
| 图片题不走视觉模型 | 核对 `config.yaml` `ai.vision_model` 与 `ai.enable: true` |
| 改了 config.yaml 不生效 | 确认键名是 `serverport` 而非 `server_port`；改完重启服务 |
| 局域网设备访问不了 | `serverport` 改 `0.0.0.0:8060` 并放行防火墙；注意暴露范围 |
