// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package utils

import "sync"

// ThreadsafeMultiMap is a thread safe implementation of a multimap.
type ThreadsafeMultiMap interface {
	Insert(key interface{}, value interface{})
	Clear(key interface{})
	Erase(key interface{})
	EraseValue(value interface{})
	ContainsPair(key interface{}, value interface{}) bool
	ContainsValue(value interface{}) bool
}

type threadsafeMultiMap struct {
	sync.Map
}

// NewThreadsafeMultimap creates a ThreadsafeMultiMap.
func NewThreadsafeMultimap() ThreadsafeMultiMap {
	return &threadsafeMultiMap{}
}

// Insert inserts a key value pair into the multimap.
func (m *threadsafeMultiMap) Insert(key interface{}, value interface{}) {
	set, _ := m.LoadOrStore(key, &sync.Map{})
	set.(*sync.Map).Store(value, struct{}{})
}

// Clear removes all the values associated with a key.
func (m *threadsafeMultiMap) Clear(key interface{}) {
	m.Store(key, &sync.Map{})
}

// Erase removes a key from the multimap.
func (m *threadsafeMultiMap) Erase(key interface{}) {
	m.Delete(key)
}

// EraseValue removes all the occurrences of a particular value from the multimap.
func (m *threadsafeMultiMap) EraseValue(value interface{}) {
	m.Range(func(k, v interface{}) bool {
		v.(*sync.Map).Delete(value)
		return true
	})
}

// ContainsPair checks if a particular key value pair exists in the multimap.
func (m *threadsafeMultiMap) ContainsPair(key interface{}, value interface{}) bool {
	set, ok := m.Load(key)
	if !ok {
		return false
	}

	s, ok := set.(*sync.Map)
	if !ok {
		return false
	}

	_, ok = s.Load(value)
	return ok
}

// ContainsValue checks if a particular value exists in the multimap.
func (m *threadsafeMultiMap) ContainsValue(value interface{}) bool {
	found := false
	m.Range(func(k, v interface{}) bool {
		_, ok := v.(*sync.Map).Load(value)
		if ok {
			found = true
			return false
		}

		return true
	})

	return found
}
