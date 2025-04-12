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
		logger.Error("Invalid request parameters in HandleUpload: %v", err)
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		connector.RespErr(c, http.StatusBadRequest, "Failed to read file: %v", err)
		logger.Error("Failed to read file in HandleUpload: %v", err)
		return
	}

	err = s.filesystem.SaveFile(c, resourceInfo, file)
	if err != nil {
		connector.RespErr(c, http.StatusInternalServerError, "Failed to save file: %v", err)
		logger.Error("Failed to save file: id=%d, dataType=%s, filepath=%s: %v",
			resourceInfo.ID, resourceInfo.DataType.String(), resourceInfo.Filepath, err)
		return
	}

	response := struct {
		Message string `json:"message"`
	}{
		Message: "File uploaded successfully",
	}
	connector.RespOK(c, &response)
}

func (s *Storage) HandleRemove(c *gin.Context) {
	resourceInfo, err := getInfoFromRequest(c)
	if err != nil {
		connector.RespErr(c, http.StatusBadRequest, "Invalid request parameters: %v", err)
		logger.Error("Invalid request parameters in HandleRemove: %v", err)
		return
	}

	err = s.filesystem.RemoveFile(resourceInfo)
	if err != nil {
		connector.RespErr(c, http.StatusInternalServerError, "Failed to remove file: %v", err)
		logger.Error("Failed to remove file: id=%d, dataType=%s, filepath=%s\n %v",
			resourceInfo.ID, resourceInfo.DataType.String(), resourceInfo.Filepath, err)
		return
	}

	response := struct {
		Message string `json:"message"`
	}{
		Message: "File removed successfully",
	}
	connector.RespOK(c, &response)
}

func (s *Storage) HandleGet(c *gin.Context) {
	resourceInfo, err := getInfoFromRequest(c)
	if err != nil {
		connector.RespErr(c, http.StatusBadRequest, "Invalid request parameters: %v", err)
		logger.Error("Invalid request parameters in HandleGet: %v", err)
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
		connector.RespErr(c, http.StatusInternalServerError, "Failed to get file: %v", err)
		return
	}

	filename := filepath.Base(fullPath)
	c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{
		"filename": filename,
	}))

	c.File(fullPath)
}
