@echo off
REM Simple compilation script for no-console version

echo Compiling no-console version...
go build -ldflags="-H windowsgui" -o pack_no_console.exe

if exist pack_no_console.exe (
    echo SUCCESS: No-console version compiled successfully
    echo You can now use pack_no_console.exe to create self-extracting files that run without a console window
) else (
    echo FAILED: Compilation of no-console version failed
)

pause