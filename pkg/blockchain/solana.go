package blockchain

import (
	"context"
	"fmt"
	"math"

	"github.com/blocto/solana-go-sdk/client"
)

type SolanaClient struct {
	rpcClient *client.Client
}

func NewSolanaClient(endpoint string) *SolanaClient {
	return &SolanaClient{
		rpcClient: client.NewClient(endpoint),
	}
}

func (s *SolanaClient) GetBalance(ctx context.Context, address string) (float64, error) {
	balance, err := s.rpcClient.GetBalance(ctx, address)
	if err != nil {
		return 0, err
	}
	// Balance is in Lamports (10^9)
	return float64(balance) / math.Pow10(9), nil
}

func (s *SolanaClient) GetConfirmedSignatures(ctx context.Context, address string, limit int) ([]string, error) {
	signatures, err := s.rpcClient.GetSignaturesForAddress(ctx, address)
	if err != nil {
		return nil, err
	}
	
	var txs []string
	for i, sig := range signatures {
		if i >= limit {
			break
		}
		txs = append(txs, sig.Signature)
	}
	return txs, nil
}

func (s *SolanaClient) GetTransaction(ctx context.Context, signature string) (float64, string, string, int, error) {
	tx, err := s.rpcClient.GetTransaction(ctx, signature)
	if err != nil {
		return 0, "", "", 0, err
	}
	
	if tx == nil {
		return 0, "", "", 0, fmt.Errorf("transaction not found")
	}

	// Basic parsing for SOL transfers
	// Note: Production implementation requires robust parsing of instructions
	amount := 0.0
	from := ""
	to := ""
	confirmations := 1 // if it's in GetTransaction, it's usually confirmed
	
	return amount, from, to, confirmations, nil
}
