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
	"fmt"
	"math/big"
	"testing"

	gethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/getamis/eth-indexer/common"
	"github.com/getamis/eth-indexer/model"
	"github.com/getamis/eth-indexer/store/reorg"
	"github.com/getamis/eth-indexer/store/sqldb"
	"github.com/getamis/sirius/test"
	"github.com/jmoiron/sqlx"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	mysql *test.MySQLContainer
	db    *sqlx.DB
)

var (
	acc0Key, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	acc0Addr   = crypto.PubkeyToAddress(acc0Key.PublicKey)
	acc1Key, _ = crypto.HexToECDSA("8a1f9a8f95be41cd7ccb6168179afb4504aefe388d1e14474d32c45c72ce7b7a")
	acc1Addr   = crypto.PubkeyToAddress(acc1Key.PublicKey)
	acc2Key, _ = crypto.HexToECDSA("49a7b37aa6f6645917e7b807e9d1c00d4fa71f18343b0d4122a4d2df64dd6fee")
	acc2Addr   = crypto.PubkeyToAddress(acc2Key.PublicKey)

	commonGasPrice = big.NewInt(5)
	commonGasUsed  = big.NewInt(4)
	commonGasLimit = big.NewInt(5)

	unknownRecipientAddr = gethCommon.HexToAddress("0xunknownrecipient")
	contractAddress      = gethCommon.HexToAddress("0x3893b9422Cd5D70a81eDeFfe3d5A1c6A978310BB")
)

