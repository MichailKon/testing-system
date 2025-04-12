package filesystem

import (
	"mime/multipart"

	"github.com/gin-gonic/gin"
)

type IFilesystem interface {
	SaveFile(c *gin.Context, resourceInfo *ResourceInfo, file *multipart.FileHeader) error
	RemoveFile(resourceInfo *ResourceInfo) error
	GetFile(resourceInfo *ResourceInfo) (string, error)
}
