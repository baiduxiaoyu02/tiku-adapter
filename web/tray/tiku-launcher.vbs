' packaging/tray/tiku-launcher.vbs - windowless launcher for the tray script.
' Created/installed by packaging/tiku-setup.iss (see layout in tiku-tray.ps1 header).
' Start-menu / desktop shortcuts and the optional autostart Run entry point here;
' the tray script locates the install root itself from its own path.
' ASCII-only on purpose: wscript misreads UTF-8 without BOM.
Option Explicit
Dim fso, sh, trayDir, installRoot, cmd
Set fso = CreateObject("Scripting.FileSystemObject")
Set sh = CreateObject("WScript.Shell")
trayDir = fso.GetParentFolderName(WScript.ScriptFullName)
installRoot = fso.GetParentFolderName(trayDir)
cmd = "powershell.exe -NoProfile -ExecutionPolicy Bypass -WindowStyle Hidden -File """ & trayDir & "\tiku-tray.ps1"""
sh.CurrentDirectory = installRoot
sh.Run cmd, 0, False
