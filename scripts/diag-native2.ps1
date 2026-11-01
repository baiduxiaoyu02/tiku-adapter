$ErrorActionPreference = 'Continue'
$root = 'D:\user\ruanjian\TikuAdapter'
$log  = Join-Path $root 'diag3-stdout.log'

Write-Output '== 1. kill all =='
Get-CimInstance Win32_Process -Filter "Name='tiku-adapter.exe'" | ForEach-Object {
    Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue
}
Start-Sleep -Seconds 2

Write-Output '== 2. start with capture =='
Start-Process -FilePath (Join-Path $root 'tiku-adapter.exe') -WorkingDirectory $root -WindowStyle Hidden -RedirectStandardOutput $log -RedirectStandardError (Join-Path $root 'diag3-stderr.log')
Start-Sleep -Seconds 4
$conn = Get-NetTCPConnection -State Listen -LocalPort 8060 -ErrorAction SilentlyContinue | Select-Object -First 1
Write-Output ('  listener PID: ' + $(if ($conn) { $conn.OwningProcess } else { 'NONE' }))

Write-Output '== 3. curl POST (max-time 60) =='
$body = '{"question":"下列哪位科学家提出了狭义相对论理论","options":["牛顿","爱因斯坦","伽利略","达尔文"],"type":0}'
$outFile = Join-Path $root 'diag3-response.json'
$code = curl.exe -s -m 60 -o $outFile -w '%{http_code} %{time_total}s' -X POST 'http://127.0.0.1:8060/api/search' -H 'Content-Type: application/json' -d $body
Write-Output ('  HTTP: ' + $code)
Write-Output ('  response: ' + (Get-Content $outFile -Raw -ErrorAction SilentlyContinue))

Write-Output '== 4. log tail (AI/GIN) =='
Get-Content $log | Where-Object { $_ -match 'AI|fallback|POST|出错|熔断' } | Select-Object -Last 8
