package filesystem

import (
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"testing_system/common/config"
	"testing_system/lib/logger"

	"github.com/gin-gonic/gin"
)

type Filesystem struct {
	Basepath  string
	BlockSize uint
}

func NewFilesystem(storageConfig *config.StorageConfig) IFilesystem {
	return &Filesystem{Basepath: storageConfig.StoragePath, BlockSize: storageConfig.BlockSize}
}

func (filesystem *Filesystem) generatePathFromID(prefix string, id string) string {
	var parts []string
	parts = append(parts, prefix)
	for i := 0; i < len(id); i += int(filesystem.BlockSize) {
		end := min(i+int(filesystem.BlockSize), len(id))
		parts = append(parts, id[i:end])
	}
	return filepath.Join(parts...)
}

func (filesystem *Filesystem) SaveFile(c *gin.Context, prefix string, id string, filename string, file *multipart.FileHeader) error {
	fullPath, err := filesystem.GetFilePath(prefix, id, filename)
	if err != nil {
		return err
	}

	fullDirPath := filepath.Dir(fullPath)
	if err := os.MkdirAll(fullDirPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if _, err := os.Stat(fullPath); err == nil {
		logger.Warn("Overwriting existing file: %s", fullPath)
	}

	return c.SaveUploadedFile(file, fullPath)
}

func (filesystem *Filesystem) RemoveFile(prefix string, id string, filename string) error {
	fullPath, err := filesystem.GetFilePath(prefix, id, filename)
	if err != nil {
		return err
	}

	if err := os.Remove(fullPath); err != nil {
		return fmt.Errorf("failed to remove file %s: %w", fullPath, err)
	}

	dir := filepath.Dir(fullPath)
	basePath, _ := filepath.Abs(filesystem.Basepath)

	for {
		absDir, _ := filepath.Abs(dir)
		if absDir == basePath || !strings.HasPrefix(absDir, basePath) {
			break
		}

		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			break
		}

		if err := os.Remove(dir); err != nil {
			return fmt.Errorf("failed to remove directory %s: %w", dir, err)
		}
		dir = filepath.Dir(dir)
	}

	return nil
}

func (filesystem *Filesystem) GetFilePath(prefix, id, filename string) (string, error) {
	if prefix == "" || id == "" || filename == "" {
		return "", fmt.Errorf("prefix, id, and filename cannot be empty")
	}

	cleanFilename := filepath.Base(filename)

	subpath := filesystem.generatePathFromID(prefix, id)
	fullPath := filepath.Join(filesystem.Basepath, subpath, cleanFilename)

	absBasepath, err := filepath.Abs(filesystem.Basepath)
	if err != nil {
		return "", err
	}

	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", err
	}

	if !strings.HasPrefix(absFullPath, absBasepath) {
		return "", fmt.Errorf("invalid path: basepath is not a prefix of fullpath")
	}

	return fullPath, nil
}
