package storage

import (
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"testing_system/lib/connector"
	"testing_system/lib/logger"

	"github.com/gin-gonic/gin"
)

func (s *Storage) HandleUpload(c *gin.Context) {
	info, err := getInfoFromRequest(c)
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

	dataTypeStr := info.dataType.String()
	idStr := strconv.FormatUint(info.id, 10)

	err = s.filesystem.SaveFile(c, dataTypeStr, idStr, info.filepath, file)
	if err != nil {
		connector.RespErr(c, http.StatusInternalServerError, "Failed to save file: %v", err)
		logger.Error("Failed to save file: id=%s, dataType=%s, filepath=%s\n %v",
			idStr, dataTypeStr, info.filepath, err)
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
	info, err := getInfoFromRequest(c)
	if err != nil {
		connector.RespErr(c, http.StatusBadRequest, "Invalid request parameters: %v", err)
		logger.Error("Invalid request parameters in HandleRemove: %v", err)
		return
	}

	dataTypeStr := info.dataType.String()
	idStr := strconv.FormatUint(info.id, 10)

	err = s.filesystem.RemoveFile(dataTypeStr, idStr, info.filepath)
	if err != nil {
		connector.RespErr(c, http.StatusInternalServerError, "Failed to remove file: %v", err)
		logger.Error("Failed to remove file: id=%s, dataType=%s, filepath=%s\n %v",
			idStr, dataTypeStr, info.filepath, err)
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
	info, err := getInfoFromRequest(c)
	if err != nil {
		connector.RespErr(c, http.StatusBadRequest, "Invalid request parameters: %v", err)
		logger.Error("Invalid request parameters in HandleGet: %v", err)
		return
	}

	dataTypeStr := info.dataType.String()
	idStr := strconv.FormatUint(info.id, 10)

	fullPath, err := s.filesystem.GetFilePath(dataTypeStr, idStr, info.filepath)
	if err != nil {
		connector.RespErr(c, http.StatusInternalServerError, "Failed to get file: %v", err)
		logger.Error("Failed to get file: id=%s, dataType=%s, filePath=%s\n %v",
			idStr, dataTypeStr, info.filepath, err)
		return
	}

	filename := filepath.Base(fullPath)
	c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{
		"filename": filename,
	}))

	c.File(fullPath)
}
