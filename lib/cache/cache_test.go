package cache

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"github.com/xorcare/pointer"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func testGet[TKey comparable, TValue comparable](t *testing.T, cache *LRUSizeCache[TKey, TValue], key TKey, expectedVal *TValue, expectedErr error) {
	val, err := cache.Get(key)
	if expectedVal == nil {
		require.Nil(t, val)
	} else {
		require.NotNil(t, val)
		require.Equal(t, *expectedVal, *val)
	}

	if expectedErr == nil {
		require.Nil(t, err)
	} else {
		require.NotNil(t, err)
		require.Equal(t, expectedErr.Error(), err.Error())
	}
}

func TestSimpleGet(t *testing.T) {
	load := func(key int) (*int, error, uint64) {
		t.Fatalf("Load function should not be called")
		return nil, nil, 1
	}

	cache := NewLRUSizeCache[int, int](
		10,
		load,
		nil,
	)

	for i := range 10 {
		require.NoError(t, cache.Insert(i, pointer.Int(i), 1))
		for j := range i + 1 {
			testGet(t, cache, j, pointer.Int(j), nil)
		}
	}
}

func TestLoad(t *testing.T) {
	counter := 0
	load := func(key int) (*int, error, uint64) {
		counter++
		if key < 0 {
			return nil, fmt.Errorf("key is %d", key), 1
		}
		return &key, nil, 1
	}

	cache := NewLRUSizeCache[int, int](
		20,
		load,
		nil,
	)

	for i := -5; i < 5; i++ {
		for j := -5; j <= i; j++ {
			if j < 0 {
				testGet(t, cache, j, nil, fmt.Errorf("key is %d", j))
			} else {
				testGet(t, cache, j, pointer.Int(j), nil)
			}
		}
	}
}

func TestDelete(t *testing.T) {
	counter := 0
	load := func(key int) (*int, error, uint64) {
		counter++
		return pointer.Int(counter - 1), nil, 1
	}
	delCount := 0
	del := func(int, *int) {
		delCount++
	}
	cache := NewLRUSizeCache[int, int](
		2,
		load,
		del,
	)

	expect := func(key int, value int) {
		testGet(t, cache, key, &value, nil)
	}

	for i := range 10 {
		expect(i, i)
	}
	for i := range 10 {
		expect(i, i+10)
	}
	expect(0, 20)
	expect(1, 21)
	expect(2, 22)
	expect(2, 22)
	expect(1, 21)
	expect(0, 23)
	expect(1, 21)
	expect(2, 24)
	expect(0, 25)
	expect(2, 24)
	expect(0, 25)
	expect(1, 26)
	expect(0, 25)
	expect(2, 27)
	require.Equal(t, counter-2, delCount)
}

func TestLock(t *testing.T) {
	counter := 0
	load := func(key int) (*int, error, uint64) {
		counter++
		return pointer.Int(counter - 1), nil, 1
	}
	cache := NewLRUSizeCache[int, int](
		2,
		load,
		nil,
	)
	expect := func(key int, value int) {
		testGet(t, cache, key, &value, nil)
	}
	cache.Lock(0)
	expect(0, 0)
	expect(1, 1)
	expect(2, 2)
	expect(1, 3)
	expect(2, 4)
	expect(0, 0)
	require.NoError(t, cache.Unlock(0))
	expect(2, 4)
	expect(1, 5)
	expect(0, 6)
	cache.Lock(10)
	expect(10, 7)
	cache.Lock(11)
	expect(11, 8)
	cache.Lock(12)
	expect(12, 9)
	expect(0, 10)
	expect(1, 11)
	expect(0, 12)
	expect(1, 13)
	expect(10, 7)
	expect(11, 8)
	expect(12, 9)
}

func TestAsyncLoad(t *testing.T) {
	var counter atomic.Int32
	load := func(key int) (*int, error, uint64) {
		time.Sleep(time.Millisecond * time.Duration(key))
		return pointer.Int(int(counter.Add(1))), nil, 1
	}
	cache := NewLRUSizeCache[int, int](
		10,
		load,
		nil,
	)
	expect := func(key int, value int) {
		testGet(t, cache, key, &value, nil)
	}

	cache.Lock(1000)
	cache.Lock(500)
	cache.Lock(0)

	expect(1000, 3)
	expect(500, 2)
	expect(0, 1)

	go func() {
		expect(1001, 6)
	}()
	go func() {
		expect(501, 5)
	}()
	go func() {
		expect(1, 4)
	}()
}

func singleTestGoroutine(t *testing.T, size uint64, maxKey int, iterations int) {
	load := func(key int) (*int, error, uint64) {
		return &key, nil, rand.Uint64() % 10
	}
	cache := NewLRUSizeCache[int, int](
		size,
		load,
		nil,
	)
	expect := func(key int, value int) {
		testGet(t, cache, key, &value, nil)
	}

	var wg sync.WaitGroup
	runThread := func() {
		lastLock := rand.Int() % maxKey
		cache.Lock(lastLock)
		for _ = range iterations {
			if rand.Int()%5 == 0 {
				require.NoError(t, cache.Unlock(lastLock))
				lastLock = rand.Int() % maxKey
				cache.Lock(lastLock)
			}
			key := rand.Int() % maxKey
			expect(key, key)
		}
		wg.Done()
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go runThread()
	}
	wg.Wait()
}

func TestGoroutines(t *testing.T) {
	singleTestGoroutine(t, 100, 10, 1000)
	singleTestGoroutine(t, 100, 100, 10000)
	singleTestGoroutine(t, 10, 10, 100000)
	singleTestGoroutine(t, 100, 100, 1000000)
}

func TestErrors(t *testing.T) {
	load := func(key int) (*int, error, uint64) {
		return &key, nil, 1
	}
	cache := NewLRUSizeCache[int, int](
		100,
		load,
		nil,
	)
	err := cache.Unlock(0)
	var errNotFound *ErrItemNotFound
	require.ErrorAs(t, err, &errNotFound)

	cache.Lock(0)
	require.NoError(t, cache.Unlock(0))

	err = cache.Unlock(0)
	var errNotLocked *ErrItemNotLocked
	require.ErrorAs(t, err, &errNotLocked)

	err = cache.Insert(0, pointer.Int(0), 1)
	var errItemExists *ErrItemAlreadyExists
	require.ErrorAs(t, err, &errItemExists)
}
