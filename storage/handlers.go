package storage

import (
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"testing_system/lib/connector"
	"testing_system/lib/logger"

	"github.com/gin-gonic/gin"
)

func (s *Storage) HandleUpload(c *gin.Context) {
	resourceInfo, err := getInfoFromRequest(c)
	if err != nil {
		connector.RespErr(c, http.StatusBadRequest, "Invalid request parameters: %v", err)
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		connector.RespErr(c, http.StatusBadRequest, "Failed to read file: %v", err)
		return
	}

	err = s.filesystem.SaveFile(c, resourceInfo, file)
	if err != nil {
		connector.RespErr(c, http.StatusInternalServerError, "Server error")
		logger.Error("Failed to save file: id=%d, dataType=%s, filepath=%s: %v",
			resourceInfo.ID, resourceInfo.DataType.String(), resourceInfo.Filepath, err)
		return
	}

	connector.RespOK(c, nil)
}

func (s *Storage) HandleRemove(c *gin.Context) {
	resourceInfo, err := getInfoFromRequest(c)
	if err != nil {
		connector.RespErr(c, http.StatusBadRequest, "Invalid request parameters: %v", err)
		return
	}

	err = s.filesystem.RemoveFile(resourceInfo)
	if err != nil {
		connector.RespErr(c, http.StatusInternalServerError, "Server error")
		logger.Error("Failed to remove file: id=%d, dataType=%s, filepath=%s\n %v",
			resourceInfo.ID, resourceInfo.DataType.String(), resourceInfo.Filepath, err)
		return
	}

	connector.RespOK(c, nil)
}

func (s *Storage) HandleGet(c *gin.Context) {
	resourceInfo, err := getInfoFromRequest(c)
	if err != nil {
		connector.RespErr(c, http.StatusBadRequest, "Invalid request parameters: %v", err)
		return
	}

	fullPath, err := s.filesystem.GetFile(resourceInfo)
	if err != nil {
		if os.IsNotExist(err) {
			connector.RespErr(c, http.StatusNotFound, "File doesn't exist: %v", err)
			return
		}
		logger.Error("Failed to get file: id=%d, dataType=%s, filePath=%s\n %v",
			resourceInfo.ID, resourceInfo.DataType.String(), resourceInfo.Filepath, err)
		connector.RespErr(c, http.StatusInternalServerError, "Server error")
		return
	}

	filename := filepath.Base(fullPath)
	c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{
		"filename": filename,
	}))

	c.File(fullPath)
}
