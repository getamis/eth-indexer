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
package store

import (
	"context"
	"errors"
	"math"
	"math/big"
	"math/rand"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	indexerCommon "github.com/maichain/eth-indexer/common"
	"github.com/maichain/eth-indexer/contracts"
	"github.com/maichain/eth-indexer/contracts/backends"
	"github.com/maichain/eth-indexer/model"
	accountMock "github.com/maichain/eth-indexer/store/account/mocks"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DB ERC 20 Test", func() {
	var auth *bind.TransactOpts
	var contract *contracts.MithrilToken
	var contractAddr common.Address
	var sim *backends.SimulatedBackend
	var db *contractDB

	var fundedAddress common.Address
	var fundedBalance *big.Int

	BeforeEach(func() {
		// pre-defined account
		key, _ := crypto.GenerateKey()
		auth = bind.NewKeyedTransactor(key)

		alloc := make(core.GenesisAlloc)
		alloc[auth.From] = core.GenesisAccount{Balance: big.NewInt(100000000000000)}
		sim = backends.NewSimulatedBackend(alloc)

		// Deploy Mithril token contract
		var err error
		contractAddr, _, contract, err = contracts.DeployMithrilToken(auth, sim)
		Expect(contract).ShouldNot(BeNil())
		Expect(err).Should(BeNil())
		sim.Commit()

		By("init token supply")
		tx, err := contract.Init(auth, big.NewInt(math.MaxInt64), auth.From, auth.From)
		type account struct {
			address common.Address
			balance *big.Int
		}
		Expect(tx).ShouldNot(BeNil())
		Expect(err).Should(BeNil())
		sim.Commit()

		By("fund some token to an address")
		fundedAddress = common.HexToAddress(getFakeAddress())
		fundedBalance = big.NewInt(int64(rand.Uint32()))
		_, err = contract.Transfer(auth, fundedAddress, fundedBalance)
		Expect(err).Should(BeNil())
		sim.Commit()

		By("get current state db")
		stateDB, err := sim.Blockchain().State()
		Expect(stateDB).ShouldNot(BeNil())
		Expect(err).Should(BeNil())

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

		db = &contractDB{
			code:    code,
			account: data,
		}
	})

	Context("Contract DB", func() {
		notSelfAddr := common.HexToAddress(getFakeAddress())
		It("self()", func() {
			Expect(db.self(contractAddr)).Should(BeTrue())
			Expect(db.err).Should(BeNil())
			Expect(db.self(notSelfAddr)).Should(BeFalse())
			Expect(db.err).Should(BeNil())
		})
		It("mustBeSelf()", func() {
			Expect(db.mustBeSelf(contractAddr)).Should(BeTrue())
			Expect(db.err).Should(BeNil())
			Expect(db.mustBeSelf(notSelfAddr)).Should(BeFalse())
			Expect(db.err).Should(Equal(ErrNotSelf))
		})
		It("Exist()", func() {
			Expect(db.Exist(contractAddr)).Should(BeTrue())
			Expect(db.err).Should(BeNil())
			Expect(db.Exist(notSelfAddr)).Should(BeFalse())
			Expect(db.err).Should(BeNil())
		})
		It("Empty()", func() {
			Expect(db.Empty(contractAddr)).Should(BeFalse())
			Expect(db.err).Should(BeNil())
			Expect(db.Empty(notSelfAddr)).Should(BeTrue())
			Expect(db.err).Should(BeNil())
		})
		It("GetBalance()", func() {
			// Currently, the balance of contract is zero because we cannot put ether in this contract
			balance, ok := new(big.Int).SetString(db.account.Balance, 10)
			Expect(ok).Should(BeTrue())
			Expect(db.GetBalance(contractAddr)).Should(Equal(balance))
			Expect(db.err).Should(BeNil())
			Expect(db.GetBalance(notSelfAddr).Int64()).Should(BeZero())
			Expect(db.err).Should(Equal(ErrNotSelf))
		})
		It("GetNonce()", func() {
			Expect(db.GetNonce(contractAddr)).Should(Equal(uint64(db.account.Nonce)))
			Expect(db.err).Should(BeNil())
			Expect(db.GetNonce(notSelfAddr)).Should(BeZero())
			Expect(db.err).Should(Equal(ErrNotSelf))
		})
		It("GetCodeHash()", func() {
			Expect(db.GetCodeHash(contractAddr)).Should(Equal(common.BytesToHash(db.code.Hash)))
			Expect(db.err).Should(BeNil())
			Expect(db.GetCodeHash(notSelfAddr)).Should(Equal(common.Hash{}))
			Expect(db.err).Should(Equal(ErrNotSelf))
		})
		It("GetCode()", func() {
			Expect(db.GetCode(contractAddr)).Should(Equal(common.Hex2Bytes(db.code.Code)))
			Expect(db.err).Should(BeNil())
			Expect(db.GetCode(notSelfAddr)).Should(Equal([]byte{}))
			Expect(db.err).Should(Equal(ErrNotSelf))
		})
		It("GetCodeSize()", func() {
			Expect(db.GetCodeSize(contractAddr)).Should(Equal(len(db.GetCode(contractAddr))))
			Expect(db.err).Should(BeNil())
			Expect(db.GetCodeSize(notSelfAddr)).Should(BeZero())
			Expect(db.err).Should(Equal(ErrNotSelf))
		})
		It("GetState()", func() {
			randomHash := common.HexToHash(getRandomString(64))
			Expect(db.GetState(contractAddr, randomHash)).Should(Equal(common.Hash{}))
			Expect(db.err).Should(BeNil())
			Expect(db.GetState(notSelfAddr, randomHash)).Should(Equal(common.Hash{}))
			Expect(db.err).Should(Equal(ErrNotSelf))
		})
	})

	Context("GetERC20Balance()", func() {
		var mockAccountStore *accountMock.Store
		var manager *serviceManager
		blockNumber := int64(10)
		stateBlock := &model.StateBlock{
			Number: 100,
		}

		BeforeEach(func() {
			mockAccountStore = new(accountMock.Store)
			manager = &serviceManager{
				accountStore: mockAccountStore,
			}
		})

		AfterEach(func() {
			mockAccountStore.AssertExpectations(GinkgoT())
		})

		Context("with valid parameters", func() {
			Context("latest block", func() {
				It("funded address", func() {
					mockAccountStore.On("FindContractCode", contractAddr).Return(db.code, nil).Once()
					mockAccountStore.On("LastStateBlock").Return(stateBlock, nil).Once()
					mockAccountStore.On("FindContract", contractAddr, stateBlock.Number).Return(db.account, nil).Once()
					expBalance, expNumber, err := manager.GetERC20Balance(context.Background(), contractAddr, fundedAddress, -1)
					Expect(err).Should(BeNil())
					Expect(expBalance).Should(Equal(fundedBalance))
					Expect(expNumber.Int64()).Should(Equal(stateBlock.Number))
				})
				It("non-funded address", func() {
					otherAddr := common.HexToAddress(getFakeAddress())
					mockAccountStore.On("FindContractCode", contractAddr).Return(db.code, nil).Once()
					mockAccountStore.On("LastStateBlock").Return(stateBlock, nil).Once()
					mockAccountStore.On("FindContract", contractAddr, stateBlock.Number).Return(db.account, nil).Once()
					expBalance, expNumber, err := manager.GetERC20Balance(context.Background(), contractAddr, otherAddr, -1)
					Expect(err).Should(BeNil())
					Expect(expBalance.Int64()).Should(BeZero())
					Expect(expNumber.Int64()).Should(Equal(stateBlock.Number))
				})
			})
			Context("non latest block", func() {
				It("funded address", func() {
					mockAccountStore.On("FindContractCode", contractAddr).Return(db.code, nil).Once()
					mockAccountStore.On("FindStateBlock", blockNumber).Return(stateBlock, nil).Once()
					mockAccountStore.On("FindContract", contractAddr, stateBlock.Number).Return(db.account, nil).Once()
					expBalance, expNumber, err := manager.GetERC20Balance(context.Background(), contractAddr, fundedAddress, blockNumber)
					Expect(err).Should(BeNil())
					Expect(expBalance).Should(Equal(fundedBalance))
					Expect(expNumber.Int64()).Should(Equal(stateBlock.Number))
				})
				It("non-funded address", func() {
					otherAddr := common.HexToAddress(getFakeAddress())
					mockAccountStore.On("FindContractCode", contractAddr).Return(db.code, nil).Once()
					mockAccountStore.On("FindStateBlock", blockNumber).Return(stateBlock, nil).Once()
					mockAccountStore.On("FindContract", contractAddr, stateBlock.Number).Return(db.account, nil).Once()
					expBalance, expNumber, err := manager.GetERC20Balance(context.Background(), contractAddr, otherAddr, blockNumber)
					Expect(err).Should(BeNil())
					Expect(expBalance.Int64()).Should(BeZero())
					Expect(expNumber.Int64()).Should(Equal(stateBlock.Number))
				})
			})
		})

		Context("with invalid parameters", func() {
			unknownErr := errors.New("unknown error")
			It("failed to find contract address", func() {
				mockAccountStore.On("FindContractCode", contractAddr).Return(db.code, nil).Once()
				mockAccountStore.On("FindStateBlock", blockNumber).Return(stateBlock, nil).Once()
				mockAccountStore.On("FindContract", contractAddr, stateBlock.Number).Return(nil, unknownErr).Once()
				expBalance, expNumber, err := manager.GetERC20Balance(context.Background(), contractAddr, fundedAddress, blockNumber)
				Expect(err).Should(Equal(unknownErr))
				Expect(expBalance).Should(BeNil())
				Expect(expNumber).Should(BeNil())
			})
			It("failed to find state block", func() {
				mockAccountStore.On("FindContractCode", contractAddr).Return(db.code, nil).Once()
				mockAccountStore.On("FindStateBlock", blockNumber).Return(nil, unknownErr).Once()
				expBalance, expNumber, err := manager.GetERC20Balance(context.Background(), contractAddr, fundedAddress, blockNumber)
				Expect(err).Should(Equal(unknownErr))
				Expect(expBalance).Should(BeNil())
				Expect(expNumber).Should(BeNil())
			})
			It("failed to find latest state block", func() {
				mockAccountStore.On("FindContractCode", contractAddr).Return(db.code, nil).Once()
				mockAccountStore.On("LastStateBlock").Return(nil, unknownErr).Once()
				expBalance, expNumber, err := manager.GetERC20Balance(context.Background(), contractAddr, fundedAddress, -1)
				Expect(err).Should(Equal(unknownErr))
				Expect(expBalance).Should(BeNil())
				Expect(expNumber).Should(BeNil())
			})
			It("failed to find state block", func() {
				mockAccountStore.On("FindContractCode", contractAddr).Return(db.code, unknownErr).Once()
				expBalance, expNumber, err := manager.GetERC20Balance(context.Background(), contractAddr, fundedAddress, blockNumber)
				Expect(err).Should(Equal(unknownErr))
				Expect(expBalance).Should(BeNil())
				Expect(expNumber).Should(BeNil())
			})
		})
	})
})
