package webapp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/godaddy-x/freego/zlog"
)

func ReadAllFilesInDir(dirPath string) ([]*KeyMeta, error) {
	// 读取目录中的所有条目（文件和子目录）
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %q: %w", dirPath, err)
	}
	walletFileList := make([]*KeyMeta, 0, len(entries))
	for _, entry := range entries {
		// 跳过子目录，只处理普通文件
		if entry.IsDir() {
			continue
		}
		filePath := filepath.Join(dirPath, entry.Name())
		content, err := os.ReadFile(filePath)
		if err != nil {
			zlog.Error("read file error", 0, zlog.String("errMsg", err.Error()))
			continue
		}
		walletFile := &KeyMeta{}
		if err := json.Unmarshal(content, walletFile); err != nil {
			zlog.Error("read file error", 0, zlog.String("errMsg", err.Error()))
			continue
		}
		walletFileList = append(walletFileList, walletFile)
	}

	return walletFileList, nil
}
