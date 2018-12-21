// Copyright 2018 AMIS Technologies
// This file is part of the hypereth library.
//
// The hypereth library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The hypereth library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the hypereth library. If not, see <http://www.gnu.org/licenses/>.

package multiclient

import (
	"sync"

	"github.com/ethereum/go-ethereum/rpc"
)

type Map struct {
	m map[string]*rpc.Client

	lock sync.RWMutex
}

func NewMap() *Map {
	return &Map{
		m: make(map[string]*rpc.Client),
	}
}

func (m *Map) Delete(key string) {
	m.lock.Lock()
	defer m.lock.Unlock()

	delete(m.m, key)
}

func (m *Map) Set(key string, value *rpc.Client) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.m[key] = value
}

func (m *Map) Get(key string) *rpc.Client {
	m.lock.RLock()
	defer m.lock.RUnlock()

	return m.m[key]
}

func (m *Map) Len() int {
	m.lock.RLock()
	defer m.lock.RUnlock()

	return len(m.m)
}

// List returns a deep copy of client list
func (m *Map) List() []*rpc.Client {
	m.lock.RLock()
	defer m.lock.RUnlock()

	l := []*rpc.Client{}
	for _, v := range m.m {
		if v != nil {
			l = append(l, v)
		}
	}
	return l
}

// Map returns a deep copy of client map
func (m *Map) Map() map[string]*rpc.Client {
	m.lock.RLock()
	defer m.lock.RUnlock()

	newMap := map[string]*rpc.Client{}
	for k, v := range m.m {
		if v != nil {
			newMap[k] = v
		}
	}
	return newMap
}

func (m *Map) Keys() []string {
	m.lock.RLock()
	defer m.lock.RUnlock()

	urls := make([]string, len(m.m))
	index := 0
	for k := range m.m {
		urls[index] = k
		index++
	}
	return urls
}
