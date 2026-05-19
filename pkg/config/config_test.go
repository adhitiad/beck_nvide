package config

import (
	"os"
	"testing"
)

func TestLoadAndGet(t *testing.T) {
	// 1. Clear relevant env variables first to ensure default or clean environment
	os.Unsetenv("SOLANA_RPC_URL")
	os.Unsetenv("USDT_RPC_URL")
	os.Unsetenv("CRYPTO_ENCRYPTION_KEY")
	os.Unsetenv("ALLOWED_REGIONS")

	// 2. Load config and verify defaults are set
	cfg := Load()
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}

	if cfg.SolanaRPCURL != "https://api.devnet.solana.com" {
		t.Errorf("expected default Solana RPC URL, got %s", cfg.SolanaRPCURL)
	}

	if cfg.CryptoEncryptionKey != "32-byte-long-aes-key-for-crypto" {
		t.Errorf("expected default Crypto Encryption Key, got %s", cfg.CryptoEncryptionKey)
	}

	// 3. Test env override
	expectedSol := "https://solana-custom-endpoint.com"
	os.Setenv("SOLANA_RPC_URL", expectedSol)
	defer os.Unsetenv("SOLANA_RPC_URL")

	expectedRegions := "indonesia,vietnam"
	os.Setenv("ALLOWED_REGIONS", expectedRegions)
	defer os.Unsetenv("ALLOWED_REGIONS")

	cfgOverridden := Load()
	if cfgOverridden.SolanaRPCURL != expectedSol {
		t.Errorf("expected overridden Solana RPC URL %s, got %s", expectedSol, cfgOverridden.SolanaRPCURL)
	}

	if cfgOverridden.AllowedRegions != expectedRegions {
		t.Errorf("expected overridden Allowed Regions %s, got %s", expectedRegions, cfgOverridden.AllowedRegions)
	}

	// 4. Test global accessor Get()
	globalCfg := Get()
	if globalCfg == nil {
		t.Fatal("expected Get() to return a non-nil config instance")
	}

	if globalCfg != globalConfig {
		t.Error("expected Get() to return the same global instance")
	}
}

func TestGetDBConnectionString(t *testing.T) {
	cfg := &Config{
		DBUser:     "testuser",
		DBPassword: "testpassword",
		DBHost:     "testhost",
		DBPort:     "5432",
		DBName:     "testdb",
		DBSSLMode:  "require",
	}

	expected := "postgres://testuser:testpassword@testhost:5432/testdb?sslmode=require"
	actual := cfg.GetDBConnectionString()
	if actual != expected {
		t.Errorf("expected connection string %s, got %s", expected, actual)
	}
}
