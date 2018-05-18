package store

import (
	"math/big"
	"os"
	"testing"

	gethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/jinzhu/gorm"
	"github.com/maichain/eth-indexer/common"
	"github.com/maichain/eth-indexer/model"
	"github.com/maichain/eth-indexer/store/account"
	"github.com/maichain/eth-indexer/store/block_header"
	"github.com/maichain/eth-indexer/store/transaction"
	"github.com/maichain/eth-indexer/store/transaction_receipt"
	"github.com/maichain/mapi/base/test"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Manager Test", func() {
	var (
		mysql    *test.MySQLContainer
		db       *gorm.DB
		blocks   []*types.Block
		receipts [][]*types.Receipt
		dumps    []*state.DirtyDump
		manager  Manager
	)
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

	AfterSuite(func() {
		mysql.Stop()
	})

	BeforeEach(func() {
		// ERC20 contract
		erc20 := &model.ERC20{
			Address:     gethCommon.HexToAddress("1234567890").Bytes(),
			Code:        []byte("1333"),
			BlockNumber: 0,
		}

		// Clean all data
		db.Table(block_header.TableName).Delete(&model.Header{})
		db.Table(transaction.TableName).Delete(&model.Transaction{})
		db.Table(transaction_receipt.TableName).Delete(&model.Receipt{})
		db.Table(account.NameAccounts).Delete(&model.Account{})
		db.Table(account.NameERC20).Delete(&model.ERC20{})
		db.DropTable(model.ERC20Storage{
			Address: erc20.Address,
		})

		// Init initial states
		blocks = []*types.Block{
			types.NewBlockWithHeader(&types.Header{
				Number: big.NewInt(100),
			}),
			types.NewBlockWithHeader(&types.Header{
				Number: big.NewInt(101),
			}),
		}
		receipts = [][]*types.Receipt{
			{
				&types.Receipt{
					TxHash: gethCommon.HexToHash("0x01"),
				},
			},
			{
				&types.Receipt{
					TxHash: gethCommon.HexToHash("0x02"),
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

		var err error
		manager, err = NewManager(db)
		Expect(err).Should(BeNil())

		err = manager.InsertERC20(erc20)
		Expect(err).Should(BeNil())

		resERC20, err := manager.FindERC20(gethCommon.BytesToAddress(erc20.Address))
		Expect(err).Should(BeNil())
		Expect(resERC20).Should(Equal(erc20))

		err = manager.UpdateBlocks(blocks, receipts, dumps, true)
		Expect(err).Should(BeNil())
	})

	Context("UpdateBlocks()", func() {
		It("can not change blocks", func() {
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
			err := manager.UpdateBlocks(newBlocks, receipts, dumps, false)
			Expect(err).Should(BeNil())

			header, err := manager.GetHeaderByNumber(100)
			Expect(err).Should(BeNil())
			Expect(header).Should(Equal(common.Header(blocks[0])))
		})

		It("can not change blocks", func() {
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
			err := manager.UpdateBlocks(newBlocks, receipts, dumps, true)
			Expect(err).Should(BeNil())

			header, err := manager.GetHeaderByNumber(100)
			Expect(err).Should(BeNil())
			Expect(header).Should(Equal(common.Header(newBlocks[0])))
		})

		It("failed due to wrong signer", func() {
			blocks[0] = types.NewBlock(
				blocks[0].Header(), []*types.Transaction{
					types.NewTransaction(0, gethCommon.Address{}, gethCommon.Big0, 0, gethCommon.Big0, []byte{}),
				}, nil, []*types.Receipt{
					types.NewReceipt([]byte{}, false, 0),
				})

			err := manager.UpdateBlocks(blocks, receipts, dumps, true)
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
