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

import "strings"

// CreateEndpoints creates a list of endpoints given the right scheme
func CreateEndpoints(addrs []string, scheme string) (entries []string) {
	for _, addr := range addrs {
		entries = append(entries, scheme+"://"+addr)
	}
	return entries
}

// Normalize the key for each store to the form:
//
//     /path/to/key
//
func Normalize(key string) string {
	return "/" + join(SplitKey(key))
}

// GetDirectory gets the full directory part of
// the key to the form:
//
//     /path/to/
//
func GetDirectory(key string) string {
	parts := SplitKey(key)
	parts = parts[:len(parts)-1]
	return "/" + join(parts)
}

// SplitKey splits the key to extract path informations
func SplitKey(key string) (path []string) {
	if strings.Contains(key, "/") {
		path = strings.Split(key, "/")
	} else {
		path = []string{key}
	}
	return path
}

// join the path parts with '/'
func join(parts []string) string {
	return strings.Join(parts, "/")
}

// ----------------------------------------------------------------------------

type keyMaker func(identifier string, args ...string) string

// NewKey given separator, package name, and tags to generate the key
func NewKey(separator string, pkgName string, tags ...string) keyMaker {
	prefix := append([]string{pkgName}, tags...)
	return func(identifier string, args ...string) string {
		return strings.Join(append(prefix, append([]string{identifier}, args...)...), separator)
	}
}
