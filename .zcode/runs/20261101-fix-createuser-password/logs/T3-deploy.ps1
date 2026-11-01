# T3 deploy: copy dist exe to install root and restart instance.
# Match processes by ExecutablePath only (same philosophy as tray script); tray process untouched.
$ErrorActionPreference = "Stop"
$dist        = "D:\user\Zcode\xubo\.zcode\workspace\default\tiku-adapter\dist\tiku-adapter.exe"
$installRoot = "D:\user\ruanjian\TikuAdapter"
$target      = Join-Path $installRoot "tiku-adapter.exe"

# 1. stop old service process (this install only)
$procs = Get-CimInstance Win32_Process -Filter "Name='tiku-adapter.exe'" -ErrorAction SilentlyContinue |
    Where-Object { $_.ExecutablePath -eq $target }
foreach ($p in $procs) { Write-Host "[deploy] stop old service PID $($p.ProcessId)"; Stop-Process -Id $p.ProcessId -Force }
Start-Sleep -Seconds 1
$left = Get-CimInstance Win32_Process -Filter "Name='tiku-adapter.exe'" -ErrorAction SilentlyContinue |
    Where-Object { $_.ExecutablePath -eq $target }
if ($left) { Write-Host "[deploy] FAIL: old process still running"; exit 1 }
if (-not $procs) { Write-Host "[deploy] WARN: no running old service process found" }

# 2. overwrite exe and verify hash
Copy-Item $dist $target -Force
$h1 = (Get-FileHash $dist -Algorithm SHA256).Hash
$h2 = (Get-FileHash $target -Algorithm SHA256).Hash
Write-Host "[deploy] dist    SHA256: $h1"
Write-Host "[deploy] deployed SHA256: $h2"
if ($h1 -ne $h2) { Write-Host "[deploy] FAIL: hash mismatch"; exit 1 }

# 3. restart service (same launch style as tray Start-App, tiku-tray.ps1 line 93)
Start-Process -FilePath $target -WorkingDirectory $installRoot -WindowStyle Hidden | Out-Null

# 4. probe GET /api/search expecting 200
$code = 0
$deadline = (Get-Date).AddSeconds(30)
while ((Get-Date) -lt $deadline) {
    $code = [int](& "$env:SystemRoot\System32\curl.exe" --noproxy "*" -s -o NUL -w "%{http_code}" --max-time 8 "http://127.0.0.1:8060/api/search" 2>$null)
    if ($code -eq 200) { break }
    Start-Sleep -Milliseconds 500
}
$new = Get-CimInstance Win32_Process -Filter "Name='tiku-adapter.exe'" -ErrorAction SilentlyContinue |
    Where-Object { $_.ExecutablePath -eq $target }
if ($code -eq 200) { Write-Host "[deploy] OK: probe 200, new service PID $($new.ProcessId)"; exit 0 }
Write-Host "[deploy] FAIL: probe returned $code"
exit 1
