$ErrorActionPreference = 'Continue'
$root = 'D:\user\ruanjian\TikuAdapter'
$log  = Join-Path $root 'diag2-stdout.log'

Write-Output '== 1. 杀全部实例 =='
Get-CimInstance Win32_Process -Filter "Name='tiku-adapter.exe'" | ForEach-Object {
    Write-Output ("  kill " + $_.ProcessId + " " + $_.ExecutablePath)
    Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue
}
Start-Sleep -Seconds 2

Write-Output '== 2. 带日志启动（CWD=安装根目录）=='
Start-Process -FilePath (Join-Path $root 'tiku-adapter.exe') -WorkingDirectory $root -WindowStyle Hidden -RedirectStandardOutput $log -RedirectStandardError (Join-Path $root 'diag2-stderr.log')
Start-Sleep -Seconds 4

$conn = Get-NetTCPConnection -State Listen -LocalPort 8060 -ErrorAction SilentlyContinue | Select-Object -First 1
if ($conn) {
    $owner = Get-CimInstance Win32_Process -Filter ('ProcessId=' + $conn.OwningProcess)
    Write-Output ('  监听者: PID ' + $conn.OwningProcess + ' | ' + $owner.ExecutablePath)
} else {
    Write-Output '  ✗ 8060 无监听，启动失败'
    Get-Content $log -Tail 10
    exit 1
}

Write-Output '== 3. 发题（计时）=='
$body = @{
    question = '下列哪位科学家提出了狭义相对论理论'
    options  = @('牛顿', '爱因斯坦', '伽利略', '达尔文')
    type     = 0
} | ConvertTo-Json
$t = Get-Date
try {
    $resp = Invoke-RestMethod -Uri 'http://127.0.0.1:8060/api/search' -Method Post -Body $body -ContentType 'application/json' -TimeoutSec 90
    $elapsed = ((Get-Date) - $t).TotalSeconds
    Write-Output ('  耗时: ' + [math]::Round($elapsed, 1) + 's')
    Write-Output ('  answerKey: ' + ($resp.answer.answerKey -join ','))
    Write-Output ('  answerText: ' + $resp.answer.answerText)
    Write-Output ('  allAnswer: ' + ($resp.answer.allAnswer | ConvertTo-Json -Compress))
    Write-Output ('  预期: B 爱因斯坦')
} catch {
    Write-Output ('  ✗ 请求失败: ' + $_.Exception.Message)
}

Write-Output '== 4. 日志里的 AI/搜题行 =='
Get-Content $log | Where-Object { $_ -match 'AI|fallback|出错|POST|dial|refused|timeout' } | Select-Object -Last 8
