package storage

import (
	"fmt"
	"github.com/xorcare/pointer"
	"os"
	"path/filepath"
	"strconv"
	"testing_system/common"
	"testing_system/common/connectors/storageconn"
	"testing_system/common/constants/resource"
	"testing_system/lib/cache"
	"testing_system/lib/logger"
)

type InvokerStorage struct {
	ts *common.TestingSystem

	cache *commonCache

	Source     *cacheGetter
	Binary     *cacheGetter
	Checker    *cacheGetter
	Interactor *cacheGetter
	TestInput  *cacheGetter
	TestAnswer *cacheGetter
}

func NewInvokerStorage(ts *common.TestingSystem) *InvokerStorage {
	s := &InvokerStorage{ts: ts}
	err := os.RemoveAll(ts.Config.Invoker.CachePath)
	if err != nil {
		logger.Panic(err.Error())
	}
	err = os.MkdirAll(ts.Config.Invoker.CachePath, 0775)
	if err != nil {
		logger.Panic(err.Error())
	}
	s.cache = cache.NewLRUSizeCache[cacheKey, string](
		ts.Config.Invoker.CacheSize,
		s.getFiles,
		cleanUpFile,
	)
	s.Source = newSourceCache(s.cache)
	s.Binary = newBinaryCache(s.cache)
	s.Checker = newCheckerCache(s.cache)
	s.Interactor = newInteractorCache(s.cache)
	s.TestInput = newTestInputCache(s.cache)
	s.TestAnswer = newTestAnswerCache(s.cache)
	logger.Info("Created invoker storage")
	return s
}

func (s *InvokerStorage) getFiles(key cacheKey) (*string, error, uint64) {
	request := &storageconn.Request{
		Resource:  key.Resource,
		ProblemID: key.ProblemID,
		SubmitID:  key.SubmitID,
		TestID:    key.TestID,
	}
	setRequestBaseFolder(request, s.ts.Config.Invoker.CachePath)
	files := s.ts.StorageConn.Download(request)
	if files.Error != nil {
		return nil, files.Error, 0
	} else {
		file, ok := files.File()
		if !ok {
			return nil, fmt.Errorf("file not exists"), 0
		}
		return pointer.String(filepath.Join(request.BaseFolder, file)), nil, files.Size
	}
}

func setRequestBaseFolder(request *storageconn.Request, parent string) {
	request.BaseFolder = filepath.Join(parent, request.Resource.String())
	switch request.Resource {
	case resource.SourceCode, resource.CompiledBinary, resource.CompileOutput:
		request.BaseFolder = filepath.Join(request.BaseFolder, strconv.FormatUint(request.SubmitID, 10))
	case resource.Checker, resource.Interactor:
		request.BaseFolder = filepath.Join(request.BaseFolder, strconv.FormatUint(request.ProblemID, 10))
	case resource.TestInput, resource.TestAnswer:
		request.BaseFolder = filepath.Join(request.BaseFolder, fmt.Sprintf("%d-%d", request.ProblemID, request.TestID))
	default:
		logger.Panic("Can not fill base folder for storageconn request of type %v", request.Resource)
	}
}

func cleanUpFile(key cacheKey, file *string) {
	if file == nil {
		return
	}
	err := os.RemoveAll(filepath.Dir(*file))
	if err != nil {
		logger.Error("can not clean up file %s, key: %+v, error: %s", *file, key, err)
	}
}