var _ = Describe("Manager Test", func() {
	var (
		blocks    []*types.Block
		uncles    [][]*types.Header
		receipts  [][]*types.Receipt
		events    [][]*types.TransferLog
		signedTxs [][]*types.Transaction
		manager   Manager
		ctx       = context.Background()
	)
	BeforeSuite(func() {
		var err error
		mysql, err = test.SetupMySQL()
		Expect(mysql).ShouldNot(BeNil())
		Expect(err).Should(Succeed())

		err = test.RunMigrationContainer(mysql, test.MigrationOptions{
			ImageRepository: "quay.io/amis/eth-indexer-db-migration",
		})
		Expect(err).Should(Succeed())

		db, err = sqldb.SimpleConnect("mysql", mysql.URL)
		Expect(err).Should(Succeed())
		Expect(db).ShouldNot(BeNil())
	})
	// ERC20 contract
	erc20 := &model.ERC20{
		Address:     gethCommon.HexToAddress("1234567890").Bytes(),
		BlockNumber: 1,
	}

	AfterSuite(func() {
		mysql.Stop()
	})

	BeforeEach(func() {
		// Init initial states
		signedTxs = [][]*types.Transaction{
			{
				signTransaction(types.NewTransaction(0, unknownRecipientAddr, big.NewInt(10000), commonGasLimit.Uint64(), commonGasPrice, []byte("test payload")), acc0Key),
			},
			{
				signTransaction(types.NewTransaction(1, unknownRecipientAddr, big.NewInt(10000), commonGasLimit.Uint64(), commonGasPrice, []byte("test payload")), acc0Key),
			},
		}
		uncles = [][]*types.Header{
			{
				types.CopyHeader(&types.Header{
					Number: big.NewInt(99),
				}),
			},
			{
				types.CopyHeader(&types.Header{
					Number: big.NewInt(99),
				}),
			},
		}
		blocks = []*types.Block{
			types.NewBlockWithHeader(&types.Header{
				Number: big.NewInt(100),
				Extra:  []byte("extra100"),
			}).WithBody(signedTxs[0], uncles[0]),
			types.NewBlockWithHeader(&types.Header{
				Number: big.NewInt(101),
				Extra:  []byte("extra101"),
			}).WithBody(signedTxs[1], uncles[1]),
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
							// transfer 99,900 tokens from 0x36928500bc1dcd7af6a2b4008875cc336b927d57 to 0xc6cde7c39eb2f0f0095f41570af89efc2c1ea828
							Topics: []gethCommon.Hash{
								gethCommon.BytesToHash(sha3TransferEvent),
								gethCommon.HexToHash("0x00000000000000000000000036928500bc1dcd7af6a2b4008875cc336b927d57"),
								gethCommon.HexToHash("0x000000000000000000000000c6cde7c39eb2f0f0095f41570af89efc2c1ea828"),
							},
							Data: gethCommon.Hex2Bytes("0000000000000000000000000000000000000000000000000000001742810700"),
						},
					},
				},
			},
			{
				&types.Receipt{
					TxHash:  signedTxs[1][0].Hash(),
					GasUsed: commonGasUsed.Uint64(),
				},
			},
		}
		events = [][]*types.TransferLog{
			{
				{
					From:   gethCommon.HexToAddress("0x01"),
					To:     gethCommon.HexToAddress("0x02"),
					Value:  big.NewInt(100),
					TxHash: gethCommon.HexToHash("0x03"),
				},
				{
					From:   gethCommon.HexToAddress("0x04"),
					To:     gethCommon.HexToAddress("0x05"),
					Value:  big.NewInt(200),
					TxHash: gethCommon.HexToHash("0x06"),
				},
			},
			{
				{
					From:   gethCommon.HexToAddress("0x07"),
					To:     gethCommon.HexToAddress("0x08"),
					Value:  big.NewInt(300),
					TxHash: gethCommon.HexToHash("0x09"),
				},
			},
		}

		var err error
		manager = NewManager(db, params.MainnetChainConfig)
		err = manager.Init(ctx)
		Expect(err).Should(BeNil())

		err = manager.InsertERC20(ctx, erc20)
		Expect(err).Should(BeNil())

		resERC20, err := manager.FindERC20(ctx, gethCommon.BytesToAddress(erc20.Address))
		Expect(err).Should(BeNil())
		Expect(resERC20).Should(Equal(erc20))

		err = manager.UpdateBlocks(ctx, nil, blocks, receipts, events, nil)
		Expect(err).Should(BeNil())
	})

	AfterEach(func() {
		// Clean all data
		_, err := db.Exec("DELETE FROM block_headers")
		Expect(err).Should(Succeed())
		_, err = db.Exec("DELETE FROM transactions")
		Expect(err).Should(Succeed())
		_, err = db.Exec("DELETE FROM transaction_receipts")
		Expect(err).Should(Succeed())
		_, err = db.Exec("DELETE FROM receipt_logs")
		Expect(err).Should(Succeed())

		_, err = db.Exec("DELETE FROM accounts")
		Expect(err).Should(Succeed())
		_, err = db.Exec("DELETE FROM eth_transfer")
		Expect(err).Should(Succeed())

		_, err = db.Exec("DELETE FROM erc20")
		Expect(err).Should(Succeed())

		_, err = db.Exec("DELETE FROM subscriptions")
		Expect(err).Should(Succeed())
		_, err = db.Exec("DELETE FROM total_balances")
		Expect(err).Should(Succeed())

		_, err = db.Exec("DELETE FROM reorgs")
		Expect(err).Should(Succeed())

		_, err = db.Exec(fmt.Sprintf("DROP TABLE %s", model.Transfer{
			Address: erc20.Address,
		}.TableName()))
		Expect(err).Should(Succeed())

		_, err = db.Exec(fmt.Sprintf("DROP TABLE %s", model.Account{
			ContractAddress: erc20.Address,
		}.TableName()))
		Expect(err).Should(Succeed())
	})

	Context("UpdateBlocks()", func() {
		It("sync mode, got duplicate key error due to the same txs", func() {
			newBlocks := []*types.Block{
				types.NewBlockWithHeader(&types.Header{
					Number:      big.NewInt(100),
					ReceiptHash: gethCommon.HexToHash("0x02"),
				}).WithBody(signedTxs[1], uncles[1]),
				types.NewBlockWithHeader(&types.Header{
					Number:      big.NewInt(101),
					ReceiptHash: gethCommon.HexToHash("0x03"),
				}).WithBody(signedTxs[0], uncles[0]),
			}
			newReceipts := [][]*types.Receipt{
				receipts[1],
				receipts[0],
			}
			err := manager.UpdateBlocks(ctx, nil, newBlocks, newReceipts, events, nil)
			Expect(common.DuplicateError(err)).Should(BeTrue())

			minerBaseReward, uncleInclusionReward, uncleCBs, unclesReward, unclesHash := common.AccumulateRewards(blocks[0].Header(), blocks[0].Uncles())
			header, err := manager.FindBlockByNumber(ctx, 100)
			Expect(err).Should(BeNil())
			h, err := common.Header(blocks[0]).AddReward(big.NewInt(20), minerBaseReward, uncleInclusionReward, unclesReward, uncleCBs, unclesHash)
			Expect(err).Should(BeNil())
			h.CreatedAt = header.CreatedAt
			h.ID = header.ID
			Expect(header).Should(Equal(h))
		})

		It("reorg mode", func() {
			newUncles := [][]*types.Header{
				uncles[1],
				uncles[0],
			}
			newBlocks := []*types.Block{
				types.NewBlockWithHeader(&types.Header{
					Number:      big.NewInt(100),
					Extra:       []byte("extra100"),
					ReceiptHash: gethCommon.HexToHash("0x02"),
				}).WithBody(signedTxs[1], newUncles[0]),
				types.NewBlockWithHeader(&types.Header{
					Number:      big.NewInt(101),
					Extra:       []byte("extra101"),
					ReceiptHash: gethCommon.HexToHash("0x03"),
				}).WithBody(signedTxs[0], newUncles[1]),
			}
			newReceipts := [][]*types.Receipt{
				receipts[1],
				receipts[0],
			}
			err := manager.UpdateBlocks(ctx, nil, newBlocks, newReceipts, events, &model.Reorg{
				From:     blocks[0].Number().Int64(),
				To:       blocks[len(blocks)-1].Number().Int64(),
				FromHash: blocks[0].Hash().Bytes(),
				ToHash:   blocks[len(blocks)-1].Hash().Bytes(),
			})
			Expect(err).Should(BeNil())

			minerBaseReward, uncleInclusionReward, uncleCBs, unclesReward, unclesHash := common.AccumulateRewards(blocks[0].Header(), blocks[0].Uncles())
			header, err := manager.FindBlockByNumber(ctx, 100)
			Expect(err).Should(BeNil())
			h, err := common.Header(newBlocks[0]).AddReward(big.NewInt(20), minerBaseReward, uncleInclusionReward, unclesReward, uncleCBs, unclesHash)
			Expect(err).Should(BeNil())
			h.CreatedAt = header.CreatedAt
			h.ID = header.ID
			Expect(header).Should(Equal(h))
			reorgStore := reorg.NewWithDB(db)
			rs, err := reorgStore.List(ctx)
			Expect(err).Should(Succeed())
			Expect(len(rs)).Should(BeNumerically("==", 1))
		})

		It("failed due to wrong signer", func() {
			blocks[0] = types.NewBlock(
				blocks[0].Header(), []*types.Transaction{
					types.NewTransaction(0, gethCommon.Address{}, gethCommon.Big0, 0, gethCommon.Big0, []byte{}),
				}, uncles[0], []*types.Receipt{
					types.NewReceipt([]byte{}, false, 0),
				})

			err := manager.UpdateBlocks(ctx, nil, blocks, receipts, events, nil)
			Expect(err).Should(Equal(common.ErrWrongSigner))
		})
	})

	Context("FindBlockByNumber()", func() {
		It("gets the right header", func() {
			for i, block := range blocks {
				minerBaseReward, uncleInclusionReward, uncleCBs, unclesReward, unclesHash := common.AccumulateRewards(blocks[i].Header(), uncles[i])
				header, err := manager.FindBlockByNumber(ctx, block.Number().Int64())
				Expect(err).Should(Succeed())

				h, err := common.Header(block).AddReward(big.NewInt(20), minerBaseReward, uncleInclusionReward, unclesReward, uncleCBs, unclesHash)
				Expect(err).Should(BeNil())
				h.CreatedAt = header.CreatedAt
				h.ID = header.ID
				Expect(header).Should(Equal(h))
			}
		})
	})
})

func TestStore(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Store Test")
}

func signTransaction(tx *types.Transaction, key *ecdsa.PrivateKey) (signedTx *types.Transaction) {
	signer := types.HomesteadSigner{}
	signedTx, _ = types.SignTx(tx, signer, key)
	return
}
