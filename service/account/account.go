// Copyright © 2018 AMIS Technologies
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package account

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// Implement vm.ContractRef
type account struct {
	address common.Address
}

func (account *account) ReturnGas(*big.Int, *big.Int) {
	// Do nothing
}
func (account *account) Address() common.Address {
	return account.address
}
func (account *account) Value() *big.Int {
	return common.Big0
}

func (account *account) SetCode(common.Hash, []byte) {
	// Do nothing
}
func (account *account) ForEachStorage(callback func(key, value common.Hash) bool) {
	// Do nothing
}
