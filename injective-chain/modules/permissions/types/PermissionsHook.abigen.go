// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package types

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// CosmosCoin is an auto generated low-level Go binding around an user-defined struct.
type CosmosCoin struct {
	Amount *big.Int
	Denom  string
}

// PermissionsHookMetaData contains all meta data concerning the PermissionsHook contract.
var PermissionsHookMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"function\",\"name\":\"isTransferRestricted\",\"inputs\":[{\"name\":\"from\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"to\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"amount\",\"type\":\"tuple\",\"internalType\":\"structCosmos.Coin\",\"components\":[{\"name\":\"amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"denom\",\"type\":\"string\",\"internalType\":\"string\"}]}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"}][{\"type\":\"function\",\"name\":\"isTransferRestricted\",\"inputs\":[{\"name\":\"from\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"to\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"amount\",\"type\":\"tuple\",\"internalType\":\"structCosmos.Coin\",\"components\":[{\"name\":\"amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"denom\",\"type\":\"string\",\"internalType\":\"string\"}]}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"}]",
}

// PermissionsHookABI is the input ABI used to generate the binding from.
// Deprecated: Use PermissionsHookMetaData.ABI instead.
var PermissionsHookABI = PermissionsHookMetaData.ABI

// PermissionsHook is an auto generated Go binding around an Ethereum contract.
type PermissionsHook struct {
	PermissionsHookCaller     // Read-only binding to the contract
	PermissionsHookTransactor // Write-only binding to the contract
	PermissionsHookFilterer   // Log filterer for contract events
}

// PermissionsHookCaller is an auto generated read-only Go binding around an Ethereum contract.
type PermissionsHookCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PermissionsHookTransactor is an auto generated write-only Go binding around an Ethereum contract.
type PermissionsHookTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PermissionsHookFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type PermissionsHookFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PermissionsHookSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type PermissionsHookSession struct {
	Contract     *PermissionsHook  // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// PermissionsHookCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type PermissionsHookCallerSession struct {
	Contract *PermissionsHookCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts          // Call options to use throughout this session
}

// PermissionsHookTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type PermissionsHookTransactorSession struct {
	Contract     *PermissionsHookTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts          // Transaction auth options to use throughout this session
}

// PermissionsHookRaw is an auto generated low-level Go binding around an Ethereum contract.
type PermissionsHookRaw struct {
	Contract *PermissionsHook // Generic contract binding to access the raw methods on
}

// PermissionsHookCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type PermissionsHookCallerRaw struct {
	Contract *PermissionsHookCaller // Generic read-only contract binding to access the raw methods on
}

// PermissionsHookTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type PermissionsHookTransactorRaw struct {
	Contract *PermissionsHookTransactor // Generic write-only contract binding to access the raw methods on
}

// NewPermissionsHook creates a new instance of PermissionsHook, bound to a specific deployed contract.
func NewPermissionsHook(address common.Address, backend bind.ContractBackend) (*PermissionsHook, error) {
	contract, err := bindPermissionsHook(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &PermissionsHook{PermissionsHookCaller: PermissionsHookCaller{contract: contract}, PermissionsHookTransactor: PermissionsHookTransactor{contract: contract}, PermissionsHookFilterer: PermissionsHookFilterer{contract: contract}}, nil
}

// NewPermissionsHookCaller creates a new read-only instance of PermissionsHook, bound to a specific deployed contract.
func NewPermissionsHookCaller(address common.Address, caller bind.ContractCaller) (*PermissionsHookCaller, error) {
	contract, err := bindPermissionsHook(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &PermissionsHookCaller{contract: contract}, nil
}

// NewPermissionsHookTransactor creates a new write-only instance of PermissionsHook, bound to a specific deployed contract.
func NewPermissionsHookTransactor(address common.Address, transactor bind.ContractTransactor) (*PermissionsHookTransactor, error) {
	contract, err := bindPermissionsHook(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &PermissionsHookTransactor{contract: contract}, nil
}

// NewPermissionsHookFilterer creates a new log filterer instance of PermissionsHook, bound to a specific deployed contract.
func NewPermissionsHookFilterer(address common.Address, filterer bind.ContractFilterer) (*PermissionsHookFilterer, error) {
	contract, err := bindPermissionsHook(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &PermissionsHookFilterer{contract: contract}, nil
}

// bindPermissionsHook binds a generic wrapper to an already deployed contract.
func bindPermissionsHook(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := PermissionsHookMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_PermissionsHook *PermissionsHookRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _PermissionsHook.Contract.PermissionsHookCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_PermissionsHook *PermissionsHookRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _PermissionsHook.Contract.PermissionsHookTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_PermissionsHook *PermissionsHookRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _PermissionsHook.Contract.PermissionsHookTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_PermissionsHook *PermissionsHookCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _PermissionsHook.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_PermissionsHook *PermissionsHookTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _PermissionsHook.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_PermissionsHook *PermissionsHookTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _PermissionsHook.Contract.contract.Transact(opts, method, params...)
}

// IsTransferRestricted is a free data retrieval call binding the contract method 0xe4e69baf.
//
// Solidity: function isTransferRestricted(address from, address to, (uint256,string) amount) view returns(bool)
func (_PermissionsHook *PermissionsHookCaller) IsTransferRestricted(opts *bind.CallOpts, from common.Address, to common.Address, amount CosmosCoin) (bool, error) {
	var out []interface{}
	err := _PermissionsHook.contract.Call(opts, &out, "isTransferRestricted", from, to, amount)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsTransferRestricted is a free data retrieval call binding the contract method 0xe4e69baf.
//
// Solidity: function isTransferRestricted(address from, address to, (uint256,string) amount) view returns(bool)
func (_PermissionsHook *PermissionsHookSession) IsTransferRestricted(from common.Address, to common.Address, amount CosmosCoin) (bool, error) {
	return _PermissionsHook.Contract.IsTransferRestricted(&_PermissionsHook.CallOpts, from, to, amount)
}

// IsTransferRestricted is a free data retrieval call binding the contract method 0xe4e69baf.
//
// Solidity: function isTransferRestricted(address from, address to, (uint256,string) amount) view returns(bool)
func (_PermissionsHook *PermissionsHookCallerSession) IsTransferRestricted(from common.Address, to common.Address, amount CosmosCoin) (bool, error) {
	return _PermissionsHook.Contract.IsTransferRestricted(&_PermissionsHook.CallOpts, from, to, amount)
}
