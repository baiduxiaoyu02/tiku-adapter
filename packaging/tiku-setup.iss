; packaging/tiku-setup.iss - Inno Setup 6 安装包（FR-4 / AC-3 / AC-4）
; 编译："<ISCC>\ISCC.exe" packaging\tiku-setup.iss -> 产物 dist\tiku-adapter-setup.exe
; 安装布局与 packaging/tray/tiku-tray.ps1 头注释一致：
;   tiku-adapter.exe  主程序（T2 产物，前端与 SQLite 内嵌）
;   config.yaml       配置模板（packaging/config.yaml；已存在则保留用户配置）
;   data\             SQLite 数据目录（运行期生成，卸载保留）
;   tray\tiku-tray.ps1 / tray\tiku-launcher.vbs / tray\tiku.ico
;   ocs-config.json   OCS 题库配置（url 指向 http://localhost:8060/api/search，FR-5）
; AppVersion 取 1.0.0（代码内无版本常量，升级走重新安装，spec 非目标不含自动更新）。

#define MyAppName "题库服务 TikuAdapter"
#define MyAppVersion "1.0.0"
#define MyAppPublisher "tikuAdapter"
#define MyAppURL "https://github.com/DokiDoki1103/tikuAdapter"

[Setup]
; AppId 为一次性生成并固定的 GUID：升级/卸载靠它识别既有安装
AppId={{56ED641C-36BD-4197-BC97-E15945B6D654}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher={#MyAppPublisher}
AppPublisherURL={#MyAppURL}
AppSupportURL={#MyAppURL}
DefaultDirName={autopf}\TikuAdapter
DefaultGroupName=题库服务
UninstallDisplayName={#MyAppName}
UninstallDisplayIcon={app}\tiku-adapter.exe
SetupIconFile=..\web\tray\tiku.ico
OutputDir=..\dist
OutputBaseFilename=tiku-adapter-setup
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
; 安装到 Program Files（FR-4）需要管理员权限；{autopf} 在 64 位进程下落到 Program Files
PrivilegesRequired=admin
ArchitecturesInstallIn64BitMode=x64compatible
ArchitecturesAllowed=x64compatible

[Tasks]
Name: "desktopicon"; Description: "创建桌面快捷方式(&D)"; GroupDescription: "附加图标:"
Name: "autostart"; Description: "开机自动启动题库服务（当前用户登录后托盘常驻）(&S)"; GroupDescription: "自启动:"; Flags: unchecked

[Dirs]
; data\ 在 Program Files 下普通用户无写权限：安装时预建并授予 Users 修改权，
; 首启生成的 data\tiku.db 才能落盘。卸载时 Inno 只删除空目录，运行期数据库不在
; 安装清单内 -> 用户数据目录保留（AC-4；手动清理路径由 T5 README 写明）。
Name: "{app}\data"; Permissions: users-modify

[Files]
Source: "..\dist\tiku-adapter.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "config.yaml"; DestDir: "{app}"; Flags: onlyifdoesntexist
Source: "tray\tiku-tray.ps1"; DestDir: "{app}\tray"; Flags: ignoreversion
Source: "tray\tiku-launcher.vbs"; DestDir: "{app}\tray"; Flags: ignoreversion
Source: "..\web\tray\tiku.ico"; DestDir: "{app}\tray"; DestName: "tiku.ico"; Flags: ignoreversion
Source: "..\web\ocs-config.json"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\题库服务"; Filename: "{app}\tray\tiku-launcher.vbs"; IconFilename: "{app}\tray\tiku.ico"; Comment: "启动题库服务（托盘常驻）"
Name: "{group}\卸载题库服务"; Filename: "{uninstallexe}"; Comment: "卸载题库服务（保留数据目录）"
Name: "{autodesktop}\题库服务"; Filename: "{app}\tray\tiku-launcher.vbs"; IconFilename: "{app}\tray\tiku.ico"; Tasks: desktopicon; Comment: "启动题库服务（托盘常驻）"

[Registry]
; 可选自启（任务 autostart，默认不勾选）：登录后经 VBS 无黑窗拉起托盘，托盘负责启动本地服务。
; HKCU 按当前用户生效；任务勾选才写入，卸载时自动删除。
Root: HKCU; Subkey: "Software\Microsoft\Windows\CurrentVersion\Run"; ValueType: string; ValueName: "TikuAdapter"; ValueData: """wscript.exe"" ""{app}\tray\tiku-launcher.vbs"""; Tasks: autostart; Flags: uninsdeletevalue

[Run]
; 安装完成页可选立即启动（静默安装跳过，避免 T5 冒烟被意外拉起的进程干扰）
Filename: "wscript.exe"; Parameters: """{app}\tray\tiku-launcher.vbs"""; Description: "启动题库服务（托盘常驻）"; Flags: postinstall skipifsilent nowait

[Code]
// 卸载前停掉本安装的服务与托盘，避免文件占用导致卸载不净（T5 端到端验证）。
// 与托盘脚本同一哲学：只按安装路径/命令行精确匹配，不动机器上其他副本。
// 注意：ExecutablePath 是 exe 全路径，必须与 '{app}\tiku-adapter.exe' 比较（与目录
// '{app}' 比较永为 false，杀不掉进程，见 T5 冒烟 ⑤）；托盘按本安装路径限定 -like
// （-like 是全串匹配，路径串前后都要有 * 通配），并排除枚举进程自身
// （$PID 的命令行也含该路径串，否则会自杀）。
procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
var
  RC: Integer;
begin
  if CurUninstallStep = usUninstall then
  begin
    Exec('powershell.exe',
      '-NoProfile -ExecutionPolicy Bypass -Command "Get-CimInstance Win32_Process -Filter \"Name=''tiku-adapter.exe''\" | Where-Object { $_.ExecutablePath -eq ''' + ExpandConstant('{app}') + '\tiku-adapter.exe'' } | ForEach-Object { Stop-Process -Id $_.ProcessId -Force }"',
      '', SW_HIDE, ewWaitUntilTerminated, RC);
    Exec('powershell.exe',
      '-NoProfile -ExecutionPolicy Bypass -Command "Get-CimInstance Win32_Process -Filter \"Name=''powershell.exe''\" | Where-Object { ($_.ProcessId -ne $PID) -and ($_.CommandLine -like ''' + '*' + ExpandConstant('{app}') + '\tray\tiku-tray.ps1*'') } | ForEach-Object { Stop-Process -Id $_.ProcessId -Force }"',
      '', SW_HIDE, ewWaitUntilTerminated, RC);
  end;
end;
