package usecase

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"go.uber.org/zap"

	"nvide-live/internal/domain"
	cryptopkg "nvide-live/pkg/crypto"
)

type payoutUsecase struct {
	payoutRepo                domain.PayoutMethodRepository
	cryptoAddrRepo            domain.CryptoPayoutAddressRepository
	encryptionKey             []byte
	microDepositVerifyEnabled bool
	logger                    *zap.Logger
}

func NewPayoutUsecase(
	payoutRepo domain.PayoutMethodRepository,
	cryptoAddrRepo domain.CryptoPayoutAddressRepository,
	encryptionKey []byte,
	microDepositVerifyEnabled bool,
	logger *zap.Logger,
) domain.PayoutUsecase {
	return &payoutUsecase{
		payoutRepo:                payoutRepo,
		cryptoAddrRepo:            cryptoAddrRepo,
		encryptionKey:             encryptionKey,
		microDepositVerifyEnabled: microDepositVerifyEnabled,
		logger:                    logger,
	}
}

func (u *payoutUsecase) ListPayoutMethods(ctx context.Context, userID domain.UUID) ([]*domain.PayoutMethod, error) {
	items, err := u.payoutRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	for _, it := range items {
		u.decryptPayout(it)
	}
	return items, nil
}

func (u *payoutUsecase) CreatePayoutMethod(ctx context.Context, userID domain.UUID, req *domain.CreatePayoutMethodRequest) (*domain.PayoutMethod, error) {
	if req == nil {
		return nil, errors.New("request tidak valid")
	}
	if err := u.validateCreate(req); err != nil {
		return nil, err
	}

	limit, err := u.payoutRepo.CountByUserIDAndType(ctx, userID, req.Type)
	if err != nil {
		return nil, err
	}
	switch req.Type {
	case domain.PayoutTypeBankTransfer:
		if limit >= 3 {
			return nil, errors.New("maksimal 3 rekening bank")
		}
	case domain.PayoutTypeEwallet:
		if limit >= 3 {
			return nil, errors.New("maksimal 3 akun e-wallet")
		}
	case domain.PayoutTypeCrypto:
		if limit >= 5 {
			return nil, errors.New("maksimal 5 metode crypto")
		}
	}

	method := &domain.PayoutMethod{
		ID:                   domain.NewUUIDv7(),
		UserID:               userID,
		Type:                 req.Type,
		IsPrimary:            req.IsPrimary,
		BankName:             req.BankName,
		AccountNumber:        req.AccountNumber,
		AccountHolderName:    req.AccountHolderName,
		EwalletProvider:      req.EwalletProvider,
		EwalletPhoneNumber:   req.EwalletPhoneNumber,
		IsVerified:           !u.microDepositVerifyEnabled,
		MicroDepositRequired: u.microDepositVerifyEnabled,
	}
	u.encryptPayout(method)

	if req.IsPrimary {
		if err := u.payoutRepo.UnsetPrimaryByUserID(ctx, userID); err != nil {
			return nil, err
		}
	}
	if err := u.payoutRepo.Create(ctx, method); err != nil {
		return nil, err
	}
	u.decryptPayout(method)
	return method, nil
}

func (u *payoutUsecase) UpdatePayoutMethod(ctx context.Context, userID, methodID domain.UUID, req *domain.UpdatePayoutMethodRequest) (*domain.PayoutMethod, error) {
	current, err := u.payoutRepo.GetByIDAndUserID(ctx, methodID, userID)
	if err != nil {
		return nil, err
	}
	u.decryptPayout(current)

	if req.BankName != nil {
		current.BankName = req.BankName
	}
	if req.AccountNumber != nil {
		current.AccountNumber = req.AccountNumber
	}
	if req.AccountHolderName != nil {
		current.AccountHolderName = req.AccountHolderName
	}
	if req.EwalletProvider != nil {
		current.EwalletProvider = req.EwalletProvider
	}
	if req.EwalletPhoneNumber != nil {
		current.EwalletPhoneNumber = req.EwalletPhoneNumber
	}
	if req.IsPrimary != nil {
		current.IsPrimary = *req.IsPrimary
	}

	switch current.Type {
	case domain.PayoutTypeBankTransfer:
		if current.BankName == nil || current.AccountNumber == nil || current.AccountHolderName == nil {
			return nil, errors.New("bank_name, account_number, account_holder_name wajib diisi")
		}
	case domain.PayoutTypeEwallet:
		if current.EwalletProvider == nil || current.EwalletPhoneNumber == nil {
			return nil, errors.New("provider dan phone_number wajib diisi")
		}
	}

	if current.IsPrimary {
		if err := u.payoutRepo.UnsetPrimaryByUserID(ctx, userID); err != nil {
			return nil, err
		}
	}
	u.encryptPayout(current)
	if err := u.payoutRepo.Update(ctx, current); err != nil {
		return nil, err
	}
	if current.IsPrimary {
		if err := u.payoutRepo.SetPrimary(ctx, current.ID, userID); err != nil {
			return nil, err
		}
	}
	u.decryptPayout(current)
	return current, nil
}

