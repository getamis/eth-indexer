package store

import (
	"fmt"
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
		db.Table(account.NameContracts).Delete(&model.Contract{})
		db.Table(account.NameContractCode).Delete(&model.ContractCode{})
		db.Table(account.NameStateBlocks).Delete(&model.StateBlock{})
	})

	Context("InsertBlock()", func() {
		It("should be ok", func() {
			manager := NewManager(db)
			header := &types.Header{
				Number: big.NewInt(10),
			}
			block := types.NewBlock(header, nil, []*types.Header{
				header,
			}, []*types.Receipt{
				types.NewReceipt([]byte{}, false, 0),
			})

			err := manager.InsertBlock(block, nil)
			Expect(err).Should(Succeed())

			By("insert the same block again, should be ok")
			err = manager.InsertBlock(block, nil)
			Expect(err).Should(Succeed())
		})

		It("failed due to wrong signer", func() {
			manager := NewManager(db)
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

			err := manager.InsertBlock(block, nil)
			Expect(err).Should(Equal(common.ErrWrongSigner))
		})
	})

	Context("GetHeaderByNumber()", func() {
		It("gets the right header", func() {
			manager := NewManager(db)
			block1 := types.NewBlockWithHeader(&types.Header{
				Number: big.NewInt(100),
			})
			block2 := types.NewBlockWithHeader(&types.Header{
				Number: big.NewInt(99),
			})
			err := manager.InsertBlock(block1, nil)
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
			manager := NewManager(db)
			block1 := types.NewBlockWithHeader(&types.Header{
				Number: big.NewInt(100),
			})
			block2 := types.NewBlockWithHeader(&types.Header{
				Number: big.NewInt(99),
			})
			err := manager.InsertBlock(block1, nil)
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
			manager := NewManager(db)
			block1 := types.NewBlockWithHeader(&types.Header{
				Number: big.NewInt(100),
				Root:   gethCommon.StringToHash("1234567890"),
			})

			dump := &state.Dump{
				Root: fmt.Sprintf("%x", block1.Root()),
				Accounts: map[string]state.DumpAccount{
					// account
					common.StringToHex("account"): {
						Nonce:   100,
						Balance: "101",
					},
					// contract
					common.StringToHex("contract"): {
						Nonce:    900,
						Balance:  "901",
						Code:     "code",
						CodeHash: common.StringToHex("codeHash"),
						Storage: map[string]string{
							"key1": "storage1",
							"key2": "storage2",
						},
					},
				},
			}
			err := manager.UpdateState(block1, dump)
			Expect(err).Should(Succeed())

			By("update the same state again, should be ok")
			err = manager.UpdateState(block1, dump)
			Expect(err).Should(Succeed())
		})

		It("failed due to wrong signer", func() {
			manager := NewManager(db)
			block1 := types.NewBlockWithHeader(&types.Header{
				Number: big.NewInt(100),
				Root:   gethCommon.StringToHash("1234567890"),
			})

			dump := &state.Dump{
				Root: "wrong root",
			}
			err := manager.UpdateState(block1, dump)
			Expect(err).Should(Equal(common.ErrInconsistentRoot))
		})
	})

	Context("LatestStateBlock()", func() {
		It("gets the latest state block", func() {
			manager := NewManager(db)
			block1 := types.NewBlockWithHeader(&types.Header{
				Number: big.NewInt(100),
				Root:   gethCommon.StringToHash("1234567890"),
			})

			dump := &state.Dump{
				Root: fmt.Sprintf("%x", block1.Root()),
			}

			err := manager.UpdateState(block1, dump)
			Expect(err).Should(Succeed())

			block, err := manager.LatestStateBlock()
			Expect(err).Should(Succeed())
			Expect(block.Number).Should(Equal(block1.Number().Int64()))
		})
	})

	Context("DeleteDataFromBlock()", func() {
		It("deletes data from a block number", func() {
			manager := NewManager(db)
			for i := int64(100); i < 120; i++ {
				block := types.NewBlockWithHeader(&types.Header{
					Number: big.NewInt(i),
					Root:   gethCommon.StringToHash("1234567890"),
				})
				err := manager.InsertBlock(block, nil)
				Expect(err).Should(Succeed())

				dump := &state.Dump{
					Root: fmt.Sprintf("%x", block.Root()),
				}
				err = manager.UpdateState(block, dump)
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
