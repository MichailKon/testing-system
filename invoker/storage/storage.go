package storage

import (
	"os"
	"testing_system/common"
	"testing_system/common/connectors/storageconn"
	"testing_system/lib/cache"
	"testing_system/lib/logger"
)

type InvokerStorage struct {
	ts               *common.TestingSystem
	storageConnector *storageconn.Connector

	cache *cache.LRUSizeCache[cacheKey, storageconn.ResponseFiles]

	Source     *cacheGetter
	Binary     *cacheGetter
	Checker    *cacheGetter
	Interactor *cacheGetter
	Test       *cacheGetter
}

func NewInvokerStorage(ts *common.TestingSystem) *InvokerStorage {
	s := &InvokerStorage{
		storageConnector: storageconn.NewStorageConnector(ts),
	}
	err := os.RemoveAll(ts.Config.Invoker.CachePath)
	if err != nil {
		logger.Panic(err.Error())
	}
	err = os.MkdirAll(ts.Config.Invoker.CachePath, 0775)
	if err != nil {
		logger.Panic(err.Error())
	}
	s.cache = cache.NewLRUSizeCache[cacheKey, storageconn.ResponseFiles](
		ts.Config.Invoker.CacheSize,
		s.getFiles,
		func(key cacheKey, files *storageconn.ResponseFiles) {
			files.CleanUp()
		},
	)
	s.Source = newSourceCache(s.cache)
	s.Binary = newBinaryCache(s.cache)
	s.Checker = newCheckerCache(s.cache)
	s.Interactor = newInteractorCache(s.cache)
	s.Test = newTestCache(s.cache)
	return s
}

func (s *InvokerStorage) getFiles(key cacheKey) (*storageconn.ResponseFiles, error, uint64) {
	request := storageconn.Request{
		Resource:  key.Resource,
		ProblemID: key.ProblemID,
		SubmitID:  key.SubmitID,
		TestID:    key.TestID,
	}
	request.FillBaseFolder(s.ts.Config.Invoker.CachePath)
	files := s.storageConnector.Download(request)
	if files.Error != nil {
		return nil, files.Error, 0
	} else {
		return files, nil, files.Size
	}
}
