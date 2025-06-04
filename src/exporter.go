package exporter

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/log"
	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/statetransfer"
	"github.com/spf13/pflag"
)

func main() {
	var genesisPath string

	// Configure console log output
	log.SetDefault(log.NewLogger(log.NewTerminalHandlerWithLevel(os.Stderr, log.LevelInfo, true)))

	// Define command line flags
	pflag.StringVarP(&genesisPath, "genesis", "g", "", "Path to genesis.json file")
	pflag.Parse()

	// Check required parameters
	if genesisPath == "" {
		fmt.Fprintf(os.Stderr, "Error: genesis path is required\n")
		pflag.Usage()
		os.Exit(1)
	}

	// Read genesis.json
	genesisJson, err := os.ReadFile(genesisPath)
	if err != nil {
		fmt.Errorf("failed to read genesis file: %v", err)
		os.Exit(1)
	}

	// Debug: Print JSON info only when it's small (to avoid overwhelming output)
	if len(genesisJson) < 1000 {
		log.Info("Raw genesis JSON content", "content", string(genesisJson))
		os.Exit(1)
	}
	log.Info("JSON file info", "bytes", len(genesisJson), "path", genesisPath)

	var gen core.Genesis
	if err := json.Unmarshal(genesisJson, &gen); err != nil {
		fmt.Errorf("failed to unmarshal genesis JSON: %v", err)
		os.Exit(1)
	}

	// Validate required fields
	if gen.Config == nil {
		fmt.Errorf("genesis config is missing")
		os.Exit(1)
	}
	if gen.Config.ArbitrumChainParams.InitialChainOwner == (common.Address{}) {
		fmt.Errorf("initial chain owner is missing in genesis config")
		os.Exit(1)
	}

	// Debug: Print the unmarshaled result
	log.Info("Successfully parsed Genesis",
		"chainId", gen.Config.ChainID,
		"gasLimit", gen.GasLimit,
		"timestamp", gen.Timestamp,
		"initialArbOSVersion", gen.Config.ArbitrumChainParams.InitialArbOSVersion,
		"initialChainOwner", gen.Config.ArbitrumChainParams.InitialChainOwner,
		"accounts", len(gen.Alloc))

	// Calculate state root
	stateRoot, blockHash, err := StateAndBlockRootFromGenesis(gen)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error calculating state root: %v\n", err)
		os.Exit(1)
	}

	fmt.Sprintf("State Root: %s\nBlock Hash: %s", stateRoot, blockHash)

}

func StateAndBlockRootFromGenesis(gen core.Genesis) (string, string, error) {

	// 2. Assemble account information (if genesis.json is empty, use empty account list)
	var accounts []statetransfer.AccountInitializationInfo
	for address, account := range gen.Alloc {
		var contractInfo *statetransfer.AccountInitContractInfo
		if len(account.Code) > 0 || len(account.Storage) > 0 {
			contractInfo = &statetransfer.AccountInitContractInfo{
				Code:            account.Code,
				ContractStorage: account.Storage,
			}
		}
		accounts = append(accounts, statetransfer.AccountInitializationInfo{
			Addr:         address,
			EthBalance:   account.Balance,
			Nonce:        account.Nonce,
			ContractInfo: contractInfo,
		})
	}

	// 3. Get chain config
	chainConfig := gen.Config

	// 4. Create initialization data reader - set correct chain owner
	initDataReader := statetransfer.NewMemoryInitDataReader(&statetransfer.ArbosInitializationInfo{
		ChainOwner: common.HexToAddress(chainConfig.ArbitrumChainParams.InitialChainOwner.Hex()),
		Accounts:   accounts,
	})

	log.Info("ArbitrumChainParams", "InitialArbOSVersion", chainConfig.ArbitrumChainParams.InitialArbOSVersion, "InitialChainOwner", chainConfig.ArbitrumChainParams.InitialChainOwner)

	// Manually construct JSON serialization of chain config, completely matching real node format (field order and content)
	serializedChainConfig := fmt.Sprintf(`{"homesteadBlock":0,"daoForkBlock":null,"daoForkSupport":true,"eip150Block":0,"eip150Hash":"0x0000000000000000000000000000000000000000000000000000000000000000","eip155Block":0,"eip158Block":0,"byzantiumBlock":0,"constantinopleBlock":0,"petersburgBlock":0,"istanbulBlock":0,"muirGlacierBlock":0,"berlinBlock":0,"londonBlock":0,"clique":{"period":0,"epoch":0},"arbitrum":{"EnableArbOS":true,"AllowDebugPrecompiles":false,"DataAvailabilityCommittee":false,"InitialArbOSVersion":%d,"GenesisBlockNum":0,"MaxCodeSize":%d,"MaxInitCodeSize":%d,"InitialChainOwner":"%s"},"chainId":%d}`,
		chainConfig.ArbitrumChainParams.InitialArbOSVersion,
		chainConfig.MaxCodeSize(),
		chainConfig.MaxInitCodeSize(),
		chainConfig.ArbitrumChainParams.InitialChainOwner.Hex(),
		chainConfig.ChainID)

	// 5. Create initialization message
	initMessage := &arbostypes.ParsedInitMessage{
		ChainId:               chainConfig.ChainID,
		InitialL1BaseFee:      big.NewInt(100000000),
		ChainConfig:           chainConfig,
		SerializedChainConfig: []byte(serializedChainConfig),
	}

	// 6. Call CalculateArbosStateHash
	stateRoot, err := CalculateArbosStateHash(initDataReader, chainConfig, initMessage, gen.Timestamp)
	if err != nil {
		return "", "", err
	}

	// 7. Create genesis block and calculate block hash
	parentHash := common.Hash{} // Genesis block has no parent, so use zero hash
	blockNumber := uint64(0)    // Genesis block number is 0
	timestamp := uint64(0)      // Use timestamp from genesis.json

	genesisBlock := arbosState.MakeGenesisBlock(parentHash, blockNumber, timestamp, stateRoot, chainConfig)
	blockHash := genesisBlock.Hash()

	// 8. Return both state root and block hash
	return stateRoot.Hex(), blockHash.Hex(), nil
}
