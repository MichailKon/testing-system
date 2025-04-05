package filesystem

import (
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"testing_system/common/config"

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
	subpath := filesystem.generatePathFromID(prefix, id)
	fullDirPath := filepath.Join(filesystem.Basepath, subpath)
	fullFilename := filepath.Join(fullDirPath, filename)

	if err := os.MkdirAll(fullDirPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return c.SaveUploadedFile(file, fullFilename)
}

func (filesystem *Filesystem) RemoveFile(prefix string, id string, filename string) error {
	subpath := filesystem.generatePathFromID(prefix, id)
	fullFilename := filepath.Join(filesystem.Basepath, subpath, filename)

	if err := os.Remove(fullFilename); err != nil {
		return fmt.Errorf("failed to remove file %s: %w", fullFilename, err)
	}

	dir := filepath.Dir(fullFilename)
	for dir != filesystem.Basepath && dir != "." {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return fmt.Errorf("failed to read directory %s: %w", dir, err)
		}

		if len(entries) > 0 {
			return nil
		}

		if err := os.Remove(dir); err != nil {
			return fmt.Errorf("failed to remove directory %s: %w", dir, err)
		}
		dir = filepath.Dir(dir)
	}

	return nil
}

func (filesystem *Filesystem) GetFilePath(prefix string, id string, filename string) (string, error) {
	subpath := filesystem.generatePathFromID(prefix, id)
	fullFilename := filepath.Join(filesystem.Basepath, subpath, filename)

	fileInfo, err := os.Stat(fullFilename)
	if err != nil {
		return "", err
	}

	if fileInfo.IsDir() {
		return "", fmt.Errorf("%s is a directory", fullFilename)
	}

	return fullFilename, nil
}
