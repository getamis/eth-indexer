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

	gethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/getamis/eth-indexer/client/mocks"
	"github.com/getamis/eth-indexer/model"
	subscriptionStore "github.com/getamis/eth-indexer/store/subscription"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Subscription Test", func() {
	var (
		blocks       []*types.Block
		receipts     [][]*types.Receipt
		dumps        []*state.DirtyDump
		events       [][]*types.TransferLog
		manager      Manager
		mockBalancer *mocks.Balancer
	)

	// ERC20 contract
	erc20 := &model.ERC20{
		Address:     gethCommon.HexToAddress("1234567892").Bytes(),
		Code:        []byte("1334"),
		BlockNumber: 0,
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
		db.Delete(&model.Account{})
		db.Delete(&model.ERC20{})
		db.DropTable(model.Transfer{
			Address: erc20.Address,
		})
	})

	It("should be successful", func() {
		// subscriptions
		subs := []*model.Subscription{
			{
				BlockNumber: 90,
				Group:       1,
				Address:     gethCommon.Hex2Bytes("36928500bc1dcd7af6a2b4008875cc336b927d57"),
			},
			{
				BlockNumber: 0,
				Group:       1,
				Address:     gethCommon.Hex2Bytes("c6cde7c39eb2f0f0095f41570af89efc2c1ea828"),
			},
			{
				BlockNumber: 0,
				Group:       2,
				Address:     gethCommon.Hex2Bytes("36928500bc1dcd7af6a2b4008875cc336b927d58"),
			},
		}
		// Insert subscription
		subStore := subscriptionStore.NewWithDB(db)
		for _, sub := range subs {
			err := subStore.Insert(sub)
			Expect(err).Should(BeNil())
		}

		// Insert ERC20 total balance
		err := subStore.InsertTotalBalance(&model.TotalBalance{
			Token:       erc20.Address,
			BlockNumber: 99,
			Group:       1,
			Balance:     "2000",
		})
		Expect(err).Should(BeNil())
		// Insert ether total balance
		err = subStore.InsertTotalBalance(&model.TotalBalance{
			Token:       model.ETHBytes,
			BlockNumber: 99,
			Group:       1,
			Balance:     "1000",
		})
		Expect(err).Should(BeNil())

		// Init initial states
		blocks = []*types.Block{
			types.NewBlockWithHeader(&types.Header{
				Number: big.NewInt(100),
			}),
			types.NewBlockWithHeader(&types.Header{
				Number: big.NewInt(101),
			}),
			types.NewBlockWithHeader(&types.Header{
				Number: big.NewInt(102),
			}),
		}
		receipts = [][]*types.Receipt{
			{
				&types.Receipt{
					TxHash: gethCommon.HexToHash("0x01"),
					Logs: []*types.Log{
						{
							Address: gethCommon.HexToAddress("0x000001"),
							Topics: []gethCommon.Hash{
								gethCommon.HexToHash("0x000011"),
								gethCommon.HexToHash("0x000012"),
								gethCommon.HexToHash("0x000013"),
								gethCommon.HexToHash("0x000014"),
							},
							Data: []byte("data"),
						},
						{
							Address: gethCommon.BytesToAddress(erc20.Address),
							// transfer 1 tokens from 0x36928500bc1dcd7af6a2b4008875cc336b927d57 to 0xc6cde7c39eb2f0f0095f41570af89efc2c1ea828
							Topics: []gethCommon.Hash{
								gethCommon.BytesToHash(sha3TransferEvent),
								gethCommon.HexToHash("0x00000000000000000000000036928500bc1dcd7af6a2b4008875cc336b927d57"),
								gethCommon.HexToHash("0x000000000000000000000000c6cde7c39eb2f0f0095f41570af89efc2c1ea828"),
							},
							Data: gethCommon.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
						},
						{
							Address: gethCommon.BytesToAddress(erc20.Address),
							// transfer 1 tokens from 0x36928500bc1dcd7af6a2b4008875cc336b927d58 to 0x36928500bc1dcd7af6a2b4008875cc336b927d57
							Topics: []gethCommon.Hash{
								gethCommon.BytesToHash(sha3TransferEvent),
								gethCommon.HexToHash("0x00000000000000000000000036928500bc1dcd7af6a2b4008875cc336b927d58"),
								gethCommon.HexToHash("0x00000000000000000000000036928500bc1dcd7af6a2b4008875cc336b927d57"),
							},
							Data: gethCommon.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
						},
					},
				},
			},
			{
				&types.Receipt{
					TxHash: gethCommon.HexToHash("0x02"),
					Logs: []*types.Log{
						{
							Address: gethCommon.HexToAddress("0x000001"),
							Topics: []gethCommon.Hash{
								gethCommon.HexToHash("0x000011"),
								gethCommon.HexToHash("0x000012"),
								gethCommon.HexToHash("0x000013"),
								gethCommon.HexToHash("0x000014"),
							},
							Data: []byte("data"),
						},
						{
							Address: gethCommon.BytesToAddress(erc20.Address),
							// transfer 1 tokens from 0x36928500bc1dcd7af6a2b4008875cc336b927d57 to 0xc6cde7c39eb2f0f0095f41570af89efc2c1ea828
							Topics: []gethCommon.Hash{
								gethCommon.BytesToHash(sha3TransferEvent),
								gethCommon.HexToHash("0x00000000000000000000000036928500bc1dcd7af6a2b4008875cc336b927d57"),
								gethCommon.HexToHash("0x000000000000000000000000c6cde7c39eb2f0f0095f41570af89efc2c1ea828"),
							},
							Data: gethCommon.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
						},
						{
							Address: gethCommon.BytesToAddress(erc20.Address),
							// transfer 1 tokens from 0x36928500bc1dcd7af6a2b4008875cc336b927d58 to 0x36928500bc1dcd7af6a2b4008875cc336b927d57
							Topics: []gethCommon.Hash{
								gethCommon.BytesToHash(sha3TransferEvent),
								gethCommon.HexToHash("0x00000000000000000000000036928500bc1dcd7af6a2b4008875cc336b927d58"),
								gethCommon.HexToHash("0x00000000000000000000000036928500bc1dcd7af6a2b4008875cc336b927d57"),
							},
							Data: gethCommon.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
						},
						// Unsubscribed accounts
						{
							Address: gethCommon.BytesToAddress(erc20.Address),
							Topics: []gethCommon.Hash{
								gethCommon.BytesToHash(sha3TransferEvent),
								gethCommon.HexToHash("0x00000000000000000000000036928500bc1dcd7af6a2b4008875cc336b927dAA"),
								gethCommon.HexToHash("0x000000000000000000000000c6cde7c39eb2f0f0095f41570af89efc2c1ea8BB"),
							},
							Data: gethCommon.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
						},
					},
				},
			},
			{
				&types.Receipt{
					TxHash: gethCommon.HexToHash("0x03"),
				},
			},
		}
		dumps = []*state.DirtyDump{
			{
				Root: "root1",
			},
			{
				Root: "root2",
			},
			{
				Root: "root3",
			},
		}
		events = [][]*types.TransferLog{
			{
				{
					From:   gethCommon.BytesToAddress(subs[0].Address),
					To:     gethCommon.BytesToAddress(subs[1].Address),
					Value:  big.NewInt(1),
					TxHash: gethCommon.HexToHash("0x03"),
				},
				{
					From:   gethCommon.BytesToAddress(subs[1].Address),
					To:     gethCommon.BytesToAddress(subs[2].Address),
					Value:  big.NewInt(1),
					TxHash: gethCommon.HexToHash("0x06"),
				},
			},
			{
				{
					From:   gethCommon.BytesToAddress(subs[2].Address),
					To:     gethCommon.BytesToAddress(subs[1].Address),
					Value:  big.NewInt(1),
					TxHash: gethCommon.HexToHash("0x03"),
				},
				{
					From:   gethCommon.BytesToAddress(subs[2].Address),
					To:     gethCommon.BytesToAddress(subs[0].Address),
					Value:  big.NewInt(1),
					TxHash: gethCommon.HexToHash("0x06"),
				},
			},
			{},
		}

		ctx := context.Background()
		manager = NewManager(db, true)

		err = manager.InsertERC20(erc20)
		Expect(err).Should(BeNil())

		err = manager.Init(mockBalancer)
		Expect(err).Should(BeNil())

		// For the 100 block
		mockBalancer.On("BalanceOf", ctx, big.NewInt(100), map[gethCommon.Address]map[gethCommon.Address]struct{}{
			model.ETHAddress: {
				gethCommon.BytesToAddress(subs[1].Address): struct{}{},
				gethCommon.BytesToAddress(subs[2].Address): struct{}{},
			},
			gethCommon.BytesToAddress(erc20.Address): {
				gethCommon.BytesToAddress(subs[1].Address): struct{}{},
				gethCommon.BytesToAddress(subs[2].Address): struct{}{},
			},
		}).Return(map[gethCommon.Address]map[gethCommon.Address]*big.Int{
			model.ETHAddress: {
				gethCommon.BytesToAddress(subs[1].Address): big.NewInt(100),
				gethCommon.BytesToAddress(subs[2].Address): big.NewInt(500),
			},
			gethCommon.BytesToAddress(erc20.Address): {
				gethCommon.BytesToAddress(subs[1].Address): big.NewInt(150),
				gethCommon.BytesToAddress(subs[2].Address): big.NewInt(1000),
			},
		}, nil).Once()
		mockBalancer.On("BalanceOf", ctx, big.NewInt(100), map[gethCommon.Address]map[gethCommon.Address]struct{}{
			model.ETHAddress: {
				gethCommon.BytesToAddress(subs[0].Address): struct{}{},
			},
			gethCommon.BytesToAddress(erc20.Address): {
				gethCommon.BytesToAddress(subs[0].Address): struct{}{},
			},
		}).Return(map[gethCommon.Address]map[gethCommon.Address]*big.Int{
			model.ETHAddress: {
				gethCommon.BytesToAddress(subs[0].Address): big.NewInt(999),
			},
			gethCommon.BytesToAddress(erc20.Address): {
				gethCommon.BytesToAddress(subs[0].Address): big.NewInt(2000),
			},
		}, nil).Once()

		// For the 101 block
		mockBalancer.On("BalanceOf", ctx, big.NewInt(101), map[gethCommon.Address]map[gethCommon.Address]struct{}{
			model.ETHAddress: {
				gethCommon.BytesToAddress(subs[0].Address): struct{}{},
				gethCommon.BytesToAddress(subs[1].Address): struct{}{},
				gethCommon.BytesToAddress(subs[2].Address): struct{}{},
			},
			gethCommon.BytesToAddress(erc20.Address): {
				gethCommon.BytesToAddress(subs[0].Address): struct{}{},
				gethCommon.BytesToAddress(subs[1].Address): struct{}{},
				gethCommon.BytesToAddress(subs[2].Address): struct{}{},
			},
		}).Return(map[gethCommon.Address]map[gethCommon.Address]*big.Int{
			model.ETHAddress: {
				gethCommon.BytesToAddress(subs[0].Address): big.NewInt(1000),
				gethCommon.BytesToAddress(subs[1].Address): big.NewInt(101),
				gethCommon.BytesToAddress(subs[2].Address): big.NewInt(498),
			},
			gethCommon.BytesToAddress(erc20.Address): {
				gethCommon.BytesToAddress(subs[0].Address): big.NewInt(2000),
				gethCommon.BytesToAddress(subs[1].Address): big.NewInt(151),
				gethCommon.BytesToAddress(subs[2].Address): big.NewInt(999),
			},
		}, nil).Once()

		err = manager.UpdateBlocks(ctx, blocks, receipts, dumps, events, ModeReOrg)
		Expect(err).Should(BeNil())

		// Verify total balances
		t1_100, err := subStore.FindTotalBalance(100, gethCommon.BytesToAddress(erc20.Address), 1)
		Expect(err).Should(BeNil())
		Expect(t1_100.Balance).Should(Equal("2150"))
		t2_100, err := subStore.FindTotalBalance(100, gethCommon.BytesToAddress(erc20.Address), 2)
		Expect(err).Should(BeNil())
		Expect(t2_100.Balance).Should(Equal("1000"))
		et1_100, err := subStore.FindTotalBalance(100, model.ETHAddress, 1)
		Expect(err).Should(BeNil())
		Expect(et1_100.Balance).Should(Equal("1099"))
		et2_100, err := subStore.FindTotalBalance(100, model.ETHAddress, 2)
		Expect(err).Should(BeNil())
		Expect(et2_100.Balance).Should(Equal("500"))

		t1_101, err := subStore.FindTotalBalance(101, gethCommon.BytesToAddress(erc20.Address), 1)
		Expect(err).Should(BeNil())
		Expect(t1_101.Balance).Should(Equal("2151"))
		t2_101, err := subStore.FindTotalBalance(101, gethCommon.BytesToAddress(erc20.Address), 2)
		Expect(err).Should(BeNil())
		Expect(t2_101.Balance).Should(Equal("999"))
		et1_101, err := subStore.FindTotalBalance(101, model.ETHAddress, 1)
		Expect(err).Should(BeNil())
		Expect(et1_101.Balance).Should(Equal("1101"))
		et2_101, err := subStore.FindTotalBalance(101, model.ETHAddress, 2)
		Expect(err).Should(BeNil())
		Expect(et2_101.Balance).Should(Equal("498"))
	})
})
