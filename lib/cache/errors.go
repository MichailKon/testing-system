package cache

import "fmt"

type ErrItemLocked struct {
	key interface{}
}

func (e *ErrItemLocked) Error() string {
	return fmt.Sprintf("size_cache: item is locked, key: %#v", e.key)
}

type ErrItemNotFound struct {
	key interface{}
}

func (e *ErrItemNotFound) Error() string {
	return fmt.Sprintf("size_cache: item not found, key: %#v", e.key)
}

type ErrItemNotLocked struct {
	key interface{}
}

func (e *ErrItemNotLocked) Error() string {
	return fmt.Sprintf("size_cache: item not locked, key: %#v", e.key)
}
