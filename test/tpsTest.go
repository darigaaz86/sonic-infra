package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Account struct {
	Index      int    `json:"index"`
	Address    string `json:"address"`
	PrivateKey string `json:"privateKey"`
}

const (
	httpTimeout       = 10 * time.Second
	maxRetries        = 3
	retryDelay        = 500 * time.Millisecond
	maxConcurrentSend = 100 // Tune this depending on your ulimit/socket
	ACCOUNTS         = "accounts2.json"
	RPC_URL          = "http://13.212.68.50:8545"
	TARGET           = "0x3e6afd653d62d7d797c7c2a0a427a17a6457c56d"
	CHAIN_ID         = 0xfa5
	INTERVAL_MS      = 500
	BATCH_TX_COUNT   = 500
	TOTAL_DURATION_MS = 6000000
	GAS_TIP_GWEI      = 1 // dynamic tip
)

func loadAccounts(file string) ([]Account, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var accounts []Account
	if err := json.Unmarshal(data, &accounts); err != nil {
		return nil, err
	}
	return accounts, nil
}

func groupAccounts(accounts []Account, size int) [][]Account {
	var groups [][]Account
	for i := 0; i < len(accounts); i += size {
		end := i + size
		if end > len(accounts) {
			end = len(accounts)
		}
		groups = append(groups, accounts[i:end])
	}
	return groups
}

func getDynamicGasParams(client *ethclient.Client) (*big.Int, *big.Int, error) {
	block, err := client.BlockByNumber(context.Background(), nil)
	if err != nil {
		return nil, nil, err
	}
	baseFee := block.BaseFee()
	tip := big.NewInt(1e9 * GAS_TIP_GWEI) // 1 gwei
	// Add buffer to base fee (e.g. 1.5x)
	buffer := big.NewInt(15)
	bufferedBaseFee := new(big.Int).Mul(baseFee, buffer)
	bufferedBaseFee.Div(bufferedBaseFee, big.NewInt(10))
	feeCap := new(big.Int).Add(bufferedBaseFee, tip)
	return feeCap, tip, nil
}

func sendTransactionWithRetry(client *ethclient.Client, tx *types.Transaction) error {
	var err error
	for attempt := 0; attempt < maxRetries; attempt++ {
		err = client.SendTransaction(context.Background(), tx)
		if err == nil {
			return nil
		}
		time.Sleep(retryDelay)
	}
	return fmt.Errorf("failed after %d retries: %w", maxRetries, err)
}

func prepareAndSendGroup(group []Account, nonceMap map[string]uint64, gasFeeCap, gasTipCap *big.Int, client *ethclient.Client, txCh chan<- int, mu *sync.Mutex, sem chan struct{}) {
	var wg sync.WaitGroup
	for _, acc := range group {
		wg.Add(1)
		sem <- struct{}{} // limit concurrent goroutines
		go func(acc Account) {
			defer func() {
				<-sem
				wg.Done()
			}()

			key, err := crypto.HexToECDSA(acc.PrivateKey[2:])
			if err != nil {
				log.Printf("❌ Invalid private key: %v", err)
				return
			}

			mu.Lock()
			nonce := nonceMap[acc.Address]
			mu.Unlock()

			to := common.HexToAddress(TARGET)
			tx := types.NewTx(&types.DynamicFeeTx{
				ChainID:   big.NewInt(CHAIN_ID),
				Nonce:     nonce,
				GasTipCap: gasTipCap,
				GasFeeCap: gasFeeCap,
				Gas:       21000,
				To:        &to,
				Value:     big.NewInt(1e15),
			})

			signedTx, err := types.SignTx(tx, types.LatestSignerForChainID(big.NewInt(CHAIN_ID)), key)
			if err != nil {
				log.Printf("❌ SignTx error: %v", err)
				return
			}

			err = sendTransactionWithRetry(client, signedTx)
			if err == nil {
    			mu.Lock()
    			nonceMap[acc.Address] = nonce + 1
    			mu.Unlock()
			} else {
    			log.Printf("❌ SendTransaction failed: %v", err)
    			// do NOT increment nonce on failure
			}

			txCh <- 1
		}(acc)
	}
	wg.Wait()
}

func runBatchLoop(groups [][]Account, nonceMap map[string]uint64, client *ethclient.Client) {
	sent := 0
	batchIndex := 0
	startTime := time.Now()
	ticker := time.NewTicker(time.Duration(INTERVAL_MS) * time.Millisecond)
	txCh := make(chan int, 10000)
	sem := make(chan struct{}, maxConcurrentSend) // semaphore
	var mu sync.Mutex

	go func() {
		for txCount := range txCh {
			sent += txCount
		}
	}()

	for now := range ticker.C {
		if now.Sub(startTime).Milliseconds() >= TOTAL_DURATION_MS {
			ticker.Stop()
			close(txCh)
			totalTime := time.Since(startTime).Seconds()
			fmt.Printf("\n✅ Done. Total txs: %d | Time: %.2fs | TPS: %.2f\n", sent, totalTime, float64(sent)/totalTime)
			break
		}

		gasFeeCap, gasTipCap, err := getDynamicGasParams(client)
		if err != nil {
			fmt.Printf("❌ Failed to fetch gas params: %v\n", err)
			continue
		}

		group := groups[batchIndex%len(groups)]
		accRange := fmt.Sprintf("[%d - %d]", batchIndex*BATCH_TX_COUNT+1, batchIndex*BATCH_TX_COUNT+len(group))
		timestamp := time.Now().Format("2006-01-02T15:04:05.000Z07:00")

		go prepareAndSendGroup(group, nonceMap, gasFeeCap, gasTipCap, client, txCh, &mu, sem)

		avgTps := float64(sent) / time.Since(startTime).Seconds()
		fmt.Printf("📦 [%s] Batch %d %s: 🚀 Sent %d txs | Avg TPS: %.2f | baseFee: %s\n",
			timestamp, batchIndex+1, accRange, len(group), avgTps, gasFeeCap.String())

		batchIndex++
	}
}

func main() {
	accounts, err := loadAccounts(ACCOUNTS)
	if err != nil {
		log.Fatal(err)
	}

	client, err := ethclient.Dial(RPC_URL)
	if err != nil {
		log.Fatal(err)
	}

	groups := groupAccounts(accounts, BATCH_TX_COUNT)
	nonceMap := make(map[string]uint64)
	ctx := context.Background()
	for _, acc := range accounts {
		nonce, err := client.PendingNonceAt(ctx, common.HexToAddress(acc.Address))
		if err != nil {
			log.Fatalf("Failed to get nonce for %s: %v", acc.Address, err)
		}
		nonceMap[acc.Address] = nonce
	}

	runBatchLoop(groups, nonceMap, client)
}