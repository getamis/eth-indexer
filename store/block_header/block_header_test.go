package block_header

import (
	"os"
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/maichain/eth-indexer/service/pb"
	"github.com/maichain/mapi/base/test"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Block Header Database Test", func() {
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

	It("should get one new record in database", func() {
		store := NewWithDB(db)
		data := &pb.BlockHeader{
			ParentHash: "ParentHash",
		}
		out := &pb.BlockHeader{}

		err := store.Upsert(data, out)
		Expect(err).Should(Succeed())
		Expect(out).ShouldNot(BeNil())
		Expect(out.ParentHash).Should(Equal(data.ParentHash))
	})

	It("should get latest header via Query function", func() {
		store := NewWithDB(db)
		data1 := &pb.BlockHeader{
			ParentHash: "ParentHash",
			Number:     100,
		}
		data2 := &pb.BlockHeader{
			ParentHash: "ParentHash",
			Number:     50,
		}

		store.Upsert(data1, &pb.BlockHeader{})
		store.Upsert(data2, &pb.BlockHeader{})

		filter := &pb.BlockHeader{}
		options := &QueryOption{
			Limit:   1,
			OrderBy: "number",
			Order:   ORDER_DESC,
		}
		result, _, err := store.Query(filter, options)
		Expect(err).Should(Succeed())
		Expect(len(result)).Should(Equal(options.Limit))
		Expect(result[0].Number).Should(Equal(data1.Number))
	})

	It("should insert one new record in database", func() {
		By("insert new one header")
		store := NewWithDB(db)
		data := &pb.BlockHeader{
			Number: 10,
		}
		err := store.Insert(data)
		Expect(err).Should(Succeed())

		By("failed to insert again")
		err = store.Insert(data)
		Expect(err).ShouldNot(BeNil())
	})
})

func TestBlockHeader(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Block Header Database Test")
}
