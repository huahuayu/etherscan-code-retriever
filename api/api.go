package api

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/huahuayu/etherscan-code-retriever/db"
	"github.com/huahuayu/etherscan-code-retriever/entity"
	"github.com/huahuayu/etherscan-code-retriever/flags"
	"github.com/huahuayu/etherscan-code-retriever/rpc"
	"github.com/huahuayu/etherscan-code-retriever/ttlmap"
	"golang.org/x/time/rate"
	"io"
	"net/http"
	"strings"
	"time"
)

var (
	limiter   = rate.NewLimiter(5, 5) // 5 requests per second
	codeCache = ttlmap.NewTTLMap(2 * 24 * time.Hour)
)

func SourceCodeHandler(w http.ResponseWriter, r *http.Request) {
	address := strings.TrimPrefix(r.URL.Path, "/code/")
	if address == "" {
		http.Error(w, "Address is required", http.StatusBadRequest)
		return
	}

	isContract, err := rpc.CheckIfContract(address)
	if err != nil {
		http.Error(w, "Failed to check if address is a contract: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if !isContract {
		http.Error(w, "Address is not a contract", http.StatusBadRequest)
		return
	}

	sourceCode, err := sourceCodeService(address)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sourceCode)
}

func sourceCodeService(address string) (*entity.SourceCode, error) {
	if code, exists := codeCache.Get(address); exists {
		return code.(*entity.SourceCode), nil
	}

	model, err := db.GetSourceCodeFromDB(address)
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

func queryAPIAndUpdate(address string) (*entity.SourceCode, error) {
	err := limiter.Wait(context.Background())
	if err != nil {
		return nil, err
	}

	api := fmt.Sprintf("https://api.etherscan.io/api?module=contract&action=getsourcecode&address=%s&apikey=%s", address, *flags.EtherscanAPIKey)
	resp, err := http.Get(api)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	response := &entity.EtherscanResponse{}
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
		code := &entity.SourceCode{}
		err = json.Unmarshal(bs, code)
		if err != nil {
			return nil, err
		}

		if code.Proxy == "1" && code.Implementation != "" {
			return sourceCodeService(code.Implementation)
		} else {
			binaryHash, err := rpc.GetBinaryHash(address)
			if err != nil {
				return nil, err
			}
			codeCache.Set(address, code, 7*24*time.Hour)
			err = db.UpsertToDB(address, code, binaryHash)
			if err != nil {
				return nil, err
			}
			return code, nil
		}
	}

	return nil, fmt.Errorf("unknown result" + string(body))
}
