# packaging/tray/tiku-tray.ps1 — Windows 桌面版托盘（FR-3）
# 桌面版托盘只管理本地 tiku-adapter.exe（不再触碰 Docker / WSL / Python 代理）。
#
# 安装根目录布局（由 packaging/tiku-setup.iss 建立）：
#   tiku-adapter.exe   主程序（T2 产物）
#   config.yaml        配置（packaging/config.yaml 模板）
#   data\              SQLite 题库数据（首启自动生成，卸载保留）
#   tray\tiku-tray.ps1 本脚本
#   tray\tiku.ico      托盘图标（可选；缺失时脚本内存生成紫色 ? 图标兜底）
#
# 用法：
#   tray\tiku-launcher.vbs          # 无黑窗启动托盘（安装包快捷方式指向它）
#   powershell -ExecutionPolicy Bypass -File tray\tiku-tray.ps1
#   powershell -ExecutionPolicy Bypass -File tray\tiku-tray.ps1 -SmokeTest
#     # 验证模式：按 config.yaml 启动 exe → 探活 /api/search 200 → 停止服务 → 退出码 0；失败退出码 1
param([switch]$SmokeTest)

$ErrorActionPreference = "Stop"
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing

# ---- 布局定位：本脚本在 <安装根>\tray\ 下 ----
$script:TrayDir     = $PSScriptRoot
$script:InstallRoot = Split-Path -Parent $PSScriptRoot
$script:ExePath     = Join-Path $script:InstallRoot "tiku-adapter.exe"
$script:ConfigPath  = Join-Path $script:InstallRoot "config.yaml"
$script:DataDir     = Join-Path $script:InstallRoot "data"
$script:IconPath    = Join-Path $script:TrayDir "tiku.ico"

# ---- 端口解析：config.yaml 的 serverport / server_port 键 ----
# 应用按 mapstructure 匹配，实际生效键是 serverport（server_port 不生效，已实测）；
# 这里两种键都识别，取首个带端口号的值，无配置时回落 8060。
function Get-ConfigPort {
    $port = 8060
    if (Test-Path -LiteralPath $script:ConfigPath) {
        foreach ($line in (Get-Content -LiteralPath $script:ConfigPath -Encoding UTF8)) {
            if ($line -match '^\s*server_?port\s*:\s*"?[0-9.]+:(\d+)"?\s*(#.*)?$') {
                $port = [int]$Matches[1]
                break
            }
        }
    }
    return $port
}
$script:Port = Get-ConfigPort

# ---- 探活：GET /api/search 期望 200（路由别名由 T1 提供） ----
function Invoke-Probe([string]$Url) {
    $curlExe = Join-Path $env:SystemRoot "System32\curl.exe"
    if (Test-Path $curlExe) {
        $code = & $curlExe --noproxy "*" -s -o NUL -w "%{http_code}" --max-time 8 $Url 2>$null
        $n = 0
        if ([int]::TryParse("$code", [ref]$n)) { return $n }
        return 0
    }
    try {
        $r = Invoke-WebRequest -Uri $Url -Method GET -UseBasicParsing -TimeoutSec 8
        return [int]$r.StatusCode
    } catch { return 0 }
}

# ---- 进程定位：只认可执行路径等于本安装根 exe 的进程 ----
# 不用进程名裸匹配：机器上可能有多份 tiku-adapter（如开发目录里的），托盘只管自己这套。
function Get-AppProcess {
    Get-CimInstance Win32_Process -Filter "Name='tiku-adapter.exe'" -ErrorAction SilentlyContinue |
        Where-Object { $_.ExecutablePath -eq $script:ExePath }
}

# 端口是否被「非本安装」的进程监听（true = 与 Docker 版等其他部署冲突，8060 互斥）
function Test-PortTakenByOther {
    if (-not (Get-Command Get-NetTCPConnection -ErrorAction SilentlyContinue)) { return $false }
    $mineIds = @(Get-AppProcess | ForEach-Object { $_.ProcessId })
    $conns = Get-NetTCPConnection -State Listen -LocalPort $script:Port -ErrorAction SilentlyContinue
    foreach ($c in $conns) {
        if ($mineIds -notcontains $c.OwningProcess) { return $true }
    }
    return $false
}

# ---- 启动服务：确保 data\ 存在 → 防重复启动 → 等 /api/search 探活 200 ----
function Start-App {
    if (-not (Test-Path -LiteralPath $script:ExePath)) {
        return @{ Ok = $false; Reason = "找不到主程序：$($script:ExePath)" }
    }
    if (-not (Test-Path -LiteralPath $script:DataDir)) {
        New-Item -ItemType Directory -Path $script:DataDir -Force | Out-Null
    }
    $already = @(Get-AppProcess).Count -gt 0
    if (-not $already) {
        if (Test-PortTakenByOther) {
            return @{ Ok = $false; Reason = "端口 $($script:Port) 已被其他进程监听（可能与 Docker 版冲突，二者互斥）" }
        }
        Start-Process -FilePath $script:ExePath -WorkingDirectory $script:InstallRoot -WindowStyle Hidden | Out-Null
    }
    $deadline = (Get-Date).AddSeconds(30)
    while ((Get-Date) -lt $deadline) {
        if ((Invoke-Probe "http://127.0.0.1:$($script:Port)/api/search") -eq 200) {
            return @{ Ok = $true; Already = $already }
        }
        Start-Sleep -Milliseconds 500
    }
    return @{ Ok = $false; Reason = "探活 30s 内未通过（http://127.0.0.1:$($script:Port)/api/search）" }
}

