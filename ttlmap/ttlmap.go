package ttlmap

import (
	"sync"
	"time"
)

// code copied from https://yooo.ltd/2020/05/24/%E5%A6%82%E4%BD%95%E5%9C%A8Golang%E9%87%8C%E5%AE%9E%E7%8E%B0%E4%B8%80%E4%B8%AA%E9%AB%98%E6%80%A7%E8%83%BD%E7%9A%84TTLMap/

type TTLMap struct {
	items map[string]item
	mux   *sync.RWMutex
	now   int64
}

type item struct {
	value  interface{}
	expire int64
}

func NewTTLMap(cleanTick time.Duration) *TTLMap {
	tm := &TTLMap{map[string]item{},
		new(sync.RWMutex), time.Now().UnixNano()}
	go tm.clean(cleanTick)
	go tm.updateNow(time.Second)
	return tm
}

func (tm *TTLMap) Get(key string) (interface{}, bool) {
	tm.mux.RLock()
	defer tm.mux.RUnlock()
	if item, ok := tm.items[key]; !ok || tm.now > item.expire {
		return nil, false
	} else {
		return item.value, true
	}
}

func (tm *TTLMap) Set(key string, val interface{}, ex time.Duration) {
	tm.mux.Lock()
	tm.items[key] = item{value: val, expire: tm.now + ex.Nanoseconds()}
	tm.mux.Unlock()
}

func (tm *TTLMap) clean(tick time.Duration) {
	for range time.Tick(tick) {
		tm.mux.Lock()
		for key, item := range tm.items {
			if tm.now >= item.expire {
				delete(tm.items, key)
			}
		}
		tm.mux.Unlock()
	}
}

func (tm *TTLMap) updateNow(tick time.Duration) {
	for range time.Tick(tick) {
		tm.mux.Lock()
		tm.now = time.Now().UnixNano()
		tm.mux.Unlock()
	}
}
