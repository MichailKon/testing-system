package storage

import (
	"net/http"
	"testing_system/lib/logger"

	"github.com/gin-gonic/gin"
)

func (s *Storage) HandleUpload(c *gin.Context) {
	id, dataType, filepath, err := getInfo(c)

	if err != nil {
		return
	}

	file, err := c.FormFile("file")

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read file"})
		return
	}

	err = s.filesystem.SaveFile(dataType, id, filepath, file, c)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		logger.Error("Failed to save file: id=%s, dataType=%s, filePath=%s\n %v", id, dataType, filepath, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "File uploaded successfully"})
}

func (s *Storage) HandleRemove(c *gin.Context) {
	id, dataType, filepath, err := getInfo(c)

	if err != nil {
		return
	}

	err = s.filesystem.RemoveFile(dataType, id, filepath)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Failed to remove file"})
		logger.Error("Failed to remove file: id=%s, dataType=%s, filePath=%s\n %v", id, dataType, filepath, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "File removed successfully"})
}

func (s *Storage) HandleGet(c *gin.Context) {
	id, dataType, filePath, err := getInfo(c)

	if err != nil {
		return
	}

	fullPath, err := s.filesystem.GetFilePath(dataType, id, filePath)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Failed to get file"})
		return
	}

	c.File(fullPath)
}
