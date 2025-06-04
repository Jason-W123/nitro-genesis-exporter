// Copyright 2021-2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package exporter

import (
	"errors"
	"fmt"
	"sort"

	"github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/triedb"

	"github.com/offchainlabs/nitro/arbos/arbosState"
	"github.com/offchainlabs/nitro/arbos/arbostypes"
	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/retryables"
	"github.com/offchainlabs/nitro/gethhook"
	"github.com/offchainlabs/nitro/statetransfer"
	"github.com/offchainlabs/nitro/util/arbmath"
)

// reuse the code from nitro
func initializeRetryables(statedb *state.StateDB, rs *retryables.RetryableState, initData statetransfer.RetryableDataReader, currentTimestamp uint64) error {
	var retryablesList []*statetransfer.InitializationDataForRetryable
	for initData.More() {
		r, err := initData.GetNext()
		if err != nil {
			return err
		}
		if r.Timeout <= currentTimestamp {
			statedb.AddBalance(r.Beneficiary, uint256.MustFromBig(r.Callvalue), tracing.BalanceChangeUnspecified)
			log.Info("After AddBalance (timeout retryable)", "beneficiary", r.Beneficiary, "amount", r.Callvalue, "root", statedb.IntermediateRoot(true))
			continue
		}
		retryablesList = append(retryablesList, r)
	}
	sort.Slice(retryablesList, func(i, j int) bool {
		a := retryablesList[i]
		b := retryablesList[j]
		if a.Timeout == b.Timeout {
			return arbmath.BigLessThan(a.Id.Big(), b.Id.Big())
		}
		return a.Timeout < b.Timeout
	})
	for _, r := range retryablesList {
		var to *common.Address
		if r.To != (common.Address{}) {
			addr := r.To
			to = &addr
		}
		statedb.AddBalance(retryables.RetryableEscrowAddress(r.Id), uint256.MustFromBig(r.Callvalue), tracing.BalanceChangeUnspecified)
		log.Info("After AddBalance (retryable escrow)", "escrowAddr", retryables.RetryableEscrowAddress(r.Id), "amount", r.Callvalue, "root", statedb.IntermediateRoot(true))

		_, err := rs.CreateRetryable(r.Id, r.Timeout, r.From, to, r.Callvalue, r.Beneficiary, r.Calldata)
		if err != nil {
			return err
		}
		log.Info("After CreateRetryable", "id", r.Id, "root", statedb.IntermediateRoot(true))
	}
	return initData.Close()
}

// reuse the code from nitro
func initializeArbosAccount(_ *state.StateDB, arbosStateInstance *arbosState.ArbosState, account statetransfer.AccountInitializationInfo) error {
	fmt.Println("initializeArbosAccount", account)
	l1pState := arbosStateInstance.L1PricingState()
	posterTable := l1pState.BatchPosterTable()
	if account.AggregatorInfo != nil {
		isPoster, err := posterTable.ContainsPoster(account.Addr)
		if err != nil {
			return err
		}
		if isPoster {
			// poster is already authorized, just set its fee collector
			poster, err := posterTable.OpenPoster(account.Addr, false)
			if err != nil {
				return err
			}
			err = poster.SetPayTo(account.AggregatorInfo.FeeCollector)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func CalculateArbosStateHash(initData statetransfer.InitDataReader, chainConfig *params.ChainConfig, initMessage *arbostypes.ParsedInitMessage, timestamp uint64) (root common.Hash, err error) {

	// Use simple in-memory database, no complex configuration needed
	db := rawdb.NewMemoryDatabase()
	stateDatabase := state.NewDatabase(triedb.NewDatabase(db, nil), nil)
	defer func() {
		err = errors.Join(err, stateDatabase.TrieDB().Close())
	}()

	statedb, err := state.New(types.EmptyRootHash, stateDatabase)
	if err != nil {
		panic("failed to init empty statedb :" + err.Error())
	}

	log.Info("Initial state", "root", statedb.IntermediateRoot(true))

	// Initialize gethhook will initialize precompiles, which is important when we calculate arbos state hash
	gethhook.RequireHookedGeth()
	log.Info("Geth hook and Precompiles initialized", "Precompilecount", len(arbosState.PrecompileMinArbOSVersions))

	burner := burn.NewSystemBurner(nil, false)

	arbosStateInstance, err := arbosState.InitializeArbosState(statedb, burner, chainConfig, nil, initMessage)
	if err != nil {
		panic("failed to open the ArbOS state :" + err.Error())
	}
	log.Info("After InitializeArbosState", "root", statedb.IntermediateRoot(true))

	// Print detailed ArbosState information
	// logArbosStateDetails(arbosStateInstance)

	// Set chain owner
	chainOwner, err := initData.GetChainOwner()
	if err != nil {
		return common.Hash{}, err
	}
	if chainOwner != (common.Address{}) {
		err = arbosStateInstance.ChainOwners().Add(chainOwner)
		if err != nil {
			return common.Hash{}, err
		}
	}

	// Initialize address table
	addrTable := arbosStateInstance.AddressTable()
	addrTableSize, err := addrTable.Size()
	if err != nil {
		return common.Hash{}, err
	}
	if addrTableSize != 0 {
		return common.Hash{}, errors.New("address table must be empty")
	}
	addressReader, err := initData.GetAddressTableReader()
	if err != nil {
		return common.Hash{}, err
	}
	for i := uint64(0); addressReader.More(); i++ {
		addr, err := addressReader.GetNext()
		if err != nil {
			return common.Hash{}, err
		}
		slot, err := addrTable.Register(*addr)
		if err != nil {
			return common.Hash{}, err
		}
		if i != slot {
			return common.Hash{}, errors.New("address table slot mismatch")
		}
		log.Info("After address table register", "addr", addr, "slot", slot, "root", statedb.IntermediateRoot(true))
	}
	if err := addressReader.Close(); err != nil {
		return common.Hash{}, err
	}

	// Initialize retryables
	retryableReader, err := initData.GetRetryableDataReader()
	if err != nil {
		return common.Hash{}, err
	}
	err = initializeRetryables(statedb, arbosStateInstance.RetryableState(), retryableReader, timestamp)
	if err != nil {
		return common.Hash{}, err
	}
	log.Info("After initializeRetryables", "root", statedb.IntermediateRoot(true))

	// Initialize account data
	accountDataReader, err := initData.GetAccountDataReader()
	if err != nil {
		return common.Hash{}, err
	}
	for accountDataReader.More() {
		account, err := accountDataReader.GetNext()
		if err != nil {
			return common.Hash{}, err
		}
		err = initializeArbosAccount(statedb, arbosStateInstance, *account)
		if err != nil {
			return common.Hash{}, err
		}

		statedb.SetBalance(account.Addr, uint256.MustFromBig(account.EthBalance), tracing.BalanceChangeUnspecified)

		statedb.SetNonce(account.Addr, account.Nonce, tracing.NonceChangeUnspecified)

		if account.ContractInfo != nil {
			statedb.SetCode(account.Addr, account.ContractInfo.Code)

			for k, v := range account.ContractInfo.ContractStorage {
				statedb.SetState(account.Addr, k, v)
			}
		}
		log.Info("After initialize Account", "addr", account.Addr, "root", statedb.IntermediateRoot(true))
	}
	if err := accountDataReader.Close(); err != nil {
		return common.Hash{}, err
	}

	// Only calculate state hash, do not write to database
	root, err = statedb.Commit(chainConfig.ArbitrumChainParams.GenesisBlockNum, true, false)
	log.Info("Final Commit", "root", root)
	return root, err
}
