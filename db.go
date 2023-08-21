package main

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
)

type Account struct {
	address string `` // add appropriate tag to scan from db
}

func (a *Account) toAddress() common.Address {
	return common.HexToAddress(a.address)
}

type DB interface {
	GetAll(context.Context) ([]*Account, error)
	GetByClaimFrequency(context.Context, uint64) ([]*Account, error)

	GetClaimPeriods(context.Context) ([]uint64, error)

	Write(context.Context, *Account) error
	WriteBatch(context.Context, []*Account) error
}

func NewDB() (DB, error) {
	return nil, nil
}
