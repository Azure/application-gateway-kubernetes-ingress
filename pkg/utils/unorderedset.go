// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package utils

// UnorderedSet implements an unordered set using GoLang map as the underlying
// data structure.
type UnorderedSet interface {
	Insert(item interface{})
	Erase(item interface{})
	Contains(item interface{}) bool
	Size() int
	Clear()
	IsEmpty() bool
	ForEach(func(interface{}))
	Union(s UnorderedSet) UnorderedSet
	Intersect(s UnorderedSet) UnorderedSet
	ToSlice() []interface{}
}

type unorderedSet struct {
	v map[interface{}]struct{}
}

var emptyKey = struct{}{}

// NewUnorderedSet creates an UnorderedSet
func NewUnorderedSet() UnorderedSet {
	s := &unorderedSet{
		v: make(map[interface{}]struct{}),
	}
	return s
}

// Insert inserts item into the set.
func (s *unorderedSet) Insert(item interface{}) {
	s.v[item] = emptyKey
}

// Erase deletes item from the set. It is a no-op if item does not exist.
func (s *unorderedSet) Erase(item interface{}) {
	delete(s.v, item)
}

// Contains checks for the existence of a given item in the set.
func (s *unorderedSet) Contains(item interface{}) bool {
	_, ok := s.v[item]
	return ok
}

// Size computes the number of items in the set.
func (s *unorderedSet) Size() int {
	return len(s.v)
}

// Clear deletes all the items in the set.
func (s *unorderedSet) Clear() {
	s.v = make(map[interface{}]struct{})
}

// IsEmpty checks if the set is empty.
func (s *unorderedSet) IsEmpty() bool {
	return len(s.v) == 0
}

// ForEach applies function to each member of the set once.
func (s *unorderedSet) ForEach(f func(interface{})) {
	for vv := range s.v {
		f(vv)
	}
}

// Union computes the union of the two sets as return value.
func (s *unorderedSet) Union(set UnorderedSet) UnorderedSet {
	s.ForEach(func(vv interface{}) {
		set.Insert(vv)
	})
	return set
}

// Intersect computes the intersection of the two sets as return value.
func (s *unorderedSet) Intersect(set UnorderedSet) UnorderedSet {
	set.ForEach(func(vv interface{}) {
		if !s.Contains(vv) {
			set.Erase(vv)
		}
	})
	return set
}

func (s *unorderedSet) ToSlice() []interface{} {
	keys := make([]interface{}, 0, len(s.v))
	for elem := range s.v {
		keys = append(keys, elem)
	}
	return keys
}
