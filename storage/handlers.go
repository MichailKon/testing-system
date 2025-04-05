package storage

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing_system/lib/connector"
	"testing_system/lib/logger"

	"github.com/gin-gonic/gin"
)

func (s *Storage) HandleUpload(c *gin.Context) {
	info, err := getInfo(c)
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

	err = s.filesystem.SaveFile(c, info.dataType, info.id, info.filepath, file)
	if err != nil {
		connector.RespErr(c, http.StatusInternalServerError, "Failed to save file: %v", err)
		logger.Error("Failed to save file: id=%s, dataType=%s, filepath=%s\n %v",
			info.id, info.dataType, info.filepath, err)
		return
	}

	response := struct {
		Message  string `json:"message"`
		Filename string `json:"filename"`
	}{
		Message:  "File uploaded successfully",
		Filename: filepath.Base(file.Filename),
	}
	connector.RespOK(c, &response)
}

func (s *Storage) HandleRemove(c *gin.Context) {
	info, err := getInfo(c)
	if err != nil {
		connector.RespErr(c, http.StatusBadRequest, "Invalid request parameters: %v", err)
		logger.Error("Invalid request parameters in HandleRemove: %v", err)
		return
	}

	err = s.filesystem.RemoveFile(info.dataType, info.id, info.filepath)
	if err != nil {
		statusCode := http.StatusInternalServerError

		var pathError *os.PathError
		if errors.As(err, &pathError) && errors.Is(pathError.Err, os.ErrNotExist) {
			statusCode = http.StatusNotFound
		}

		connector.RespErr(c, statusCode, "Failed to remove file: %v", err)
		logger.Error("Failed to remove file: id=%s, dataType=%s, filePath=%s\n %v",
			info.id, info.dataType, info.filepath, err)
		return
	}

	response := struct {
		Message  string `json:"message"`
		Filename string `json:"filename"`
	}{
		Message:  "File removed successfully",
		Filename: info.filepath,
	}
	connector.RespOK(c, &response)
}

func (s *Storage) HandleGet(c *gin.Context) {
	info, err := getInfo(c)
	if err != nil {
		connector.RespErr(c, http.StatusBadRequest, "Invalid request parameters: %v", err)
		logger.Error("Invalid request parameters in HandleGet: %v", err)
		return
	}

	fullPath, err := s.filesystem.GetFilePath(info.dataType, info.id, info.filepath)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorMsg := "Failed to access file: %v"

		var pathError *os.PathError
		if errors.As(err, &pathError) {
			switch {
			case errors.Is(pathError.Err, os.ErrNotExist):
				statusCode = http.StatusNotFound
				errorMsg = "File not found: %v"
			case errors.Is(pathError.Err, os.ErrPermission):
				statusCode = http.StatusForbidden
				errorMsg = "Access denied: %v"
			}
		}

		connector.RespErr(c, statusCode, errorMsg, err)
		logger.Error("Failed to get file: id=%s, dataType=%s, filePath=%s\n %v",
			info.id, info.dataType, info.filepath, err)
		return
	}

	filename := filepath.Base(fullPath)
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.File(fullPath)
}
