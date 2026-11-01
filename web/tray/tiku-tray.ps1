Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing

$trayDir = $PSScriptRoot
$webDir = (Resolve-Path (Join-Path $trayDir "..")).Path
$iconPath = Join-Path $trayDir "tiku.ico"

# Tray icon
$notify = New-Object System.Windows.Forms.NotifyIcon
$notify.Icon = New-Object System.Drawing.Icon($iconPath)
$notify.Text = "Tiku Adapter - GLM 5.2"
$notify.Visible = $true

# Context menu
$menu = New-Object System.Windows.Forms.ContextMenuStrip
$menu.Items.Add("Search Page") | Out-Null
$menu.Items.Add("Admin Panel") | Out-Null
$menu.Items.Add("-") | Out-Null
$menu.Items.Add("Restart Service") | Out-Null
$menu.Items.Add("Stop Service") | Out-Null
$menu.Items.Add("-") | Out-Null
$menu.Items.Add("Exit") | Out-Null

$global:proxyPid = $null

function Start-Services {
    $status = & wsl -e docker inspect tiku-adapter --format "{{.State.Running}}" 2>$null
    if ($status -ne "true") {
        & wsl -e docker start tiku-adapter 2>$null | Out-Null
    }
    $existing = Get-Process pythonw -ErrorAction SilentlyContinue
    if (-not $existing) {
        $p = Start-Process pythonw -ArgumentList "server.py" -WorkingDirectory $webDir -PassThru -WindowStyle Hidden
        $global:proxyPid = $p.Id
    }
}

function Stop-Services {
    & wsl -e docker stop tiku-adapter 2>$null | Out-Null
    Get-Process pythonw -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
    $global:proxyPid = $null
}

# Menu events
$menu.Items[0].Add_Click({ Start-Process "http://localhost:8061" })
$menu.Items[1].Add_Click({ Start-Process "http://localhost:8060" })
$menu.Items[3].Add_Click({
    Stop-Services
    Start-Sleep -Seconds 2
    Start-Services
    $notify.BalloonTipTitle = "Tiku Adapter"
    $notify.BalloonTipText = "Service restarted."
    $notify.ShowBalloonTip(2000)
})
$menu.Items[4].Add_Click({
    Stop-Services
    $notify.BalloonTipTitle = "Tiku Adapter"
    $notify.BalloonTipText = "Service stopped."
    $notify.ShowBalloonTip(2000)
})
$menu.Items[6].Add_Click({
    $notify.Visible = $false
    $notify.Dispose()
    [System.Windows.Forms.Application]::Exit()
})

$notify.ContextMenuStrip = $menu

# Left-click: open search page
$notify.Add_MouseUp({
    if ($_.Button -eq [System.Windows.Forms.MouseButtons]::Left) {
        Start-Process "http://localhost:8061"
    }
})

# Double-click: also open search page
$notify.Add_DoubleClick({
    Start-Process "http://localhost:8061"
})

# Start services on launch
Start-Services

# Show startup notification
$notify.BalloonTipTitle = "Tiku Adapter"
$notify.BalloonTipText = "Service started. Left-click or right-click the icon."
$notify.ShowBalloonTip(3000)

# Run message loop (blocks until Exit)
[System.Windows.Forms.Application]::Run()
