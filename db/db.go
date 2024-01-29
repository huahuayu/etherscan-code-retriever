package db

import (
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/huahuayu/etherscan-code-retriever/entity"
	"github.com/huahuayu/etherscan-code-retriever/flags"
	"sync"
	"time"
)

type Model struct {
	Address      string             `db:"address"` // length 42
	ContractName string             `db:"contract_name"`
	SourceCode   *entity.SourceCode `db:"source_code"`
	BinaryHash   string             `db:"binary_hash"` // 32 bytes sha256 hash string with 0x prefix, length 66
	CreatedAt    time.Time          `db:"created_at"`
	UpdatedAt    time.Time          `db:"updated_at"`
}

var (
	db   *sql.DB
	once sync.Once
)

func initDB() {
	once.Do(func() {
		var err error
		db, err = sql.Open("postgres", *flags.Dsn)
		if err != nil {
			panic(err)
		}
		err = db.Ping()
		if err != nil {
			panic(err)
		}
	})
}

func UpsertToDB(address string, sourceCode *entity.SourceCode, binaryHash string) error {
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

func GetSourceCodeFromDB(address string) (*Model, error) {
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
