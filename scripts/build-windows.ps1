# scripts/build-windows.ps1 — Windows 原生构建（FR-2 / AC-2）
# 产出 dist/tiku-adapter.exe：前端与 SQLite 内嵌（go:embed + glebarez 纯 Go 驱动），无外部运行时依赖。
# 工具链锁 GOTOOLCHAIN=go1.21.13：golang.org/x/tools@v0.13.0 与 Go 1.22+ 不兼容，
# 本机 Go 版本更新时由 Go 自动经 GOPROXY（goproxy.cn）下载 1.21.13 工具链并校验。
#
# 用法（项目根执行）：
#   powershell -ExecutionPolicy Bypass -File scripts\build-windows.ps1
#   # 冒烟默认绑 127.0.0.1:8060；该地址被现有 Docker 部署占用时（二者 8060 互斥）换绑定地址：
#   powershell -ExecutionPolicy Bypass -File scripts\build-windows.ps1 -SmokeBind "10.11.172.155:8060"
#
# 退出码：0 成功；2 环境问题（go 不存在/项目根不对）；3 前端构建失败；
#         4 产物缺失；5 冒烟地址被占用/应用绑定被拒；6 冒烟探活失败；
#         1 go build 透传的非 0 退出码或其他未预期错误。
param(
    [string]$OutFile = "dist/tiku-adapter.exe",  # 产物路径（相对项目根）
    [string]$SmokeBind = "127.0.0.1:8060",       # 冒烟绑定地址 host:port（写入临时 config.yaml 的 server_port）
    [switch]$BuildWeb,                           # 强制重建前端（默认 cmd/adapter-service/dist 缺失才构建，需 npm）
    [switch]$SkipSmoke                           # 跳过启动冒烟
)

$ErrorActionPreference = "Stop"

# ---- 定位项目根（本脚本位于 scripts/ 下）----
$ProjectRoot = Split-Path -Parent $PSScriptRoot
if (-not (Test-Path (Join-Path $ProjectRoot "go.mod"))) {
    Write-Host "[build-windows] 找不到项目根（go.mod）：$ProjectRoot"
    exit 2
}

# ---- 工具链锁定：GOTOOLCHAIN=go1.21.13 ----
$env:GOTOOLCHAIN = "go1.21.13"
$env:CGO_ENABLED = "0"      # SQLite 走 glebarez（纯 Go），禁 CGO 得静态 exe
# 不设 GO111MODULE：Go 1.24+ 已移除该变量，设置即报错；模块模式是默认行为

$go = (Get-Command go -ErrorAction SilentlyContinue).Source
if (-not $go -and (Test-Path "C:\Program Files\Go\bin\go.exe")) { $go = "C:\Program Files\Go\bin\go.exe" }
if (-not $go) {
    Write-Host "[build-windows] 未找到 go.exe：请安装 Go 或将其加入 PATH"
    exit 2
}
$goVer = (& $go version)
Write-Host "[1/4] 工具链：$goVer | GOTOOLCHAIN=$($env:GOTOOLCHAIN) | CGO_ENABLED=$($env:CGO_ENABLED)"

# ---- 前端：内嵌进 exe（main.go 的 //go:embed dist）----
$webDistIndex = Join-Path $ProjectRoot "cmd/adapter-service/dist/index.html"
if ($BuildWeb -or -not (Test-Path $webDistIndex)) {
    Write-Host "[2/4] 构建前端：web/ -> cmd/adapter-service/dist（npm）"
    Push-Location (Join-Path $ProjectRoot "web")
    try {
        npm install
        if ($LASTEXITCODE -ne 0) { Write-Host "[build-windows] npm install 失败（退出码 $LASTEXITCODE）"; exit 3 }
        npm run build
        if ($LASTEXITCODE -ne 0) { Write-Host "[build-windows] npm run build 失败（退出码 $LASTEXITCODE）"; exit 3 }
    } finally { Pop-Location }
    $built = Join-Path $ProjectRoot "web/dist"
    $target = Join-Path $ProjectRoot "cmd/adapter-service/dist"
    if (-not (Test-Path (Join-Path $built "index.html"))) {
        Write-Host "[build-windows] 前端产物缺失：$built"
        exit 3
    }
    if (Test-Path $target) { Remove-Item -Recurse -Force $target }
    Move-Item $built $target
} else {
    Write-Host "[2/4] 复用已有前端：cmd/adapter-service/dist"
}

