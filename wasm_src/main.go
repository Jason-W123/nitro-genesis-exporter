//go:build wasm
// +build wasm

package wasm_exporter

import (
	"encoding/json"
	"fmt"
	exporter "go-batchhandler/src"
	"syscall/js"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
)

// WASM exported function
func wasm_calculateStateRoot(this js.Value, args []js.Value) interface{} {
	if len(args) != 1 {
		return js.ValueOf(map[string]interface{}{
			"error": "requires one parameter: genesis JSON string",
		})
	}

	genesisJsonStr := args[0].String()

	// blockGasPrice := args[1].Int()

	// Parse genesis JSON
	var gen core.Genesis
	if err := json.Unmarshal([]byte(genesisJsonStr), &gen); err != nil {
		return js.ValueOf(map[string]interface{}{
			"error": fmt.Sprintf("failed to parse genesis JSON: %v", err),
		})
	}

	// Validate required fields
	if gen.Config == nil {
		return js.ValueOf(map[string]interface{}{
			"error": "genesis config is missing",
		})
	}
	if gen.Config.ArbitrumChainParams.InitialChainOwner == (common.Address{}) {
		return js.ValueOf(map[string]interface{}{
			"error": "initial chain owner is missing in genesis config",
		})
	}

	// Note: The actual StateAndBlockRootFromGenesis function uses dependencies
	// (like fastcache, system calls, etc.) that are not compatible with WASM.
	// This is a demonstration that parses the input but cannot perform the full calculation.

	stateRoot, blockHash, err := exporter.StateAndBlockRootFromGenesis(gen)
	if err != nil {
		return js.ValueOf(map[string]interface{}{
			"error": fmt.Sprintf("failed to calculate state root: %v", err),
		})
	}

	// TODO: Implement actual state root calculation here
	// Since full nitro dependencies are not available in WASM, return a sample result
	return js.ValueOf(map[string]interface{}{
		"success":   true,
		"message":   "WASM version does not support full calculation yet, please use native version",
		"stateRoot": stateRoot,
		"blockHash": blockHash,
		"chainId":   gen.Config.ChainID.String(),
		"gasLimit":  fmt.Sprintf("0x%x", gen.GasLimit),
		"timestamp": fmt.Sprintf("0x%x", gen.Timestamp),
		"accounts":  len(gen.Alloc),
	})
}

func main() {
	// Register WASM export function
	js.Global().Set("calculateStateRoot", js.FuncOf(wasm_calculateStateRoot))

	// Keep the program running
	select {}
}
