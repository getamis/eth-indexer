// Copyright 2018 AMIS Technologies
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

package model

// StateBlock represents the state is at the given block
type StateBlock struct {
	Number int64
}

// Account represents the state of externally owned accounts in Ethereum at given block
type Account struct {
	BlockNumber int64
	Address     []byte
	Balance     string
	Nonce       int64
}

// ContractCode represents the contract code
type ContractCode struct {
	Address []byte
	Hash    []byte
	Code    string
}

// Contract represents the state of contract accounts in Ethereum at given block
type Contract struct {
	BlockNumber int64
	Address     []byte
	Balance     string
	Nonce       int64
	Root        []byte
	Storage     []byte
}
