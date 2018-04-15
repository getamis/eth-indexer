package block_header

import (
	"github.com/jinzhu/gorm"
	"github.com/maichain/eth-indexer/service/pb"
	mpb "github.com/maichain/mapi/api/pb"
)

const (
	TableName = "block_headers"
)

type Store interface {
	Insert(data *pb.BlockHeader) error
	Upsert(data, result *pb.BlockHeader) error
	Find(filter *pb.BlockHeader) (result []*pb.BlockHeader, err error)
	Query(filter interface{}, queryOpt *QueryOption) (result []*pb.BlockHeader, pag *mpb.Pagination, err error)
	// Last retruns the header with the greatest number
	Last() (result *pb.BlockHeader, err error)
}

type store struct {
	db *gorm.DB
}

func NewWithDB(db *gorm.DB) Store {
	return &store{
		db: db.Table(TableName),
	}
}

func (t *store) Insert(data *pb.BlockHeader) error {
	return t.db.Create(data).Error
}

func (t *store) Upsert(data, result *pb.BlockHeader) error {
	filter := pb.BlockHeader{Number: data.Number}
	return t.db.Where(filter).Attrs(data).FirstOrCreate(result).Error
}

func (t *store) Find(filter *pb.BlockHeader) (result []*pb.BlockHeader, err error) {
	err = t.db.Where(filter).Find(&result).Error
	return
}

// Get returns records matched filter condition and query options.
func (t *store) Query(filter interface{}, queryOpt *QueryOption) (result []*pb.BlockHeader, pag *mpb.Pagination, err error) {
	var total int64
	offset := queryOpt.Limit * (queryOpt.Page - 1)

	db := t.db
	db = db.Where(filter)

	err = db.Count(&total).Error
	if err != nil {
		return
	}

	if queryOpt != nil {
		if orderBy := queryOpt.OrderString(); len(orderBy) > 0 {
			db = db.Order(orderBy)
		}
		if queryOpt.Limit > 0 {
			db = db.Limit(queryOpt.Limit)
		}
		if offset > 0 {
			db = db.Offset(offset)
		}
	}

	err = db.Find(&result).Error
	if err != nil {
		return
	}

	pag = &mpb.Pagination{
		Page:       uint64(queryOpt.Page),
		Limit:      uint64(queryOpt.Limit),
		Order:      queryOpt.OrderString(),
		TotalCount: uint64(total),
	}
	return
}

func (t *store) Last() (result *pb.BlockHeader, err error) {
	result = &pb.BlockHeader{}
	err = t.db.Order("number DESC").Limit(1).Find(result).Error
	return
}
