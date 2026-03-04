#ifndef MyAppVersion
  #define MyAppVersion "0.0.0"
#endif

[Setup]
AppName=logcurse
AppVersion={#MyAppVersion}
AppPublisher=wasson-ece
AppPublisherURL=https://github.com/wasson-ece/logcurse
AppSupportURL=https://github.com/wasson-ece/logcurse/issues
DefaultDirName={autopf}\logcurse
DefaultGroupName=logcurse
DisableProgramGroupPage=yes
OutputBaseFilename=logcurse-setup-amd64
Compression=lzma2
SolidCompression=yes
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
PrivilegesRequired=lowest
ChangesEnvironment=yes
UninstallDisplayIcon={app}\logcurse.exe
OutputDir=output

[Files]
Source: "logcurse-windows-amd64.exe"; DestDir: "{app}"; DestName: "logcurse.exe"; Flags: ignoreversion

[Code]
const
  EnvironmentKey = 'Environment';

procedure AddToPath(Dir: string);
var
  OldPath: string;
begin
  if not RegQueryStringValue(HKEY_CURRENT_USER, EnvironmentKey, 'Path', OldPath) then
    OldPath := '';
  if Pos(Uppercase(Dir), Uppercase(OldPath)) > 0 then
    exit;
  if OldPath <> '' then
    OldPath := OldPath + ';';
  RegWriteStringValue(HKEY_CURRENT_USER, EnvironmentKey, 'Path', OldPath + Dir);
end;

procedure RemoveFromPath(Dir: string);
var
  OldPath, NewPath, Item: string;
  I: Integer;
begin
  if not RegQueryStringValue(HKEY_CURRENT_USER, EnvironmentKey, 'Path', OldPath) then
    exit;
  NewPath := '';
  while Length(OldPath) > 0 do
  begin
    I := Pos(';', OldPath);
    if I = 0 then
    begin
      Item := OldPath;
      OldPath := '';
    end
    else
    begin
      Item := Copy(OldPath, 1, I - 1);
      OldPath := Copy(OldPath, I + 1, Length(OldPath));
    end;
    if CompareText(Item, Dir) <> 0 then
    begin
      if NewPath <> '' then
        NewPath := NewPath + ';';
      NewPath := NewPath + Item;
    end;
  end;
  RegWriteStringValue(HKEY_CURRENT_USER, EnvironmentKey, 'Path', NewPath);
end;

procedure CurStepChanged(CurStep: TSetupStep);
begin
  if CurStep = ssPostInstall then
    AddToPath(ExpandConstant('{app}'));
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
begin
  if CurUninstallStep = usPostUninstall then
    RemoveFromPath(ExpandConstant('{app}'));
end;
