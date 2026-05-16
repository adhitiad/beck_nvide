package blockchain

import (
	"context"
	"fmt"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

type EVMClient struct {
	client *ethclient.Client
}

func NewEVMClient(endpoint string) (*EVMClient, error) {
	c, err := ethclient.Dial(endpoint)
	if err != nil {
		return nil, err
	}
	return &EVMClient{client: c}, nil
}

func (e *EVMClient) GetNativeBalance(ctx context.Context, address string) (float64, error) {
	account := common.HexToAddress(address)
	balance, err := e.client.BalanceAt(ctx, account, nil)
	if err != nil {
		return 0, err
	}
	
	// Convert Wei to Ether (10^18)
	fbalance := new(big.Float).SetInt(balance)
	ethValue := new(big.Float).Quo(fbalance, big.NewFloat(math.Pow10(18)))
	val, _ := ethValue.Float64()
	return val, nil
}

func (e *EVMClient) GetERC20Balance(ctx context.Context, tokenAddress, walletAddress string, decimals int) (float64, error) {
	// Minimal ERC20 ABI for balance
	// In production, use abigen
	return 0, fmt.Errorf("not implemented")
}

func (e *EVMClient) GetConfirmations(ctx context.Context, txHash string) (int, error) {
	hash := common.HexToHash(txHash)
	_, isPending, err := e.client.TransactionByHash(ctx, hash)
	if err != nil {
		return 0, err
	}
	if isPending {
		return 0, nil
	}

	receipt, err := e.client.TransactionReceipt(ctx, hash)
	if err != nil {
		return 0, err
	}

	header, err := e.client.HeaderByNumber(ctx, nil)
	if err != nil {
		return 0, err
	}

	confirmations := header.Number.Uint64() - receipt.BlockNumber.Uint64() + 1
	return int(confirmations), nil
}
