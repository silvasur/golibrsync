package librsync

import (
	"sync"
)

// pointerMap holds Go *Patcher objects to pass to C.
// Don't touch this data structure, instead use the storePatcher, getPatcher,
// and dropPatcher functions.
var patcherStore = struct {
	lock  sync.Mutex
	store map[uintptr]*Patcher
}{
	store: make(map[uintptr]*Patcher),
}

// storePatcher stores the value and returns a reference to it, for use in a CGo
// call. Use the same reference for dropPatcher. C callbacks can use getPatcher
// to get the original value.
func storePatcher(patcher *Patcher, id uintptr) {
	patcherStore.lock.Lock()
	defer patcherStore.lock.Unlock()

	if _, ok := patcherStore.store[id]; ok {
		// Just to be on the safe side.
		panic("pointer already stored")
	}
	patcherStore.store[id] = patcher
}

// getPatcher returns the value for the reference id. It returns nil if there
// is no such reference.
func getPatcher(id uintptr) *Patcher {
	patcherStore.lock.Lock()
	defer patcherStore.lock.Unlock()

	return patcherStore.store[id]
}

// dropPatcher unreferences the value so the garbage collector can free it's
// memory.
func dropPatcher(id uintptr) {
	patcherStore.lock.Lock()
	defer patcherStore.lock.Unlock()

	if _, ok := patcherStore.store[id]; !ok {
		panic("pointer not stored")
	}

	delete(patcherStore.store, id)
}
