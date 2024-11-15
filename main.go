package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type ConfigItem struct {
	Name         string `json:"name"`
	OriginalPath string `json:"originalPath"`
	TargetPath   string `json:"targetPath"`
}

type Config struct {
	Items []ConfigItem `json:"items"`
}

// 计算文件的MD5值
func getFileMd5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// 创建目录
func createDirIfNotExist(dirPath string) error {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return os.MkdirAll(dirPath, 0755)
	}
	return nil
}

// 遍历目录并计算所有文件的MD5值
func calculateDirMd5(dirPath string) (map[string]string, error) {
	filesMd5 := make(map[string]string)
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		md5Value, err := getFileMd5(path)
		if err != nil {
			return err
		}
		outpath := strings.Replace(path, dirPath, "", 1)
		filesMd5[outpath] = md5Value
		return nil
	})
	return filesMd5, err
}

// 复制文件
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destinationFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, sourceFile)
	return err
}

func processFile(config ConfigItem) error {
	err := createDirIfNotExist(config.TargetPath)
	if err != nil {
		return err
	}
	originalMd5, err := calculateDirMd5(config.OriginalPath)
	if err != nil {
		fmt.Printf("计算原始目录MD5失败：%v\n", err)
		return nil
	}
	targetMd5, err := calculateDirMd5(config.TargetPath)
	if err != nil {
		fmt.Printf("计算目标目录MD5失败：%v\n", err)
		return nil
	}
	// 确保目标目录结构存在
	err = filepath.Walk(config.OriginalPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode().IsDir() {
			relativePath, _ := filepath.Rel(config.OriginalPath, path)
			targetDir := filepath.Join(config.TargetPath, relativePath)
			err := createDirIfNotExist(targetDir)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		fmt.Printf("确保目标目录结构存在失败：%v\n", err)
		return nil
	}
	// 比较MD5值并复制文件
	for fileName, originalMd5Value := range originalMd5 {
		targetMd5Value, exists := targetMd5[fileName]
		if !exists || originalMd5Value != targetMd5Value {
			fmt.Printf("文件 %s MD5不匹配，需要复制\n", fileName)
			sourceFilePath := filepath.Join(config.OriginalPath, fileName)
			targetFilePath := filepath.Join(config.TargetPath, fileName)
			fmt.Println(sourceFilePath)
			fmt.Println(targetFilePath)
			if err := copyFile(sourceFilePath, targetFilePath); err != nil {
				fmt.Printf("复制文件失败：%v\n", err)
				continue
			}
		}
	}
	return nil
}

func main() {
	// 配置文件路径
	configPath := "config"
	// 读取配置文件
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		fmt.Println("Error reading config file:", err)
		return
	}
	var configItems []ConfigItem
	err = json.Unmarshal(data, &configItems)
	if err != nil {
		fmt.Println("Error parsing JSON:", err)
		return
	}
	for _, item := range configItems {
		err := processFile(item)
		if err != nil {
			return
		}
		fmt.Printf("Name: %s, OriginalPath: %s, TargetPath: %s\n", item.Name, item.OriginalPath, item.TargetPath)
	}
}
