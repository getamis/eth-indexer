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
	"github.com/getamis/eth-indexer/common"
	"github.com/getamis/eth-indexer/model"
	"github.com/getamis/eth-indexer/store/account"
	subsStore "github.com/getamis/eth-indexer/store/subscription"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Subscription Test", func() {
	var (
		blocks    []*types.Block
		signedTxs [][]*types.Transaction
		receipts  [][]*types.Receipt
		dumps     []*state.DirtyDump
		events    [][]*types.TransferLog
		manager   Manager

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
		db.Delete(&model.Subscription{})
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
			Token:       erc20.Address,
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

		// Init initial states
		signedTxs = [][]*types.Transaction{
			{
				signTransaction(types.NewTransaction(0, gethCommon.BytesToAddress(subs[1].Address), big.NewInt(1), 9000000, commonGasPrice, []byte("test payload")), acc0Key),
				signTransaction(types.NewTransaction(0, gethCommon.BytesToAddress(subs[2].Address), big.NewInt(1), 9000000, commonGasPrice, []byte("test payload")), acc1Key),
			},
			{
				signTransaction(types.NewTransaction(0, gethCommon.BytesToAddress(subs[1].Address), big.NewInt(1), 9000000, commonGasPrice, []byte("test payload")), acc2Key),
				signTransaction(types.NewTransaction(1, gethCommon.BytesToAddress(subs[0].Address), big.NewInt(1), 9000000, commonGasPrice, []byte("test payload")), acc2Key),
			},
			{
				// mimic a calling a contract without any value transfer (not represented in events)
				signTransaction(types.NewTransaction(2, contractAddress, big.NewInt(0), 9000000, commonGasPrice, []byte("test payload")), acc2Key),
			},
		}

		blocks = []*types.Block{
			types.NewBlockWithHeader(&types.Header{
				Number:   big.NewInt(100),
				Coinbase: acc0Addr,
			}).WithBody(signedTxs[0], nil),
			types.NewBlockWithHeader(&types.Header{
				Number:   big.NewInt(101),
				Coinbase: unknownRecipientAddr,
			}).WithBody(signedTxs[1], nil),
			types.NewBlockWithHeader(&types.Header{
				Number:   big.NewInt(102),
				Coinbase: acc1Addr,
			}).WithBody(signedTxs[2], nil),
		}
		receipts = [][]*types.Receipt{
			{
				&types.Receipt{
					TxHash:  signedTxs[0][0].Hash(),
					GasUsed: commonGasUsed.Uint64(),
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
							// transfer 1 tokens from subs[0] to subs[1]
							Topics: []gethCommon.Hash{
								gethCommon.BytesToHash(sha3TransferEvent),
								gethCommon.BytesToHash(subs[0].Address),
								gethCommon.BytesToHash(subs[1].Address),
							},
							Data: gethCommon.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
						},
						{
							Address: gethCommon.BytesToAddress(erc20.Address),
							// transfer 1 tokens from subs[2] to subs[0]
							Topics: []gethCommon.Hash{
								gethCommon.BytesToHash(sha3TransferEvent),
								gethCommon.BytesToHash(subs[2].Address),
								gethCommon.BytesToHash(subs[0].Address),
							},
							Data: gethCommon.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
						},
					},
				},
				&types.Receipt{
					TxHash:  signedTxs[0][1].Hash(),
					GasUsed: commonGasUsed.Uint64(),
				},
			},
			{
				&types.Receipt{
					TxHash:  signedTxs[1][0].Hash(),
					GasUsed: commonGasUsed.Uint64(),
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
							// transfer 1 tokens from subs[0] to subs[1]
							Topics: []gethCommon.Hash{
								gethCommon.BytesToHash(sha3TransferEvent),
								gethCommon.BytesToHash(subs[0].Address),
								gethCommon.BytesToHash(subs[1].Address),
							},
							Data: gethCommon.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001"),
						},
						{
							Address: gethCommon.BytesToAddress(erc20.Address),
							// transfer 1 tokens from subs[2] to subs[0]
							Topics: []gethCommon.Hash{
								gethCommon.BytesToHash(sha3TransferEvent),
								gethCommon.BytesToHash(subs[2].Address),
								gethCommon.BytesToHash(subs[0].Address),
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
				&types.Receipt{
					TxHash:  signedTxs[1][1].Hash(),
					GasUsed: commonGasUsed.Uint64(),
				},
			},
			{
				&types.Receipt{
					TxHash:  signedTxs[2][0].Hash(),
					GasUsed: commonGasUsed.Uint64(),
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
					TxHash: signedTxs[0][0].Hash(),
				},
				{
					From:   gethCommon.BytesToAddress(subs[1].Address),
					To:     gethCommon.BytesToAddress(subs[2].Address),
					Value:  big.NewInt(1),
					TxHash: signedTxs[0][1].Hash(),
				},
			},
			{
				{
					From:   gethCommon.BytesToAddress(subs[2].Address),
					To:     gethCommon.BytesToAddress(subs[1].Address),
					Value:  big.NewInt(1),
					TxHash: signedTxs[1][0].Hash(),
				},
				{
					From:   gethCommon.BytesToAddress(subs[2].Address),
					To:     gethCommon.BytesToAddress(subs[0].Address),
					Value:  big.NewInt(1),
					TxHash: signedTxs[1][1].Hash(),
				},
			},
			{},
		}

		ctx := context.Background()
		manager = NewManager(db)

		err = manager.InsertERC20(erc20)
		Expect(err).Should(BeNil())

		acctStore := account.NewWithDB(db)
		// Insert previous ERC20 balance for the old subscriptions
		err = acctStore.InsertAccount(&model.Account{
			ContractAddress: erc20.Address,
			BlockNumber:     99,
			Address:         subs[0].Address,
			Balance:         "2000",
		})
		Expect(err).Should(BeNil())
		// Insert previous ether balance for the old subscriptions
		err = acctStore.InsertAccount(&model.Account{
			ContractAddress: model.ETHBytes,
			BlockNumber:     99,
			Address:         subs[0].Address,
			Balance:         "1000",
		})
		Expect(err).Should(BeNil())

		err = manager.Init(mockBalancer)
		Expect(err).Should(BeNil())

		// For the 100 block
		mockBalancer.On("BalanceOf", ctx, big.NewInt(100), map[gethCommon.Address]map[gethCommon.Address]struct{}{
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
				gethCommon.BytesToAddress(subs[0].Address): big.NewInt(999),
				gethCommon.BytesToAddress(subs[1].Address): big.NewInt(100),
				gethCommon.BytesToAddress(subs[2].Address): big.NewInt(500),
			},
			gethCommon.BytesToAddress(erc20.Address): {
				gethCommon.BytesToAddress(subs[0].Address): big.NewInt(2000),
				gethCommon.BytesToAddress(subs[1].Address): big.NewInt(150),
				gethCommon.BytesToAddress(subs[2].Address): big.NewInt(1000),
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
				gethCommon.BytesToAddress(subs[2].Address): big.NewInt(458),
			},
			gethCommon.BytesToAddress(erc20.Address): {
				gethCommon.BytesToAddress(subs[0].Address): big.NewInt(2000),
				gethCommon.BytesToAddress(subs[1].Address): big.NewInt(151),
				gethCommon.BytesToAddress(subs[2].Address): big.NewInt(999),
			},
		}, nil).Once()

		// For the 102 block
		mockBalancer.On("BalanceOf", ctx, big.NewInt(102), map[gethCommon.Address]map[gethCommon.Address]struct{}{
			model.ETHAddress: {
				gethCommon.BytesToAddress(subs[1].Address): struct{}{},
				gethCommon.BytesToAddress(subs[2].Address): struct{}{},
			},
		}).Return(map[gethCommon.Address]map[gethCommon.Address]*big.Int{
			model.ETHAddress: {
				gethCommon.BytesToAddress(subs[1].Address): big.NewInt(201),
				gethCommon.BytesToAddress(subs[2].Address): big.NewInt(438),
			},
		}, nil).Once()

		err = manager.UpdateBlocks(ctx, blocks, receipts, dumps, events, ModeReOrg)
		Expect(err).Should(BeNil())

		// Verify total balances
		t1_100, err := subStore.FindTotalBalance(100, gethCommon.BytesToAddress(erc20.Address), 1)
		Expect(err).Should(BeNil())
		Expect(t1_100.Balance).Should(Equal("2150"))
		Expect(t1_100.TxFee).Should(Equal("0"))
		t2_100, err := subStore.FindTotalBalance(100, gethCommon.BytesToAddress(erc20.Address), 2)
		Expect(err).Should(BeNil())
		Expect(t2_100.Balance).Should(Equal("1000"))
		Expect(t2_100.TxFee).Should(Equal("0"))
		et1_100, err := subStore.FindTotalBalance(100, model.ETHAddress, 1)
		Expect(err).Should(BeNil())
		Expect(et1_100.Balance).Should(Equal("1099"))
		Expect(et1_100.TxFee).Should(Equal("40"))
		et2_100, err := subStore.FindTotalBalance(100, model.ETHAddress, 2)
		Expect(err).Should(BeNil())
		Expect(et2_100.Balance).Should(Equal("500"))
		Expect(et2_100.TxFee).Should(Equal("0"))

		t1_101, err := subStore.FindTotalBalance(101, gethCommon.BytesToAddress(erc20.Address), 1)
		Expect(err).Should(BeNil())
		Expect(t1_101.Balance).Should(Equal("2151"))
		Expect(t1_101.TxFee).Should(Equal("0"))
		t2_101, err := subStore.FindTotalBalance(101, gethCommon.BytesToAddress(erc20.Address), 2)
		Expect(err).Should(BeNil())
		Expect(t2_101.Balance).Should(Equal("999"))
		Expect(t2_101.TxFee).Should(Equal("0"))
		et1_101, err := subStore.FindTotalBalance(101, model.ETHAddress, 1)
		Expect(err).Should(BeNil())
		Expect(et1_101.Balance).Should(Equal("1101"))
		Expect(et1_101.TxFee).Should(Equal("0"))
		et2_101, err := subStore.FindTotalBalance(101, model.ETHAddress, 2)
		Expect(err).Should(BeNil())
		Expect(et2_101.Balance).Should(Equal("458"))
		Expect(et2_101.TxFee).Should(Equal("40"))

		t1_102, err := subStore.FindTotalBalance(102, gethCommon.BytesToAddress(erc20.Address), 1)
		Expect(err).Should(BeNil())
		Expect(t1_102).Should(Equal(t1_101))
		t2_102, err := subStore.FindTotalBalance(102, gethCommon.BytesToAddress(erc20.Address), 2)
		Expect(err).Should(BeNil())
		Expect(t2_102).Should(Equal(t2_101))
		et1_102, err := subStore.FindTotalBalance(102, model.ETHAddress, 1)
		Expect(err).Should(BeNil())
		Expect(et1_102.Balance).Should(Equal("1201"))
		Expect(et1_102.TxFee).Should(Equal("0"))
		et2_102, err := subStore.FindTotalBalance(102, model.ETHAddress, 2)
		Expect(err).Should(BeNil())
		Expect(et2_102.Balance).Should(Equal("438"))
		Expect(et2_102.TxFee).Should(Equal("20"))

		// Verify new subscriptions' block numbers updated
		res, err := subStore.FindOldSubscriptions([][]byte{subs[0].Address, subs[1].Address, subs[2].Address})
		Expect(err).Should(BeNil())
		Expect(res[0].BlockNumber).Should(Equal(int64(90)))
		Expect(res[1].BlockNumber).Should(Equal(int64(100)))
		Expect(res[2].BlockNumber).Should(Equal(int64(100)))

		// Verify recorded eth transfers
		ts, err := acctStore.FindAllTransfers(model.ETHAddress, acc0Addr)
		Expect(err).Should(BeNil())
		Expect(len(ts)).Should(Equal(2))
		for _, t := range ts {
			t.Address = model.ETHBytes
		}
		Expect(ts[0]).Should(Equal(common.EthTransferEvent(blocks[1], events[1][1])))
		Expect(ts[1]).Should(Equal(common.EthTransferEvent(blocks[0], events[0][0])))

		ts, err = acctStore.FindAllTransfers(model.ETHAddress, acc1Addr)
		Expect(err).Should(BeNil())
		Expect(len(ts)).Should(Equal(3))
		for _, t := range ts {
			t.Address = model.ETHBytes
		}
		Expect(ts[0]).Should(Equal(common.EthTransferEvent(blocks[1], events[1][0])))
		Expect(ts[1]).Should(Equal(common.EthTransferEvent(blocks[0], events[0][0])))
		Expect(ts[2]).Should(Equal(common.EthTransferEvent(blocks[0], events[0][1])))

		ts, err = acctStore.FindAllTransfers(model.ETHAddress, acc2Addr)
		Expect(err).Should(BeNil())
		Expect(len(ts)).Should(Equal(3))
		for _, t := range ts {
			t.Address = model.ETHBytes
		}
		Expect(ts[0]).Should(Equal(common.EthTransferEvent(blocks[1], events[1][0])))
		Expect(ts[1]).Should(Equal(common.EthTransferEvent(blocks[1], events[1][1])))
		Expect(ts[2]).Should(Equal(common.EthTransferEvent(blocks[0], events[0][1])))

		// Verify recorded erc20 transfers
		ts, err = acctStore.FindAllTransfers(gethCommon.BytesToAddress(erc20.Address), acc0Addr)
		Expect(err).Should(BeNil())
		Expect(len(ts)).Should(Equal(4))
		ts, err = acctStore.FindAllTransfers(gethCommon.BytesToAddress(erc20.Address), acc1Addr)
		Expect(err).Should(BeNil())
		Expect(len(ts)).Should(Equal(2))
		ts, err = acctStore.FindAllTransfers(gethCommon.BytesToAddress(erc20.Address), acc2Addr)
		Expect(err).Should(BeNil())
		Expect(len(ts)).Should(Equal(2))
	})
})
