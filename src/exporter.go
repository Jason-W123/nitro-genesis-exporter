package main

import (
	"fmt"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
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

	// Calculate state root
	stateRoot, err := StateRootFromGenesis(genesisPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error calculating state root: %v\n", err)
		os.Exit(1)
	}

	// Output result
	fmt.Printf("State Root: %s\n", stateRoot)
}

func StateRootFromGenesis(genesisPath string) (string, error) {
	// 1. Read genesis.json
	// genesisJson, err := os.ReadFile(genesisPath)
	// if err != nil {
	// 	return "", err
	// }
	// var gen core.Genesis
	// if err := json.Unmarshal(genesisJson, &gen); err != nil {
	// 	return "", err
	// }

	// Provide default values for missing required fields
	// if gen.GasLimit == 0 {
	// 	gen.GasLimit = 30000000 // default gas limit
	// }
	// if gen.Difficulty == nil {
	// 	gen.Difficulty = big.NewInt(1) // default difficulty
	// }
	// if gen.Timestamp == 0 {
	// 	gen.Timestamp = 0 // default timestamp
	// }

	// 2. Assemble account information (if genesis.json is empty, use empty account list)
	// var accounts []statetransfer.AccountInitializationInfo
	// for address, account := range gen.Alloc {
	// 	var contractInfo *statetransfer.AccountInitContractInfo
	// 	if len(account.Code) > 0 || len(account.Storage) > 0 {
	// 		contractInfo = &statetransfer.AccountInitContractInfo{
	// 			Code:            account.Code,
	// 			ContractStorage: account.Storage,
	// 		}
	// 	}
	// 	accounts = append(accounts, statetransfer.AccountInitializationInfo{
	// 		Addr:         address,
	// 		EthBalance:   account.Balance,
	// 		Nonce:        account.Nonce,
	// 		ContractInfo: contractInfo,
	// 	})
	// }

	initData := statetransfer.ArbosInitializationInfo{
		NextBlockNumber: 0,
		ChainOwner:      common.HexToAddress("0x860C58951A77ac2F2Cb452cC9F36B1d1FdF7c521"),
	}
	initDataReader := statetransfer.NewMemoryInitDataReader(&initData)
	// 3. Create initialization data reader - set correct chain owner
	// initDataReader := statetransfer.NewMemoryInitDataReader(&statetransfer.ArbosInitializationInfo{
	// 	ChainOwner: common.HexToAddress("0x860C58951A77ac2F2Cb452cC9F36B1d1FdF7c521"),
	// 	Accounts:   accounts,
	// })

	// 4. Create chain configuration - use complete configuration for target chain
	chainConfig := &params.ChainConfig{
		ChainID:        big.NewInt(71268946402), // Target chain ID
		HomesteadBlock: big.NewInt(0),
		DAOForkBlock:   nil,
		DAOForkSupport: true,
		EIP150Block:    big.NewInt(0),

		EIP155Block:         big.NewInt(0),
		EIP158Block:         big.NewInt(0),
		ByzantiumBlock:      big.NewInt(0),
		ConstantinopleBlock: big.NewInt(0),
		PetersburgBlock:     big.NewInt(0),
		IstanbulBlock:       big.NewInt(0),
		MuirGlacierBlock:    big.NewInt(0),
		BerlinBlock:         big.NewInt(0),
		LondonBlock:         big.NewInt(0),
		ArbitrumChainParams: params.ArbitrumChainParams{
			EnableArbOS:               true,
			AllowDebugPrecompiles:     false,
			DataAvailabilityCommittee: false,
			InitialArbOSVersion:       32,
			InitialChainOwner:         common.HexToAddress("0x860C58951A77ac2F2Cb452cC9F36B1d1FdF7c521"),
			GenesisBlockNum:           0,
			MaxCodeSize:               24576,
			MaxInitCodeSize:           49152,
		},
		Clique: &params.CliqueConfig{
			Period: 0,
			Epoch:  0,
		},
	}
	// Manually construct JSON serialization of chain config, completely matching real node format (field order and content)
	serializedChainConfig := fmt.Sprintf(`{"homesteadBlock":0,"daoForkBlock":null,"daoForkSupport":true,"eip150Block":0,"eip150Hash":"0x0000000000000000000000000000000000000000000000000000000000000000","eip155Block":0,"eip158Block":0,"byzantiumBlock":0,"constantinopleBlock":0,"petersburgBlock":0,"istanbulBlock":0,"muirGlacierBlock":0,"berlinBlock":0,"londonBlock":0,"clique":{"period":0,"epoch":0},"arbitrum":{"EnableArbOS":true,"AllowDebugPrecompiles":false,"DataAvailabilityCommittee":false,"InitialArbOSVersion":%d,"GenesisBlockNum":0,"MaxCodeSize":24576,"MaxInitCodeSize":49152,"InitialChainOwner":"%s"},"chainId":%d}`,
		chainConfig.ArbitrumChainParams.InitialArbOSVersion, chainConfig.ArbitrumChainParams.InitialChainOwner.Hex(), chainConfig.ChainID)

	// 5. Create initialization message
	initMessage := &arbostypes.ParsedInitMessage{
		ChainId:               chainConfig.ChainID,
		InitialL1BaseFee:      big.NewInt(109320000),
		ChainConfig:           chainConfig,
		SerializedChainConfig: []byte(serializedChainConfig),
	}
	log.Info("chainConfig", "chainConfig", chainConfig)
	log.Info("serializedChainConfig", "serializedChainConfig", string(serializedChainConfig))
	log.Info("initMessage", "initMessage", initMessage)

	// 6. Call CalculateArbosStateHash
	stateRoot, err := CalculateArbosStateHash(initDataReader, chainConfig, initMessage, 0)
	if err != nil {
		return "", err
	}

	// 7. Output state root
	return stateRoot.Hex(), nil
	// return "", nil
}
