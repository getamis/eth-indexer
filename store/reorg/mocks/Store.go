// Code generated by mockery v1.0.0
package mocks

import context "context"
import mock "github.com/stretchr/testify/mock"
import model "github.com/getamis/eth-indexer/model"

// Store is an autogenerated mock type for the Store type
type Store struct {
	mock.Mock
}

// Insert provides a mock function with given fields: ctx, data
func (_m *Store) Insert(ctx context.Context, data *model.Reorg) error {
	ret := _m.Called(ctx, data)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *model.Reorg) error); ok {
		r0 = rf(ctx, data)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// List provides a mock function with given fields: ctx
func (_m *Store) List(ctx context.Context) ([]*model.Reorg, error) {
	ret := _m.Called(ctx)

	var r0 []*model.Reorg
	if rf, ok := ret.Get(0).(func(context.Context) []*model.Reorg); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*model.Reorg)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
