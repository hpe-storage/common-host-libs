// Copyright 2019 Hewlett Packard Enterprise Development LP

package etcd

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDBClientSuite(t *testing.T) {
	// TODO: Uncomment this to run integration tests
	// _TestAll(t)
}

func _TestAll(t *testing.T) {
	_TestDBClientSuite1(t)
	_TestDBClientSuite2(t)
	// TestDBClientSuiteLockUnlock(t)
}

func _TestDBClientSuiteLockUnlock(t *testing.T) {
	key := "mylock"

	// Create DB Client
	endPoints := []string{fmt.Sprintf("%s:%s", "localhost", DefaultPort)}
	dbClient, err := NewClient(endPoints, DefaultVersion)
	if err != nil {
		t.Errorf("NewClient() error = %v", err)
		return
	}
	defer dbClient.CloseClient()

	// Check the lock status. This must be in 'UNLOCKED' state
	locked, err := dbClient.IsLocked(key)
	if err != nil {
		t.Errorf("Failed to check if the key '%s' is locked, err: %s", key, err.Error())
	}
	assert.Equal(t, false /* unlocked */, locked)

	// Acquire the lock
	lck, err := dbClient.WaitAcquireLock(key, 30)
	if err != nil {
		t.Errorf("Failed to lock key '%s', err: %s", key, err.Error())
	}
	//time.Sleep(10 * time.Second)
	// Check the lock status. This must be in 'LOCKED' state
	locked, err = dbClient.IsLocked(key)
	if err != nil {
		t.Errorf("Failed to check if the key '%s' is locked, err: %s", key, err.Error())
	}
	assert.Equal(t, true /* locked */, locked)

	// Try to acquire lock and expect error
	lck1, err := dbClient.AcquireLock(key, 30)
	assert.Nil(t, lck1)
	assert.NotNil(t, err)

	// Release the lock
	assert.Nil(t, dbClient.ReleaseLock(lck))

	// Check the lock status. This must be in 'UNLOCKED' stat
	locked, err = dbClient.IsLocked(key)
	if err != nil {
		t.Errorf("Failed to check if the key '%s' is locked, err: %s", key, err.Error())
	}
	assert.Equal(t, false /* unlocked */, locked)
}

func _TestDBClientSuite1(t *testing.T) {
	// Create DB Client
	endPoints := []string{fmt.Sprintf("%s:%s", "localhost", DefaultPort)}
	dbClient, err := NewClient(endPoints, DefaultVersion)
	if err != nil {
		t.Errorf("NewClient() error = %v", err)
		return
	}
	defer dbClient.CloseClient()

	key := "TestFoo1"
	value := "TestBar"

	// Put
	err = dbClient.Put(key, value)
	if err != nil {
		t.Errorf("PUT error = %v", err)
		return
	}

	// Get
	gotVal, err := dbClient.Get(key)
	if err != nil {
		t.Errorf("GET error = %v", err)
		return
	}
	assert.Equal(t, value, *gotVal, fmt.Sprintf("Get() = Expected: %v, Got: %v", value, *gotVal))

	// Delete
	err = dbClient.Delete(key)
	if err != nil {
		t.Errorf("DELETE error = %v", err)
		return
	}

	// Get again
	gotVal, err = dbClient.Get(key)
	if err != nil {
		t.Errorf("GET error = %v", err)
		return
	}
	assert.Nil(t, gotVal, fmt.Sprintf("Get() = Expected: nil, Got: %v", gotVal))

	// Put with lease expiry of 5 seconds
	err = dbClient.PutWithLeaseExpiry("SUN", value, 5)
	if err != nil {
		t.Errorf("PUT error = %v", err)
		return
	}
	// Sleep for 6 seconds for the lease to expiry
	time.Sleep(6 * time.Second)

	// Get - Check if the key is being removed
	gotVal, err = dbClient.Get(key)
	if err != nil {
		t.Errorf("GET error = %v", err)
		return
	}
	assert.Nil(t, gotVal, fmt.Sprintf("Get() = Expected: nil, Got: %v", gotVal))
}

func _TestDBClientSuite2(t *testing.T) {

	key := "TestFoo2"
	value := "TestBar"

	// Put
	err := Put(key, value)
	if err != nil {
		t.Errorf("PUT error = %v", err)
		return
	}

	// Get
	gotVal, err := Get(key)
	if err != nil {
		t.Errorf("GET error = %v", err)
		return
	}
	assert.Equal(t, value, *gotVal, fmt.Sprintf("Get() = Expected: %v, Got: %v", value, *gotVal))

	// Delete
	err = Delete(key)
	if err != nil {
		t.Errorf("DELETE error = %v", err)
		return
	}

	// Get again
	gotVal, err = Get(key)
	if err != nil {
		t.Errorf("GET error = %v", err)
		return
	}
	assert.Nil(t, gotVal, fmt.Sprintf("Get() = Expected: nil, Got: %v", gotVal))
}
