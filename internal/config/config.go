package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all application configuration loaded from environment variables
type Config struct {
	Port             string
	DatabaseURL      string
	BaseRPCURL       string
	EscrowContract   string
	RelayerPrivKey   string
	USDCContract     string
	FeeTreasury      string
	BondOpsWallet    string
	S3Endpoint       string
	S3Bucket         string
	S3AccessKey      string
	S3SecretKey      string
	JWTSecret        string
	ChainID          uint64
	PlatformFeeBps   uint16
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		Port:           getEnv("PORT", ":8080"),
		DatabaseURL:    getEnv("DATABASE_URL", ""),
		BaseRPCURL:     getEnv("BASE_RPC_URL", ""),
		EscrowContract: getEnv("ESCROW_CONTRACT", ""),
		RelayerPrivKey: getEnv("RELAYER_PRIVATE_KEY", ""),
		USDCContract:   getEnv("USDC_CONTRACT", "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"),
		FeeTreasury:    getEnv("FEE_TREASURY", ""),
		BondOpsWallet:  getEnv("BOND_OPS_WALLET", ""),
		S3Endpoint:     getEnv("S3_ENDPOINT", ""),
		S3Bucket:       getEnv("S3_BUCKET", "artifacts"),
		S3AccessKey:    getEnv("S3_ACCESS_KEY", ""),
		S3SecretKey:    getEnv("S3_SECRET_KEY", ""),
		JWTSecret:      getEnv("JWT_SECRET", ""),
	}

	// Parse ChainID
	chainIDStr := getEnv("CHAIN_ID", "8453")
	chainID, err := strconv.ParseUint(chainIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid CHAIN_ID: %w", err)
	}
	cfg.ChainID = chainID

	// Parse PlatformFeeBps
	feeBpsStr := getEnv("PLATFORM_FEE_BPS", "500")
	feeBps, err := strconv.ParseUint(feeBpsStr, 10, 16)
	if err != nil {
		return nil, fmt.Errorf("invalid PLATFORM_FEE_BPS: %w", err)
	}
	cfg.PlatformFeeBps = uint16(feeBps)

	// Validate required fields
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.BaseRPCURL == "" {
		return nil, fmt.Errorf("BASE_RPC_URL is required")
	}
	if cfg.EscrowContract == "" {
		return nil, fmt.Errorf("ESCROW_CONTRACT is required")
	}
	if cfg.RelayerPrivKey == "" {
		return nil, fmt.Errorf("RELAYER_PRIVATE_KEY is required")
	}
	if cfg.FeeTreasury == "" {
		return nil, fmt.Errorf("FEE_TREASURY is required")
	}
	if cfg.BondOpsWallet == "" {
		return nil, fmt.Errorf("BOND_OPS_WALLET is required")
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	return cfg, nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
