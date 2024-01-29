package entity

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
