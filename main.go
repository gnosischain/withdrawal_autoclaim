package main

import (
	"context"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
)

const rpcUrl = "https://rpc.gnosischain.com"

var (
	BatchSize int

	DepositContractAddress common.Address
	TokenContractAddress   common.Address

	PrivateKey string

	metrics *Metrics

	db DB
)

func init() {
	var err error

	DepositContractAddress = common.HexToAddress(os.Getenv("DEPOSIT_CONTRACT_ADDRESS"))
	if DepositContractAddress == common.HexToAddress("") {
		log.Fatal("no DEPOSIT_CONTRACT_ADDRESS provided")
	}
	TokenContractAddress = common.HexToAddress(os.Getenv("TOKEN_CONTRACT_ADDRESS"))
	if TokenContractAddress == common.HexToAddress("") {
		log.Fatal("no TOKEN_CONTRACT_ADDRESS provided")
	}
	PrivateKey = os.Getenv("PRIVATE_KEY")
	if PrivateKey == "" {
		log.Fatal("no PRIVATE_KEY provided")
	}
	BatchSize, err = strconv.Atoi(os.Getenv("BATCH_SIZE"))
	if err != nil {
		log.Fatal("invalid batch size value provided")
	}
	if BatchSize == 0 {
		BatchSize = 1000
	}

	metrics = NewMetrics()
	db, err = NewDB()
	if err != nil {
		log.Fatal("can't connect to db")
	}
}

func Run(ctx context.Context, client *ethclient.Client, periods []uint64) {
	for _, p := range periods {
		go func(ctx context.Context, client *ethclient.Client, period uint64) {
			for {
				select {
				case <-ctx.Done():
					return
				case <-time.Tick(time.Duration(period) * time.Second):
					accounts, err := db.GetByClaimFrequency(ctx, period)
					if err != nil {
						// TODO:
						log.Errorf("can't get accounts from db: %s", err.Error())
						continue
					}
					if len(accounts) == 0 {
						continue
					}
					if err := claimBatches(client, accountsToAddresses(accounts)); err != nil {
						// TODO: add retry logic ?
						log.Errorf("can't claim withdrawals: %s", err.Error())
						continue
					}
				}
			}
		}(ctx, client, p)
	}
}

func main() {
	client, err := ethclient.Dial(rpcUrl)
	if err != nil {
		log.Fatal(err)
	}
	// parent context, all goroutines will stop on stop() call
	ctx, stop := context.WithCancel(context.Background())

	go metrics.serve(ctx)
	periods, err := db.GetClaimPeriods(ctx)
	if err != nil {
		log.Fatal("can't get claim periods from db")
	}
	go Run(ctx, client, periods)

	c := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(c, os.Interrupt)

	// Block until receive SIGINT (kill -2)
	<-c
	stop()
}
