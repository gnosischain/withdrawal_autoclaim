package main

import (
	"github.com/ethereum/go-ethereum/common"
)

func accountsToAddresses(accounts []*Account) []common.Address {
	var addresses []common.Address
	for _, a := range accounts {
		addresses = append(addresses, a.toAddress())
	}
	return addresses
}