# ---- 停止服务：只停本安装根的进程 ----
function Stop-App {
    Get-AppProcess | ForEach-Object {
        Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue
    }
}

# ---- 验证模式：启动 → 探活 → 停止 → 报告（不进托盘消息循环） ----
if ($SmokeTest) {
    Write-Host "[tiku-tray] 安装根：$($script:InstallRoot) | 端口：$($script:Port)"
    $r = Start-App
    if ($r.Ok) {
        $tag = if ($r.Already) { "已在运行" } else { "新启动" }
        Write-Host "[tiku-tray] SMOKE OK：exe $tag，GET /api/search -> 200"
        Stop-App
        Write-Host "[tiku-tray] 验证进程已停止"
        exit 0
    }
    Write-Host "[tiku-tray] SMOKE FAIL：$($r.Reason)"
    exit 1
}

# ---- 托盘 UI ----
# 单实例锁：同一会话重复启动只保留一个托盘图标
# 第三参是 out bool，PS 5.1 下必须用 [ref] 传入，否则 New-Object 绑定构造函数失败
# （参数"3"应为 System.Management.Automation.PSReference），$ErrorActionPreference='Stop' 下托盘必崩。
$createdNew = $false
$mutex = New-Object System.Threading.Mutex($true, "Local\tiku-adapter-tray", [ref]$createdNew)
if (-not $mutex.WaitOne(0)) { exit 0 }

$notify = New-Object System.Windows.Forms.NotifyIcon
if (Test-Path -LiteralPath $script:IconPath) {
    $notify.Icon = New-Object System.Drawing.Icon($script:IconPath)
} else {
    # 兜底：内存生成紫色 ? 图标（与 web/tray/create-icon.ps1 同款视觉，无需落盘文件）
    $size = 64
    $bmp = New-Object System.Drawing.Bitmap($size, $size)
    $g = [System.Drawing.Graphics]::FromImage($bmp)
    $g.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::AntiAlias
    $g.TextRenderingHint = [System.Drawing.Text.TextRenderingHint]::AntiAliasGridFit
    $g.Clear([System.Drawing.Color]::Transparent)
    $brush = New-Object System.Drawing.SolidBrush([System.Drawing.Color]::FromArgb(79, 70, 229))
    $g.FillEllipse($brush, 2, 2, $size - 4, $size - 4)
    $font = New-Object System.Drawing.Font('Segoe UI', [int]($size * 0.6), [System.Drawing.FontStyle]::Bold)
    $white = New-Object System.Drawing.SolidBrush([System.Drawing.Color]::White)
    $sf = New-Object System.Drawing.StringFormat
    $sf.Alignment = [System.Drawing.StringAlignment]::Center
    $sf.LineAlignment = [System.Drawing.StringAlignment]::Center
    $g.DrawString('?', $font, $white, (New-Object System.Drawing.RectangleF(0, -3, $size, $size)), $sf)
    $notify.Icon = [System.Drawing.Icon]::FromHandle($bmp.GetHicon())
    $script:TrayBmp = $bmp   # 保活引用，防止 GC 回收图标句柄
}
$notify.Text = "题库服务 TikuAdapter（:$($script:Port)）"
$notify.Visible = $true

$menu = New-Object System.Windows.Forms.ContextMenuStrip
$menu.Items.Add("打开管理面板") | Out-Null
$menu.Items.Add("打开数据目录") | Out-Null
$menu.Items.Add("-") | Out-Null
$menu.Items.Add("重新启动服务") | Out-Null
$menu.Items.Add("停止服务") | Out-Null
$menu.Items.Add("-") | Out-Null
$menu.Items.Add("退出（停止服务）") | Out-Null

$menu.Items[0].Add_Click({ Start-Process "http://127.0.0.1:$($script:Port)/" })
$menu.Items[1].Add_Click({ Start-Process "explorer.exe" $script:DataDir })
$menu.Items[3].Add_Click({
    Stop-App
    Start-Sleep -Seconds 1
    $r = Start-App
    $notify.BalloonTipTitle = "题库服务"
    $notify.BalloonTipText = if ($r.Ok) { "服务已重启（127.0.0.1:$($script:Port)）。" } else { "重启失败：$($r.Reason)" }
    $notify.ShowBalloonTip(2000)
})
$menu.Items[4].Add_Click({
    Stop-App
    $notify.BalloonTipTitle = "题库服务"
    $notify.BalloonTipText = "服务已停止。"
    $notify.ShowBalloonTip(2000)
})
$menu.Items[6].Add_Click({
    Stop-App
    $notify.Visible = $false
    $notify.Dispose()
    $mutex.ReleaseMutex()
    [System.Windows.Forms.Application]::Exit()
})
$notify.ContextMenuStrip = $menu

# 左键：打开管理面板（打包版 exe 内嵌的是管理 SPA；搜题主入口是 OCS 面板）
$notify.Add_MouseUp({
    if ($_.Button -eq [System.Windows.Forms.MouseButtons]::Left) {
        Start-Process "http://127.0.0.1:$($script:Port)/"
    }
})

# 启动即拉起服务并气泡提示
$r = Start-App
$notify.BalloonTipTitle = "题库服务"
if ($r.Ok) {
    $notify.BalloonTipText = "服务已启动（127.0.0.1:$($script:Port)）。左键打开管理面板，右键菜单管理服务。"
} else {
    $notify.BalloonTipText = "服务启动失败：$($r.Reason)"
}
$notify.ShowBalloonTip(3000)

# 消息循环（阻塞至「退出」）
[System.Windows.Forms.Application]::Run()
