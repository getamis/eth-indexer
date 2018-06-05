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
	"errors"
	"math"
	"math/big"
	"math/rand"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/params"
	indexerCommon "github.com/getamis/eth-indexer/common"
	"github.com/getamis/eth-indexer/contracts"
	"github.com/getamis/eth-indexer/contracts/backends"
	"github.com/getamis/eth-indexer/model"
	accountMock "github.com/getamis/eth-indexer/store/account/mocks"
	hdrMock "github.com/getamis/eth-indexer/store/block_header/mocks"
	"github.com/jinzhu/gorm"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("DB ERC 20 Test", func() {
	var auth *bind.TransactOpts
	var contract *contracts.MithrilToken
	var contractAddr common.Address
	var sim *backends.SimulatedBackend
	var db *contractDB
	var mockAccountStore *accountMock.Store
	var storages map[string]*model.ERC20Storage
	var fundedAddress common.Address
	var fundedBalance *big.Int

	BeforeEach(func() {
		mockAccountStore = new(accountMock.Store)

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

		By("get dirty storage")
		now := sim.Blockchain().CurrentBlock().NumberU64()
		dump, err := eth.GetDirtyStorage(params.AllEthashProtocolChanges, sim.Blockchain(), now)
		Expect(err).Should(BeNil())

		By("find the contract code and data")
		blockNumber := int64(sim.Blockchain().CurrentBlock().NumberU64())
		var code *model.ERC20
		storages = make(map[string]*model.ERC20Storage)
		for addrStr, account := range dump.Accounts {
			if contractAddr == common.HexToAddress(addrStr) {
				c, _ := sim.CodeAt(context.Background(), contractAddr, nil)
				code = &model.ERC20{
					Address: contractAddr.Bytes(),
					Code:    c,
				}

				for k, v := range account.Storage {
					storages[k] = &model.ERC20Storage{
						BlockNumber: blockNumber,
						Address:     contractAddr.Bytes(),
						Key:         common.Hex2Bytes(k),
						Value:       common.Hex2Bytes(v),
					}
				}
				break
			}
		}
		Expect(code).ShouldNot(BeNil())

		db = &contractDB{
			blockNumber:  blockNumber,
			code:         code,
			accountStore: mockAccountStore,
			account: &model.Account{
				Address: contractAddr.Bytes(),
				Balance: "0",
			},
		}
	})
	AfterEach(func() {
		mockAccountStore.AssertExpectations(GinkgoT())
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
			// Currently, we cannot send ether to Mirthril contract because its contract implementation.
			// The balance of contract is always zero.
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
			Expect(db.GetCodeHash(contractAddr)).Should(Equal(crypto.Keccak256Hash(db.code.Code)))
			Expect(db.err).Should(BeNil())
			Expect(db.GetCodeHash(notSelfAddr)).Should(Equal(common.Hash{}))
			Expect(db.err).Should(Equal(ErrNotSelf))
		})
		It("GetCode()", func() {
			Expect(db.GetCode(contractAddr)).Should(Equal(db.code.Code))
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
			mockFindREC20Storage(mockAccountStore, storages)
			Expect(db.GetState(contractAddr, randomHash)).Should(Equal(common.Hash{}))
			Expect(db.err).Should(BeNil())
			Expect(db.GetState(notSelfAddr, randomHash)).Should(Equal(common.Hash{}))
			Expect(db.err).Should(Equal(ErrNotSelf))
		})
	})

	Context("GetERC20Balance()", func() {
		var mockAccountStore *accountMock.Store
		var mockHdrStore *hdrMock.Store
		var manager *serviceManager
		blockNumber := int64(10)
		header := &model.Header{
			Number: 100,
		}

		BeforeEach(func() {
			mockAccountStore = new(accountMock.Store)
			mockHdrStore = new(hdrMock.Store)
			manager = &serviceManager{
				accountStore:     mockAccountStore,
				blockHeaderStore: mockHdrStore,
			}
		})

		Context("with valid parameters", func() {
			Context("latest block", func() {
				It("funded address", func() {
					mockFindREC20Storage(mockAccountStore, storages)
					mockAccountStore.On("FindERC20", contractAddr).Return(db.code, nil).Once()
					mockHdrStore.On("FindLatestBlock").Return(header, nil).Once()
					mockAccountStore.On("FindAccount", contractAddr, header.Number).Return(db.account, nil).Once()
					expBalance, expNumber, err := manager.GetERC20Balance(context.Background(), contractAddr, fundedAddress, -1)
					Expect(err).Should(BeNil())
					Expect(expBalance.String()).Should(Equal(fundedBalance.String()))
					Expect(expNumber.Int64()).Should(Equal(header.Number))
				})
				It("non-funded address", func() {
					otherAddr := common.HexToAddress(getFakeAddress())
					mockFindREC20Storage(mockAccountStore, storages)
					mockAccountStore.On("FindERC20", contractAddr).Return(db.code, nil).Once()
					mockHdrStore.On("FindLatestBlock").Return(header, nil).Once()
					mockAccountStore.On("FindAccount", contractAddr, header.Number).Return(db.account, nil).Once()
					expBalance, expNumber, err := manager.GetERC20Balance(context.Background(), contractAddr, otherAddr, -1)
					Expect(err).Should(BeNil())
					Expect(expBalance.IntPart()).Should(BeZero())
					Expect(expNumber.Int64()).Should(Equal(header.Number))
				})
			})
			Context("non latest block", func() {
				It("funded address", func() {
					mockFindREC20Storage(mockAccountStore, storages)
					mockAccountStore.On("FindERC20", contractAddr).Return(db.code, nil).Once()
					mockHdrStore.On("FindBlockByNumber", blockNumber).Return(header, nil).Once()
					mockAccountStore.On("FindAccount", contractAddr, header.Number).Return(db.account, nil).Once()
					expBalance, expNumber, err := manager.GetERC20Balance(context.Background(), contractAddr, fundedAddress, blockNumber)
					Expect(err).Should(BeNil())
					Expect(expBalance.String()).Should(Equal(fundedBalance.String()))
					Expect(expNumber.Int64()).Should(Equal(header.Number))
				})
				It("non-funded address", func() {
					mockFindREC20Storage(mockAccountStore, storages)
					otherAddr := common.HexToAddress(getFakeAddress())
					mockAccountStore.On("FindERC20", contractAddr).Return(db.code, nil).Once()
					mockHdrStore.On("FindBlockByNumber", blockNumber).Return(header, nil).Once()
					mockAccountStore.On("FindAccount", contractAddr, header.Number).Return(db.account, nil).Once()
					expBalance, expNumber, err := manager.GetERC20Balance(context.Background(), contractAddr, otherAddr, blockNumber)
					Expect(err).Should(BeNil())
					Expect(expBalance.IntPart()).Should(BeZero())
					Expect(expNumber.Int64()).Should(Equal(header.Number))
				})
			})
		})

		Context("with invalid parameters", func() {
			unknownErr := errors.New("unknown error")
			It("failed to execute state db", func() {
				mockAccountStore.On("FindERC20", contractAddr).Return(db.code, nil).Once()
				mockHdrStore.On("FindBlockByNumber", blockNumber).Return(header, nil).Once()
				mockAccountStore.On("FindAccount", contractAddr, header.Number).Return(db.account, nil).Once()
				mockAccountStore.On("FindERC20Storage", mock.AnythingOfType("common.Address"), mock.AnythingOfType("common.Hash"), mock.AnythingOfType("int64")).Return(nil, unknownErr).Once()
				expBalance, expNumber, err := manager.GetERC20Balance(context.Background(), contractAddr, fundedAddress, blockNumber)
				Expect(err).Should(Equal(unknownErr))
				Expect(expBalance).Should(BeNil())
				Expect(expNumber).Should(BeNil())
			})
			It("failed to find contract address", func() {
				mockAccountStore.On("FindERC20", contractAddr).Return(db.code, nil).Once()
				mockHdrStore.On("FindBlockByNumber", blockNumber).Return(header, nil).Once()
				mockAccountStore.On("FindAccount", contractAddr, header.Number).Return(nil, unknownErr).Once()
				expBalance, expNumber, err := manager.GetERC20Balance(context.Background(), contractAddr, fundedAddress, blockNumber)
				Expect(err).Should(Equal(unknownErr))
				Expect(expBalance).Should(BeNil())
				Expect(expNumber).Should(BeNil())
			})
			It("failed to find state block", func() {
				mockAccountStore.On("FindERC20", contractAddr).Return(db.code, nil).Once()
				mockHdrStore.On("FindBlockByNumber", blockNumber).Return(nil, unknownErr).Once()
				expBalance, expNumber, err := manager.GetERC20Balance(context.Background(), contractAddr, fundedAddress, blockNumber)
				Expect(err).Should(Equal(unknownErr))
				Expect(expBalance).Should(BeNil())
				Expect(expNumber).Should(BeNil())
			})
			It("failed to find latest state block", func() {
				mockAccountStore.On("FindERC20", contractAddr).Return(db.code, nil).Once()
				mockHdrStore.On("FindLatestBlock").Return(nil, unknownErr).Once()
				expBalance, expNumber, err := manager.GetERC20Balance(context.Background(), contractAddr, fundedAddress, -1)
				Expect(err).Should(Equal(unknownErr))
				Expect(expBalance).Should(BeNil())
				Expect(expNumber).Should(BeNil())
			})
			It("failed to find state block", func() {
				mockAccountStore.On("FindERC20", contractAddr).Return(db.code, unknownErr).Once()
				expBalance, expNumber, err := manager.GetERC20Balance(context.Background(), contractAddr, fundedAddress, blockNumber)
				Expect(err).Should(Equal(unknownErr))
				Expect(expBalance).Should(BeNil())
				Expect(expNumber).Should(BeNil())
			})
		})
	})
})

func mockFindREC20Storage(mockAccountStore *accountMock.Store, storages map[string]*model.ERC20Storage) {
	mockAccountStore.On("FindERC20Storage", mock.AnythingOfType("common.Address"), mock.AnythingOfType("common.Hash"), mock.AnythingOfType("int64")).Return(
		func(address common.Address, key common.Hash, blockNr int64) *model.ERC20Storage {
			v, ok := storages[indexerCommon.HashHex(key)]
			if ok {
				return v
			}
			return nil
		}, func(address common.Address, key common.Hash, blockNr int64) error {
			_, ok := storages[indexerCommon.HashHex(key)]
			if ok {
				return nil
			}
			return gorm.ErrRecordNotFound
		}).Once()
}
