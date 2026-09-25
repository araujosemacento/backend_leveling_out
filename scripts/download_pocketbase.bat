@echo off
setlocal enabledelayedexpansion

set PB_VERSION=0.40.4
set OS=windows
set ARCH=amd64

if /i "%PROCESSOR_ARCHITECTURE%"=="ARM64" set ARCH=arm64
if /i "%PROCESSOR_ARCHITEW6432%"=="ARM64" set ARCH=arm64

set FILENAME=pocketbase_%PB_VERSION%_%OS%_%ARCH%.zip
set DOWNLOAD_URL=https://github.com/pocketbase/pocketbase/releases/download/v%PB_VERSION%/%FILENAME%

set TARGET_DIR=%~dp0..
set TARGET_ZIP=%TARGET_DIR%\%FILENAME%
set TARGET_BIN=%TARGET_DIR%\pocketbase.exe

echo ========================================================
echo   Instalador do PocketBase v%PB_VERSION% para Windows (%ARCH%)
echo ========================================================
echo.
echo ==^> Baixando: %DOWNLOAD_URL%
echo.

where curl >nul 2>nul
if %ERRORLEVEL% equ 0 (
    curl -L -o "%TARGET_ZIP%" "%DOWNLOAD_URL%"
) else (
    echo [INFO] curl nao detectado. Utilizando PowerShell como fallback para download...
    powershell -NoProfile -ExecutionPolicy Bypass -Command "[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12; (New-Object Net.WebClient).DownloadFile('%DOWNLOAD_URL%', '%TARGET_ZIP%')"
)

if not exist "%TARGET_ZIP%" (
    echo [ERRO] Falha ao baixar o arquivo compactado. Verifique sua conexao com a internet.
    goto :error
)

echo.
echo ==^> Extraindo pocketbase.exe...

where tar >nul 2>nul
if %ERRORLEVEL% equ 0 (
    tar -xf "%TARGET_ZIP%" -C "%TARGET_DIR%" pocketbase.exe
) else (
    echo [INFO] tar nao detectado. Utilizando PowerShell como fallback para extracao...
    powershell -NoProfile -ExecutionPolicy Bypass -Command "Expand-Archive -Path '%TARGET_ZIP%' -DestinationPath '%TARGET_DIR%' -Force"
)

if exist "%TARGET_ZIP%" (
    del "%TARGET_ZIP%" >nul 2>nul
)

if not exist "%TARGET_BIN%" (
    echo [ERRO] Nao foi possivel extrair o executavel pocketbase.exe.
    goto :error
)

echo.
echo ========================================================
echo ==^> Concluido com sucesso!
echo ==^> Binario instalado em: %TARGET_BIN%
echo ==^> Para iniciar o servidor local execute:
echo.
echo     .\pocketbase.exe serve --http=127.0.0.1:8090
echo ========================================================
echo.
pause
exit /b 0

:error
echo.
echo [FALHA] A instalacao nao foi concluida com sucesso.
echo.
pause
exit /b 1
