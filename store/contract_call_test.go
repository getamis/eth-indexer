// Copyright 2018 The eth-indexer Authors
// This file is part of the eth-indexer library.
//
// The eth-indexer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The eth-indexer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the eth-indexer library. If not, see <http://www.gnu.org/licenses/>.

package store

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"math/rand"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/params"
	"github.com/getamis/eth-indexer/contracts"
	"github.com/getamis/eth-indexer/contracts/backends"
	"github.com/getamis/eth-indexer/model"
	accountMocks "github.com/getamis/eth-indexer/store/account/mocks"
	"github.com/stretchr/testify/mock"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Call Test", func() {
	var auth *bind.TransactOpts
	var contract *contracts.MithrilToken
	var contractAddr common.Address
	var sim *backends.SimulatedBackend
	var mockAccountStore *accountMocks.Store
	BeforeEach(func() {
		mockAccountStore = new(accountMocks.Store)
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

	AfterEach(func() {
		mockAccountStore.AssertExpectations(GinkgoT())
	})

	It("BalanceOf", func() {
		By("init token supply")
		tx, err := contract.Init(auth, big.NewInt(math.MaxInt64), auth.From, auth.From)
		type account struct {
			address      common.Address
			balance      *big.Int
			dirtyStateDB map[string]state.DirtyDumpAccount
		}
		Expect(tx).ShouldNot(BeNil())
		Expect(err).Should(BeNil())
		sim.Commit()

		By("transfer token to accounts")
		accounts := make(map[uint64]*account)
		for i := 0; i < 100; i++ {
			acc := &account{
				address: common.HexToAddress(getFakeAddress()),
				balance: big.NewInt(int64(rand.Uint32())),
			}
			tx, err := contract.Transfer(auth, acc.address, acc.balance)
			Expect(tx).ShouldNot(BeNil())
			Expect(err).Should(BeNil())
			accounts[sim.Blockchain().CurrentBlock().NumberU64()+1] = acc
			sim.Commit()
		}

		By("get current state db")
		stateDB, err := sim.Blockchain().State()
		Expect(stateDB).ShouldNot(BeNil())

		By("ensure all account token balance are expected")
		for _, account := range accounts {
			result, err := BalanceOf(stateDB, contractAddr, account.address)
			Expect(err).Should(BeNil())
			Expect(account.balance).Should(Equal(result))
		}

		By("get dirty storage")
		for blockNumber, account := range accounts {
			dump, err := eth.GetDirtyStorage(params.AllEthashProtocolChanges, sim.Blockchain(), blockNumber)
			account.dirtyStateDB = dump.Accounts
			Expect(err).Should(BeNil())
			accounts[blockNumber] = account
		}

		By("find the contract code")
		code, err := sim.CodeAt(context.Background(), contractAddr, nil)
		Expect(code).ShouldNot(BeNil())
		Expect(err).Should(BeNil())

		contractCode := &model.ERC20{
			Address: contractAddr.Bytes(),
			Code:    code,
		}

		By("mock account store")
		mockAccountStore.On("FindERC20Storage", mock.AnythingOfType("common.Address"), mock.AnythingOfType("common.Hash"), mock.AnythingOfType("int64")).Return(func(address common.Address, key common.Hash, blockNr int64) *model.ERC20Storage {
			s, _ := accounts[uint64(blockNr)].dirtyStateDB[common.Bytes2Hex(address.Bytes())]
			kayHash := common.Bytes2Hex(key.Bytes())
			value, _ := s.Storage[kayHash]
			return &model.ERC20Storage{
				Address:     address.Bytes(),
				BlockNumber: blockNr,
				Key:         key.Bytes(),
				Value:       common.Hex2Bytes(value),
			}
		}, nil)

		By("ensure all account token balance are expected based on contract code and data")
		for blockNumber, account := range accounts {
			db := &contractDB{
				blockNumber:  int64(blockNumber),
				code:         contractCode,
				accountStore: mockAccountStore,
				account: &model.Account{
					Address: contractAddr.Bytes(),
				},
			}
			result, err := BalanceOf(db, contractAddr, account.address)
			Expect(err).Should(BeNil())
			Expect(db.err).Should(BeNil())
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
