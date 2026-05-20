package config

import (
	"os"
)

type StorageConfig struct {
	OCI      *OCIConfig
	Storj    *StorjConfig
	Filebase *FilebaseConfig
}

type OCIConfig struct {
	Enabled       bool
	Namespace     string
	CompartmentID string
	Region        string
	TenancyOCID   string
	UserOCID      string
	PrivateKey    string
	Fingerprint   string
}

type StorjConfig struct {
	Enabled   bool
	AccessKey string
	SecretKey string
	Endpoint  string
}

type FilebaseConfig struct {
	Enabled   bool
	AccessKey string
	SecretKey string
	Endpoint  string
}

func LoadStorageConfig() *StorageConfig {
	return &StorageConfig{
		OCI:      loadOCIConfig(),
		Storj:    loadStorjConfig(),
		Filebase: loadFilebaseConfig(),
	}
}

func loadOCIConfig() *OCIConfig {
	return &OCIConfig{
		Enabled:       getEnvBool("OCI_ENABLED", false),
		Namespace:     os.Getenv("OCI_NAMESPACE"),
		CompartmentID: os.Getenv("OCI_COMPARTMENT_ID"),
		Region:        os.Getenv("OCI_REGION"),
		TenancyOCID:   os.Getenv("OCI_TENANCY_OCID"),
		UserOCID:      os.Getenv("OCI_USER_OCID"),
		PrivateKey:    os.Getenv("OCI_PRIVATE_KEY"),
		Fingerprint:   os.Getenv("OCI_FINGERPRINT"),
	}
}

func loadStorjConfig() *StorjConfig {
	endpoint := os.Getenv("STORJ_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://gateway.storjshare.io"
	}
	return &StorjConfig{
		Enabled:   getEnvBool("STORJ_ENABLED", false),
		AccessKey: os.Getenv("STORJ_ACCESS_KEY"),
		SecretKey: os.Getenv("STORJ_SECRET_KEY"),
		Endpoint:  endpoint,
	}
}

func loadFilebaseConfig() *FilebaseConfig {
	endpoint := os.Getenv("FILEBASE_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://s3.filebase.com"
	}
	return &FilebaseConfig{
		Enabled:   getEnvBool("FILEBASE_ENABLED", false),
		AccessKey: os.Getenv("FILEBASE_ACCESS_KEY"),
		SecretKey: os.Getenv("FILEBASE_SECRET_KEY"),
		Endpoint:  endpoint,
	}
}