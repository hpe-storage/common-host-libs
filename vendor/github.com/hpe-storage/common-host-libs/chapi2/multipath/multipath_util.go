// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

package multipath

import (
	"sort"
	"strings"
	"sync"
	"time"

	log "github.com/hpe-storage/common-host-libs/logger"
)

const (
	defaultTargetMaxEntries = 128              // Default maximum number of target entries to store in the cache
	defaultTargetExpiration = (24 * time.Hour) // Default amount of time to keep cache entry around if not accessed (-1 == no expiration)
	defaultCleanupFrequency = (1 * time.Hour)  // Default background thread cache cleanup frequency (e.g. once per hour)
)

// TargetTypeCache is used to maintain a cache of target types (group or volume) with the iSCSI
// target iqn used as the map key.
//
// NOTE:  A VST iqn is very different than a GST iqn.  That is why it is safe to cache the target
// type per iqn (i.e. "volume" or "group").  However, it is technically possible for a user, using
// the CLI, to change a GST iqn to make it look like a VST iqn.  If the cache has set a VST iqn as
// "volume", and the user removes the VST, and gives the GST the VST iqn, then the cache will have
// the wrong target type.  I can't imagine a scenario where this would actually take place.  The
// SetTargetType() method, with an empty target type, can be called to clear any target entry from
// the cache.  Alternatively, the CHAPI service can be restarted to clear the cache.
type TargetTypeCache struct {
	lock             sync.RWMutex            // RWMutex for thread safety
	maxCachedEntries int                     // Maximum number of targets to cache
	expiration       time.Duration           // If object not accessed within this duration, object will be freed (-1 == no expiration)
	threadSleep      time.Duration           // Background thread cache cleanup frequency (e.g. once per hour)
	cache            map[string]*iscsiTarget // Cached entries, key is target iqn, value is iscsiTarget object
}

// iscsiTarget is the value portion of our key/value map (key is target iqn)
type iscsiTarget struct {
	targetType   string    // Nimble iSCSI target type (i.e. "group" or "volume")
	lastAccessed time.Time // Time target type was last accessed
}

// NewTargetTypeCache allocates a new TargetTypeCache object with default settings
func NewTargetTypeCache() *TargetTypeCache {
	log.Trace(">>>>> NewTargetTypeCache")
	defer log.Trace("<<<<< NewTargetTypeCache")
	return NewCustomTargetTypeCache(defaultTargetMaxEntries, defaultTargetExpiration, defaultCleanupFrequency)
}

// NewCustomTargetTypeCache allocates a new TargetTypeCache object with the specified settings
func NewCustomTargetTypeCache(maxCachedEntries int, expiration time.Duration, threadSleep time.Duration) *TargetTypeCache {
	log.Tracef(">>>>> NewCustomTargetTypeCache, maxCachedEntries=%v, expiration=%v, threadSleep=%v", maxCachedEntries, expiration, threadSleep)
	defer log.Trace("<<<<< NewCustomTargetTypeCache")

	// Allocate the TargetTypeCache object
	c := &TargetTypeCache{
		maxCachedEntries: maxCachedEntries,
		expiration:       expiration,
		threadSleep:      threadSleep,
		cache:            make(map[string]*iscsiTarget),
	}

	// If a cache expiration setting is used, have a background thread periodically check the
	// cache and remove any expired entries.
	if c.expiration != -1 {
		go c.maintainCacheThread()
	}

	// Return the initialized TargetTypeCache object
	return c
}

// ClearCache clears all entries from the target type cache
func (c *TargetTypeCache) ClearCache() {
	// Lock for thread safety
	c.lock.Lock()
	defer c.lock.Unlock()

	// Simply allocate a new map to release all map entries
	c.cache = make(map[string]*iscsiTarget)
}

// len returns the number of cached entries
func (c *TargetTypeCache) len() int {
	// Lock for thread safety
	c.lock.RLock()
	defer c.lock.RUnlock()

	return len(c.cache)
}

