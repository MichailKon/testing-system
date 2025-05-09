package storage

import (
	"testing_system/common/constants/resource"
	"testing_system/lib/cache"
	"testing_system/lib/logger"
)

// The main idea of cache is following: We have one cache of specific size to hold all types of files.
// However, we have multiple file types so we must have different getter for each file.
//
// So we have single LRUSizeCache that holds all file types. Key for this cache is cacheKey, the internal struct with which we can determine the file type and it's keys.
//
// To access cache we have CacheGetter structs. Each CacheGetter responds for single file type.
// Each CacheGetter accepts any number of uint64 that are transformed to cacheKey struct using CacheGetter.keyGen func.
//
// So to access files, we call methods of CacheGetter, that transforms our request to request for LRUSizeCache and LRUSizeCache then does all the cache work.

type commonCache = cache.LRUSizeCache[cacheKey, string]

type cacheKey struct {
	Epoch int

	Resource resource.Type `json:"resource"`

	// If resource is part of problem, ProblemID is used
	ProblemID uint64 `json:"problemID"`
	// If resource is part of submit, SubmitID is used
	SubmitID uint64 `json:"submitID"`
	// If resource is a test, TestID should be specified
	TestID uint64 `json:"testID"`
}

type CacheGetter struct {
	Cache  *commonCache
	keyGen func(epoch int, vals ...uint64) cacheKey
}

func (c *CacheGetter) Get(epoch int, vals ...uint64) (*string, error) {
	return c.Cache.Get(c.keyGen(epoch, vals...))
}

func (c *CacheGetter) Lock(epoch int, vals ...uint64) {
	c.Cache.Lock(c.keyGen(epoch, vals...))
}

func (c *CacheGetter) Unlock(epoch int, vals ...uint64) error {
	return c.Cache.Unlock(c.keyGen(epoch, vals...))
}

// Insert can be used only for testing
func (c *CacheGetter) Insert(epoch int, file string, vals ...uint64) error {
	return c.Cache.Insert(c.keyGen(epoch, vals...), &file, 1)
}

func newSourceCache(commonCache *commonCache) *CacheGetter {
	return &CacheGetter{
		Cache: commonCache,
		keyGen: func(epoch int, vals ...uint64) cacheKey {
			return submitKeyGen(epoch, resource.SourceCode, vals)
		},
	}
}

func newBinaryCache(commonCache *commonCache) *CacheGetter {
	return &CacheGetter{
		Cache: commonCache,
		keyGen: func(epoch int, vals ...uint64) cacheKey {
			return submitKeyGen(epoch, resource.CompiledBinary, vals)
		},
	}
}

func newCheckerCache(commonCache *commonCache) *CacheGetter {
	return &CacheGetter{
		Cache: commonCache,
		keyGen: func(epoch int, vals ...uint64) cacheKey {
			return problemIDKeyGen(epoch, resource.Checker, vals)
		},
	}
}

func newInteractorCache(commonCache *commonCache) *CacheGetter {
	return &CacheGetter{
		Cache: commonCache,
		keyGen: func(epoch int, vals ...uint64) cacheKey {
			return submitKeyGen(epoch, resource.Interactor, vals)
		},
	}
}

func newTestInputCache(commonCache *commonCache) *CacheGetter {
	return &CacheGetter{
		Cache:  commonCache,
		keyGen: func(epoch int, vals ...uint64) cacheKey { return testKeyGen(epoch, resource.TestInput, vals) },
	}
}

func newTestAnswerCache(commonCache *commonCache) *CacheGetter {
	return &CacheGetter{
		Cache:  commonCache,
		keyGen: func(epoch int, vals ...uint64) cacheKey { return testKeyGen(epoch, resource.TestAnswer, vals) },
	}
}

func problemIDKeyGen(epoch int, resource resource.Type, vals []uint64) cacheKey {
	if len(vals) != 1 {
		logger.PanicLevel(3,
			"wrong usage of storage cache, can not get problem %s for id %v, too many ids passed",
			resource.String(), vals)
	}
	key := cacheKey{
		Epoch:     epoch,
		Resource:  resource,
		ProblemID: vals[0],
	}
	return key
}

func submitKeyGen(epoch int, resource resource.Type, vals []uint64) cacheKey {
	if len(vals) != 1 {
		logger.PanicLevel(3,
			"wrong usage of storage cache, can not get submit %s for id %v, too many ids passed",
			resource.String(), vals)
	}
	key := cacheKey{
		Epoch:    epoch,
		Resource: resource,
		SubmitID: vals[0],
	}
	return key
}

func testKeyGen(epoch int, resource resource.Type, vals []uint64) cacheKey {
	if len(vals) != 2 {
		logger.PanicLevel(3,
			"wrong usage of storage cache, can not get problem test for ids %v, exactly 2 ids should be passed",
			vals)
	}
	key := cacheKey{
		Epoch:     epoch,
		Resource:  resource,
		ProblemID: vals[0],
		TestID:    vals[1],
	}
	return key
}
