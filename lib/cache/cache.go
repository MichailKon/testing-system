package cache

import (
	"container/list"
	"sync"
	"testing_system/lib/logger"
)

type valHolder[TValue any] struct {
	Value *TValue
	Error error

	LockCount     uint64
	Size          uint64
	LoadingStatus *sync.WaitGroup
	ListPosition  *list.Element
}

// LRUSizeCache is simple key value LRU cache that accepts size bound for values
// Cache has Getter and may have Remover functions
// Cache removes the least recently used value if the total size bound is exceeded
//
// Getter will be called to get value for specified key
// If value can be loaded, getter must return value, and size of value
// If value can not be loaded, getter must return error and size of value (even with error)
//
// # If Remover is specified, and item has been loaded without error, Remover will be called before value removal
//
// Lock and Unlock methods can be used to avoid removal of specific keys
type LRUSizeCache[TKey comparable, TValue any] struct {
	mutex        sync.Mutex
	valueHolders map[TKey]*valHolder[TValue]

	getter  func(TKey) (*TValue, error, uint64)
	remover func(TKey, *TValue)

	sizeBound uint64
	totalSize uint64

	recentRank *list.List
}

// NewLRUSizeCache creates new cache for given size bound
func NewLRUSizeCache[TKey comparable, TValue any](
	sizeBound uint64,
	getter func(TKey) (*TValue, error, uint64),
	remover func(TKey, *TValue),
) *LRUSizeCache[TKey, TValue] {
	return &LRUSizeCache[TKey, TValue]{
		valueHolders: make(map[TKey]*valHolder[TValue]),

		getter:  getter,
		remover: remover,

		sizeBound: sizeBound,
		totalSize: 0,

		recentRank: list.New(),
	}
}

// Get returns item from cache.
//
// If id is not found, it will be loaded and get method will wait until value is loaded.
// If loader returned error, this error will be returned along with item
func (c *LRUSizeCache[TKey, TValue]) Get(key TKey) (*TValue, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	valueHolder := c.lockAndGetHolder(key)
	if valueHolder.LoadingStatus == nil {
		valueHolder.LockCount--
		c.itemUsed(key, valueHolder)
		return valueHolder.Value, valueHolder.Error
	}

	loadingStatus := valueHolder.LoadingStatus
	c.mutex.Unlock()

	loadingStatus.Wait()
	c.mutex.Lock()

	// Lock count is increased, so value can not be removed
	valueHolder = c.valueHolders[key]
	valueHolder.LockCount--
	c.itemUsed(key, valueHolder)
	// Mutex is deferred at the beginning
	return valueHolder.Value, valueHolder.Error
}

// Lock locks item in cache to avoid its removal.
//
// If element is not present in cache, it will start loading in background
// Multiple lock can be added to item, they will require multiple unlock calls
func (c *LRUSizeCache[TKey, TValue]) Lock(key TKey) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	valueHolder := c.lockAndGetHolder(key)
	if valueHolder.LoadingStatus == nil {
		c.itemUsed(key, valueHolder) // Only loaded items can appear in RankList
	}
}

// Unlock unlocks item in cache so that it can be removed.
//
// If item is not found, ErrItemNotFound will be returned
// If item is not locked, ErrNotLocked will be returned
func (c *LRUSizeCache[TKey, TValue]) Unlock(key TKey) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	valueHolder, ok := c.valueHolders[key]
	if ok {
		if valueHolder.LockCount > 0 {
			valueHolder.LockCount--
		} else {
			return &ErrItemNotLocked{key: key}
		}
		c.removeItemsIfNeeded()
		return nil
	} else {
		return &ErrItemNotFound{key: key}
	}
}

// Remove removes item from cache.
//
// This method is not tested and not recommended for usage
// If item does not exist in cache, returns nil.
// If item is locked, returns ErrItemLocked.
func (c *LRUSizeCache[TKey, TValue]) Remove(key TKey) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	valueHolder, ok := c.valueHolders[key]
	if !ok {
		return nil
	}

	if valueHolder.LockCount > 0 {
		return &ErrItemLocked{key: key}
	}

	c.removeSingleItem(key)
	return nil
}

