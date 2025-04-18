package filesystem

import (
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"strconv"
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

func (filesystem *Filesystem) generatePathFromID(id string) string {
	parts := []string{}
	for i := 0; i < len(id); i += int(filesystem.BlockSize) {
		end := min(i+int(filesystem.BlockSize), len(id))
		parts = append(parts, id[i:end])
	}
	return filepath.Join(parts...)
}

func (filesystem *Filesystem) SaveFile(c *gin.Context, resourceInfo *ResourceInfo, file *multipart.FileHeader) error {
	fullPath, err := filesystem.BuildFilePath(resourceInfo)
	if err != nil {
		return err
	}

	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	if _, err := os.Stat(fullPath); err == nil {
		logger.Warn("Overwriting existing file: %s", fullPath)
	}

	return c.SaveUploadedFile(file, fullPath)
}

func (filesystem *Filesystem) RemoveFile(resourceInfo *ResourceInfo) error {
	fullPath, err := filesystem.BuildFilePath(resourceInfo)
	if err != nil {
		return err
	}

	if err := os.Remove(fullPath); err != nil {
		return fmt.Errorf("failed to remove file %s: %w", fullPath, err)
	}

	dir := filepath.Dir(fullPath)

	for {
		if dir == filesystem.Basepath {
			break
		}

		entries, err := os.ReadDir(dir)

		if err != nil {
			return fmt.Errorf("failed to read directory %s: %w", dir, err)
		}
		if len(entries) > 0 {
			break
		}

		if err := os.Remove(dir); err != nil {
			return fmt.Errorf("failed to remove directory %s: %w", dir, err)
		}
		dir = filepath.Dir(dir)
	}

	return nil
}

func (filesystem *Filesystem) GetFile(resourceInfo *ResourceInfo) (string, error) {
	fullPath, err := filesystem.BuildFilePath(resourceInfo)
	if err != nil {
		return "", err
	}

	_, err = os.Stat(fullPath)

	if err != nil {
		return "", err
	}

	return fullPath, nil
}

func (filesystem *Filesystem) BuildFilePath(resourceInfo *ResourceInfo) (string, error) {
	subpathID := filesystem.generatePathFromID(strconv.FormatUint(resourceInfo.ID, 10))
	fullPath := filepath.Join(filesystem.Basepath, resourceInfo.DataType.String(), subpathID, resourceInfo.Filepath)

	if resourceInfo.EmptyStorageFilename {
		entries, err := os.ReadDir(fullPath)
		if err != nil {
			return "", fmt.Errorf("failed to read directory %s: %v", fullPath, err)
		}
		if len(entries) != 1 || entries[0].IsDir() {
			return "", fmt.Errorf("StorageFilename is not specified, but directory %s does not contain exactly one file", fullPath)
		}
		fullPath = filepath.Join(fullPath, entries[0].Name())
	}

	return fullPath, nil
}