# ---- Go 编译 ----
Write-Host "[3/4] go build -> $OutFile"
Push-Location $ProjectRoot
try {
    & $go build -ldflags "-s -w" -o $OutFile ./cmd/adapter-service
    if ($LASTEXITCODE -ne 0) {
        Write-Host "[build-windows] go build 失败，退出码 $LASTEXITCODE"
        exit $LASTEXITCODE
    }
} finally { Pop-Location }

$exePath = Join-Path $ProjectRoot $OutFile
if (-not (Test-Path $exePath)) {
    Write-Host "[build-windows] 产物不存在：$exePath"
    exit 4
}
$exeMB = [math]::Round((Get-Item $exePath).Length / 1MB, 1)
Write-Host "    产物就绪：$exePath（$exeMB MB）"

# ---- 启动冒烟：探活 200 ----
if ($SkipSmoke) {
    Write-Host "[4/4] 已跳过冒烟（-SkipSmoke）"
    exit 0
}

# 解析绑定地址；占用判定不做自绑预检（预检套接字刚释放的端口可能与应用的 bind
# 毫秒级相撞，产生假「占用」），改用监听表：见 Test-AddressCovered。
try {
    $bindIP = [System.Net.IPAddress]::Parse($SmokeBind.Split(':')[0])
} catch {
    $bindIP = ([System.Net.Dns]::GetHostAddresses($SmokeBind.Split(':')[0]))[0]
}
$bindPort = [int]($SmokeBind.Split(':')[1])

$hasTcpTable = [bool](Get-Command Get-NetTCPConnection -ErrorAction SilentlyContinue)

# 目标地址是否已被 LISTEN 套接字覆盖（通配 0.0.0.0/:: 或同址同端口均算覆盖）。
# 覆盖即拒绝冒烟：同端口上跑着等价服务时，探活可能打到旧服务造成假通过。
function Test-AddressCovered([string]$ip, [int]$port) {
    $conns = Get-NetTCPConnection -State Listen -LocalPort $port -ErrorAction SilentlyContinue
    foreach ($c in $conns) {
        if ($c.LocalAddress -eq $ip -or $c.LocalAddress -eq "0.0.0.0" -or $c.LocalAddress -eq "::") {
            return $true
        }
    }
    return $false
}

# 冒烟临时目录：独立 config.yaml + 独立 data/（SQLite），不触碰仓库与本机现有部署
$smokeDir = Join-Path ([System.IO.Path]::GetTempPath()) ("tiku-smoke-" + [guid]::NewGuid().ToString("N").Substring(0, 8))
New-Item -ItemType Directory -Path $smokeDir | Out-Null
Copy-Item $exePath (Join-Path $smokeDir "tiku-adapter.exe")
# 双键写入 server_port 与 serverport：应用的 Config 结构体只有 yaml 标签，而
# viper.Unmarshal 按 mapstructure 规则匹配（键名去下划线才命中 ServerPort 字段）。
# 两个键写同一地址，无论当前/修复后的映射行为如何，冒烟端口都确定生效。
$configLines = @(
    "server_port: `"$SmokeBind`"",
    "serverport: `"$SmokeBind`""
)
$configLines | Set-Content -Path (Join-Path $smokeDir "config.yaml") -Encoding UTF8
$appOut = Join-Path $smokeDir "app-stdout.log"
$appErr = Join-Path $smokeDir "app-stderr.log"

function Show-AppLog {
    foreach ($f in @($appOut, $appErr)) {
        if (Test-Path $f) {
            Write-Host "---- $(Split-Path -Leaf $f) 尾部 ----"
            Get-Content $f -Tail 15 | ForEach-Object { Write-Host "    $_" }
        }
    }
}

