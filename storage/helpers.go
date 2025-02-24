package storage

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

var validDataTypes = map[string]struct{}{
	"submission": {},
	"problem":    {},
}

func isValidDataType(dataType string) bool {
	_, exists := validDataTypes[dataType]
	return exists
}

func getInfo(c *gin.Context) (string, string, string, error) {
	id := c.Query("id")
	dataType := c.Query("dataType")
	filepath := c.Query("filepath")

	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing id"})
		return "", "", "", fmt.Errorf("missing id")
	}

	if filepath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing filepath"})
		return "", "", "", fmt.Errorf("missing id")
	}

	if !isValidDataType(dataType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid dataType"})
		return "", "", "", fmt.Errorf("invalid dataType")
	}

	return id, dataType, filepath, nil
}
