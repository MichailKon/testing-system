package filesystem

import (
	"mime/multipart"

	"github.com/gin-gonic/gin"
)

type IFilesystem interface {
	SaveFile(c *gin.Context, prefix string, id string, filename string, file *multipart.FileHeader) error
	RemoveFile(prefix string, id string, filename string) error
	GetFilePath(prefix string, id string, filename string) (string, error)
}
