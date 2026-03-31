@echo off
setlocal EnableExtensions

set BINARY_NAME=iclang.exe
set TARGET_DIR=release\windows-amd64
set VERSION=%~1

if not "%VERSION%"=="" goto build

for /f "delims=" %%i in ('git describe --tags --always --dirty 2^>nul') do set VERSION=%%i
if not "%VERSION%"=="" goto build

set VERSION=dev

:build
echo Building %BINARY_NAME% version %VERSION%...

if not exist "%TARGET_DIR%" mkdir "%TARGET_DIR%"
if errorlevel 1 exit /b 1

go build -trimpath -ldflags "-s -w -X main.VERSION=%VERSION%" -o "%TARGET_DIR%\%BINARY_NAME%" .\cmd\iclang
if errorlevel 1 exit /b 1

echo Build OK: %TARGET_DIR%\%BINARY_NAME%
echo Version: %VERSION%
endlocal
