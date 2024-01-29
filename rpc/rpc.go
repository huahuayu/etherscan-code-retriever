package rpc

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/huahuayu/etherscan-code-retriever/flags"
	"io"
	"net/http"
)

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

func CheckIfContract(address string) (bool, error) {
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

	resp, err := http.Post(*flags.RpcURL, "application/json", bytes.NewBuffer(payloadBytes))
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

func GetBinaryHash(address string) (string, error) {
	isContract, err := CheckIfContract(address)
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

	resp, err := http.Post(*flags.RpcURL, "application/json", bytes.NewBuffer(payloadBytes))
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
