package common

import (
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
	"os"
	"path/filepath"
	"testing_system/clients/common/clientconfig"
	"testing_system/common/connectors/masterconn"
	"testing_system/common/connectors/storageconn"
	"testing_system/common/db"
	"testing_system/lib/logger"
)

type ClientBase struct {
	Config clientconfig.Config

	Router *gin.Engine

	StorageConnection *storageconn.Connector
	MasterConnection  *masterconn.Connector
	DB                *gorm.DB
}

func NewClientBase(configPath string) *ClientBase {
	var config clientconfig.Config
	configData, err := os.ReadFile(configPath)
	if err != nil {
		panic(err)
	}
	err = yaml.Unmarshal(configData, &config)
	if err != nil {
		panic(err)
	}

	logger.InitLogger(config.Logger)

	base := &ClientBase{
		Config:            config,
		StorageConnection: storageconn.NewConnector(config.StorageConnection),
		MasterConnection:  masterconn.NewConnector(config.MasterConnection),
	}

	base.DB, err = db.NewDB(config.DB)
	if err != nil {
		logger.Panic("Can not set up testing system db, error: %v", err)
	}

	base.Router = gin.Default()

	base.Router.Static("/static", filepath.Join(base.Config.ResourcesPath, "static"))
	base.Router.LoadHTMLGlob(filepath.Join(base.Config.ResourcesPath, "templates/*"))
	return base
}

func (b *ClientBase) Run() {
	err := b.Router.Run(b.Config.Address)
	if err != nil {
		logger.Panic("Can not start client handler, error: %v", err)
	}
}