func (u *payoutUsecase) DeletePayoutMethod(ctx context.Context, userID, methodID domain.UUID) error {
	return u.payoutRepo.Delete(ctx, methodID, userID)
}

func (u *payoutUsecase) SetPrimaryPayoutMethod(ctx context.Context, userID, methodID domain.UUID) error {
	_, err := u.payoutRepo.GetByIDAndUserID(ctx, methodID, userID)
	if err != nil {
		return err
	}
	return u.payoutRepo.SetPrimary(ctx, methodID, userID)
}

func (u *payoutUsecase) ListCryptoPayoutAddresses(ctx context.Context, userID domain.UUID) ([]*domain.CryptoPayoutAddress, error) {
	list, err := u.cryptoAddrRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	for _, it := range list {
		plain, decErr := cryptopkg.Decrypt(it.Address, u.encryptionKey)
		if decErr == nil {
			it.Address = plain
		}
	}
	return list, nil
}

func (u *payoutUsecase) CreateCryptoPayoutAddress(ctx context.Context, userID domain.UUID, req *domain.CreateCryptoPayoutAddressRequest) (*domain.CryptoPayoutAddress, error) {
	if req == nil {
		return nil, errors.New("request tidak valid")
	}
	if req.Network != domain.CryptoNetworkSolana && req.Network != domain.CryptoNetworkBTC && req.Network != domain.CryptoNetworkBSC {
		return nil, errors.New("network tidak didukung")
	}
	if !validateCryptoAddress(req.Network, req.Address) {
		return nil, errors.New("format alamat crypto tidak valid")
	}
	count, err := u.cryptoAddrRepo.CountByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if count >= 5 {
		return nil, errors.New("maksimal 5 alamat crypto")
	}
	encAddr, err := cryptopkg.Encrypt(strings.TrimSpace(req.Address), u.encryptionKey)
	if err != nil {
		return nil, err
	}
	row := &domain.CryptoPayoutAddress{
		ID:      domain.NewUUIDv7(),
		UserID:  userID,
		Network: req.Network,
		Address: encAddr,
		Label:   req.Label,
	}
	if err := u.cryptoAddrRepo.Create(ctx, row); err != nil {
		return nil, err
	}
	row.Address = req.Address
	return row, nil
}

func (u *payoutUsecase) DeleteCryptoPayoutAddress(ctx context.Context, userID, id domain.UUID) error {
	return u.cryptoAddrRepo.Delete(ctx, id, userID)
}

func (u *payoutUsecase) ResolveWithdrawalTarget(ctx context.Context, userID domain.UUID) (string, map[string]interface{}, error) {
	pm, err := u.payoutRepo.GetPrimaryByUserID(ctx, userID)
	if err != nil {
		return "", nil, errors.New("metode payout primary belum diatur")
	}
	u.decryptPayout(pm)

	switch pm.Type {
	case domain.PayoutTypeBankTransfer:
		return pm.Type, map[string]interface{}{
			"bank_name":           valueOrEmpty(pm.BankName),
			"account_number":      valueOrEmpty(pm.AccountNumber),
			"account_holder_name": valueOrEmpty(pm.AccountHolderName),
		}, nil
	case domain.PayoutTypeEwallet:
		return pm.Type, map[string]interface{}{
			"provider":     valueOrEmpty(pm.EwalletProvider),
			"phone_number": valueOrEmpty(pm.EwalletPhoneNumber),
		}, nil
	case domain.PayoutTypeCrypto:
		list, err := u.cryptoAddrRepo.ListByUserID(ctx, userID)
		if err != nil || len(list) == 0 {
			return "", nil, errors.New("alamat crypto belum tersedia")
		}
		addr := list[0]
		plainAddr, err := cryptopkg.Decrypt(addr.Address, u.encryptionKey)
		if err != nil {
			return "", nil, err
		}
		return pm.Type, map[string]interface{}{
			"network":  addr.Network,
			"address":  plainAddr,
			"label":    addr.Label,
			"provider": "nowpayments",
		}, nil
	default:
		return "", nil, fmt.Errorf("tipe payout tidak didukung: %s", pm.Type)
	}
}

