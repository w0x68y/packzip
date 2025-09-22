package main

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// 主函数
func main() {
	if isSelfExtracting() {
		extractAndRunFiles()
		return
	}

	var outputPath string
	flag.StringVar(&outputPath, "o", "output.exe", "输出的可执行文件路径")
	flag.Parse()

	inputFiles := flag.Args()
	if len(inputFiles) < 2 {
		fmt.Println("使用方法: pack [选项] 文件1 文件2 [更多文件...]")
		fmt.Println("选项:")
		fmt.Println("  -o string   输出的可执行文件路径 (默认为 'output.exe')")
		os.Exit(1)
	}

	if !strings.HasSuffix(strings.ToLower(outputPath), ".exe") {
		outputPath += ".exe"
	}

	createSelfExtractingExe(inputFiles, outputPath)
}

func isSelfExtracting() bool {
	exePath, err := os.Executable()
	if err != nil {
		return false
	}

	fileInfo, err := os.Stat(exePath)
	if err != nil {
		return false
	}
	fileSize := fileInfo.Size()

	if fileSize < 10000 {
		return false
	}

	readSize := int64(2048)
	if fileSize < readSize {
		readSize = fileSize
	}

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

	return bytes.Contains(buffer, []byte("PACKZIP_BASE64_END"))
}

func extractAndRunFiles() {
	exePath, err := os.Executable()
	if err != nil {
		os.Exit(1)
	}

	file, err := os.Open(exePath)
	if err != nil {
		os.Exit(1)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		os.Exit(1)
	}
	fileSize := fileInfo.Size()

	readSize := int64(2048)
	if fileSize < readSize {
		readSize = fileSize
	}

	buffer := make([]byte, readSize)
	_, err = file.ReadAt(buffer, fileSize-readSize)
	if err != nil {
		os.Exit(1)
	}

	const marker = "PACKZIP_BASE64_END"
	markerIndex := bytes.LastIndex(buffer, []byte(marker))
	if markerIndex == -1 {
		os.Exit(1)
	}

	startIndex := markerIndex - 10
	if startIndex < 0 {
		startIndex = 0
	}
	zipSizeBytes := buffer[startIndex:markerIndex]
	zipSizeStr := strings.TrimSpace(string(zipSizeBytes))

	var zipSize int64
	_, err = fmt.Sscanf(zipSizeStr, "%d", &zipSize)
	if err != nil {
		os.Exit(1)
	}

	zipStartPos := fileSize - readSize + int64(startIndex) - zipSize

	tempDir, err := os.MkdirTemp(os.Getenv("TEMP"), "packzip_")
	if err != nil {
		os.Exit(1)
	}

	file.Seek(zipStartPos, 0)
	encodedData := make([]byte, zipSize)
	_, err = io.ReadFull(file, encodedData)
	if err != nil {
		os.Exit(1)
	}

	zipData, err := base64.StdEncoding.DecodeString(string(encodedData))
	if err != nil {
		os.Exit(1)
	}

	zipReader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		os.Exit(1)
	}

	var extractedFiles []string
	for _, zipFile := range zipReader.File {
		dstPath := filepath.Join(tempDir, zipFile.Name)
		extractedFiles = append(extractedFiles, dstPath)

		dir := filepath.Dir(dstPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			continue
		}

		r, err := zipFile.Open()
		if err != nil {
			r.Close()
			continue
		}

		dst, err := os.Create(dstPath)
		if err != nil {
			r.Close()
			continue
		}

		_, err = io.Copy(dst, r)
		dst.Close()
		r.Close()

		if err != nil {
			continue
		}

		os.Chmod(dstPath, zipFile.Mode())
	}

	for _, filePath := range extractedFiles {
		runFile(filepath.Clean(filePath))
		time.Sleep(100 * time.Millisecond)
	}

	time.Sleep(3 * time.Second)
}

