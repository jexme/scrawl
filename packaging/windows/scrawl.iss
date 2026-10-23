; Inno Setup 6 script: a per-user installer for Scrawl (no administrator rights needed).
;
; Build (paths below are relative to this file; run scripts\build-windows.ps1 first so that
; bin\scrawl.exe exists):
;   iscc /DAppVersion=0.1.0-rc3 /DNumVersion=0.1.0.0 packaging\windows\scrawl.iss
; The result is dist\scrawl-v0.1.0-rc3-windows-amd64-setup.exe.
;
; Keep this file ASCII: Inno reads a script without a BOM in the ANSI code page.

#ifndef AppVersion
  #define AppVersion "0.0.0-dev"
#endif
; The file version resource must be numeric (four parts); AppVersion may carry a suffix like -rc3.
#ifndef NumVersion
  #define NumVersion "0.0.0.0"
#endif
#ifndef SourceExe
  #define SourceExe "..\..\bin\scrawl.exe"
#endif
#ifndef OutputDir
  #define OutputDir "..\..\dist"
#endif

[Setup]
; Never change AppId: it is how an upgrade finds the previous installation.
AppId={{CF20B818-C622-4988-88E8-A3895472DE4F}
AppName=Scrawl
AppVersion={#AppVersion}
AppVerName=Scrawl {#AppVersion}
AppPublisher=jexme
AppPublisherURL=https://github.com/jexme/scrawl
AppSupportURL=https://github.com/jexme/scrawl/issues
AppUpdatesURL=https://github.com/jexme/scrawl/releases
VersionInfoVersion={#NumVersion}
VersionInfoProductName=Scrawl
VersionInfoDescription=Scrawl setup
; Per user, no UAC prompt. With PrivilegesRequired=lowest, {autopf} is
; %LOCALAPPDATA%\Programs, so the program lands in %LOCALAPPDATA%\Programs\Scrawl.
PrivilegesRequired=lowest
DefaultDirName={autopf}\Scrawl
DisableProgramGroupPage=yes
DisableDirPage=auto
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
; The running tray is stopped by hand in PrepareToInstall (see [Code]); the Restart Manager
; dialog would only get in the way.
CloseApplications=no
SetupIconFile=scrawl.ico
UninstallDisplayIcon={app}\scrawl.exe
UninstallDisplayName=Scrawl
OutputDir={#OutputDir}
OutputBaseFilename=scrawl-v{#AppVersion}-windows-amd64-setup
Compression=lzma2
SolidCompression=yes
WizardStyle=modern

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
; Optional and off by default, like the checkbox in the program's settings.
Name: "autostart"; Description: "Start Scrawl when I sign in to Windows"; GroupDescription: "Additional options:"; Flags: unchecked

[Files]
Source: "{#SourceExe}"; DestDir: "{app}"; DestName: "scrawl.exe"; Flags: ignoreversion
Source: "..\..\LICENSE"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\..\THIRD_PARTY_NOTICES.md"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{autoprograms}\Scrawl"; Filename: "{app}\scrawl.exe"; Parameters: "-tray"; Comment: "Screenshots with annotations (runs in the notification area)"
Name: "{autoprograms}\Scrawl settings"; Filename: "{app}\scrawl.exe"; Parameters: "-settings"

[Registry]
; The same value and command that the program itself writes (internal/autostart). It is removed
; on uninstall; an entry made later by the program's own settings is removed in [Code].
Root: HKCU; Subkey: "Software\Microsoft\Windows\CurrentVersion\Run"; ValueType: string; ValueName: "Scrawl"; ValueData: """{app}\scrawl.exe"" -tray -autostart"; Flags: uninsdeletevalue; Tasks: autostart

[Run]
Filename: "{app}\scrawl.exe"; Parameters: "-tray"; Description: "Start Scrawl now"; Flags: nowait postinstall skipifsilent

[Code]
{ Quit every running scrawl.exe: first politely (WM_CLOSE reaches the hidden tray window),
  then by force. Without this the old tray would keep the files locked and the hotkey. }
procedure StopScrawl;
var
  ResultCode: Integer;
begin
  Exec(ExpandConstant('{sys}\taskkill.exe'), '/IM scrawl.exe', '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
  Sleep(500);
  Exec(ExpandConstant('{sys}\taskkill.exe'), '/F /IM scrawl.exe', '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
  Sleep(300);
end;

function PrepareToInstall(var NeedsRestart: Boolean): String;
begin
  StopScrawl;
  Result := '';
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
begin
  if CurUninstallStep = usUninstall then
    StopScrawl
  else if CurUninstallStep = usPostUninstall then
    { The user may have switched autostart on in the program's settings, not in this installer. }
    RegDeleteValue(HKEY_CURRENT_USER, 'Software\Microsoft\Windows\CurrentVersion\Run', 'Scrawl');
end;
