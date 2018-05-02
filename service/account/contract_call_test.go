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
	"fmt"
	"math"
	"math/big"
	"math/rand"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	indexerCommon "github.com/maichain/eth-indexer/common"
	"github.com/maichain/eth-indexer/model"
	"github.com/maichain/eth-indexer/service/account/contracts"
	"github.com/maichain/eth-indexer/service/account/contracts/backends"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Call Test", func() {
	var auth *bind.TransactOpts
	var contract *contracts.MithrilToken
	var contractAddr common.Address
	var sim *backends.SimulatedBackend

	BeforeEach(func() {
		// pre-defined account
		key, _ := crypto.GenerateKey()
		auth = bind.NewKeyedTransactor(key)

		alloc := make(core.GenesisAlloc)
		alloc[auth.From] = core.GenesisAccount{Balance: big.NewInt(100000000)}
		sim = backends.NewSimulatedBackend(alloc)

		// Deploy Mithril token contract
		var err error
		contractAddr, _, contract, err = contracts.DeployMithrilToken(auth, sim)
		Expect(contract).ShouldNot(BeNil())
		Expect(err).Should(BeNil())
		sim.Commit()
	})

	It("BalanceOf", func() {
		By("init token supply")
		tx, err := contract.Init(auth, big.NewInt(math.MaxInt64), auth.From, auth.From)
		type account struct {
			address common.Address
			balance *big.Int
		}
		Expect(tx).ShouldNot(BeNil())
		Expect(err).Should(BeNil())
		sim.Commit()

		By("transfer token to accounts")
		var accounts []*account
		for i := 0; i < 100; i++ {
			acc := &account{
				address: common.HexToAddress(getFakeAddress()),
				balance: big.NewInt(int64(rand.Uint32())),
			}
			tx, err := contract.Transfer(auth, acc.address, acc.balance)
			Expect(tx).ShouldNot(BeNil())
			Expect(err).Should(BeNil())
			accounts = append(accounts, acc)
			sim.Commit()
		}

		By("get current state db")
		stateDB, err := sim.Blockchain().State()
		Expect(stateDB).ShouldNot(BeNil())
		Expect(err).Should(BeNil())

		By("ensure all account token balance are expected")
		for _, account := range accounts {
			result, err := BalanceOf(stateDB, contractAddr, account.address)
			Expect(err).Should(BeNil())
			Expect(account.balance).Should(Equal(result))
		}

		By("find the contract code and data")
		dump := stateDB.RawDump()
		var code *model.ContractCode
		var data *model.Contract
		for addrStr, account := range dump.Accounts {
			if contractAddr == common.HexToAddress(addrStr) {
				code, data, err = indexerCommon.Contract(sim.Blockchain().CurrentBlock().Number().Int64(), addrStr, account)
				Expect(err).Should(BeNil())
				break
			}
		}
		Expect(code).ShouldNot(BeNil())
		Expect(data).ShouldNot(BeNil())

		By("ensure all account token balance are expected based on contract code and data")
		db := &contractDB{
			code:    code,
			account: data,
		}
		for _, account := range accounts {
			result, err := BalanceOf(db, contractAddr, account.address)
			Expect(err).Should(BeNil())
			Expect(account.balance).Should(Equal(result))
		}
	})
})

// ----------------------------------------------------------------------------
var letters = []rune("abcdef0123456789")

func getRandomString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func getFakeAddress() string {
	return fmt.Sprintf("0x%s", getRandomString(40))
}
