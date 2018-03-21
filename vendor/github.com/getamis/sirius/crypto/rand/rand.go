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

package rand

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"hash"
)

const (
	defaultRandomBytesLength = 128
)

type Generator interface {
	Key() []byte
	KeyEncoded() string
}

func New(opts ...Option) Generator {
	generator := &generator{
		hasher:  sha256.New,
		encoder: hex.EncodeToString,
		len:     defaultRandomBytesLength,
	}

	for _, opt := range opts {
		opt(generator)
	}

	return generator
}

// ----------------------------------------------------------------------------

type generator struct {
	hasher  func() hash.Hash
	encoder func(src []byte) string
	len     int
}

func (gen *generator) Key() []byte {
	b := make([]byte, gen.len)
	_, err := rand.Read(b)
	if err != nil {
		return nil
	}

	hash := gen.hasher()
	_, err = hash.Write(b)
	if err != nil {
		return nil
	}

	return hash.Sum(nil)
}

func (gen *generator) KeyEncoded() string {
	b := gen.Key()
	if b == nil {
		return ""
	}
	return gen.encoder(b)
}
