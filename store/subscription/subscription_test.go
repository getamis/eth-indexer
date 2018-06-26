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
			err := store.Insert(data1)
			Expect(err).Should(Succeed())

			By("failed to subscription again")
			err = store.Insert(data1)
			Expect(err).ShouldNot(BeNil())

			data2 := &model.Subscription{
				BlockNumber: 100,
				Address:     common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A893"),
			}

			By("insert another new subscription")
			err = store.Insert(data2)
			Expect(err).Should(Succeed())

			By("Update subscriptions")
			data2.BlockNumber = 10000
			err = store.UpdateBlockNumber(data2)
			Expect(err).Should(Succeed())

			subs, err := store.Find(data2.BlockNumber)
			Expect(err).Should(Succeed())
			Expect(len(subs)).Should(Equal(1))
			Expect(subs[0].Address).Should(Equal(data2.Address))
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
			for _, d := range data {
				err := store.Insert(d)
				Expect(err).Should(Succeed())
			}

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
			for _, d := range data {
				err := store.Insert(d)
				Expect(err).Should(Succeed())
			}
			res, err := store.FindByAddresses([][]byte{
				data0.Address,
				data1.Address,
				data2.Address,
				common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A895"),
			})
			Expect(err).Should(Succeed())
			Expect(len(res)).Should(BeNumerically("==", 2))

			res, err = store.FindByAddresses([][]byte{
				data0.Address,
				data1.Address,
				common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A895"),
				common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A896"),
			})
			Expect(err).Should(Succeed())
			Expect(len(res)).Should(BeNumerically("==", 1))

			res, err = store.FindByAddresses([][]byte{
				data0.Address,
				common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A895"),
				common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A896"),
				common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A897"),
			})
			Expect(err).Should(Succeed())
			Expect(len(res)).Should(BeZero())

			err = store.Reset(100, 102)
			Expect(err).Should(Succeed())

			res, err = store.FindByAddresses([][]byte{
				data0.Address,
				data1.Address,
				data2.Address,
				data3.Address,
			})
			Expect(err).Should(Succeed())
			Expect(len(res)).Should(BeZero())
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
			}
			data2 := &model.TotalBalance{
				BlockNumber: 101,
				Token:       common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A893"),
				Group:       2,
				Balance:     "1000",
			}
			data3 := &model.TotalBalance{
				BlockNumber: 102,
				Token:       common.HexToBytes("0xB287a379e6caCa6732E50b88D23c290aA990A894"),
				Group:       3,
				Balance:     "1000",
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
			Expect(err).Should(Succeed())
			Expect(res).Should(Equal(data1))

			res, err = store.FindTotalBalance(data2.BlockNumber, gethCommon.BytesToAddress(data2.Token), data2.Group)
			data2.Balance = "0"
			Expect(err).Should(Succeed())
			Expect(res).Should(Equal(data2))

			res, err = store.FindTotalBalance(data3.BlockNumber, gethCommon.BytesToAddress(data3.Token), data3.Group)
			data3.Balance = "0"
			Expect(err).Should(Succeed())
			Expect(res).Should(Equal(data3))
		})
	})
})

func TestSubscription(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Subscription Test")
}
