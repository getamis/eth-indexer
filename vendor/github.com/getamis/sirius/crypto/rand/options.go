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
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"hash"
	"math/rand"

	"github.com/satori/go.uuid"
)

type Option func(*generator)

// ----------------------------------------------------------------------------
// Random bytes length

func Randomness(len int) Option {
	return func(g *generator) {
		g.len = len
	}
}

// ----------------------------------------------------------------------------
// Hashing Policy

func Hash(hashFn func() hash.Hash) Option {
	return func(g *generator) {
		g.hasher = hashFn
	}
}

func Sha1Hash() Option {
	return Hash(sha1.New)
}

func Sha256Hash() Option {
	return Hash(sha256.New)
}

func Sha512Hash() Option {
	return Hash(sha512.New)
}

// ----------------------------------------------------------------------------
// Encoding Policy

func Encoder(encoder func(src []byte) string) Option {
	return func(h *generator) {
		h.encoder = encoder
	}
}

func Base64Encoder() Option {
	return Encoder(base64.StdEncoding.EncodeToString)
}

func HexEncoder() Option {
	return Encoder(hex.EncodeToString)
}

func UUIDEncoder() Option {
	return Encoder(func(src []byte) string {
		b := make([]byte, 16)
		_, _ = rand.Read(b)
		copy(b, src)
		uuid, _ := uuid.FromBytes(b)
		return uuid.String()
	})
}
