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

package service

import (
	"github.com/getamis/eth-indexer/common"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrInvalidToken        = status.Error(codes.InvalidArgument, "invalid token")
	ErrInvalidAddress      = status.Error(codes.InvalidArgument, "invalid address")
	ErrInvalidHash         = status.Error(codes.InvalidArgument, "invalid hash")
	ErrInvalidBlockNumber  = status.Error(codes.InvalidArgument, "invalid block number")
	ErrInvalidOffset       = status.Error(codes.InvalidArgument, "invalid offset")
	ErrBlockNotFound       = status.Error(codes.NotFound, "block not found")
	ErrTransactionNotFound = status.Error(codes.NotFound, "transaction not found")
	ErrInternal            = status.Error(codes.Internal, "internal server error")
)

func WrapTransactionNotFoundError(err error) error {
	if common.NotFoundError(err) {
		return ErrTransactionNotFound
	}

	return ErrInternal
}

func WrapBlockNotFoundError(err error) error {
	if common.NotFoundError(err) {
		return ErrBlockNotFound
	}

	return ErrInternal
}
