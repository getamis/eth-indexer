// Copyright 2017 AMIS Technologies
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kv

import "errors"

var (
	// ErrNotSupported is returned when a method is not implemented/supported by the current backend
	ErrNotSupported = errors.New("not supported")
	// ErrKeyModified is returned during an atomic operation if the index does not match the one in the store
	ErrKeyModified = errors.New("key was modified")
	// ErrKeyNotFound is returned when the key is not found in the store during a Get operation
	ErrKeyNotFound = errors.New("key not found")
	// ErrKeyExists is returned when the previous value exists in the case of an AtomicPut
	ErrKeyExists = errors.New("key exists")
	// ErrUnableToLock is returned when there is an error when acquiring a lock on a key
	ErrUnableToLock = errors.New("failed to acquire the lock")
)

//go:generate mockery -name Store

// Store is the interface to access the backend
type Store interface {
	// Put a value with the specified key
	Put(key string, value []byte, opts ...PutOption) error

	// AtomicPut puts a single value and gets previous one if exists.
	// Pass previous = nil to create a new key.
	AtomicPut(key string, value []byte, expected *KeyValue, opts ...PutOption) (*KeyValue, error)

	// Get a value with given key
	Get(key string) (*KeyValue, error)

	// List the content with given prefix
	List(prefix string) ([]*KeyValue, error)

	// Delete the value with the specified key
	Delete(key string) error

	// Atomic delete of a single value
	AtomicDelete(key string, expected *KeyValue) (bool, error)

	// DeleteTree deletes a range of keys under a given prefix
	DeleteTree(prefix string) error

	// Exists checks if a key exists in the backend
	Exists(key string) (bool, error)

	// Watch for changes on a key
	Watch(key string, stopCh <-chan struct{}) (<-chan *KeyValue, error)

	// WatchTree watches for changes on child nodes under a given prefix
	WatchTree(prefix string, stopCh <-chan struct{}) (<-chan []*KeyValue, error)

	// Lock locks the given key.
	// The returned Locker is not held and must be acquired
	Lock(key string, opts ...LockOption) (Locker, error)

	// Close the connection
	Close()
}

// KeyValue represents {Key, Value, Revision} tuple
type KeyValue struct {
	Key      string
	Value    []byte
	Revision uint64
}

// Locker provides locking mechanism on top of the backend.
type Locker interface {
	Lock(stopChan chan struct{}) (<-chan struct{}, error)
	Unlock() error
}
