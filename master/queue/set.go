package queue

type Set[T comparable] struct {
	items map[T]struct{}
}

func NewSet[T comparable]() Set[T] {
	return Set[T]{
		items: make(map[T]struct{}),
	}
}

func (s *Set[T]) Add(item T) {
	s.items[item] = struct{}{}
}

func (s *Set[T]) SafeRemove(item T) {
	if _, ok := s.items[item]; ok {
		delete(s.items, item)
	}
}

func (s *Set[T]) Remove(item T) {
	delete(s.items, item)
}

func (s *Set[T]) Contains(item T) bool {
	_, ok := s.items[item]
	return ok
}

func (s *Set[T]) Len() int {
	return len(s.items)
}

func (s *Set[T]) Empty() bool {
	return s.Len() == 0
}