# 启动前：地址已被别的监听覆盖 => 拒绝冒烟（防假通过）
if ($hasTcpTable -and (Test-AddressCovered $bindIP.ToString() $bindPort)) {
    $holders = (Get-NetTCPConnection -State Listen -LocalPort $bindPort -ErrorAction SilentlyContinue |
        Select-Object -ExpandProperty OwningProcess -Unique) -join ","
    Write-Host (@"
[build-windows] 冒烟地址 $SmokeBind 已被监听（PID: $holders）。打包版与现有 Docker 部署共用 8060、二者互斥：
在占用服务运行时对新 exe 探活会打到旧服务、得到假通过。停止占用进程后重跑，或用 -SmokeBind 换空闲地址。
"@)
    exit 5
}

# 探活函数：curl.exe 直连（--noproxy '*' 排除系统代理干扰），输出状态码
function Invoke-Probe([string]$Url) {
    $curlExe = Join-Path $env:SystemRoot "System32\curl.exe"
    if (Test-Path $curlExe) {
        $code = & $curlExe --noproxy "*" -s -o NUL -w "%{http_code}" --max-time 8 $Url 2>$null
        return [int]($code -as [int])
    }
    try {
        $r = Invoke-WebRequest -Uri $Url -Method GET -UseBasicParsing -TimeoutSec 8
        return [int]$r.StatusCode
    } catch { return 0 }
}

$proc = $null
$listened = $false
$apiOk = $false
$webOk = $false
try {
    Write-Host "[4/4] 启动冒烟：http://$SmokeBind（临时目录 $smokeDir）"
    $proc = Start-Process -FilePath (Join-Path $smokeDir "tiku-adapter.exe") `
        -WorkingDirectory $smokeDir -PassThru -WindowStyle Hidden `
        -RedirectStandardOutput $appOut -RedirectStandardError $appErr

    # 监听确认：我们启动的进程 PID 必须出现在 <bindIP>:<bindPort> 的 LISTEN 表里，
    # 才允许探活——证明后续 200 响应来自新 exe 而非同口旧服务。
    if ($hasTcpTable) {
        $listenDeadline = (Get-Date).AddSeconds(20)
        while ((Get-Date) -lt $listenDeadline) {
            if ($proc.HasExited) { break }
            $mine = Get-NetTCPConnection -State Listen -LocalPort $bindPort -OwningProcess $proc.Id -ErrorAction SilentlyContinue
            if ($mine | Where-Object { $_.LocalAddress -eq $bindIP.ToString() }) { $listened = $true; break }
            Start-Sleep -Milliseconds 300
        }
    } else {
        Write-Host "    （无 Get-NetTCPConnection，跳过监听确认，仅靠应用日志与探活判断）"
        $listened = $true
    }

    if ($listened) {
        $deadline = (Get-Date).AddSeconds(60)
        while ((Get-Date) -lt $deadline) {
            if ($proc.HasExited) { break }
            if ((Invoke-Probe "http://$SmokeBind/api/search") -eq 200) { $apiOk = $true; break }
            Start-Sleep -Milliseconds 500
        }
        if ($apiOk) {
            $webOk = ((Invoke-Probe "http://$SmokeBind/") -eq 200)
        }
    }
} finally {
    if ($proc -and -not $proc.HasExited) {
        Stop-Process -Id $proc.Id -Force
        $proc.WaitForExit(5000) | Out-Null
    }
}

if (-not $listened) {
    Show-AppLog
    $bindConflict = $false
    foreach ($f in @($appOut, $appErr)) {
        if (Test-Path $f) { if (Select-String -Path $f -Pattern "bind:" -SimpleMatch -Quiet) { $bindConflict = $true } }
    }
    Remove-Item -Recurse -Force $smokeDir -ErrorAction SilentlyContinue
    if ($bindConflict) {
        Write-Host (@"
[build-windows] 冒烟失败：应用绑定 $SmokeBind 被拒（地址被占用或刚释放未完全）。
打包版与现有 Docker 部署共用 8060、二者互斥：停止占用进程后重跑，或用 -SmokeBind 换空闲地址。
"@)
        exit 5
    }
    Write-Host "[build-windows] 冒烟失败：应用未在 $SmokeBind 进入监听（日志见上）"
    exit 6
}
if (-not $apiOk) {
    $exitInfo = if ($proc -and $proc.HasExited) { "进程退出码 $($proc.ExitCode)" }
               elseif ($proc) { "进程仍在运行（60s 内探活未通过）" }
               else { "进程未启动" }
    Show-AppLog
    Remove-Item -Recurse -Force $smokeDir -ErrorAction SilentlyContinue
    Write-Host "[build-windows] 冒烟失败：/api/search 60s 内未返回 200（$exitInfo；日志见上）"
    exit 6
}
if (-not $webOk) {
    Show-AppLog
    Remove-Item -Recurse -Force $smokeDir -ErrorAction SilentlyContinue
    Write-Host "[build-windows] 冒烟失败：前端页 GET / 未返回 200（日志见上）"
    exit 6
}
Write-Host "    冒烟通过：PID $($proc.Id) 监听 $SmokeBind | GET /api/search -> 200 | GET / -> 200（exe 启动正常）"
Remove-Item -Recurse -Force $smokeDir -ErrorAction SilentlyContinue
exit 0
