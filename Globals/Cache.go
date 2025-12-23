package Globals

import (
	"sync"
	"time"
)

type CacheEntry struct {

	Value      interface{}
	ExpiresAt  time.Time
	
}

type Cache struct {

	Data  map[string]CacheEntry
	Mutex sync.RWMutex

}

var Caches = make(map[string]*Cache)
var CachesMutex = sync.RWMutex{}

func GetOrCreateCache(CacheName string) *Cache {

	CachesMutex.Lock()
	defer CachesMutex.Unlock()

	if CacheInstance, Exists := Caches[CacheName]; Exists {

		return CacheInstance

	}

	NewCache := &Cache{

		Data: make(map[string]CacheEntry),

	}

	Caches[CacheName] = NewCache

	return NewCache

}

func (C *Cache) Set(Key string, Value interface{}, TTL time.Duration) {

	C.Mutex.Lock()
	defer C.Mutex.Unlock()

	ExpiresAt := time.Now().Add(TTL)

	if TTL == 0 {

		ExpiresAt = time.Time{}

	}

	C.Data[Key] = CacheEntry{

		Value:     Value,
		ExpiresAt: ExpiresAt,

	}

}

func (C *Cache) Get(Key string) (any, bool) {

	C.Mutex.RLock()
	defer C.Mutex.RUnlock()

	Entry, Exists := C.Data[Key]

	if !Exists { return nil, false }

	if !Entry.ExpiresAt.IsZero() && time.Now().After(Entry.ExpiresAt) {

		return nil, false

	}

	return Entry.Value, true

}

func (C *Cache) Delete(Key string) {

	C.Mutex.Lock()
	defer C.Mutex.Unlock()

	delete(C.Data, Key)

}

func (C *Cache) Clear() {

	C.Mutex.Lock()
	defer C.Mutex.Unlock()

	C.Data = make(map[string]CacheEntry)

}

func (C *Cache) CleanExpired() {

	C.Mutex.Lock()
	defer C.Mutex.Unlock()

	Now := time.Now()

	for Key, Entry := range C.Data {

		if !Entry.ExpiresAt.IsZero() && Now.After(Entry.ExpiresAt) {

			delete(C.Data, Key)

		}

	}

}

func (C *Cache) StartAutoCleanup(Interval time.Duration) {

	go func() {

		for {

			time.Sleep(Interval)
			C.CleanExpired()

		}

	}()

}