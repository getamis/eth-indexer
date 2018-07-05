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

package subscription

import (
	"os"
	"testing"
	"time"

	gethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/getamis/eth-indexer/common"
	"github.com/getamis/eth-indexer/model"
	"github.com/getamis/sirius/test"
	"github.com/jinzhu/gorm"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Database Test", func() {
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
		db.Delete(&model.Subscription{})
		db.Delete(&model.TotalBalance{})
	})

	Context("Subscription database", func() {
		It("should insert", func() {
			store := NewWithDB(db)
			data1 := &model.Subscription{
				BlockNumber: 100,
				Address:     common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A892"),
			}

			By("insert new subscription")
			duplicated, err := store.BatchInsert([]*model.Subscription{data1})
			Expect(err).Should(Succeed())
			Expect(len(duplicated)).Should(Equal(0))

			By("duplicated should be 1")
			duplicated, err = store.BatchInsert([]*model.Subscription{data1})
			Expect(err).Should(Succeed())
			Expect(len(duplicated)).Should(Equal(1))

			data2 := &model.Subscription{
				BlockNumber: 100,
				Address:     common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A893"),
			}

			By("insert another new subscription")
			duplicated, err = store.BatchInsert([]*model.Subscription{data2})
			Expect(len(duplicated)).Should(Equal(0))
			Expect(err).Should(Succeed())
		})

		It("should get subscriptions by block number", func() {
			store := NewWithDB(db)
			data1 := &model.Subscription{
				BlockNumber: 100,
				Group:       1,
				Address:     common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A892"),
			}
			data2 := &model.Subscription{
				BlockNumber: 100,
				Group:       2,
				Address:     common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A893"),
			}
			data3 := &model.Subscription{
				BlockNumber: 101,
				Group:       3,
				Address:     common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A894"),
			}
			By("insert three new subscriptions")
			data := []*model.Subscription{data1, data2, data3}
			_, err := store.BatchInsert(data)
			Expect(err).Should(Succeed())

			res, err := store.Find(data1.BlockNumber)
			Expect(err).Should(Succeed())
			Expect(len(res)).Should(BeNumerically("==", 2))

			res, err = store.Find(data3.BlockNumber)
			Expect(err).Should(Succeed())
			Expect(len(res)).Should(BeNumerically("==", 1))

			res, err = store.Find(0)
			Expect(err).Should(Succeed())
			Expect(len(res)).Should(BeZero())
		})

		It("should get subscriptions by addresses", func() {
			store := NewWithDB(db)
			data0 := &model.Subscription{
				BlockNumber: 0,
				Address:     common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A891"),
			}
			data1 := &model.Subscription{
				BlockNumber: 100,
				Address:     common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A892"),
			}
			data2 := &model.Subscription{
				BlockNumber: 100,
				Address:     common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A893"),
			}
			data3 := &model.Subscription{
				BlockNumber: 101,
				Address:     common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A894"),
			}
			By("insert three new subscriptions")
			data := []*model.Subscription{data1, data2, data3}
			_, err := store.BatchInsert(data)
			Expect(err).Should(Succeed())

			res, err := store.FindOldSubscriptions([][]byte{
				data0.Address,
				data1.Address,
				data2.Address,
				common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A895"),
			})
			Expect(err).Should(Succeed())
			Expect(len(res)).Should(BeNumerically("==", 2))

			res, err = store.FindOldSubscriptions([][]byte{
				data0.Address,
				data1.Address,
				common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A895"),
				common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A896"),
			})
			Expect(err).Should(Succeed())
			Expect(len(res)).Should(BeNumerically("==", 1))

			res, err = store.FindOldSubscriptions([][]byte{
				data0.Address,
				common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A895"),
				common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A896"),
				common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A897"),
			})
			Expect(err).Should(Succeed())
			Expect(len(res)).Should(BeZero())

			err = store.Reset(100, 102)
			Expect(err).Should(Succeed())

			res, err = store.FindOldSubscriptions([][]byte{
				data0.Address,
				data1.Address,
				data2.Address,
				data3.Address,
			})
			Expect(err).Should(Succeed())
			Expect(len(res)).Should(BeZero())
		})

		Context("FindByGroup", func() {
			var (
				store   Store
				groupID int64
			)

			BeforeEach(func() {
				store = NewWithDB(db)
				groupID = time.Now().UnixNano()
			})

			It("should get subscriptions by group id", func() {
				subs := []*model.Subscription{
					{
						Group:   groupID,
						Address: common.HexToBytes("0xdfbba377a6d55d26d7dc6acd28279dc1f31308ed"),
					},
					{
						Group:   groupID,
						Address: common.HexToBytes("0x52384b72f5582996d30f493ffc8518f6dc93f7c8"),
					},
				}

				By("Should be successful to insert", func() {
					_, err := store.BatchInsert(subs)
					Expect(err).Should(Succeed())
				})

				By("Should be successful to get subscriptions with page 1", func() {
					result, total, err := store.FindByGroup(groupID, &model.QueryParameters{
						Page:    1,
						Limit:   1,
						OrderBy: "created_at",
						Order:   "asc",
					})
					Expect(err).Should(Succeed())
					Expect(total).Should(Equal(uint64(len(subs))))
					Expect(len(result)).Should(Equal(1))
					Expect(result[0].Group).Should(Equal(groupID))
					Expect(result[0].Address).Should(Equal(subs[0].Address))
				})

				By("Should be successful to get subscriptions with page 2", func() {
					result, total, err := store.FindByGroup(groupID, &model.QueryParameters{
						Page:    2,
						Limit:   1,
						OrderBy: "created_at",
						Order:   "asc",
					})
					Expect(err).Should(Succeed())
					Expect(total).Should(Equal(uint64(len(subs))))
					Expect(len(result)).Should(Equal(1))
					Expect(result[0].Group).Should(Equal(groupID))
					Expect(result[0].Address).Should(Equal(subs[1].Address))
				})
			})

			It("should get empty subscriptions if group id doesn't exist", func() {
				result, total, err := store.FindByGroup(groupID, &model.QueryParameters{
					Page:    1,
					Limit:   1,
					OrderBy: "created_at",
					Order:   "asc",
				})
				Expect(err).Should(Succeed())
				Expect(total).Should(Equal(uint64(0)))
				Expect(len(result)).Should(Equal(0))
			})

		})

		It("update block number in batch", func() {
			store := NewWithDB(db)
			data1 := &model.Subscription{
				BlockNumber: 100,
				Address:     common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A892"),
			}
			data2 := &model.Subscription{
				BlockNumber: 100,
				Address:     common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A893"),
			}
			data3 := &model.Subscription{
				BlockNumber: 101,
				Address:     common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A894"),
			}
			By("insert three new subscriptions")
			data := []*model.Subscription{data1, data2, data3}
			_, err := store.BatchInsert(data)
			Expect(err).Should(Succeed())

			res, err := store.Find(0)
			Expect(err).Should(Succeed())
			Expect(len(res)).Should(BeNumerically("==", 0))

			err = store.BatchUpdateBlockNumber(0,
				[][]byte{data1.Address, data2.Address, data3.Address})
			Expect(err).Should(Succeed())

			res, err = store.Find(0)
			Expect(err).Should(Succeed())
			Expect(len(res)).Should(BeNumerically("==", 3))
		})
	})

	Context("Total balance database", func() {
		It("should insert", func() {
			store := NewWithDB(db)
			data1 := &model.TotalBalance{
				BlockNumber: 100,
				Token:       common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A892"),
				Group:       1,
				Balance:     "1000",
				TxFee:       "99",
			}

			By("insert new total balance")
			err := store.InsertTotalBalance(data1)
			Expect(err).Should(Succeed())

			By("failed to total balance again")
			err = store.InsertTotalBalance(data1)
			Expect(err).ShouldNot(BeNil())

			data2 := &model.TotalBalance{
				BlockNumber: 101,
				Token:       common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A892"),
				Group:       1,
				Balance:     "1000",
				TxFee:       "101",
			}

			By("insert another new subscription")
			err = store.InsertTotalBalance(data2)
			Expect(err).Should(Succeed())
		})

		It("should get total balances, then reset them", func() {
			store := NewWithDB(db)
			data1 := &model.TotalBalance{
				BlockNumber: 100,
				Token:       common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A892"),
				Group:       1,
				Balance:     "1000",
				TxFee:       "99",
			}
			data2 := &model.TotalBalance{
				BlockNumber: 101,
				Token:       common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A893"),
				Group:       2,
				Balance:     "1000",
				TxFee:       "99",
			}
			data3 := &model.TotalBalance{
				BlockNumber: 102,
				Token:       common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A894"),
				Group:       3,
				Balance:     "1000",
				TxFee:       "99",
			}
			By("insert three new total balances")
			data := []*model.TotalBalance{data1, data2, data3}
			for _, d := range data {
				err := store.InsertTotalBalance(d)
				Expect(err).Should(Succeed())
			}

			res, err := store.FindTotalBalance(data1.BlockNumber, gethCommon.BytesToAddress(data1.Token), data1.Group)
			Expect(err).Should(Succeed())
			Expect(res).Should(Equal(data1))

			res, err = store.FindTotalBalance(data2.BlockNumber, gethCommon.BytesToAddress(data2.Token), data2.Group)
			Expect(err).Should(Succeed())
			Expect(res).Should(Equal(data2))

			res, err = store.FindTotalBalance(data3.BlockNumber, gethCommon.BytesToAddress(data3.Token), data3.Group)
			Expect(err).Should(Succeed())
			Expect(res).Should(Equal(data3))

			err = store.Reset(100, 102)
			Expect(err).Should(Succeed())

			res, err = store.FindTotalBalance(data1.BlockNumber, gethCommon.BytesToAddress(data1.Token), data1.Group)
			data1.Balance = "0"
			data1.TxFee = "0"
			Expect(err).Should(Succeed())
			Expect(res).Should(Equal(data1))

			res, err = store.FindTotalBalance(data2.BlockNumber, gethCommon.BytesToAddress(data2.Token), data2.Group)
			data2.Balance = "0"
			data2.TxFee = "0"
			Expect(err).Should(Succeed())
			Expect(res).Should(Equal(data2))

			res, err = store.FindTotalBalance(data3.BlockNumber, gethCommon.BytesToAddress(data3.Token), data3.Group)
			data3.Balance = "0"
			data3.TxFee = "0"
			Expect(err).Should(Succeed())
			Expect(res).Should(Equal(data3))
		})
	})
})

func TestSubscription(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Subscription Test")
}
