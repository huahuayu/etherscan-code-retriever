package flags

import (
	"flag"
	"github.com/joho/godotenv"
	"log"
	"os"
)

var (
	Dsn             = flag.String("dsn", "", "Database connection string, e.g. localhost/etherscan?user=postgres&password=passwd&sslmode=disable")
	EtherscanAPIKey = flag.String("apikey", "", "Etherscan API Key")
	RpcURL          = flag.String("rpc", "", "Ethereum RPC URL")
)

func Init() {
	// Load .env file
	_ = godotenv.Overload()
	flag.Parse()
	// Check if required flags are provided
	if *Dsn == "" {
		*Dsn = os.Getenv("DSN")
		if *Dsn == "" {
			log.Fatal("Database connection string is required")
		}
	}

	if *EtherscanAPIKey == "" {
		*EtherscanAPIKey = os.Getenv("APIKEY")
		if *EtherscanAPIKey == "" {
			log.Fatal("Etherscan API Key is required")
		}
	}

	if *RpcURL == "" {
		*RpcURL = os.Getenv("RPCURL")
		if *RpcURL == "" {
			log.Fatal("Ethereum RPC URL is required")
		}
	}
}