func runFile(filePath string) {
	cmd := exec.Command("cmd", "/c", "start", "", filePath)
	if cmd.Run() == nil {
		return
	}

	cmd = exec.Command("powershell", "Start-Process", filePath)
	if cmd.Run() == nil {
		return
	}

	if strings.ToLower(filepath.Ext(filePath)) == ".exe" {
		cmd = exec.Command(filePath)
		cmd.Start()
	}
}

func createSelfExtractingExe(inputFiles []string, outputPath string) {
	var zipBuffer bytes.Buffer
	zipWriter := zip.NewWriter(&zipBuffer)

	for _, filePath := range inputFiles {
		fmt.Println("添加文件到ZIP:", filePath)
		srcFile, err := os.Open(filePath)
		if err != nil {
			fmt.Printf("无法打开文件 '%s': %s\n", filePath, err)
			continue
		}
		defer srcFile.Close()

		srcInfo, err := srcFile.Stat()
		if err != nil {
			fmt.Printf("无法获取文件信息 '%s': %s\n", filePath, err)
			continue
		}

		head, err := zip.FileInfoHeader(srcInfo)
		if err != nil {
			fmt.Printf("无法创建ZIP头部 '%s': %s\n", filePath, err)
			continue
		}

		head.Name = filepath.Base(filePath)
		head.SetMode(srcInfo.Mode())

		writer, err := zipWriter.CreateHeader(head)
		if err != nil {
			fmt.Printf("无法创建ZIP条目 '%s': %s\n", filePath, err)
			continue
		}

		_, err = io.Copy(writer, srcFile)
		if err != nil {
			fmt.Printf("无法复制文件内容 '%s': %s\n", filePath, err)
		}
	}

	zipWriter.Close()

	zipData := zipBuffer.Bytes()
	zipSize := int64(len(zipData))
	fmt.Println("ZIP数据大小:", zipSize, "字节")

	selfPath, err := os.Executable()
	if err != nil {
		fmt.Println("无法获取当前可执行文件路径:", err)
		time.Sleep(5 * time.Second)
		os.Exit(1)
	}

	if _, err := os.Stat(selfPath); os.IsNotExist(err) {
		fmt.Println("错误: 当前可执行文件不存在。请先编译程序。")
		time.Sleep(5 * time.Second)
		os.Exit(1)
	}

	selfFile, err := os.Open(selfPath)
	if err != nil {
		fmt.Println("无法打开当前可执行文件:", err)
		time.Sleep(5 * time.Second)
		os.Exit(1)
	}
	defer selfFile.Close()

	selfInfo, err := selfFile.Stat()
	if err != nil {
		fmt.Println("无法获取当前可执行文件信息:", err)
		time.Sleep(5 * time.Second)
		os.Exit(1)
	}

	destFile, err := os.Create(outputPath)
	if err != nil {
		fmt.Println("无法创建目标可执行文件:", err)
		time.Sleep(5 * time.Second)
		os.Exit(1)
	}
	defer destFile.Close()

	fmt.Println("复制可执行文件内容...")
	_, err = io.Copy(destFile, selfFile)
	if err != nil {
		fmt.Println("复制可执行文件内容失败:", err)
		time.Sleep(5 * time.Second)
		os.Exit(1)
	}

	encodedData := base64.StdEncoding.EncodeToString(zipData)
	encodedSize := int64(len(encodedData))
	fmt.Println("ZIP数据已进行base64加密编码")

	fmt.Println("追加base64编码数据到可执行文件...")
	destFile.Write([]byte(encodedData))

	fmt.Println("写入结束标记...")
	fmt.Fprintf(destFile, "%10dPACKZIP_BASE64_END", encodedSize)

	if err := os.Chmod(outputPath, selfInfo.Mode()|0111); err != nil {
		fmt.Println("设置可执行权限失败:", err)
	}

	fmt.Printf("\n自解压可执行文件已创建: %s\n", outputPath)
	fmt.Println("双击该文件将会自动解压并打开所有打包的文件。")
	fmt.Println("注意: 如果双击无法运行，请尝试右键点击并选择'以管理员身份运行'。")
}
