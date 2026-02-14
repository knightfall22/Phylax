package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

var ServiceName = "phylax"

type Config struct {
	TLSEnabled bool
	ClientCert string
	ClientKey  string
	RootCA     string

	NATSURL    string
	DBUser     string
	DBPassword string
	DBHost     string
	DBName     string
	DBPort     string
}

func LoadConfigurations() *Config {
	if err := godotenv.Load(); err != nil {
		if err := godotenv.Load("../.env"); err != nil {
			log.Println("No .env file found, using system environment variables")
		}
	}

	missing := []string{}
	requiredVars := []string{
		"NATS_URL",
		"DB_USER",
		"DB_PASSWORD",
		"DB_HOST",
		"DB_NAME",
		"DB_PORT",
	}

	for _, v := range requiredVars {
		if os.Getenv(v) == "" {
			missing = append(missing, v)
		}
	}

	if len(missing) > 0 {
		log.Fatalf("Missing required environment variables: %s", strings.Join(missing, ", "))
	}

	tlsEnabled, err := strconv.ParseBool(os.Getenv("TLSEnabled"))
	if err != nil {
		log.Fatalf("Invalid value for environment variable 'TLSEnabled': %v", err)
	}

	clientCert := os.Getenv("ClientCert")
	clientKey := os.Getenv("ClientKey")
	rootCA := os.Getenv("RootCA")

	if tlsEnabled {
		missing := []string{}
		if clientCert == "" {
			missing = append(missing, "ClientCert")
		}
		if clientKey == "" {
			missing = append(missing, "ClientKey")
		}
		if rootCA == "" {
			missing = append(missing, "RootCA")
		}

		if len(missing) > 0 {
			log.Fatalf("TLSEnabled is true, but the following variables are missing: %s", strings.Join(missing, ", "))
		}
	}

	return &Config{
		TLSEnabled: tlsEnabled,
		ClientCert: clientCert,
		ClientKey:  clientKey,
		RootCA:     rootCA,
		NATSURL:    os.Getenv("NATS_URL"),
		DBUser:     os.Getenv("DB_USER"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		DBHost:     os.Getenv("DB_HOST"),
		DBName:     os.Getenv("DB_NAME"),
		DBPort:     os.Getenv("DB_PORT"),
	}
}
