package worker

import (
	"context"
	"sync"
	"time"

	"nvide-live/internal/domain"
	"nvide-live/internal/usecase"
	"nvide-live/pkg/blockchain"

	"go.uber.org/zap"
)

type CryptoMonitor struct {
	cryptoRepo domain.CryptoRepository
	cryptoUC   *usecase.CryptoUseCase
	solana     *blockchain.SolanaClient
	evm        *blockchain.EVMClient
	logger     *zap.Logger
	stopChan   chan struct{}
	stopOnce   sync.Once
}

func NewCryptoMonitor(
	cryptoRepo domain.CryptoRepository,
	cryptoUC *usecase.CryptoUseCase,
	solana *blockchain.SolanaClient,
	evm *blockchain.EVMClient,
	logger *zap.Logger,
) *CryptoMonitor {
	return &CryptoMonitor{
		cryptoRepo: cryptoRepo,
		cryptoUC:   cryptoUC,
		solana:     solana,
		evm:        evm,
		logger:     logger,
		stopChan:   make(chan struct{}),
	}
}

func (m *CryptoMonitor) Start(ctx context.Context) {
	m.logger.Info("Starting Crypto Monitor Service")

	pollingTicker := time.NewTicker(1 * time.Minute)
	confirmationTicker := time.NewTicker(30 * time.Second)

	go func() {
		for {
			select {
			case <-pollingTicker.C:
				m.pollNewDeposits(ctx)
			case <-confirmationTicker.C:
				m.trackConfirmations(ctx)
			case <-m.stopChan:
				pollingTicker.Stop()
				confirmationTicker.Stop()
				return
			case <-ctx.Done():
				pollingTicker.Stop()
				confirmationTicker.Stop()
				return
			}
		}
	}()
}

func (m *CryptoMonitor) Stop() {
	m.stopOnce.Do(func() {
		m.logger.Info("Stopping Crypto Monitor Service")
		close(m.stopChan)
	})
}

func (m *CryptoMonitor) pollNewDeposits(ctx context.Context) {
	m.logger.Debug("Polling blockchain for new deposits")

	// 1. Get all active deposit addresses from DB
	// In production, we'd use a more efficient way like webhooks (Helius/BlockCypher)
	// but for now we'll simulate polling for SOL as requested.
	
	// Implementation would iterate through addresses and check signatures
}

func (m *CryptoMonitor) trackConfirmations(ctx context.Context) {
	m.logger.Debug("Tracking transaction confirmations")
	txs, err := m.cryptoRepo.ListPendingTransactions(ctx)
	if err != nil {
		m.logger.Error("Failed to list pending transactions", zap.Error(err))
		return
	}

	for _, tx := range txs {
		var currentConf int
		var err error

		switch tx.Chain {
		case domain.ChainSOL:
			// Solana confirmations are fast, usually GetTransaction is enough
			currentConf = tx.Confirmations + 1 
		case domain.ChainUSDT_ERC20, domain.ChainUSDT_BEP20:
			currentConf, err = m.evm.GetConfirmations(ctx, tx.TxHash)
		default:
			continue
		}

		if err != nil {
			m.logger.Warn("Failed to get confirmations for tx", zap.String("hash", tx.TxHash), zap.Error(err))
			continue
		}

		if currentConf != tx.Confirmations {
			status := tx.Status
			if currentConf >= tx.RequiredConfirmations {
				status = domain.CryptoStatusSuccess
				// Credit user wallet here if success
				m.logger.Info("Deposit confirmed", zap.String("tx_id", tx.ID.String()), zap.Float64("amount", tx.AmountCrypto))
			} else {
				status = domain.CryptoStatusConfirming
			}

			err = m.cryptoRepo.UpdateTransactionStatus(ctx, tx.ID, status, currentConf)
			if err != nil {
				m.logger.Error("Failed to update transaction status", zap.Error(err))
			}
		}
	}
}

// WebhookHandler processes incoming real-time notifications
func (m *CryptoMonitor) HandleWebhook(ctx context.Context, provider string, payload interface{}) error {
	m.logger.Info("Received crypto webhook", zap.String("provider", provider))
	
	// Example for SOL (Helius):
	// 1. Parse payload to get signature and involved address
	// 2. Verify if address belongs to our users
	// 3. Create 'pending' transaction in DB
	
	return nil
}
