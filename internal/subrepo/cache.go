package subrepo

import (
	"sync"
	"time"
)

// HashCacheEntry hash缓存条目
// Hash cache entry
type HashCacheEntry struct {
	Hash    string
	ModTime time.Time
	Size    int64
}

// HashCache hash缓存
// Hash cache
type HashCache struct {
	cache map[string]HashCacheEntry
	mu    sync.RWMutex
}

// NewHashCache 创建hash缓存
// Creates a new hash cache
func NewHashCache() *HashCache {
	return &HashCache{
		cache: make(map[string]HashCacheEntry),
	}
}

// Get 获取缓存的hash
// Gets cached hash
func (hc *HashCache) Get(path string, modTime time.Time, size int64) (string, bool) {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	
	entry, exists := hc.cache[path]
	if !exists {
		return "", false
	}
	
	// 检查文件是否被修改
	// Check if file has been modified
	if entry.ModTime.Equal(modTime) && entry.Size == size {
		return entry.Hash, true
	}
	
	return "", false
}

// Set 设置hash缓存
// Sets hash cache
func (hc *HashCache) Set(path, hash string, modTime time.Time, size int64) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	
	hc.cache[path] = HashCacheEntry{
		Hash:    hash,
		ModTime: modTime,
		Size:    size,
	}
}

// Clear 清空缓存
// Clears cache
func (hc *HashCache) Clear() {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	
	hc.cache = make(map[string]HashCacheEntry)
}

// Size 获取缓存大小
// Gets cache size
func (hc *HashCache) Size() int {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	
	return len(hc.cache)
}