// Insert inserts custom value inside cache.
//
// This method should be used only for testing purpose.
// The value must not be present inside cache, otherwise ErrItemAlreadyExists is returned
func (c *LRUSizeCache[TKey, TValue]) Insert(key TKey, val *TValue, size uint64) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	_, ok := c.valueHolders[key]
	if ok {
		return &ErrItemAlreadyExists{key: key}
	}
	c.valueHolders[key] = &valHolder[TValue]{
		Value:     val,
		Error:     nil,
		LockCount: 0,
		Size:      size,
	}
	c.itemUsed(key, c.valueHolders[key])
	return nil
}

// Mutex must be locked
// Increases lock count
// If value is loaded, returns valueHolder
// If value is loading, returns valueHolder with active waitgroup
// If value is absent, starts loading in background and returns valueHolder with active waitgroup
func (c *LRUSizeCache[TKey, TValue]) lockAndGetHolder(key TKey) *valHolder[TValue] {
	valueHolder, ok := c.valueHolders[key]
	if ok {
		valueHolder.LockCount++
		return valueHolder
	}
	valueHolder = &valHolder[TValue]{
		LoadingStatus: &sync.WaitGroup{},
		LockCount:     1,
	}

	valueHolder.LoadingStatus.Add(1)
	c.valueHolders[key] = valueHolder

	go c.loadAbsentValue(key)

	return valueHolder
}

// Mutex must not be locked or this should be called in different goroutine
// Value must be absent and no concurrent load for same key must be present
func (c *LRUSizeCache[TKey, TValue]) loadAbsentValue(key TKey) {
	value, err, size := c.getter(key)

	c.mutex.Lock()
	defer c.mutex.Unlock()
	valueHolder := c.valueHolders[key]

	if valueHolder.Value != nil || valueHolder.Error != nil {
		logger.Panic("Error in LRUSizeCache. loadAbsentValue is called for already loaded key, key: %v", key)
	}

	valueHolder.Value = value
	valueHolder.Error = err

	c.totalSize += size
	valueHolder.Size = size
	valueHolder.LoadingStatus.Done()
	valueHolder.LoadingStatus = nil // It will stay in other pointers

	c.itemUsed(key, valueHolder)
	c.removeItemsIfNeeded()
}

// Mutex must be locked
func (c *LRUSizeCache[TKey, TValue]) itemUsed(key TKey, valueHolder *valHolder[TValue]) {
	if valueHolder.ListPosition != nil {
		c.recentRank.MoveToBack(valueHolder.ListPosition)
	} else {
		valueHolder.ListPosition = c.recentRank.PushBack(key)
	}
}

// Mutex must be locked
func (c *LRUSizeCache[TKey, TValue]) removeItemsIfNeeded() {
	elem := c.recentRank.Front()
	for c.totalSize > c.sizeBound && elem != nil {
		key := elem.Value.(TKey)
		valueHolder := c.valueHolders[key]
		elem = elem.Next()

		if valueHolder.LockCount == 0 {
			c.removeSingleItem(key)
		}
	}
}

// Mutex must be locked
// Key, Value must be present and lock count should be zero
func (c *LRUSizeCache[TKey, TValue]) removeSingleItem(key TKey) {
	valueHolder := c.valueHolders[key]
	if valueHolder.LockCount != 0 {
		logger.Panic("Error in LRUSizeCache. Removing key with non zero lock count, key: %#v", key)
	}
	if c.remover != nil && valueHolder.Error == nil {
		c.remover(key, valueHolder.Value)
	}

	delete(c.valueHolders, key)
	c.totalSize -= valueHolder.Size
	c.recentRank.Remove(valueHolder.ListPosition)
}
