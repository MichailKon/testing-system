package filesystem

import (
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

type Filesystem struct {
	Basepath  string
	BlockSize int
}

func CreateFilesystem(basepath string) IFilesystem {
	return &Filesystem{Basepath: basepath, BlockSize: 3}
}

func (filesystem *Filesystem) generatePathFromID(prefix string, id string) string {
	var parts []string
	parts = append(parts, prefix)
	for i := 0; i < len(id); i += filesystem.BlockSize {
		end := min(i+filesystem.BlockSize, len(id))
		parts = append(parts, id[i:end])
	}
	return filepath.Join(parts...)
}

func (filesystem *Filesystem) SaveFile(prefix string, id string, filename string, file *multipart.FileHeader, c *gin.Context) error {
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
		return fmt.Errorf("failed to remove file: %w", err)
	}

	dir := filepath.Dir(fullFilename)
	for dir != filesystem.Basepath && dir != "." {
		if err := os.Remove(dir); err != nil {
			break
		}
		dir = filepath.Dir(dir)
	}

	return nil
}

func (filesystem *Filesystem) GetFilePath(prefix string, id string, filename string) (string, error) {
	subpath := filesystem.generatePathFromID(prefix, id)
	fullFilename := filepath.Join(filesystem.Basepath, subpath, filename)

	if _, err := os.Stat(fullFilename); os.IsNotExist(err) {
		return "", fmt.Errorf("file not found")
	}
	return fullFilename, nil
}
