package syncset

import "sync"

// A set which supports concurrent use through an internal mutex; operations may
// block.
type Set struct {
    set map[string]struct{}
    mutex sync.RWMutex
}

func New() Set {
    return Set{make(map[string]struct{}), sync.RWMutex{}}
}

func (set *Set) Insert(value string) {
    set.mutex.Lock()
    set.set[value] = struct{}{}
    set.mutex.Unlock()
}

func (set *Set) Contains(value string) bool {
    set.mutex.RLock()
    _, ok := set.set[value]
    set.mutex.RUnlock()

    return ok
}
