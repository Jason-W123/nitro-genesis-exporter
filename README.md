# Nitro Genesis Exporter

A tool for calculating Arbitrum state roots and block hashes from genesis.json files using the Nitro codebase.

## Overview

This tool leverages the official Arbitrum Nitro codebase to accurately calculate state roots and block hashes for Arbitrum chains. It's particularly useful for verifying genesis state calculations and understanding how Arbitrum initializes its state.

## Features

- **State Root Calculation**: Computes the state root hash after initializing ArbOS state
- **Block Hash Calculation**: Generates the complete genesis block hash
- **Genesis JSON Support**: Reads standard Ethereum/Arbitrum genesis.json files
- **Detailed Logging**: Provides comprehensive logging of the initialization process
- **Chain Config Handling**: Properly handles Arbitrum-specific chain configuration

## Requirements

- Go 1.21 or higher
- Git (for submodule management)
- Rust toolchain (required for Nitro dependencies)

## Installation

1. Clone the repository with submodules:
```bash
git clone --recursive https://github.com/your-username/nitro-genesis-exporter.git
cd nitro-genesis-exporter
```

2. If you already cloned without submodules, initialize them:
```bash
git submodule update --init --recursive
```

3. Build the tool:
```bash
go build -o nitro-genesis-exporter ./src/
```

## Usage

### Basic Usage

```bash
./nitro-genesis-exporter -g path/to/genesis.json
```

### Command Line Options

- `-g, --genesis`: Path to the genesis.json file (required)

### Example

```bash
./nitro-genesis-exporter -g example-genesis.json
```

**Output:**
```
State Root: 0xdd4441cbdde99d67976c9ac698d4088edb6129d9b0f9da875eb61ad80e01ff0d
Block Hash: 0x1234567890abcdef...
```
(If you get a different hash, it might because your rollup created or will be created with a different gas price, this code default it as 100000000 wei)
## Genesis JSON Format

The tool expects a standard Arbitrum genesis.json file with the following structure:

```json
{
  "config": {
    "chainId": 71268946402,
    "homesteadBlock": 0,
    "eip150Block": 0,
    "eip155Block": 0,
    "eip158Block": 0,
    "byzantiumBlock": 0,
    "constantinopleBlock": 0,
    "petersburgBlock": 0,
    "istanbulBlock": 0,
    "berlinBlock": 0,
    "londonBlock": 0,
    "arbitrum": {
      "EnableArbOS": true,
      "AllowDebugPrecompiles": false,
      "DataAvailabilityCommittee": false,
      "InitialArbOSVersion": 32,
      "GenesisBlockNum": 0,
      "MaxCodeSize": 24576,
      "MaxInitCodeSize": 49152,
      "InitialChainOwner": "0xAddress"
    }
  },
  "nonce": "0x0",
  "timestamp": "0x0",
  "extraData": "0x",
  "gasLimit": "0x1c9c380",
  "difficulty": "0x1",
  "mixHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
  "coinbase": "0x0000000000000000000000000000000000000000",
  "alloc": {
    // Account allocations
  }
}
```

## Technical Details

### State Initialization Process

1. **Genesis Parsing**: Reads and validates the genesis.json file
2. **Account Setup**: Processes account allocations from the genesis file
3. **ArbOS Initialization**: Initializes the Arbitrum Operating System state
4. **Precompile Setup**: Registers all required ArbOS precompiles
5. **Chain Owner Setup**: Configures the chain owner from genesis config
6. **Address Table**: Initializes the address table if needed
7. **Retryables**: Processes any retryable transactions
8. **State Commitment**: Commits the final state and calculates the root hash
9. **Block Creation**: Creates the genesis block and calculates its hash

### Key Components

- **ArbOS State**: Uses the official Nitro ArbOS state management
- **Precompiles**: Includes all standard ArbOS precompiles
- **Chain Configuration**: Handles Arbitrum-specific chain parameters
- **State Database**: Uses in-memory state database for calculations

