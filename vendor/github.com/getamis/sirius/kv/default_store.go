// +build go1.9

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

import "sync"

// New creates in-memory key-value store
func New() Store {
	return &defaultStore{}
}

// ----------------------------------------------------------------------------

type defaultStore struct {
	m sync.Map
}

func (s *defaultStore) Put(key string, value []byte, opts ...PutOption) error {
	o := &PutOptions{}
	for _, opt := range opts {
		opt(o)
	}

	if o.IsPrefix || o.TTL > 0 {
		return ErrNotSupported
	}

	s.m.Store(key, value)

	return nil
}

func (s *defaultStore) AtomicPut(key string, value []byte, expected *KeyValue, opts ...PutOption) (*KeyValue, error) {
	return nil, ErrNotSupported
}

func (s *defaultStore) Get(key string) (*KeyValue, error) {
	if v, ok := s.m.Load(key); ok {
		return &KeyValue{
			Key:      key,
			Value:    v.([]byte),
			Revision: 1,
		}, nil
	}

	return nil, ErrKeyNotFound
}

func (s *defaultStore) List(prefix string) ([]*KeyValue, error) {
	return nil, ErrNotSupported
}

func (s *defaultStore) Delete(key string) error {
	exist, err := s.Exists(key)
	if err != nil {
		return err
	}

	if exist {
		s.m.Delete(key)
		return nil
	}

	return ErrKeyNotFound
}

func (s *defaultStore) AtomicDelete(key string, expected *KeyValue) (bool, error) {
	return false, ErrNotSupported
}

func (s *defaultStore) DeleteTree(prefix string) error {
	return ErrNotSupported
}

func (s *defaultStore) Exists(key string) (bool, error) {
	if _, ok := s.m.Load(key); ok {
		return true, nil
	}

	return false, nil
}

func (s *defaultStore) Watch(key string, stopCh <-chan struct{}) (<-chan *KeyValue, error) {
	return nil, ErrNotSupported
}

func (s *defaultStore) WatchTree(prefix string, stopCh <-chan struct{}) (<-chan []*KeyValue, error) {
	return nil, ErrNotSupported
}

func (s *defaultStore) Lock(key string, opts ...LockOption) (Locker, error) {
	return nil, ErrNotSupported
}

func (s *defaultStore) Close() {}
