# PackZip - 自解压文件打包工具

## 项目简介

PackZip 是一个用 Go 语言编写的自解压文件打包工具，可以将多个文件打包成一个可执行文件。当用户双击生成的可执行文件时，它会自动解压并运行所有打包的文件。

## 演示效果

![PackZip 演示](res/1.gif)

## 功能特点

- **自解压功能**：将多个文件打包成单个可执行文件
- **自动运行**：解压后自动打开所有打包的文件
- **静默模式**：解压过程完全静默，无任何用户界面
- **Base64 编码**：使用 Base64 编码保护嵌入的文件数据
- **跨平台支持**：主要针对 Windows 平台优化
- **临时目录管理**：自动在系统临时目录中解压文件

## 工作原理

1. **打包模式**：
   - 将输入文件压缩成 ZIP 格式
   - 对 ZIP 数据进行 Base64 编码
   - 将编码后的数据追加到当前可执行文件末尾
   - 添加特殊标记 `PACKZIP_BASE64_END` 标识数据结束

2. **解压模式**：
   - 检测文件末尾的 `PACKZIP_BASE64_END` 标记
   - 提取 Base64 编码的数据
   - 解码并解压 ZIP 文件到临时目录
   - 自动运行所有解压的文件

## 编译说明

### 无控制台窗口编译
```bash
go build -ldflags="-H windowsgui" -o pack.exe main.go
```

### 添加自定义图标

如果需要为生成的可执行文件添加自定义图标，可以使用 `rsrc` 工具：

#### 1. 安装 rsrc 工具
```bash
go install github.com/akavel/rsrc@latest
```

#### 2. 准备图标文件
- 准备一个 `.ico` 格式的图标文件（如 `icon.ico`）
- 建议使用 256x256 像素的图标以获得最佳效果

#### 3. 生成资源文件
```bash
rsrc -ico icon.ico -o resource.syso
```

#### 4. 编译程序
编译程序时，Go 编译器会自动识别同目录下的 `.syso` 文件：
```bash
go build -o yourprogram.exe
```

#### 完整示例
```bash
# 1. 安装工具
go install github.com/akavel/rsrc@latest

# 2. 生成资源文件（假设有 icon.ico 文件）
rsrc -ico icon.ico -o resource.syso

# 3. 编译程序
go build -ldflags="-H windowsgui" -o pack.exe main.go
```

#### 注意事项
- `.syso` 文件必须与 `main.go` 在同一目录下
- 编译完成后可以删除 `.syso` 文件
- 图标文件必须是 `.ico` 格式
- 建议图标尺寸为 16x16, 32x32, 48x48, 64x64, 128x128, 256x256 像素

## 使用方法

### 基本语法
```bash
pack [选项] 文件1 文件2 [更多文件...]
```

### 参数说明
- `-o string`：指定输出的可执行文件路径（默认为 'output.exe'）

### 使用示例

1. **打包多个文件**：
   ```bash
   pack file1.txt file2.exe document.pdf
   ```

2. **指定输出文件名**：
   ```bash
   pack -o mypackage.exe file1.txt file2.exe
   ```

3. **打包可执行文件**：
   ```bash
   pack -o installer.exe setup.exe config.ini readme.txt
   ```

## 文件结构

```
packzip/
├── main.go              # 主程序文件
├── go.mod              # Go 模块文件
├── README.md           # 项目说明文档
├── build_no_console.bat # 无控制台编译脚本
├── compile_simple.bat   # 简单编译脚本
└── test_*.bat          # 各种测试脚本
```

## 核心函数说明

### `main()`
- 程序入口点
- 检测运行模式（打包或解压）
- 解析命令行参数

### `isSelfExtracting()`
- 检测当前程序是否为自解压模式
- 通过查找文件末尾的 `PACKZIP_BASE64_END` 标记判断

### `extractAndRunFiles()`
- 自解压模式的核心函数
- 提取并解码嵌入的文件数据
- 解压到临时目录并运行所有文件

### `createSelfExtractingExe()`
- 打包模式的核心函数
- 创建 ZIP 压缩包
- 将数据编码并追加到可执行文件

### `runFile()`
- 运行解压后的文件
- 支持多种运行方式（cmd start、PowerShell、直接执行）

## 技术细节

### 数据存储格式
```
[原始可执行文件内容] + [Base64编码的ZIP数据] + [数据大小(10位数字)] + [PACKZIP_BASE64_END标记]
```

### 临时目录
- 使用系统环境变量 `%TEMP%` 指定的临时目录
- 目录名格式：`packzip_xxxxxxxx`
- 程序运行完成后自动清理临时文件

### 文件权限
- 保持原始文件的权限设置
- 确保可执行文件能够正确运行

## 注意事项

1. **文件大小限制**：程序会检查文件大小，小于 10KB 的文件不会被识别为自解压文件
2. **权限要求**：某些文件可能需要管理员权限才能运行
3. **防病毒软件**：某些防病毒软件可能会将生成的自解压文件标记为可疑文件
4. **临时文件**：解压的文件会在程序运行完成后自动清理

## 故障排除

### 双击无法运行
- 尝试右键点击并选择"以管理员身份运行"
- 检查防病毒软件是否阻止了程序运行
- 确认文件没有损坏

### 文件解压失败
- 检查原始可执行文件是否完整
- 确认文件末尾包含正确的标记
- 检查磁盘空间是否充足

### 文件无法打开
- 确认文件路径中没有特殊字符
- 检查文件权限设置
- 尝试手动运行解压后的文件

## 开发环境

- **编程语言**：Go 1.x
- **目标平台**：Windows
- **依赖包**：标准库（archive/zip, encoding/base64, os/exec 等）

## 更新日志

- **v1.0.0**：初始版本，支持基本的自解压功能
- 支持 Base64 编码保护
- 支持静默解压和自动运行
- 优化了 Windows 平台的兼容性
