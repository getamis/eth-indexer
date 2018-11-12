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
	"context"
	"testing"
	"time"

	gethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/getamis/eth-indexer/common"
	"github.com/getamis/eth-indexer/model"
	"github.com/getamis/eth-indexer/store/sqldb"
	"github.com/getamis/sirius/test"
	"github.com/jmoiron/sqlx"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Database Test", func() {
	var (
		mysql *test.MySQLContainer
		db    *sqlx.DB
		ctx   = context.Background()
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

	AfterSuite(func() {
		mysql.Stop()
	})

	BeforeEach(func() {
		_, err := db.Exec("DELETE FROM subscriptions")
		Expect(err).Should(Succeed())
		_, err = db.Exec("DELETE FROM total_balances")
		Expect(err).Should(Succeed())
	})

	Context("Subscription database", func() {
		It("should insert", func() {
			store := NewWithDB(db)
			data1 := &model.Subscription{
				BlockNumber: 100,
				Address:     common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A892"),
			}

			By("insert new subscription")
			duplicated, err := store.BatchInsert(ctx, []*model.Subscription{data1})
			Expect(err).Should(Succeed())
			Expect(len(duplicated)).Should(Equal(0))

			By("duplicated should be 1")
			duplicated, err = store.BatchInsert(ctx, []*model.Subscription{data1})
			Expect(err).Should(Succeed())
			Expect(len(duplicated)).Should(Equal(1))

			data2 := &model.Subscription{
				BlockNumber: 100,
				Address:     common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A893"),
			}

			By("insert another new subscription")
			duplicated, err = store.BatchInsert(ctx, []*model.Subscription{data2})
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
			_, err := store.BatchInsert(ctx, data)
			Expect(err).Should(Succeed())

			res, total, err := store.Find(ctx, data1.BlockNumber, &model.QueryParameters{
				Page:  1,
				Limit: 1,
			})
			Expect(err).Should(Succeed())
			Expect(len(res)).Should(BeNumerically("==", 1))
			Expect(total).Should(BeNumerically("==", 2))

			res, total, err = store.Find(ctx, data3.BlockNumber, &model.QueryParameters{
				Page:  1,
				Limit: 1,
			})
			Expect(err).Should(Succeed())
			Expect(len(res)).Should(BeNumerically("==", 1))
			Expect(total).Should(BeNumerically("==", 1))

			res, total, err = store.Find(ctx, 0, &model.QueryParameters{
				Page:  1,
				Limit: 1,
			})
			Expect(err).Should(Succeed())
			Expect(len(res)).Should(BeZero())
			Expect(total).Should(BeZero())
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
			_, err := store.BatchInsert(ctx, data)
			Expect(err).Should(Succeed())

			res, err := store.FindOldSubscriptions(ctx, [][]byte{
				data0.Address,
				data1.Address,
				data2.Address,
				common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A895"),
			})
			Expect(err).Should(Succeed())
			Expect(len(res)).Should(BeNumerically("==", 2))

			res, err = store.FindOldSubscriptions(ctx, [][]byte{
				data0.Address,
				data1.Address,
				common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A895"),
				common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A896"),
			})
			Expect(err).Should(Succeed())
			Expect(len(res)).Should(BeNumerically("==", 1))

			res, err = store.FindOldSubscriptions(ctx, [][]byte{
				data0.Address,
				common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A895"),
				common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A896"),
				common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A897"),
			})
			Expect(err).Should(Succeed())
			Expect(len(res)).Should(BeZero())

			err = store.Reset(ctx, 100, 102)
			Expect(err).Should(Succeed())

			res, err = store.FindOldSubscriptions(ctx, [][]byte{
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
					_, err := store.BatchInsert(ctx, subs)
					Expect(err).Should(Succeed())
				})

				By("Should be successful to get subscriptions with page 1", func() {
					result, total, err := store.FindByGroup(ctx, groupID, &model.QueryParameters{
						Page:  1,
						Limit: 1,
					})
					Expect(err).Should(Succeed())
					Expect(total).Should(Equal(uint64(len(subs))))
					Expect(len(result)).Should(Equal(1))
					Expect(result[0].Group).Should(Equal(groupID))
					Expect(result[0].Address).Should(Equal(subs[0].Address))
				})

				By("Should be successful to get subscriptions with page 2", func() {
					result, total, err := store.FindByGroup(ctx, groupID, &model.QueryParameters{
						Page:  2,
						Limit: 1,
					})
					Expect(err).Should(Succeed())
					Expect(total).Should(Equal(uint64(len(subs))))
					Expect(len(result)).Should(Equal(1))
					Expect(result[0].Group).Should(Equal(groupID))
					Expect(result[0].Address).Should(Equal(subs[1].Address))
				})
			})

			It("should get empty subscriptions if group id doesn't exist", func() {
				result, total, err := store.FindByGroup(ctx, groupID, &model.QueryParameters{
					Page:  1,
					Limit: 1,
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
			_, err := store.BatchInsert(ctx, data)
			Expect(err).Should(Succeed())

			res, total, err := store.Find(ctx, 0, &model.QueryParameters{
				Page:  1,
				Limit: 100,
			})
			Expect(err).Should(Succeed())
			Expect(len(res)).Should(BeNumerically("==", 0))
			Expect(total).Should(BeNumerically("==", 0))

			err = store.BatchUpdateBlockNumber(ctx, 0,
				[][]byte{data1.Address, data2.Address, data3.Address})
			Expect(err).Should(Succeed())

			res, total, err = store.Find(ctx, 0, &model.QueryParameters{
				Page:  1,
				Limit: 100,
			})
			Expect(err).Should(Succeed())
			Expect(len(res)).Should(BeNumerically("==", 3))
			Expect(total).Should(BeNumerically("==", 3))
		})
	})

	Context("ListOldSubscriptions", func() {
		var (
			store Store
		)

		BeforeEach(func() {
			store = NewWithDB(db)
		})

		It("should get subscriptions", func() {
			subs := []*model.Subscription{
				{
					Group:       1,
					Address:     common.HexToBytes("0xdfbba377a6d55d26d7dc6acd28279dc1f31308ed"),
					BlockNumber: 100,
				},
				{
					Group:       2,
					Address:     common.HexToBytes("0x52384b72f5582996d30f493ffc8518f6dc93f7c8"),
					BlockNumber: 101,
				},
				// Cannot get the new subscription
				{
					Group:       3,
					Address:     common.HexToBytes("0x52384b72f5582996d30f493ffc8518f6dc93f7c9"),
					BlockNumber: 0,
				},
			}

			By("Should be successful to insert", func() {
				_, err := store.BatchInsert(ctx, subs)
				Expect(err).Should(Succeed())
			})

			By("Should be successful to get subscriptions with page 1", func() {
				result, total, err := store.ListOldSubscriptions(ctx, &model.QueryParameters{
					Page:  1,
					Limit: 1,
				})
				Expect(err).Should(Succeed())
				Expect(total).Should(Equal(uint64(2)))
				Expect(len(result)).Should(Equal(1))
				Expect(result[0].Group).Should(Equal(subs[0].Group))
				Expect(result[0].Address).Should(Equal(subs[0].Address))
			})

			By("Should be successful to get subscriptions with page 2", func() {
				result, total, err := store.ListOldSubscriptions(ctx, &model.QueryParameters{
					Page:  2,
					Limit: 1,
				})
				Expect(err).Should(Succeed())
				Expect(total).Should(Equal(uint64(2)))
				Expect(len(result)).Should(Equal(1))
				Expect(result[0].Group).Should(Equal(subs[1].Group))
				Expect(result[0].Address).Should(Equal(subs[1].Address))
			})

			By("Should be successful to get subscriptions with page 3", func() {
				result, total, err := store.ListOldSubscriptions(ctx, &model.QueryParameters{
					Page:  3,
					Limit: 1,
				})
				Expect(err).Should(Succeed())
				Expect(total).Should(Equal(uint64(2)))
				Expect(len(result)).Should(BeZero())
			})
		})

		It("should get empty subscriptions", func() {
			result, total, err := store.ListOldSubscriptions(ctx, &model.QueryParameters{
				Page:  1,
				Limit: 1,
			})
			Expect(err).Should(Succeed())
			Expect(total).Should(Equal(uint64(0)))
			Expect(len(result)).Should(Equal(0))
		})

	})

	Context("Total balance database", func() {
		It("should insert", func() {
			store := NewWithDB(db)
			data1 := &model.TotalBalance{
				BlockNumber:  100,
				Token:        common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A892"),
				Group:        1,
				Balance:      "1000",
				TxFee:        "99",
				MinerReward:  "0",
				UnclesReward: "0",
			}

			By("insert new total balance")
			err := store.InsertTotalBalance(ctx, data1)
			Expect(err).Should(Succeed())

			By("failed to total balance again")
			err = store.InsertTotalBalance(ctx, data1)
			Expect(err).ShouldNot(BeNil())

			data2 := &model.TotalBalance{
				BlockNumber:  101,
				Token:        common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A892"),
				Group:        1,
				Balance:      "1000",
				TxFee:        "101",
				MinerReward:  "0",
				UnclesReward: "0",
			}

			By("insert another new subscription")
			err = store.InsertTotalBalance(ctx, data2)
			Expect(err).Should(Succeed())
		})

		It("should get total balances, then reset them", func() {
			store := NewWithDB(db)
			data1 := &model.TotalBalance{
				BlockNumber:  100,
				Token:        common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A892"),
				Group:        1,
				Balance:      "1000",
				TxFee:        "99",
				MinerReward:  "0",
				UnclesReward: "0",
			}
			data2 := &model.TotalBalance{
				BlockNumber:  101,
				Token:        common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A893"),
				Group:        2,
				Balance:      "1000",
				TxFee:        "99",
				MinerReward:  "0",
				UnclesReward: "0",
			}
			data3 := &model.TotalBalance{
				BlockNumber:  102,
				Token:        common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A894"),
				Group:        3,
				Balance:      "1000",
				TxFee:        "99",
				MinerReward:  "0",
				UnclesReward: "0",
			}
			By("insert three new total balances")
			data := []*model.TotalBalance{data1, data2, data3}
			for _, d := range data {
				err := store.InsertTotalBalance(ctx, d)
				Expect(err).Should(Succeed())
			}

			res, err := store.FindTotalBalance(ctx, data1.BlockNumber, gethCommon.BytesToAddress(data1.Token), data1.Group)
			Expect(err).Should(Succeed())
			Expect(res).Should(Equal(data1))

			res, err = store.FindTotalBalance(ctx, data2.BlockNumber, gethCommon.BytesToAddress(data2.Token), data2.Group)
			Expect(err).Should(Succeed())
			Expect(res).Should(Equal(data2))

			res, err = store.FindTotalBalance(ctx, data3.BlockNumber, gethCommon.BytesToAddress(data3.Token), data3.Group)
			Expect(err).Should(Succeed())
			Expect(res).Should(Equal(data3))

			// Find total balance in a large block number should return the latest one
			res, err = store.FindTotalBalance(ctx, 999999, gethCommon.BytesToAddress(data3.Token), data3.Group)
			Expect(err).Should(Succeed())
			Expect(res).Should(Equal(data3))

			err = store.Reset(ctx, 100, 102)
			Expect(err).Should(Succeed())

			res, err = store.FindTotalBalance(ctx, data1.BlockNumber, gethCommon.BytesToAddress(data1.Token), data1.Group)
			Expect(err).ShouldNot(Succeed())
			Expect(res).Should(BeNil())

			res, err = store.FindTotalBalance(ctx, data2.BlockNumber, gethCommon.BytesToAddress(data2.Token), data2.Group)
			Expect(err).ShouldNot(Succeed())
			Expect(res).Should(BeNil())

			res, err = store.FindTotalBalance(ctx, data3.BlockNumber, gethCommon.BytesToAddress(data3.Token), data3.Group)
			Expect(err).ShouldNot(Succeed())
			Expect(res).Should(BeNil())
		})
	})
})

func TestSubscription(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Subscription Test")
}
