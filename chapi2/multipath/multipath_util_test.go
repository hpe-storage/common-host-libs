// (c) Copyright 2019 Hewlett Packard Enterprise Development LP

package multipath

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

const (
	minCacheEntries  = 10                       // Minimum number of cached entries we'll test
	maxCacheEntries  = 32                       // Maximum number of cached entries we'll test
	targetExpiration = (2 * time.Second)        // Amount of time to keep cache entry around if not accessed
	cleanupFrequency = (100 * time.Millisecond) // Background thread cache cleanup frequency
)

func TestTargetTypeCache(t *testing.T) {

	// Randomly pick the number of cache entries we'll exercise
	rand.Seed(time.Now().UnixNano())
	maxCacheEntries := minCacheEntries + int(rand.Float64()*(maxCacheEntries-minCacheEntries+1))

	// Allocate a new TargetTypeCache object
	c := NewCustomTargetTypeCache(maxCacheEntries, targetExpiration, cleanupFrequency)

	// Store 2x the number of cached entries
	var lastCachedIndex int
	for i := 0; i < 2*maxCacheEntries; i++ {
		// In case host doesn't have a high precision timer, we're going to sleep 25 msec
		// between the first and second half entries.  We want to give additional time
		// so that only the first half entries are removed and not the second half.
		if i == maxCacheEntries {
			time.Sleep(25 * time.Millisecond)
		}
		c.SetTargetType(getTargetName(i), "group")
		lastCachedIndex = i
	}

	// Only the last half entries should be in the cache (i.e. maxCacheEntries)
	for i := 0; i < 2*maxCacheEntries; i++ {
		iqn := getTargetName(i)
		targetType := c.GetTargetType(iqn)
		if ((i < maxCacheEntries) && (targetType != "")) || ((i >= maxCacheEntries) && (targetType == "")) {
			t.Errorf("Maximum cache entry test failure, index=%v, maxCacheEntries=%v, iqn=%v, targetType=%v", i, maxCacheEntries, iqn, targetType)
		}
	}

	// Sleep one second and then read the last cached entry to refresh its last accessed time
	time.Sleep(1 * time.Second)
	targetType := c.GetTargetType(getTargetName(lastCachedIndex))
	if targetType == "" {
		t.Errorf("Last cached index %v should not be empty, count=%v", lastCachedIndex, c.len())
	}

	// At this point, the cache should have maxCacheEntries within it.  The last cached index will
	// expire in targetExpiration time.  All the rest will expire in targetExpiration minus one
	// second.  Wait, up until targetExpiration, for the cache count to drop down to one entry.
	timeStart := time.Now()
	for ; (time.Since(timeStart) < targetExpiration) && (c.len() != 1); time.Sleep(10 * time.Millisecond) {
	}

	// At this point, we should only have one cached entry remaining
	if cacheCount := c.len(); (cacheCount != 1) || (c.GetTargetType(getTargetName(lastCachedIndex)) == "") {
		t.Errorf("Expected only single last cache entry, cacheCount=%v", cacheCount)
	}
}

// getTargetName returns a unique string for the given index
func getTargetName(index int) string {
	return fmt.Sprintf("target%v", index)
}
