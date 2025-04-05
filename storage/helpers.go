package storage

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

var validDataTypes = map[string]struct{}{
	"submission": {},
	"problem":    {},
}

type fileInfo struct {
	id       string `form:"id"`
	dataType string `form:"dataType"`
	filepath string `form:"filepath"`
}

func isValidDataType(dataType string) bool {
	_, exists := validDataTypes[dataType]
	return exists
}

func getInfo(c *gin.Context) (fileInfo, error) {
	var info fileInfo

	if err := c.ShouldBind(&info); err != nil {
		return fileInfo{}, fmt.Errorf("invalid request parameters: %w", err)
	}

	if info.id == "" {
		return fileInfo{}, fmt.Errorf("missing id")
	}

	if info.filepath == "" {
		return fileInfo{}, fmt.Errorf("missing filepath")
	}

	if !isValidDataType(info.dataType) {
		return fileInfo{}, fmt.Errorf("invalid dataType")
	}

	return info, nil
}
