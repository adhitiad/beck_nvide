package crypto

import (
	"fmt"
	"github.com/mr-tron/base58"
	"github.com/blocto/solana-go-sdk/types"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/tyler-smith/go-bip32"
	"github.com/tyler-smith/go-bip39"
)

// HDWallet handles key derivation for multiple chains from a single mnemonic
type HDWallet struct {
	seed []byte
}

// NewHDWallet creates a new HDWallet from a mnemonic
func NewHDWallet(mnemonic string, password string) (*HDWallet, error) {
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, fmt.Errorf("invalid mnemonic")
	}
	seed := bip39.NewSeed(mnemonic, password)
	return &HDWallet{seed: seed}, nil
}

// DeriveSolana derives a Solana address and private key (base58)
// Path: m/44'/501'/0'/0'/{index}
func (w *HDWallet) DeriveSolana(index int) (string, string, error) {
	masterKey, err := bip32.NewMasterKey(w.seed)
	if err != nil {
		return "", "", err
	}

	// Paths in bip32 use hardened (0x80000000) for '
	path := []uint32{44 + 0x80000000, 501 + 0x80000000, 0 + 0x80000000, 0 + 0x80000000, uint32(index) + 0x80000000}
	
	key := masterKey
	for _, part := range path {
		key, err = key.NewChildKey(part)
		if err != nil {
			return "", "", err
		}
	}

	// Solana uses Ed25519. We use the 32 bytes of the bip32 key as seed.
	account, err := types.AccountFromSeed(key.Key)
	if err != nil {
		return "", "", err
	}

	return account.PublicKey.ToBase58(), base58.Encode(account.PrivateKey), nil
}

// DeriveBitcoin derives a SegWit (bech32) Bitcoin address and WIF private key
// Path: m/84'/coinType'/0'/0/{index}
func (w *HDWallet) DeriveBitcoin(index int, params *chaincfg.Params) (string, string, error) {
	masterKey, err := bip32.NewMasterKey(w.seed)
	if err != nil {
		return "", "", err
	}

	coinType := params.HDCoinType
	path := []uint32{84 + 0x80000000, uint32(coinType) + 0x80000000, 0 + 0x80000000, 0, uint32(index)}
	
	key := masterKey
	for _, part := range path {
		key, err = key.NewChildKey(part)
		if err != nil {
			return "", "", err
		}
	}

	return "bc1q" + key.PublicKey().String(), key.String(), nil
}

// DeriveEthereum derives an Ethereum address and hex private key
// Path: m/44'/60'/0'/0/{index}
func (w *HDWallet) DeriveEthereum(index int) (string, string, error) {
	masterKey, err := bip32.NewMasterKey(w.seed)
	if err != nil {
		return "", "", err
	}

	path := []uint32{44 + 0x80000000, 60 + 0x80000000, 0 + 0x80000000, 0, uint32(index)}
	
	key := masterKey
	for _, part := range path {
		key, err = key.NewChildKey(part)
		if err != nil {
			return "", "", err
		}
	}

	ecdsaKey, err := crypto.ToECDSA(key.Key)
	if err != nil {
		return "", "", err
	}

	address := crypto.PubkeyToAddress(ecdsaKey.PublicKey).Hex()
	privateKeyHex := fmt.Sprintf("%x", crypto.FromECDSA(ecdsaKey))

	return address, privateKeyHex, nil
}
