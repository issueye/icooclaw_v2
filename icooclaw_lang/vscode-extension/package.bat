@echo off
setlocal EnableExtensions

if not exist node_modules (
    echo Installing npm dependencies...
    call npm.cmd install
    if errorlevel 1 exit /b 1
)

echo Packaging VSIX...
call npm.cmd run package:out
if errorlevel 1 exit /b 1

echo Done: iclang-vscode.vsix
endlocal
