package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"etherscan-code-retriever/ttlmap"
	"flag"
	"fmt"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"golang.org/x/time/rate"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// EtherscanResponse represents the response structure from Etherscan API
type EtherscanResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Result  any    `json:"result"`
}

// SourceCode represents the structure of contract source code information
type SourceCode struct {
	SourceCode           string `json:"sourceCode"`
	ABI                  string `json:"ABI"`
	ContractName         string `json:"contractName"`
	CompilerVersion      string `json:"compilerVersion"`
	OptimizationUsed     string `json:"optimizationUsed"`
	Runs                 string `json:"runs"`
	ConstructorArguments string `json:"constructorArguments"`
	EVMVersion           string `json:"EVMVersion"`
	Library              string `json:"library"`
	LicenseType          string `json:"licenseType"`
	Proxy                string `json:"proxy"`
	Implementation       string `json:"implementation"`
	SwarmSource          string `json:"swarmSource"`
}

// RPCRequest represents a JSON-RPC request payload
type RPCRequest struct {
	Jsonrpc string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

// RPCResponse represents a JSON-RPC response payload
type RPCResponse struct {
	Jsonrpc string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Result  string      `json:"result"`
	Error   interface{} `json:"error"`
}

// Model represents the structure for storing information about the contract in the database
type Model struct {
	Address      string      `db:"address"` // length 42
	ContractName string      `db:"contract_name"`
	SourceCode   *SourceCode `db:"source_code"`
	BinaryHash   string      `db:"binary_hash"` // 32 bytes sha256 hash string with 0x prefix, length 66
	CreatedAt    time.Time   `db:"created_at"`
	UpdatedAt    time.Time   `db:"updated_at"`
}

var (
	dsn             = flag.String("dsn", "", "Database connection string, e.g. localhost/etherscan?user=postgres&password=passwd&sslmode=disable")
	etherscanAPIKey = flag.String("apikey", "", "Etherscan API Key")
	rpcURL          = flag.String("rpc", "", "Ethereum RPC URL")
	limiter         = rate.NewLimiter(5, 5) // 5 requests per second
	codeCache       = ttlmap.NewTTLMap(2 * 24 * time.Hour)
	db              *sql.DB
	once            sync.Once
	url             string
)

func init() {
	// Load .env file
	_ = godotenv.Overload()
	flag.Parse()
	// Check if required flags are provided
	if *dsn == "" {
		*dsn = os.Getenv("DSN")
		if *dsn == "" {
			log.Fatal("Database connection string is required")
		}
	}

	if *etherscanAPIKey == "" {
		*etherscanAPIKey = os.Getenv("APIKEY")
		if *etherscanAPIKey == "" {
			log.Fatal("Etherscan API Key is required")
		}
	}

	if *rpcURL == "" {
		*rpcURL = os.Getenv("RPCURL")
		if *rpcURL == "" {
			log.Fatal("Ethereum RPC URL is required")
		}
	}
	url = fmt.Sprintf("https://api.etherscan.io/api?module=contract&action=getsourcecode&address=%%s&apikey=%s", *etherscanAPIKey)
}

func initDB() {
	once.Do(func() {
		var err error
		db, err = sql.Open("postgres", *dsn)
		if err != nil {
			panic(err)
		}
		err = db.Ping()
		if err != nil {
			panic(err)
		}
	})
}

func main() {
	http.HandleFunc("/code/", loggerMiddleware(sourceCodeHandler))
	log.Println("Server starting on port 8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("ListenAndServe error:", err)
	}
}

func sourceCodeHandler(w http.ResponseWriter, r *http.Request) {
	address := strings.TrimPrefix(r.URL.Path, "/code/")
	if address == "" {
		http.Error(w, "Address is required", http.StatusBadRequest)
		return
	}

	isContract, err := checkIfContract(address)
	if err != nil {
		http.Error(w, "Failed to check if address is a contract: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if !isContract {
		http.Error(w, "Address is not a contract", http.StatusBadRequest)
		return
	}

	sourceCode, err := getSourceCode(address)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sourceCode)
}

func checkIfContract(address string) (bool, error) {
	payload := RPCRequest{
		Jsonrpc: "2.0",
		Method:  "eth_getCode",
		Params:  []interface{}{address, "latest"},
		ID:      1,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return false, err
	}

	resp, err := http.Post(*rpcURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	var rpcResponse RPCResponse
	err = json.Unmarshal(body, &rpcResponse)
	if err != nil {
		return false, err
	}

	return rpcResponse.Result != "0x", nil
}

func upsertToDB(address string, sourceCode *SourceCode, binaryHash string) error {
	initDB()
	query := `
        INSERT INTO code (address, contract_name, source_code, binary_hash, created_at, updated_at)
        VALUES ($1, $2, $3, $4, NOW(), NOW())
        ON CONFLICT (address) DO UPDATE
        SET contract_name = EXCLUDED.contract_name,
            source_code = EXCLUDED.source_code,
            binary_hash = EXCLUDED.binary_hash,
            updated_at = NOW();
    `
	sourceCodeJSON, err := json.Marshal(sourceCode)
	if err != nil {
		return err
	}

	_, err = db.Exec(query, address, sourceCode.ContractName, sourceCodeJSON, binaryHash)
	return err
}

func getSourceCodeFromDB(address string) (*Model, error) {
	initDB()
	query := "SELECT address, contract_name, source_code, binary_hash, created_at, updated_at FROM code WHERE address = $1"

	var model Model
	var sourceCodeJSON string
	err := db.QueryRow(query, address).Scan(&model.Address, &model.ContractName, &sourceCodeJSON, &model.BinaryHash, &model.CreatedAt, &model.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	err = json.Unmarshal([]byte(sourceCodeJSON), &model.SourceCode)
	if err != nil {
		return nil, err
	}

	return &model, nil
}

func getSourceCode(address string) (*SourceCode, error) {
	if code, exists := codeCache.Get(address); exists {
		return code.(*SourceCode), nil
	}

	model, err := getSourceCodeFromDB(address)
	if err != nil {
		return nil, err
	}

	if model != nil {
		if time.Since(model.UpdatedAt) < 30*24*time.Hour {
			return model.SourceCode, nil
		}
	}

	return queryAPIAndUpdate(address)
}

func queryAPIAndUpdate(address string) (*SourceCode, error) {
	err := limiter.Wait(context.Background())
	if err != nil {
		return nil, err
	}

	api := fmt.Sprintf(url, address)
	resp, err := http.Get(api)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	response := &EtherscanResponse{}
	err = json.Unmarshal(body, response)
	if err != nil {
		return nil, err
	}

	if msg, ok := response.Result.(string); ok {
		return nil, fmt.Errorf(msg)
	}

	if result, ok := response.Result.([]any); ok {
		if len(result) == 0 {
			return nil, fmt.Errorf("no result")
		}

		bs, err := json.Marshal(result[0])
		if err != nil {
			return nil, err
		}
		code := &SourceCode{}
		err = json.Unmarshal(bs, code)
		if err != nil {
			return nil, err
		}

		if code.Proxy == "1" && code.Implementation != "" {
			return getSourceCode(code.Implementation)
		} else {
			binaryHash, err := getBinaryHash(address)
			if err != nil {
				return nil, err
			}
			codeCache.Set(address, code, 7*24*time.Hour)
			err = upsertToDB(address, code, binaryHash)
			if err != nil {
				return nil, err
			}
			return code, nil
		}
	}

	return nil, fmt.Errorf("unknown result" + string(body))
}

func getBinaryHash(address string) (string, error) {
	isContract, err := checkIfContract(address)
	if err != nil {
		return "", err
	}

	if !isContract {
		return "", fmt.Errorf("address is not a contract")
	}

	payload := RPCRequest{
		Jsonrpc: "2.0",
		Method:  "eth_getCode",
		Params:  []interface{}{address, "latest"},
		ID:      1,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(*rpcURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var rpcResponse RPCResponse
	err = json.Unmarshal(body, &rpcResponse)
	if err != nil {
		return "", err
	}

	if rpcResponse.Result == "0x" {
		return "", fmt.Errorf("no code found at address")
	}

	hash := sha256.Sum256([]byte(rpcResponse.Result))
	return "0x" + hex.EncodeToString(hash[:]), nil
}

func loggerMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now() // Record the start time

		// Extract the IP address from the request
		ip := r.RemoteAddr
		if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
			ip = forwardedFor // Use X-Forwarded-For header if present
		}

		// Call the actual handler
		next.ServeHTTP(w, r)

		// Calculate the duration and log the IP, URL, and the time taken
		duration := time.Since(start)
		log.Printf("IP: %s Request: %s Time: %v", ip, r.URL.Path, duration)
	}
}
