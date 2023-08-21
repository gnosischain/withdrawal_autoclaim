package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gnosischain/withdrawal_autoclaim/bindings"
	log "github.com/sirupsen/logrus"
)

// claimBatches сalls claim for every batch of withdrawal addresses
// where number of batches == (withdrawal address / BatchSize).
func claimBatches(client *ethclient.Client, addrs []common.Address) error {
	batches := len(addrs) / BatchSize

	for i := 0; i <= batches; i++ {
		var list []common.Address

		if len(addrs) < BatchSize {
			list = addrs
		} else {
			list = addrs[:BatchSize]
			addrs = addrs[BatchSize:]
		}
		if err := claim(client, list); err != nil {
			return fmt.Errorf("can't claim withdrawals: %w", err)
		}
	}
	// sets addressesCounter (withdrawalAddresses) metric
	// to the number of unique withdrawal addresses
	metrics.addressesCounter = uint64(len(addrs))
	return nil
}

// claim send claimWithdrawals tx to deposit contract with provided list of addresses.
func claim(client *ethclient.Client, addrs []common.Address) error {
	opts, err := txOpts(client)
	if err != nil {
		return fmt.Errorf("transaction options build error: %w", err)
	}
	contract, err := bindings.NewDeposit(DepositContractAddress, client)
	if err != nil {
		return fmt.Errorf("can't create deposit contract binding: %w", err)
	}

	tx, err := contract.ClaimWithdrawals(opts, addrs)
	if err != nil {
		return fmt.Errorf("claim tx error: %w", err)
	}
	log.Infof("sent claim tx hash: %s", tx.Hash())
	r, err := bind.WaitMined(context.Background(), client, tx)
	if err != nil {
		return fmt.Errorf("can't cant get tx receipt: %w", err)
	}
	if r.Status == 0 {
		return fmt.Errorf("tx reverted: hash %s source: %v", r.TxHash.String(), r)
	}
	log.Infof("tx mined on block: %d", r.BlockNumber.Uint64())
	return nil
}

// txOpts creates tx signer from private key and fills tx options.
func txOpts(client *ethclient.Client) (*bind.TransactOpts, error) {
	privateKey, err := crypto.HexToECDSA(PrivateKey)
	if err != nil {
		log.Fatal(err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("can't cast public key to ECDSA")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		return nil, fmt.Errorf("can't get account nonce: %w", err)
	}
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return nil, fmt.Errorf("can't get gas price: %w", err)
	}
	chainID, err := client.ChainID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("can't get chain id: %w", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		return nil, fmt.Errorf("can't create transactor: %w", err)
	}
	auth.Nonce = big.NewInt(int64(nonce))
	auth.Value = big.NewInt(0) // in wei
	auth.GasLimit = uint64(0)  // estimate
	auth.GasPrice = gasPrice

	return auth, nil
}
