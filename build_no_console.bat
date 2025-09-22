@echo off
REM 编译无控制台窗口的自解压程序
go build -ldflags="-H windowsgui" -o pack_no_console.exe

if %errorlevel% equ 0 (
    echo 编译成功！已生成无控制台版本的pack_no_console.exe
    echo 使用方法: pack_no_console.exe 文件1 文件2 [更多文件...]
) else (
    echo 编译失败，请检查错误信息
    pause
)