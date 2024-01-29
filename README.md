# Etherscan code retriever

This is a Go project that helps to retrieve ethereum contract code from the Etherscan API.

For most of the data on chain, you can query your own node, but for contract source code you need to use Etherscan API.

Why this project useful? The Etherscan API has a rate limit of 5 requests per second(for free). This project can be used to cache the contract source code in database to avoid hitting the rate limit.

Since the number of contract is limited, especially the well-known contracts or protocols, so it's won't be a problem to store all the contract in database.

## Getting Started

### Prerequisites

- Docker
- Docker Compose

### Installing

1. Clone the repository.
2. Navigate to the project directory.
3. Copy the `.env.example` file to `.env` and fill in the environment variables.
4. Run `docker-compose up -d` to start the PostgreSQL database & app.
5. Run `docker-compose logs -f` to check the logs.

## Usage

If the app is running, you can use the following http endpoint to query the contract code, e.g. http://localhost:8080/code/0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48

- `/code/{address}`: Get the contract code by address.

You get the same result as the Etherscan API.

```go
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
```

*When the contract is a proxy, by default the app will return the implementation contract code. If your intention is to get the proxy contract code, you need to do some modification.*

## Flowchart

For better understanding, here is the flowchart of the project.

![](https://cdn.liushiming.cn/img/etherscan_code_retriever_flowchart.png)

## License

This project is licensed under the MIT License.