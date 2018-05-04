package rpc

import (
	"github.com/maichain/eth-indexer/common"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrInvalidBlockNumber  = status.Error(codes.InvalidArgument, "invalid block number")
	ErrInvalidOffset       = status.Error(codes.InvalidArgument, "invalid offset")
	ErrBlockNotFound       = status.Error(codes.NotFound, "block not found")
	ErrTransactionNotFound = status.Error(codes.NotFound, "transaction not found")
	ErrInternal            = status.Error(codes.Internal, "internal server error")
)

func wrapTransactionNotFoundError(err error) error {
	if common.NotFoundError(err) {
		return ErrTransactionNotFound
	}

	return ErrInternal
}

func wrapBlockNotFoundError(err error) error {
	if common.NotFoundError(err) {
		return ErrBlockNotFound
	}

	return ErrInternal
}
