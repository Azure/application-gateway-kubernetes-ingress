// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package utils

import (
	"github.com/deckarep/golang-set"
	"sync"
)

// ThreadsafeMultiMap is a thread safe implementation of a multimap.
type ThreadsafeMultiMap interface {
	Insert(key interface{}, value interface{})
	Clear(key interface{})
	Erase(key interface{}) bool
	EraseValue(value interface{}) bool
	ContainsPair(key interface{}, value interface{}) bool
	ContainsValue(value interface{}) bool
}

type threadsafeMultiMap struct {

	// TODO: Evaluate if sync.Map is needed here
	//       for our usage pattern.
	sync.RWMutex
	v map[interface{}]mapset.Set
}

// NewThreadsafeMultimap creates a ThreadsafeMultiMap.
func NewThreadsafeMultimap() ThreadsafeMultiMap {
	return &threadsafeMultiMap{
		v: make(map[interface{}]mapset.Set),
	}
}

// Insert inserts a key value pair into the multimap.
func (m *threadsafeMultiMap) Insert(key interface{}, value interface{}) {
	m.Lock()
	defer m.Unlock()

	if m.v[key] == nil {
		m.v[key] = mapset.NewSet()
	}
	m.v[key].Add(value)
}

// Clear removes all the values associated with a key.
func (m *threadsafeMultiMap) Clear(key interface{}) {
	m.Lock()
	defer m.Unlock()

	if m.v[key] != nil {
		m.v[key].Clear()
	}
}

// Erase removes a key from the multimap.
func (m *threadsafeMultiMap) Erase(key interface{}) bool {
	m.Lock()
	defer m.Unlock()

	_, exists := m.v[key]
	if exists {
		delete(m.v, key)
		return true
	}

	return false
}

// EraseValue removes all the occurrences of a particular value from the multimap.
func (m *threadsafeMultiMap) EraseValue(value interface{}) bool {
	m.Lock()
	defer m.Unlock()

	erased := false
	for i := range m.v {
		if m.v[i] != nil && m.v[i].Contains(value) {
			erased = true
			m.v[i].Remove(value)
		}
	}

	return erased
}

// ContainsPair checks if a particular key value pair exists in the multimap.
func (m *threadsafeMultiMap) ContainsPair(key interface{}, value interface{}) bool {
	m.RLock()
	defer m.RUnlock()

	if m.v[key] != nil && m.v[key].Contains(value) {
		return true
	}

	return false
}

// ContainsValue checks if a particular value exists in the multimap.
func (m *threadsafeMultiMap) ContainsValue(value interface{}) bool {
	m.RLock()
	defer m.RUnlock()

	for i := range m.v {
		if m.v[i] != nil && m.v[i].Contains(value) {
			return true
		}
	}

	return false
}
