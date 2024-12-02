/*
(c) Copyright 2018 Hewlett Packard Enterprise Development LP
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package concurrent

import (
	"sync"
	"testing"
	"time"
)

const (
	testCount = 999
)

var (
	mm *MapMutex
	wg sync.WaitGroup
)

//TestManyKeys takes over 7 seconds (999 x 999 go routines)
//func TestManyKeys(t *testing.T) {
//	tests := make(map[string][]int)
//	for index := 0; index < testCount; index++ {
//		tests[fmt.Sprintf("%d", index)] = make([]int, testCount)
//	}
//	testWithMap(tests, t)
//}

// TestManyLocks has been run with -race at a count of 999
// TestManyLocks has been run with a count of 99999
func TestManyLocks(t *testing.T) {

	tests := map[string][]int{
		"foo":                            make([]int, testCount),
		"bar":                            make([]int, testCount),
		"foobar":                         make([]int, testCount),
		"fun":                            make([]int, testCount),
		"haha":                           make([]int, testCount),
		"a Really        Long      Key!": make([]int, testCount),
	}

	testWithMap(tests, t)
}
func TestLockUnlock(t *testing.T) {
	mm = NewMapMutex()
	lockName := "testLock"

	// Lock the mutex
	mm.Lock(lockName)

	// Try to lock the mutex again in a different goroutine
	locked := make(chan bool)
	go func() {
		mm.Lock(lockName)
		locked <- true
	}()

	select {
	case <-locked:
		t.Error("Lock should not be re-entrant")
	default:
	}

	// Unlock the mutex
	mm.Unlock(lockName)

	// Now the other goroutine should be able to lock the mutex
	select {
	case <-locked:
	case <-time.After(time.Second):
		t.Error("Lock should be acquired after unlock")
	}

	// Clean up
	mm.Unlock(lockName)
}

func testWithMap(tests map[string][]int, t *testing.T) {

	mm = NewMapMutex()

	wg.Add(testCount * len(tests))
	for lockName, data := range tests {
		go load(lockName, data)
	}

	wg.Wait()

	for index := 0; index < testCount; index++ {
		for lockName, data := range tests {
			verify(lockName, data, index, t)
		}
	}

}

func load(lock string, data []int) {
	for index := 0; index < testCount; index++ {
		go loadItem(lock, data, index)
	}
}

func loadItem(lock string, data []int, value int) {
	defer wg.Done()
	mm.Lock(lock)
	data[value] = value
	mm.Unlock(lock)
}

func verify(lock string, data []int, index int, t *testing.T) {
	if data[index] != index {
		t.Error(
			"For", lock, "array,index", index,
			"expected", index,
			"got", data[index],
		)
	}
}

// TODO: instead of Sleep below, introduce a Map to simulate the actual usage of lock.
func TestConcurrentLockUnlock(t *testing.T) {
	mm = NewMapMutex()
	lockName := "concurrentLock"

	const goroutineCount = 100
	var wg sync.WaitGroup
	wg.Add(goroutineCount)

	for i := 0; i < goroutineCount; i++ {
		go func() {
			defer wg.Done()
			mm.Lock(lockName)
			time.Sleep(10 * time.Millisecond) // Simulate some work
			mm.Unlock(lockName)
		}()
	}

	wg.Wait()
}
