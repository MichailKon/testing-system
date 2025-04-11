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
// To access cache we have cacheGetter structs. Each cacheGetter responds for single file type.
// Each cacheGetter accepts any number of uint64 that are transformed to cacheKey struct using cacheGetter.keyGen func.
//
// So to access files, we call methods of cacheGetter, that transforms our request to request for LRUSizeCache and LRUSizeCache then does all the cache work.

type commonCache = cache.LRUSizeCache[cacheKey, string]

type cacheKey struct {
	Resource resource.Type `json:"resource"`

	// If resource is part of problem, ProblemID is used
	ProblemID uint64 `json:"problemID"`
	// If resource is part of submit, SubmitID is used
	SubmitID uint64 `json:"submitID"`
	// If resource is a test, TestID should be specified
	TestID uint64 `json:"testID"`
}

type cacheGetter struct {
	Cache  *commonCache
	keyGen func(vals ...uint64) cacheKey
}

func (c *cacheGetter) Get(vals ...uint64) (*string, error) {
	return c.Cache.Get(c.keyGen(vals...))
}

func (c *cacheGetter) Lock(vals ...uint64) {
	c.Cache.Lock(c.keyGen(vals...))
}

func (c *cacheGetter) Unlock(vals ...uint64) error {
	return c.Cache.Unlock(c.keyGen(vals...))
}

func newSourceCache(commonCache *commonCache) *cacheGetter {
	return &cacheGetter{
		Cache: commonCache,
		keyGen: func(vals ...uint64) cacheKey {
			return submitKeyGen(resource.SourceCode, vals)
		},
	}
}

func newBinaryCache(commonCache *commonCache) *cacheGetter {
	return &cacheGetter{
		Cache: commonCache,
		keyGen: func(vals ...uint64) cacheKey {
			return submitKeyGen(resource.CompiledBinary, vals)
		},
	}
}

func newCheckerCache(commonCache *commonCache) *cacheGetter {
	return &cacheGetter{
		Cache: commonCache,
		keyGen: func(vals ...uint64) cacheKey {
			return problemIDKeyGen(resource.Checker, vals)
		},
	}
}

func newInteractorCache(commonCache *commonCache) *cacheGetter {
	return &cacheGetter{
		Cache: commonCache,
		keyGen: func(vals ...uint64) cacheKey {
			return submitKeyGen(resource.Interactor, vals)
		},
	}
}

func newTestInputCache(commonCache *commonCache) *cacheGetter {
	return &cacheGetter{
		Cache:  commonCache,
		keyGen: func(vals ...uint64) cacheKey { return testKeyGen(resource.TestInput, vals) },
	}
}

func newTestAnswerCache(commonCache *commonCache) *cacheGetter {
	return &cacheGetter{
		Cache:  commonCache,
		keyGen: func(vals ...uint64) cacheKey { return testKeyGen(resource.TestAnswer, vals) },
	}
}

func problemIDKeyGen(resource resource.Type, vals []uint64) cacheKey {
	if len(vals) != 1 {
		logger.PanicLevel(3,
			"wrong usage of storage cache, can not get problem %s for id %v, too many ids passed",
			resource.String(), vals)
	}
	key := cacheKey{
		Resource:  resource,
		ProblemID: vals[0],
	}
	return key
}

func submitKeyGen(resource resource.Type, vals []uint64) cacheKey {
	if len(vals) != 1 {
		logger.PanicLevel(3,
			"wrong usage of storage cache, can not get submit %s for id %v, too many ids passed",
			resource.String(), vals)
	}
	key := cacheKey{
		Resource: resource,
		SubmitID: vals[0],
	}
	return key
}

func testKeyGen(resource resource.Type, vals []uint64) cacheKey {
	if len(vals) != 2 {
		logger.PanicLevel(3,
			"wrong usage of storage cache, can not get problem test for ids %v, exactly 2 ids should be passed",
			vals)
	}
	key := cacheKey{
		Resource:  resource,
		ProblemID: vals[0],
		TestID:    vals[1],
	}
	return key
}