func (u *payoutUsecase) validateCreate(req *domain.CreatePayoutMethodRequest) error {
	switch req.Type {
	case domain.PayoutTypeBankTransfer:
		if req.BankName == nil || req.AccountNumber == nil || req.AccountHolderName == nil {
			return errors.New("bank_name, account_number, account_holder_name wajib diisi")
		}
	case domain.PayoutTypeEwallet:
		if req.EwalletProvider == nil || req.EwalletPhoneNumber == nil {
			return errors.New("provider dan phone_number wajib diisi")
		}
		provider := strings.ToLower(strings.TrimSpace(*req.EwalletProvider))
		switch provider {
		case domain.EwalletGopay, domain.EwalletOVO, domain.EwalletDANA, domain.EwalletShopeePay, domain.EwalletLinkAja:
		default:
			return errors.New("provider e-wallet tidak valid")
		}
	case domain.PayoutTypeCrypto:
		// Optional as selector; actual target from crypto_payout_addresses table.
	default:
		return errors.New("type payout tidak valid")
	}
	return nil
}

func (u *payoutUsecase) encryptPayout(m *domain.PayoutMethod) {
	if m.BankName != nil {
		if enc, err := cryptopkg.Encrypt(*m.BankName, u.encryptionKey); err == nil {
			m.BankName = &enc
		}
	}
	if m.AccountNumber != nil {
		if enc, err := cryptopkg.Encrypt(*m.AccountNumber, u.encryptionKey); err == nil {
			m.AccountNumber = &enc
		}
	}
	if m.AccountHolderName != nil {
		if enc, err := cryptopkg.Encrypt(*m.AccountHolderName, u.encryptionKey); err == nil {
			m.AccountHolderName = &enc
		}
	}
	if m.EwalletPhoneNumber != nil {
		if enc, err := cryptopkg.Encrypt(*m.EwalletPhoneNumber, u.encryptionKey); err == nil {
			m.EwalletPhoneNumber = &enc
		}
	}
}

func (u *payoutUsecase) decryptPayout(m *domain.PayoutMethod) {
	if m.BankName != nil {
		if dec, err := cryptopkg.Decrypt(*m.BankName, u.encryptionKey); err == nil {
			m.BankName = &dec
		}
	}
	if m.AccountNumber != nil {
		if dec, err := cryptopkg.Decrypt(*m.AccountNumber, u.encryptionKey); err == nil {
			m.AccountNumber = &dec
		}
	}
	if m.AccountHolderName != nil {
		if dec, err := cryptopkg.Decrypt(*m.AccountHolderName, u.encryptionKey); err == nil {
			m.AccountHolderName = &dec
		}
	}
	if m.EwalletPhoneNumber != nil {
		if dec, err := cryptopkg.Decrypt(*m.EwalletPhoneNumber, u.encryptionKey); err == nil {
			m.EwalletPhoneNumber = &dec
		}
	}
}

func valueOrEmpty(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func validateCryptoAddress(network, address string) bool {
	addr := strings.TrimSpace(address)
	switch network {
	case domain.CryptoNetworkSolana:
		// base58 32..44 chars
		return regexp.MustCompile(`^[1-9A-HJ-NP-Za-km-z]{32,44}$`).MatchString(addr)
	case domain.CryptoNetworkBTC:
		return regexp.MustCompile(`^(bc1|[13])[a-zA-HJ-NP-Z0-9]{25,62}$`).MatchString(addr)
	case domain.CryptoNetworkBSC:
		return regexp.MustCompile(`^0x[a-fA-F0-9]{40}$`).MatchString(addr)
	default:
		return false
	}
}
