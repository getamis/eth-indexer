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
		mysql *test.MySQLContainer
		db    *gorm.DB
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
		db.Table(block_header.TableName).Delete(&model.Header{})
		db.Table(transaction.TableName).Delete(&model.Transaction{})
		db.Table(transaction_receipt.TableName).Delete(&model.Receipt{})
		db.Table(account.NameStateBlocks).Delete(&model.StateBlock{})
		db.Table(account.NameAccounts).Delete(&model.Account{})
		db.Table(account.NameERC20).Delete(&model.ERC20{})
		db.Table(account.NameStateBlocks).Delete(&model.StateBlock{})
	})

	Context("InsertBlock()", func() {
		It("should be ok", func() {
			manager, err := NewManager(db)
			Expect(err).Should(BeNil())

			header := &types.Header{
				Number: big.NewInt(10),
			}
			block := types.NewBlock(header, nil, []*types.Header{
				header,
			}, []*types.Receipt{
				types.NewReceipt([]byte{}, false, 0),
			})

			err = manager.InsertBlock(block, nil)
			Expect(err).Should(Succeed())

			By("insert the same block again, should be ok")
			err = manager.InsertBlock(block, nil)
			Expect(err).Should(Succeed())
		})

		It("failed due to wrong signer", func() {
			manager, err := NewManager(db)
			Expect(err).Should(BeNil())

			header := &types.Header{
				Number: big.NewInt(11),
			}
			block := types.NewBlock(header, []*types.Transaction{
				types.NewTransaction(0, gethCommon.Address{}, gethCommon.Big0, 0, gethCommon.Big0, []byte{}),
			}, []*types.Header{
				header,
			}, []*types.Receipt{
				types.NewReceipt([]byte{}, false, 0),
			})

			err = manager.InsertBlock(block, nil)
			Expect(err).Should(Equal(common.ErrWrongSigner))
		})
	})

	Context("GetHeaderByNumber()", func() {
		It("gets the right header", func() {
			manager, err := NewManager(db)
			Expect(err).Should(BeNil())

			block1 := types.NewBlockWithHeader(&types.Header{
				Number: big.NewInt(100),
			})
			block2 := types.NewBlockWithHeader(&types.Header{
				Number: big.NewInt(99),
			})
			err = manager.InsertBlock(block1, nil)
			Expect(err).Should(Succeed())
			err = manager.InsertBlock(block2, nil)
			Expect(err).Should(Succeed())

			header, err := manager.GetHeaderByNumber(100)
			Expect(err).Should(Succeed())
			Expect(header).Should(Equal(common.Header(block1)))

			header, err = manager.GetHeaderByNumber(99)
			Expect(err).Should(Succeed())
			Expect(header).Should(Equal(common.Header(block2)))

			header, err = manager.GetHeaderByNumber(199)
			Expect(common.NotFoundError(err)).Should(BeTrue())
		})
	})

	Context("LatestHeader()", func() {
		It("gets the latest header", func() {
			manager, err := NewManager(db)
			Expect(err).Should(BeNil())

			block1 := types.NewBlockWithHeader(&types.Header{
				Number: big.NewInt(100),
			})
			block2 := types.NewBlockWithHeader(&types.Header{
				Number: big.NewInt(99),
			})
			err = manager.InsertBlock(block1, nil)
			Expect(err).Should(Succeed())
			err = manager.InsertBlock(block2, nil)
			Expect(err).Should(Succeed())

			header, err := manager.LatestHeader()
			Expect(err).Should(Succeed())
			Expect(header).Should(Equal(common.Header(block1)))
		})
	})

	Context("UpdateState()", func() {
		It("should be ok", func() {
			accountStore := account.NewWithDB(db)
			erc20 := &model.ERC20{
				Address:     gethCommon.HexToAddress("0x01").Bytes(),
				Code:        common.HexToBytes("0x02"),
				TotalSupply: "1000000",
				Name:        "erc20",
				Decimals:    18,
			}
			err := accountStore.InsertERC20(erc20)
			Expect(err).Should(Succeed())
			tmp := model.ERC20Storage{
				Address: erc20.Address,
			}
			Expect(db.HasTable(tmp)).Should(BeTrue())

			manager, err := NewManager(db)
			Expect(err).Should(BeNil())
			block1 := types.NewBlockWithHeader(&types.Header{
				Number: big.NewInt(100),
				Root:   gethCommon.StringToHash("1234567890"),
			})

			value := gethCommon.HexToHash("0x0a")
			key := gethCommon.HexToHash("0x0b")
			dump := map[string]state.DumpDirtyAccount{
				"fffffffffff": {
					Nonce:   100,
					Balance: "101",
				},
				// contract
				common.BytesToHex(erc20.Address): {
					Nonce:   900,
					Balance: "901",
					Storage: map[string]string{
						common.HashHex(key): common.HashHex(value),
					},
				},
			}
			err = manager.UpdateState(block1, dump)
			Expect(err).Should(Succeed())

			By("update the same state again, should be ok")
			err = manager.UpdateState(block1, dump)
			Expect(err).Should(Succeed())

			s, err := account.NewWithDB(db).FindERC20Storage(gethCommon.BytesToAddress(erc20.Address), key, block1.Number().Int64())
			Expect(err).Should(Succeed())
			Expect(s.Address).Should(Equal(erc20.Address))
			Expect(s.Key).Should(Equal(key.Bytes()))
			Expect(s.Value).Should(Equal(value.Bytes()))
		})
	})

	Context("LatestStateBlock()", func() {
		It("gets the latest state block", func() {
			manager, err := NewManager(db)
			Expect(err).Should(BeNil())

			block1 := types.NewBlockWithHeader(&types.Header{
				Number: big.NewInt(100),
				Root:   gethCommon.StringToHash("1234567890"),
			})

			err = manager.UpdateState(block1, nil)
			Expect(err).Should(Succeed())

			block, err := manager.LatestStateBlock()
			Expect(err).Should(Succeed())
			Expect(block.Number).Should(Equal(block1.Number().Int64()))
		})
	})

	Context("DeleteDataFromBlock()", func() {
		It("deletes data from a block number", func() {
			manager, err := NewManager(db)
			Expect(err).Should(BeNil())

			for i := int64(100); i < 120; i++ {
				block := types.NewBlockWithHeader(&types.Header{
					Number: big.NewInt(i),
					Root:   gethCommon.StringToHash("1234567890"),
				})
				err := manager.InsertBlock(block, nil)
				Expect(err).Should(Succeed())

				err = manager.UpdateState(block, nil)
				Expect(err).Should(Succeed())
			}
			manager.DeleteDataFromBlock(111)

			block, err := manager.LatestStateBlock()
			Expect(err).Should(Succeed())
			Expect(block.Number).Should(Equal(int64(110)))

			header, err := manager.LatestHeader()
			Expect(err).Should(Succeed())
			Expect(header.Number).Should(Equal(int64(110)))

			for i := int64(111); i < 120; i++ {
				header, err = manager.GetHeaderByNumber(i)
				Expect(common.NotFoundError(err)).Should(BeTrue())
			}
		})
	})
})

func TestStore(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Store Test")
}
