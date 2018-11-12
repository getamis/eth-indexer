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

package reorg

import (
	"context"
	"testing"

	"github.com/getamis/eth-indexer/model"
	"github.com/getamis/eth-indexer/store/sqldb"
	"github.com/getamis/sirius/test"
	"github.com/jmoiron/sqlx"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Reorg Database Test", func() {
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
		_, err := db.Exec("DELETE FROM reorgs")
		Expect(err).Should(Succeed())
	})

	It("should insert", func() {
		store := NewWithDB(db)

		data1 := &model.Reorg{
			From:     100,
			FromHash: []byte("hash1"),
			To:       110,
			ToHash:   []byte("hash2"),
		}

		By("insert new reorg")
		err := store.Insert(ctx, data1)
		Expect(err).Should(Succeed())

		By("insert the reorg again")
		err = store.Insert(ctx, data1)
		Expect(err).Should(Succeed())

		By("check reorgs size")
		rs, err := store.List(ctx)
		Expect(err).Should(Succeed())
		Expect(len(rs)).Should(BeNumerically("==", 2))
	})
})

func TestReorg(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Reorg Test")
}
