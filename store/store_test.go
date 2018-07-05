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
	"os"
	"testing"

	gethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/getamis/eth-indexer/common"
	"github.com/getamis/eth-indexer/model"
	"github.com/getamis/eth-indexer/store/account"
	"github.com/getamis/sirius/test"
	"github.com/jinzhu/gorm"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	mysql *test.MySQLContainer
	db    *gorm.DB
)

var _ = Describe("Manager Test", func() {
	var (
		blocks    []*types.Block
		receipts  [][]*types.Receipt
		dumps     []*state.DirtyDump
		events    [][]*types.TransferLog
		signedTxs [][]*types.Transaction
		manager   Manager
	)

	newErc20Addr := gethCommon.HexToAddress("1234567891")
	newErc20 := &model.ERC20{
		Address:     newErc20Addr.Bytes(),
		Code:        []byte("1332"),
		BlockNumber: 0,
	}
	newErc20Storage := map[string]string{
		common.BytesToHex(gethCommon.HexToHash("0x0a").Bytes()): common.BytesToHex(gethCommon.HexToHash("0x0b").Bytes()),
		common.BytesToHex(gethCommon.HexToHash("0x0c").Bytes()): common.BytesToHex(gethCommon.HexToHash("0x0d").Bytes()),
	}

	BeforeSuite(func() {
		var err error
		mysql, err = test.NewMySQLContainer("quay.io/amis/eth-indexer-db-migration")
		Expect(mysql).ShouldNot(BeNil())
		Expect(err).Should(Succeed())
		Expect(mysql.Start()).Should(Succeed())

		db, err = gorm.Open("mysql", mysql.URL)
		Expect(err).Should(Succeed())
		Expect(db).ShouldNot(BeNil())

		db.LogMode(os.Getenv("ENABLE_DB_LOG_IN_TEST") != "")
	})
	// ERC20 contract
	erc20 := &model.ERC20{
		Address:     gethCommon.HexToAddress("1234567890").Bytes(),
		Code:        []byte("1333"),
		BlockNumber: 0,
	}

	AfterSuite(func() {
		mysql.Stop()
	})

	BeforeEach(func() {
		// Init initial states
		signedTxs = [][]*types.Transaction{
			{
				signTransaction(types.NewTransaction(0, unknownRecipientAddr, big.NewInt(10000), 9000000, big.NewInt(9000000), []byte("test payload")), acc0Key),
			},
			{
				signTransaction(types.NewTransaction(1, unknownRecipientAddr, big.NewInt(10000), 9000000, big.NewInt(9000000), []byte("test payload")), acc0Key),
			},
		}
		blocks = []*types.Block{
			types.NewBlockWithHeader(&types.Header{
				Number: big.NewInt(100),
			}).WithBody(signedTxs[0], nil),
			types.NewBlockWithHeader(&types.Header{
				Number: big.NewInt(101),
			}).WithBody(signedTxs[1], nil),
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
		dumps = []*state.DirtyDump{
			{
				Root: "root1",
				Accounts: map[string]state.DirtyDumpAccount{
					common.BytesToHex(erc20.Address): {
						Storage: map[string]string{
							"1": "2",
						},
					},
					common.BytesToHex(newErc20.Address): {
						Storage: newErc20Storage,
					},
				},
			},
			{
				Root: "root2",
				Accounts: map[string]state.DirtyDumpAccount{
					"3": {
						Storage: map[string]string{
							"4": "5",
						},
					},
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
		manager = NewManager(db)
		err = manager.Init(nil)
		Expect(err).Should(BeNil())

		err = manager.InsertERC20(erc20)
		Expect(err).Should(BeNil())

		resERC20, err := manager.FindERC20(gethCommon.BytesToAddress(erc20.Address))
		Expect(err).Should(BeNil())
		Expect(resERC20).Should(Equal(erc20))

		err = manager.UpdateBlocks(context.Background(), blocks, receipts, dumps, events, ModeReOrg)
		Expect(err).Should(BeNil())
	})

	AfterEach(func() {
		// Clean all data
		db.Delete(&model.Header{})
		db.Delete(&model.Transaction{})
		db.Delete(&model.Receipt{})
		db.Delete(&model.Account{})
		db.Delete(&model.ERC20{})
		db.DropTable(model.ERC20Storage{
			Address: erc20.Address,
		})
		db.DropTable(model.ERC20Storage{
			Address: newErc20.Address,
		})
		db.DropTable(model.Account{
			ContractAddress: erc20.Address,
		})
		db.DropTable(model.Account{
			ContractAddress: newErc20.Address,
		})
		db.DropTable(model.Transfer{
			Address: erc20.Address,
		})
		db.DropTable(model.Transfer{
			Address: newErc20.Address,
		})
	})

	Context("UpdateBlocks()", func() {
		It("sync mode", func() {
			newBlocks := []*types.Block{
				types.NewBlockWithHeader(&types.Header{
					Number:      big.NewInt(100),
					ReceiptHash: gethCommon.HexToHash("0x02"),
				}),
				types.NewBlockWithHeader(&types.Header{
					Number:      big.NewInt(101),
					ReceiptHash: gethCommon.HexToHash("0x03"),
				}),
			}
			err := manager.UpdateBlocks(context.Background(), newBlocks, receipts, dumps, events, ModeSync)
			Expect(err).Should(BeNil())

			header, err := manager.GetHeaderByNumber(100)
			Expect(err).Should(BeNil())
			Expect(header).Should(Equal(common.Header(blocks[0])))
		})

		It("reorg mode", func() {
			newBlocks := []*types.Block{
				types.NewBlockWithHeader(&types.Header{
					Number:      big.NewInt(100),
					ReceiptHash: gethCommon.HexToHash("0x02"),
				}).WithBody(signedTxs[1], nil),
				types.NewBlockWithHeader(&types.Header{
					Number:      big.NewInt(101),
					ReceiptHash: gethCommon.HexToHash("0x03"),
				}).WithBody(signedTxs[0], nil),
			}
			newReceipts := [][]*types.Receipt{
				receipts[1],
				receipts[0],
			}
			err := manager.UpdateBlocks(context.Background(), newBlocks, newReceipts, dumps, events, ModeReOrg)
			Expect(err).Should(BeNil())

			header, err := manager.GetHeaderByNumber(100)
			Expect(err).Should(BeNil())
			Expect(header).Should(Equal(common.Header(newBlocks[0])))
		})

		It("force sync mode", func() {
			accountStore := account.NewWithDB(db)

			// Cannot find new erc20 storage
			for k := range newErc20Storage {
				_, err := accountStore.FindERC20Storage(newErc20Addr, gethCommon.HexToHash(k), blocks[0].Number().Int64())
				Expect(err).ShouldNot(BeNil())
			}

			// Create find new erc20
			err := manager.InsertERC20(newErc20)
			Expect(err).Should(BeNil())

			// Reload manager
			manager = NewManager(db)
			err = manager.Init(nil)
			Expect(err).Should(BeNil())

			// Force update blocks
			err = manager.UpdateBlocks(context.Background(), blocks, receipts, dumps, events, ModeForceSync)
			Expect(err).Should(BeNil())

			// Got blocks 0
			header, err := manager.GetHeaderByNumber(100)
			Expect(err).Should(BeNil())
			Expect(header).Should(Equal(common.Header(blocks[0])))

			// Found new erc20 storage
			for k, v := range newErc20Storage {
				value, err := accountStore.FindERC20Storage(newErc20Addr, gethCommon.HexToHash(k), blocks[0].Number().Int64())
				Expect(err).Should(BeNil())
				Expect(value.Value).Should(Equal(gethCommon.HexToHash(v).Bytes()))
			}

			db.DropTable(model.ERC20Storage{
				Address: newErc20.Address,
			})
		})

		It("failed due to wrong signer", func() {
			blocks[0] = types.NewBlock(
				blocks[0].Header(), []*types.Transaction{
					types.NewTransaction(0, gethCommon.Address{}, gethCommon.Big0, 0, gethCommon.Big0, []byte{}),
				}, nil, []*types.Receipt{
					types.NewReceipt([]byte{}, false, 0),
				})

			err := manager.UpdateBlocks(context.Background(), blocks, receipts, dumps, events, ModeReOrg)
			Expect(err).Should(Equal(common.ErrWrongSigner))
		})
	})

	Context("InsertTd/GetTd()", func() {
		It("saves and get TD", func() {
			err := manager.InsertTd(blocks[0], new(big.Int).SetInt64(123456789))
			Expect(err).Should(Succeed())

			_, err = manager.GetTd(blocks[0].Hash().Bytes())
			Expect(err).Should(Succeed())

			err = manager.InsertTd(blocks[0], new(big.Int).SetInt64(123456789))
			Expect(common.DuplicateError(err)).Should(BeTrue())

			_, err = manager.GetTd(blocks[0].Hash().Bytes())
			Expect(err).Should(Succeed())
		})
	})

	Context("GetHeaderByNumber()", func() {
		It("gets the right header", func() {
			for _, block := range blocks {
				header, err := manager.GetHeaderByNumber(block.Number().Int64())
				Expect(err).Should(Succeed())
				Expect(header).Should(Equal(common.Header(block)))
			}
		})
	})

	Context("LatestHeader()", func() {
		It("gets the latest header", func() {
			header, err := manager.LatestHeader()
			Expect(err).Should(Succeed())
			Expect(header).Should(Equal(common.Header(blocks[1])))
		})
	})
})

func TestStore(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Store Test")
}
