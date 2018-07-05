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
	"crypto/ecdsa"
	"math/big"

	gethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/getamis/eth-indexer/client/mocks"
	"github.com/getamis/eth-indexer/model"
	subsStore "github.com/getamis/eth-indexer/store/subscription"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	acc0Key, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	acc0Addr   = crypto.PubkeyToAddress(acc0Key.PublicKey)

	acc1Key, _ = crypto.HexToECDSA("8a1f9a8f95be41cd7ccb6168179afb4504aefe388d1e14474d32c45c72ce7b7a")
	acc2Key, _ = crypto.HexToECDSA("49a7b37aa6f6645917e7b807e9d1c00d4fa71f18343b0d4122a4d2df64dd6fee")
	acc1Addr   = crypto.PubkeyToAddress(acc1Key.PublicKey)
	acc2Addr   = crypto.PubkeyToAddress(acc2Key.PublicKey)

	commonGasPrice = big.NewInt(5)
	commonGasUsed  = big.NewInt(4)

	unknownRecipientAddr = gethCommon.HexToAddress("0xunknownrecipient")
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
		err := subStore.BatchInsert(subs)
		Expect(err).Should(BeNil())

		// Insert ERC20 total balance
		err = subStore.InsertTotalBalance(&model.TotalBalance{
			Token:        erc20.Address,
			BlockNumber:  99,
			Group:        1,
			Balance:      "2000",
			TxFee:        "0",
			MinerReward:  "0",
			UnclesReward: "0",
		})
		Expect(err).Should(BeNil())
		// Insert ether total balance
		err = subStore.InsertTotalBalance(&model.TotalBalance{
			Token:        model.ETHBytes,
			BlockNumber:  99,
			Group:        1,
			Balance:      "1000",
			TxFee:        "0",
			MinerReward:  "0",
			UnclesReward: "0",
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
				signTransaction(types.NewTransaction(2, unknownRecipientAddr, big.NewInt(1), 9000000, commonGasPrice, []byte("test payload")), acc2Key),
			},
		}

		blocks = []*types.Block{
			types.NewBlockWithHeader(&types.Header{
				Number:   big.NewInt(100),
				Coinbase: acc0Addr,
			}).WithBody(signedTxs[0], nil),
			types.NewBlockWithHeader(&types.Header{
				Number: big.NewInt(101),
			}).WithBody(signedTxs[1], nil),
			types.NewBlockWithHeader(&types.Header{
				Number: big.NewInt(102),
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
				gethCommon.BytesToAddress(subs[0].Address): big.NewInt(5000000000000000040 + 999),
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
		// 999+100-20(gasPrice * gasUsed)+5000000000000000040(miner reward)
		Expect(et1_100.Balance).Should(Equal("5000000000000001119"))
		Expect(et1_100.MinerReward).Should(Equal("5000000000000000040"))
		Expect(et1_100.UnclesReward).Should(Equal("0"))
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
		// 1000+101-20(gasPrice * gasUsed)+5000000000000000040(latest block balance)
		Expect(et1_101.Balance).Should(Equal("5000000000000001121"))
		Expect(et1_101.MinerReward).Should(Equal("0"))
		Expect(et1_101.UnclesReward).Should(Equal("0"))
		et2_101, err := subStore.FindTotalBalance(101, model.ETHAddress, 2)
		Expect(err).Should(BeNil())
		// 498-20(gasPrice * gasUsed)-20(gasPrice * gasUsed)
		Expect(et2_101.Balance).Should(Equal("458"))

		// Verify new subscriptions' block numbers updated
		res, err := subStore.FindOldSubscriptions([][]byte{subs[0].Address, subs[1].Address, subs[2].Address})
		Expect(err).Should(BeNil())
		Expect(res[0].BlockNumber).Should(Equal(int64(90)))
		Expect(res[1].BlockNumber).Should(Equal(int64(100)))
		Expect(res[2].BlockNumber).Should(Equal(int64(100)))
	})
})

func signTransaction(tx *types.Transaction, key *ecdsa.PrivateKey) (signedTx *types.Transaction) {
	signer := types.HomesteadSigner{}
	signedTx, _ = types.SignTx(tx, signer, key)
	return
}