// GetTargetType returns the cached target type for the given iSCSI iqn.  If no cache entry was
// found, an empty string is returned.
func (c *TargetTypeCache) GetTargetType(targetIqn string) string {
	log.Tracef(">>>>> GetTargetType, targetIqn=%v", targetIqn)
	defer log.Trace("<<<<< GetTargetType")

	// Lock for thread safety
	c.lock.RLock()
	defer c.lock.RUnlock()

	// Get the cached target details; return "" if no cache entry
	target := c.cache[strings.ToLower(targetIqn)]
	if target == nil {
		return ""
	}

	// Update the last accessed time and return the cached target type
	target.lastAccessed = time.Now()
	log.Tracef("targetIqn=%v, targetType=%v", targetIqn, target.targetType)
	return target.targetType
}

// SetTargetType sets the cached target type for the given iSCSI iqn.  If "targetType" is an empty
// string, any cache entry for the given iqn is removed.
func (c *TargetTypeCache) SetTargetType(targetIqn, targetType string) {
	log.Tracef(">>>>> SetTargetType, targetIqn=%v, targetType=%v", targetIqn, targetType)
	defer log.Trace("<<<<< SetTargetType")

	// Lock for thread safety
	c.lock.Lock()
	defer c.lock.Unlock()

	// If passed in an empty string, simply delete the entry from the cache (if present)
	targetIqn = strings.ToLower(targetIqn)
	if targetType == "" {
		delete(c.cache, targetIqn)
		return
	}

	// Get the cached target details; allocate target object if not in cache
	target := c.cache[targetIqn]
	if target == nil {
		target = new(iscsiTarget)
	}

	// Update the last access and target type properties; update the cache
	target.lastAccessed = time.Now()
	target.targetType = targetType
	c.cache[targetIqn] = target

	// Force a cache refresh, on exit, only if maximum count reached
	if len(c.cache) > c.maxCachedEntries {
		c.maintainCache(false)
	}
}

// maintainCacheThread is our background cache maintenance thread
func (c *TargetTypeCache) maintainCacheThread() {
	log.Trace(">>>>> maintainCacheThread")
	defer log.Trace("<<<<< maintainCacheThread")

	// Determine how long the thread should sleep before waking and performing a maintenance pass.
	timeChan := time.Tick(c.threadSleep)

	for {
		select {
		case <-timeChan:
			// Cache frequency period has expired
			log.Tracef("Periodic cache maintenance signal received, sleepTime=%v", c.threadSleep)
			c.maintainCache(true)
		}
	}
}

// maintainCache is an internal routine that is called to maintain the cache.  If there are entries
// that have expired, or if the number of cache entries has exceeds its limits, this routine will
// perform the necessary cleanup.
func (c *TargetTypeCache) maintainCache(lockNeeded bool) {
	log.Trace(">>>>> maintainCache")
	defer log.Trace("<<<<< maintainCache")

	// Lock for thread safety
	if lockNeeded {
		c.lock.Lock()
		defer c.lock.Unlock()
	}

	// If there are no cache entries, there is no maintenance needed
	if len(c.cache) == 0 {
		return
	}

	// If the cached entries have not exceeded the maximum count, and no expiration has been set,
	// there is no maintenance needed.
	if (len(c.cache) <= c.maxCachedEntries) && (c.expiration == -1) {
		return
	}

	// Convert the cache from a map into a kv slice
	type kv struct {
		Key   string
		Value time.Duration
	}
	var ss []kv
	for k, v := range c.cache {
		ss = append(ss, kv{k, time.Since(v.lastAccessed)})
	}

	// Sort the slice from newest to oldest entries
	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Value < ss[j].Value
	})

	// Is a cache expiration set?
	if c.expiration != -1 {
		// Scanning from the oldest to the newest entries, remove any expired entries from cache
		for i := len(ss) - 1; i >= 0; i-- {
			// If the current entry hasn't expired, break out of loop as there are no further expired entries
			if ss[i].Value < c.expiration {
				break
			}
			// Remove the expired entry from the cache
			log.Tracef("Cached target type entry expired, iqn=%v, lastAccessed=%v", ss[i].Key, ss[i].Value)
			delete(c.cache, ss[i].Key)
		}
	}

	// Remove any entries needed to stay within the cache limits
	if len(c.cache) > c.maxCachedEntries {
		for i := c.maxCachedEntries; i < len(ss); i++ {
			log.Tracef("Removing cached target entry, iqn=%v, lastAccessed=%v", ss[i].Key, ss[i].Value)
			delete(c.cache, ss[i].Key)
		}
	}
}
