' Tiku Tray Launcher - starts PowerShell hidden (no window flash)
Set shell = CreateObject("WScript.Shell")
shell.Run "powershell.exe -ExecutionPolicy Bypass -WindowStyle Hidden -File """ & Replace(WScript.ScriptFullName, "tiku-launcher.vbs", "tiku-tray.ps1") & """", 0, False
