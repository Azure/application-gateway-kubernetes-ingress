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
