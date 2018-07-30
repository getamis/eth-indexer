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
	"math/big"
	"strconv"

	gethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/getamis/eth-indexer/client/mocks"
	"github.com/getamis/eth-indexer/model"
	"github.com/getamis/eth-indexer/store/account"
	subsStore "github.com/getamis/eth-indexer/store/subscription"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("New ERC20 Test", func() {
	var (
		blocks   []*types.Block
		receipts [][]*types.Receipt
		events   [][]*types.TransferLog
		manager  Manager

		mockBalancer *mocks.Balancer
	)

	// ERC20 contract
	erc20s := []*model.ERC20{
		{
			Address:     gethCommon.HexToAddress("1234567893").Bytes(),
			BlockNumber: 1,
		},
		{
			Address:     gethCommon.HexToAddress("1234567894").Bytes(),
			BlockNumber: 0,
		},
	}

	BeforeEach(func() {
		mockBalancer = new(mocks.Balancer)
	})

	AfterEach(func() {
		mockBalancer.AssertExpectations(GinkgoT())

		// Clean all data
		db.Delete(&model.Header{})
		db.Delete(&model.Transaction{})
		db.Delete(&model.Receipt{})
		db.Delete(&model.Account{
			ContractAddress: model.ETHBytes,
		})
		db.Delete(&model.ERC20{})
		db.Delete(&model.TotalBalance{})
		db.Delete(&model.Subscription{})
		db.Delete(&model.Reorg{})
		db.DropTable(model.Transfer{
			Address: erc20s[0].Address,
		})
		db.DropTable(model.Transfer{
			Address: erc20s[1].Address,
		})
		db.DropTable(model.Account{
			ContractAddress: erc20s[0].Address,
		})
		db.DropTable(model.Account{
			ContractAddress: erc20s[1].Address,
		})
	})

	It("should be successful", func() {
		By("Normal blocks comes")
		subLimit = 1
		// subscriptions
		subs := []*model.Subscription{
			{
				BlockNumber: 90,
				Group:       1,
				Address:     acc0Addr.Bytes(),
			},
			{
				BlockNumber: 0,
				Group:       1,
				Address:     acc1Addr.Bytes(),
			},
			{
				BlockNumber: 0,
				Group:       2,
				Address:     acc2Addr.Bytes(),
			},
		}
		// Insert subscription
		subStore := subsStore.NewWithDB(db)
		duplicated, err := subStore.BatchInsert(subs)
		Expect(err).Should(BeNil())
		Expect(len(duplicated)).Should(Equal(0))

		// Insert ERC20 total balance
		err = subStore.InsertTotalBalance(&model.TotalBalance{
			Token:       erc20s[0].Address,
			BlockNumber: 99,
			Group:       1,
			Balance:     "2000",
			TxFee:       "0",
		})
		Expect(err).Should(BeNil())

		// Insert ether total balance
		err = subStore.InsertTotalBalance(&model.TotalBalance{
			Token:       model.ETHBytes,
			BlockNumber: 99,
			Group:       1,
			Balance:     "1000",
			TxFee:       "0",
		})
		Expect(err).Should(BeNil())

		blocks = []*types.Block{
			types.NewBlockWithHeader(&types.Header{
				Number:   big.NewInt(100),
				Coinbase: unknownRecipientAddr,
			}),
			types.NewBlockWithHeader(&types.Header{
				Number:   big.NewInt(101),
				Coinbase: unknownRecipientAddr,
			}),
			types.NewBlockWithHeader(&types.Header{
				Number:   big.NewInt(102),
				Coinbase: unknownRecipientAddr,
			}),
		}
		receipts = [][]*types.Receipt{{}, {}, {}}
		events = [][]*types.TransferLog{{}, {}, {}}

		ctx := context.Background()
		manager = NewManager(db)

		for _, erc20 := range erc20s {
			err = manager.InsertERC20(erc20)
			Expect(err).Should(BeNil())
		}

		acctStore := account.NewWithDB(db)
		// Insert previous ERC20 balance for the old subscriptions
		err = acctStore.InsertAccount(&model.Account{
			ContractAddress: erc20s[0].Address,
			BlockNumber:     99,
			Address:         subs[0].Address,
			Balance:         "2000",
		})
		Expect(err).Should(BeNil())

		err = manager.Init(mockBalancer)
		Expect(err).Should(BeNil())

		// For the 100 block
		mockBalancer.On("BalanceOf", ctx, big.NewInt(100), map[gethCommon.Address]map[gethCommon.Address]struct{}{
			model.ETHAddress: {
				gethCommon.BytesToAddress(subs[1].Address): struct{}{},
				gethCommon.BytesToAddress(subs[2].Address): struct{}{},
			},
			gethCommon.BytesToAddress(erc20s[0].Address): {
				gethCommon.BytesToAddress(subs[1].Address): struct{}{},
				gethCommon.BytesToAddress(subs[2].Address): struct{}{},
			},
		}).Return(map[gethCommon.Address]map[gethCommon.Address]*big.Int{
			model.ETHAddress: {
				gethCommon.BytesToAddress(subs[1].Address): big.NewInt(112),
				gethCommon.BytesToAddress(subs[2].Address): big.NewInt(113),
			},
			gethCommon.BytesToAddress(erc20s[0].Address): {
				gethCommon.BytesToAddress(subs[1].Address): big.NewInt(212),
				gethCommon.BytesToAddress(subs[2].Address): big.NewInt(213),
			},
		}, nil).Once()

		// For new token
		for i, sub := range subs {
			mockBalancer.On("BalanceOf", ctx, big.NewInt(100), map[gethCommon.Address]map[gethCommon.Address]struct{}{
				gethCommon.BytesToAddress(erc20s[1].Address): {
					gethCommon.BytesToAddress(sub.Address): struct{}{},
				},
			}).Return(map[gethCommon.Address]map[gethCommon.Address]*big.Int{
				gethCommon.BytesToAddress(erc20s[1].Address): {
					gethCommon.BytesToAddress(sub.Address): big.NewInt(310 + int64(i)),
				},
			}, nil).Once()
		}

		err = manager.UpdateBlocks(ctx, blocks, receipts, events, nil)
		Expect(err).Should(BeNil())

		// block 100
		t0_1_100, err := subStore.FindTotalBalance(100, gethCommon.BytesToAddress(erc20s[0].Address), 1)
		Expect(err).Should(BeNil())
		Expect(t0_1_100.Balance).Should(Equal("2212"))
		Expect(t0_1_100.TxFee).Should(Equal("0"))
		t0_2_100, err := subStore.FindTotalBalance(100, gethCommon.BytesToAddress(erc20s[0].Address), 2)
		Expect(err).Should(BeNil())
		Expect(t0_2_100.Balance).Should(Equal("213"))
		Expect(t0_2_100.TxFee).Should(Equal("0"))
		t1_1_100, err := subStore.FindTotalBalance(100, gethCommon.BytesToAddress(erc20s[1].Address), 1)
		Expect(err).Should(BeNil())
		Expect(t1_1_100.Balance).Should(Equal("621"))
		Expect(t1_1_100.TxFee).Should(Equal("0"))
		t1_2_100, err := subStore.FindTotalBalance(100, gethCommon.BytesToAddress(erc20s[1].Address), 2)
		Expect(err).Should(BeNil())
		Expect(t1_2_100.Balance).Should(Equal("312"))
		Expect(t1_2_100.TxFee).Should(Equal("0"))

		// block 101
		t0_1_101, err := subStore.FindTotalBalance(101, gethCommon.BytesToAddress(erc20s[0].Address), 1)
		Expect(err).Should(BeNil())
		Expect(t0_1_101).Should(Equal(t0_1_100))
		t0_2_101, err := subStore.FindTotalBalance(101, gethCommon.BytesToAddress(erc20s[0].Address), 2)
		Expect(err).Should(BeNil())
		Expect(t0_2_101).Should(Equal(t0_2_100))
		t1_1_101, err := subStore.FindTotalBalance(101, gethCommon.BytesToAddress(erc20s[1].Address), 1)
		Expect(err).Should(BeNil())
		Expect(t1_1_101).Should(Equal(t1_1_100))
		t1_2_101, err := subStore.FindTotalBalance(101, gethCommon.BytesToAddress(erc20s[1].Address), 2)
		Expect(err).Should(BeNil())
		Expect(t1_2_101).Should(Equal(t1_2_100))

		t0_1_102, err := subStore.FindTotalBalance(102, gethCommon.BytesToAddress(erc20s[0].Address), 1)
		Expect(err).Should(BeNil())
		Expect(t0_1_102).Should(Equal(t0_1_100))
		t0_2_102, err := subStore.FindTotalBalance(102, gethCommon.BytesToAddress(erc20s[0].Address), 2)
		Expect(err).Should(BeNil())
		Expect(t0_2_102).Should(Equal(t0_2_100))
		t1_1_102, err := subStore.FindTotalBalance(102, gethCommon.BytesToAddress(erc20s[1].Address), 1)
		Expect(err).Should(BeNil())
		Expect(t1_1_102).Should(Equal(t1_1_100))
		t1_2_102, err := subStore.FindTotalBalance(102, gethCommon.BytesToAddress(erc20s[1].Address), 2)
		Expect(err).Should(BeNil())
		Expect(t1_2_102).Should(Equal(t1_2_100))

		// Verify new subscriptions' block numbers updated
		res, err := subStore.FindOldSubscriptions([][]byte{subs[0].Address, subs[1].Address, subs[2].Address})
		Expect(err).Should(BeNil())
		Expect(res[0].BlockNumber).Should(Equal(int64(90)))
		Expect(res[1].BlockNumber).Should(Equal(int64(100)))
		Expect(res[2].BlockNumber).Should(Equal(int64(100)))
		erc20, err := acctStore.FindERC20(gethCommon.BytesToAddress(erc20s[1].Address))
		Expect(err).Should(BeNil())
		Expect(erc20.BlockNumber).Should(Equal(int64(101)))
		// Check the balances of new token
		for i, sub := range subs {
			acc, err := acctStore.FindAccount(gethCommon.BytesToAddress(erc20s[1].Address), gethCommon.BytesToAddress(sub.Address))
			Expect(err).Should(BeNil())
			Expect(acc.Balance).Should(Equal("31" + strconv.Itoa(i)))
		}

		By("Reorg blocks comes")
		// For the 100 block
		mockBalancer.On("BalanceOf", ctx, big.NewInt(100), map[gethCommon.Address]map[gethCommon.Address]struct{}{
			model.ETHAddress: {
				gethCommon.BytesToAddress(subs[1].Address): struct{}{},
				gethCommon.BytesToAddress(subs[2].Address): struct{}{},
			},
			gethCommon.BytesToAddress(erc20s[0].Address): {
				gethCommon.BytesToAddress(subs[1].Address): struct{}{},
				gethCommon.BytesToAddress(subs[2].Address): struct{}{},
			},
		}).Return(map[gethCommon.Address]map[gethCommon.Address]*big.Int{
			model.ETHAddress: {
				gethCommon.BytesToAddress(subs[1].Address): big.NewInt(1112),
				gethCommon.BytesToAddress(subs[2].Address): big.NewInt(1113),
			},
			gethCommon.BytesToAddress(erc20s[0].Address): {
				gethCommon.BytesToAddress(subs[1].Address): big.NewInt(1212),
				gethCommon.BytesToAddress(subs[2].Address): big.NewInt(1213),
			},
		}, nil).Once()

		// For new token
		for i, sub := range subs {
			mockBalancer.On("BalanceOf", ctx, big.NewInt(100), map[gethCommon.Address]map[gethCommon.Address]struct{}{
				gethCommon.BytesToAddress(erc20s[1].Address): {
					gethCommon.BytesToAddress(sub.Address): struct{}{},
				},
			}).Return(map[gethCommon.Address]map[gethCommon.Address]*big.Int{
				gethCommon.BytesToAddress(erc20s[1].Address): {
					gethCommon.BytesToAddress(sub.Address): big.NewInt(1310 + int64(i)),
				},
			}, nil).Once()
		}
		err = manager.UpdateBlocks(ctx, blocks, receipts, events, &model.Reorg{
			From:     blocks[0].Number().Int64(),
			To:       blocks[len(blocks)-1].Number().Int64(),
			FromHash: blocks[0].Hash().Bytes(),
			ToHash:   blocks[len(blocks)-1].Hash().Bytes(),
		})
		Expect(err).Should(BeNil())

		// block 100
		t0_1_100, err = subStore.FindTotalBalance(100, gethCommon.BytesToAddress(erc20s[0].Address), 1)
		Expect(err).Should(BeNil())
		Expect(t0_1_100.Balance).Should(Equal("3212"))
		Expect(t0_1_100.TxFee).Should(Equal("0"))
		t0_2_100, err = subStore.FindTotalBalance(100, gethCommon.BytesToAddress(erc20s[0].Address), 2)
		Expect(err).Should(BeNil())
		Expect(t0_2_100.Balance).Should(Equal("1213"))
		Expect(t0_2_100.TxFee).Should(Equal("0"))
		t1_1_100, err = subStore.FindTotalBalance(100, gethCommon.BytesToAddress(erc20s[1].Address), 1)
		Expect(err).Should(BeNil())
		Expect(t1_1_100.Balance).Should(Equal("2621"))
		Expect(t1_1_100.TxFee).Should(Equal("0"))
		t1_2_100, err = subStore.FindTotalBalance(100, gethCommon.BytesToAddress(erc20s[1].Address), 2)
		Expect(err).Should(BeNil())
		Expect(t1_2_100.Balance).Should(Equal("1312"))
		Expect(t1_2_100.TxFee).Should(Equal("0"))

		// block 101
		t0_1_101, err = subStore.FindTotalBalance(101, gethCommon.BytesToAddress(erc20s[0].Address), 1)
		Expect(err).Should(BeNil())
		Expect(t0_1_101).Should(Equal(t0_1_100))
		t0_2_101, err = subStore.FindTotalBalance(101, gethCommon.BytesToAddress(erc20s[0].Address), 2)
		Expect(err).Should(BeNil())
		Expect(t0_2_101).Should(Equal(t0_2_100))
		t1_1_101, err = subStore.FindTotalBalance(101, gethCommon.BytesToAddress(erc20s[1].Address), 1)
		Expect(err).Should(BeNil())
		Expect(t1_1_101).Should(Equal(t1_1_100))
		t1_2_101, err = subStore.FindTotalBalance(101, gethCommon.BytesToAddress(erc20s[1].Address), 2)
		Expect(err).Should(BeNil())
		Expect(t1_2_101).Should(Equal(t1_2_100))

		t0_1_102, err = subStore.FindTotalBalance(102, gethCommon.BytesToAddress(erc20s[0].Address), 1)
		Expect(err).Should(BeNil())
		Expect(t0_1_102).Should(Equal(t0_1_100))
		t0_2_102, err = subStore.FindTotalBalance(102, gethCommon.BytesToAddress(erc20s[0].Address), 2)
		Expect(err).Should(BeNil())
		Expect(t0_2_102).Should(Equal(t0_2_100))
		t1_1_102, err = subStore.FindTotalBalance(102, gethCommon.BytesToAddress(erc20s[1].Address), 1)
		Expect(err).Should(BeNil())
		Expect(t1_1_102).Should(Equal(t1_1_100))
		t1_2_102, err = subStore.FindTotalBalance(102, gethCommon.BytesToAddress(erc20s[1].Address), 2)
		Expect(err).Should(BeNil())
		Expect(t1_2_102).Should(Equal(t1_2_100))

		// Verify new subscriptions' block numbers updated
		erc20, err = acctStore.FindERC20(gethCommon.BytesToAddress(erc20s[1].Address))
		Expect(err).Should(BeNil())
		Expect(erc20.BlockNumber).Should(Equal(int64(101)))

		// Check the balances of new token
		for i, sub := range subs {
			acc, err := acctStore.FindAccount(gethCommon.BytesToAddress(erc20s[1].Address), gethCommon.BytesToAddress(sub.Address))
			Expect(err).Should(BeNil())
			Expect(acc.Balance).Should(Equal("131" + strconv.Itoa(i)))
		}
	})
})
