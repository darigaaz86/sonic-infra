package main

import (
	"sync/atomic"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"
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
	Mnemonic   string `json:"mnemonic"`
}

const (
	rpcURL        = "http://13.212.159.55:8545"
	chainID       = 0xfa5 // Replace with actual Sonic chain ID
	gasLimit      = uint64(21000)
	GAS_TIP_GWEI  = int64(2) // Tip: 2 Gwei
	valueStr      = "1000000000000000000000"
	concurrency   = 100
	txsPerSender  = 100
)

func loadAccounts(filename string) ([]Account, error) {
	file, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var accounts []Account
	decoder := json.NewDecoder(strings.NewReader(string(file)))
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&accounts)
	return accounts, err
}

func getDynamicGasParams(client *ethclient.Client) (*big.Int, *big.Int, error) {
	block, err := client.BlockByNumber(context.Background(), nil)
	if err != nil {
		return nil, nil, err
	}
	baseFee := block.BaseFee()
	tip := new(big.Int).Mul(big.NewInt(GAS_TIP_GWEI), big.NewInt(1e9)) // Gwei to Wei
	buffer := big.NewInt(15)
	bufferedBaseFee := new(big.Int).Mul(baseFee, buffer)
	bufferedBaseFee.Div(bufferedBaseFee, big.NewInt(10)) // 1.5x baseFee
	feeCap := new(big.Int).Add(bufferedBaseFee, tip)
	return feeCap, tip, nil
}

func sendTx(client *ethclient.Client, from Account, to string, nonce uint64, feeCap, tip *big.Int) error {
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(from.PrivateKey, "0x"))
	if err != nil {
		return err
	}

	toAddress := common.HexToAddress(to)
	valueToSend := new(big.Int)
	value, _ := valueToSend.SetString(valueStr, 10)
	new(big.Int).SetString(valueStr, 10)
	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   big.NewInt(chainID),
		Nonce:     nonce,
		GasFeeCap: feeCap,
		GasTipCap: tip,
		Gas:       gasLimit,
		To:        &toAddress,
		Value:     value,
	})

	signer := types.NewLondonSigner(big.NewInt(chainID))
	signedTx, err := types.SignTx(tx, signer, privateKey)
	if err != nil {
		return err
	}

	return client.SendTransaction(context.Background(), signedTx)
}

func main() {
	var transferCount uint64
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	senders, err := loadAccounts("accountsGenesis.json")
	if err != nil {
		log.Fatal("Failed to load funded accounts:", err)
	}

	receivers, err := loadAccounts("accountsTest.json")
	if err != nil {
		log.Fatal("Failed to load recipient accounts:", err)
	}

	if len(receivers) < len(senders) {
		log.Fatal("Not enough recipient accounts")
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)

	for i, sender := range senders {
		wg.Add(1)
		sem <- struct{}{}

		go func(i int, sender Account) {
		defer wg.Done()
		defer func() { <-sem }()

		fromAddress := common.HexToAddress(sender.Address)
		nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
		if err != nil {
			log.Printf("Nonce error for %s: %v\n", sender.Address, err)
			return
		}

		feeCap, tip, err := getDynamicGasParams(client)
		if err != nil {
			log.Printf("Gas param error: %v\n", err)
			return
		}

		for j := 0; j < txsPerSender; j++ {
			receiverIndex := i*txsPerSender + j
			if receiverIndex >= len(receivers) {
				break
			}
			receiver := receivers[receiverIndex]

			err = sendTx(client, sender, receiver.Address, nonce+uint64(j), feeCap, tip)
			if err != nil {
				log.Printf("Tx error from %s to %s: %v\n", sender.Address, receiver.Address, err)
			} else {
				count := atomic.AddUint64(&transferCount, 1)
				if count%1000 == 0 {
					log.Printf("Transferred %d transactions so far\n", count)
				}
			}
			time.Sleep(300 * time.Millisecond)
		}
	}(i, sender)
	}

	wg.Wait()
	fmt.Println("All transactions sent.")
}
