package proxy

import (
	"container/list"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

// ! LRUCache represents the LRU cache
type LRUCache struct {
	capacity int
	cache    map[string]*list.Element
	list     *list.List
	mu       sync.Mutex
}

// ! CacheItem represents an item in the cache
type CacheItem struct {
	key   string
	value []byte
}

// ! NewLRUCache creates a new LRU cache with the given capacity
func NewLRUCache(capacity int) *LRUCache {
	return &LRUCache{
		capacity: capacity,
		cache:    make(map[string]*list.Element),
		list:     list.New(),
	}
}

// ! Get retrieves a value from the cache
func (lru *LRUCache) Get(key string) ([]byte, bool) {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	if elem, found := lru.cache[key]; found {
		lru.list.MoveToFront(elem) // Mark as recently used
		return elem.Value.(*CacheItem).value, true
	}
	return nil, false
}

// ! Put adds a value to the cache
func (lru *LRUCache) Put(key string, value []byte) {
	lru.mu.Lock()
	defer lru.mu.Unlock()

	if elem, found := lru.cache[key]; found {
		lru.list.MoveToFront(elem) //? Update existing item
		elem.Value.(*CacheItem).value = value
		return
	}

	if len(lru.cache) >= lru.capacity {
		//? Evict the least recently used item
		lastElem := lru.list.Back()
		if lastElem != nil {
			delete(lru.cache, lastElem.Value.(*CacheItem).key)
			lru.list.Remove(lastElem)
		}
	}

	//! Add new item to the cache
	newItem := &CacheItem{key, value}
	elem := lru.list.PushFront(newItem)
	lru.cache[key] = elem
}

// ! Global cache instance
var cache = NewLRUCache(10)

// ! HTTP client with a timeout
var client = &http.Client{
	Timeout: 10 * time.Second,
}

// ! Function to handle incoming requests
func handleRequest(w http.ResponseWriter, r *http.Request) {
	cacheKey := r.URL.String()

	//? Check if response is cached
	if cachedResp, found := cache.Get(cacheKey); found {
		fmt.Println("Cache hit:", cacheKey)
		w.Write(cachedResp)
		return
	}

	//? If not cached, forward the request to the target server
	resp, err := client.Get(r.URL.String())
	if err != nil {
		http.Error(w, "Failed to fetch from target", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	//? Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read response", http.StatusInternalServerError)
		return
	}

	//? Store response in cache
	cache.Put(cacheKey, body)

	//? Write the response back to the client
	w.Write(body)
}

// ? Main function to start the proxy server
func main() {
	http.HandleFunc("/", handleRequest)

	//? Start the HTTP server
	fmt.Println("Starting proxy server on port 8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("Server failed:", err)
	}
}
