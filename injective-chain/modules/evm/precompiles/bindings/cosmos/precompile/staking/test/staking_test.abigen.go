// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package staking

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

// StakingTestMetaData contains all meta data concerning the StakingTest contract.
var StakingTestMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"string\",\"name\":\"validatorAddress\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"delegate\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"delegatorAddress\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"validatorAddress\",\"type\":\"string\"}],\"name\":\"delegation\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"shares\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"string\",\"name\":\"denom\",\"type\":\"string\"}],\"internalType\":\"structCosmos.Coin\",\"name\":\"balance\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"validatorSrc\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"validatorDst\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"redelegate\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"validatorAddress\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"undelegate\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"validatorAddress\",\"type\":\"string\"}],\"name\":\"withdrawDelegatorRewards\",\"outputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"string\",\"name\":\"denom\",\"type\":\"string\"}],\"internalType\":\"structCosmos.Coin[]\",\"name\":\"amount\",\"type\":\"tuple[]\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x608060405260665f806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555034801561004f575f80fd5b50610e628061005d5f395ff3fe608060405234801561000f575f80fd5b5060043610610055575f3560e01c806303f24de114610059578063241774e6146100895780636636125e146100ba5780637dd0209d146100ea5780638dfc88971461011a575b5f80fd5b610073600480360381019061006e9190610626565b61014a565b604051610080919061069a565b60405180910390f35b6100a3600480360381019061009e919061070d565b6101ee565b6040516100b1929190610839565b60405180910390f35b6100d460048036038101906100cf9190610867565b61029e565b6040516100e191906109a3565b60405180910390f35b61010460048036038101906100ff91906109c3565b610342565b604051610111919061069a565b60405180910390f35b610134600480360381019061012f9190610626565b6103e9565b604051610141919061069a565b60405180910390f35b5f805f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166303f24de184846040518363ffffffff1660e01b81526004016101a6929190610a93565b6020604051808303815f875af11580156101c2573d5f803e3d5ffd5b505050506040513d601f19601f820116820180604052508101906101e69190610aeb565b905092915050565b5f6101f761048d565b5f8054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1663241774e685856040518363ffffffff1660e01b8152600401610251929190610b25565b5f60405180830381865afa15801561026b573d5f803e3d5ffd5b505050506040513d5f823e3d601f19601f820116820180604052508101906102939190610c46565b915091509250929050565b60605f8054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16636636125e836040518263ffffffff1660e01b81526004016102f89190610ca0565b5f604051808303815f875af1158015610313573d5f803e3d5ffd5b505050506040513d5f823e3d601f19601f8201168201806040525081019061033b9190610da2565b9050919050565b5f805f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16637dd0209d8585856040518463ffffffff1660e01b81526004016103a093929190610de9565b6020604051808303815f875af11580156103bc573d5f803e3d5ffd5b505050506040513d601f19601f820116820180604052508101906103e09190610aeb565b90509392505050565b5f805f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16638dfc889784846040518363ffffffff1660e01b8152600401610445929190610a93565b6020604051808303815f875af1158015610461573d5f803e3d5ffd5b505050506040513d601f19601f820116820180604052508101906104859190610aeb565b905092915050565b60405180604001604052805f8152602001606081525090565b5f604051905090565b5f80fd5b5f80fd5b5f80fd5b5f80fd5b5f601f19601f8301169050919050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52604160045260245ffd5b610505826104bf565b810181811067ffffffffffffffff82111715610524576105236104cf565b5b80604052505050565b5f6105366104a6565b905061054282826104fc565b919050565b5f67ffffffffffffffff821115610561576105606104cf565b5b61056a826104bf565b9050602081019050919050565b828183375f83830152505050565b5f61059761059284610547565b61052d565b9050828152602081018484840111156105b3576105b26104bb565b5b6105be848285610577565b509392505050565b5f82601f8301126105da576105d96104b7565b5b81356105ea848260208601610585565b91505092915050565b5f819050919050565b610605816105f3565b811461060f575f80fd5b50565b5f81359050610620816105fc565b92915050565b5f806040838503121561063c5761063b6104af565b5b5f83013567ffffffffffffffff811115610659576106586104b3565b5b610665858286016105c6565b925050602061067685828601610612565b9150509250929050565b5f8115159050919050565b61069481610680565b82525050565b5f6020820190506106ad5f83018461068b565b92915050565b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f6106dc826106b3565b9050919050565b6106ec816106d2565b81146106f6575f80fd5b50565b5f81359050610707816106e3565b92915050565b5f8060408385031215610723576107226104af565b5b5f610730858286016106f9565b925050602083013567ffffffffffffffff811115610751576107506104b3565b5b61075d858286016105c6565b9150509250929050565b610770816105f3565b82525050565b61077f816105f3565b82525050565b5f81519050919050565b5f82825260208201905092915050565b5f5b838110156107bc5780820151818401526020810190506107a1565b5f8484015250505050565b5f6107d182610785565b6107db818561078f565b93506107eb81856020860161079f565b6107f4816104bf565b840191505092915050565b5f604083015f8301516108145f860182610776565b506020830151848203602086015261082c82826107c7565b9150508091505092915050565b5f60408201905061084c5f830185610767565b818103602083015261085e81846107ff565b90509392505050565b5f6020828403121561087c5761087b6104af565b5b5f82013567ffffffffffffffff811115610899576108986104b3565b5b6108a5848285016105c6565b91505092915050565b5f81519050919050565b5f82825260208201905092915050565b5f819050602082019050919050565b5f604083015f8301516108ec5f860182610776565b506020830151848203602086015261090482826107c7565b9150508091505092915050565b5f61091c83836108d7565b905092915050565b5f602082019050919050565b5f61093a826108ae565b61094481856108b8565b935083602082028501610956856108c8565b805f5b8581101561099157848403895281516109728582610911565b945061097d83610924565b925060208a01995050600181019050610959565b50829750879550505050505092915050565b5f6020820190508181035f8301526109bb8184610930565b905092915050565b5f805f606084860312156109da576109d96104af565b5b5f84013567ffffffffffffffff8111156109f7576109f66104b3565b5b610a03868287016105c6565b935050602084013567ffffffffffffffff811115610a2457610a236104b3565b5b610a30868287016105c6565b9250506040610a4186828701610612565b9150509250925092565b5f82825260208201905092915050565b5f610a6582610785565b610a6f8185610a4b565b9350610a7f81856020860161079f565b610a88816104bf565b840191505092915050565b5f6040820190508181035f830152610aab8185610a5b565b9050610aba6020830184610767565b9392505050565b610aca81610680565b8114610ad4575f80fd5b50565b5f81519050610ae581610ac1565b92915050565b5f60208284031215610b0057610aff6104af565b5b5f610b0d84828501610ad7565b91505092915050565b610b1f816106d2565b82525050565b5f604082019050610b385f830185610b16565b8181036020830152610b4a8184610a5b565b90509392505050565b5f81519050610b61816105fc565b92915050565b5f80fd5b5f80fd5b5f610b81610b7c84610547565b61052d565b905082815260208101848484011115610b9d57610b9c6104bb565b5b610ba884828561079f565b509392505050565b5f82601f830112610bc457610bc36104b7565b5b8151610bd4848260208601610b6f565b91505092915050565b5f60408284031215610bf257610bf1610b67565b5b610bfc604061052d565b90505f610c0b84828501610b53565b5f83015250602082015167ffffffffffffffff811115610c2e57610c2d610b6b565b5b610c3a84828501610bb0565b60208301525092915050565b5f8060408385031215610c5c57610c5b6104af565b5b5f610c6985828601610b53565b925050602083015167ffffffffffffffff811115610c8a57610c896104b3565b5b610c9685828601610bdd565b9150509250929050565b5f6020820190508181035f830152610cb88184610a5b565b905092915050565b5f67ffffffffffffffff821115610cda57610cd96104cf565b5b602082029050602081019050919050565b5f80fd5b5f610d01610cfc84610cc0565b61052d565b90508083825260208201905060208402830185811115610d2457610d23610ceb565b5b835b81811015610d6b57805167ffffffffffffffff811115610d4957610d486104b7565b5b808601610d568982610bdd565b85526020850194505050602081019050610d26565b5050509392505050565b5f82601f830112610d8957610d886104b7565b5b8151610d99848260208601610cef565b91505092915050565b5f60208284031215610db757610db66104af565b5b5f82015167ffffffffffffffff811115610dd457610dd36104b3565b5b610de084828501610d75565b91505092915050565b5f6060820190508181035f830152610e018186610a5b565b90508181036020830152610e158185610a5b565b9050610e246040830184610767565b94935050505056fea2646970667358221220d5ea018b5a500c46e8531813cc837309d9f2aa0849d1fdcf88eae299da99359764736f6c63430008180033",
}

// StakingTestABI is the input ABI used to generate the binding from.
// Deprecated: Use StakingTestMetaData.ABI instead.
var StakingTestABI = StakingTestMetaData.ABI

// StakingTestBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use StakingTestMetaData.Bin instead.
var StakingTestBin = StakingTestMetaData.Bin

// DeployStakingTest deploys a new Ethereum contract, binding an instance of StakingTest to it.
func DeployStakingTest(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *StakingTest, error) {
	parsed, err := StakingTestMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(StakingTestBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &StakingTest{StakingTestCaller: StakingTestCaller{contract: contract}, StakingTestTransactor: StakingTestTransactor{contract: contract}, StakingTestFilterer: StakingTestFilterer{contract: contract}}, nil
}

// StakingTest is an auto generated Go binding around an Ethereum contract.
type StakingTest struct {
	StakingTestCaller     // Read-only binding to the contract
	StakingTestTransactor // Write-only binding to the contract
	StakingTestFilterer   // Log filterer for contract events
}

// StakingTestCaller is an auto generated read-only Go binding around an Ethereum contract.
type StakingTestCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StakingTestTransactor is an auto generated write-only Go binding around an Ethereum contract.
type StakingTestTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StakingTestFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type StakingTestFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StakingTestSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type StakingTestSession struct {
	Contract     *StakingTest      // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// StakingTestCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type StakingTestCallerSession struct {
	Contract *StakingTestCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts      // Call options to use throughout this session
}

// StakingTestTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type StakingTestTransactorSession struct {
	Contract     *StakingTestTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts      // Transaction auth options to use throughout this session
}

// StakingTestRaw is an auto generated low-level Go binding around an Ethereum contract.
type StakingTestRaw struct {
	Contract *StakingTest // Generic contract binding to access the raw methods on
}

// StakingTestCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type StakingTestCallerRaw struct {
	Contract *StakingTestCaller // Generic read-only contract binding to access the raw methods on
}

// StakingTestTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type StakingTestTransactorRaw struct {
	Contract *StakingTestTransactor // Generic write-only contract binding to access the raw methods on
}

// NewStakingTest creates a new instance of StakingTest, bound to a specific deployed contract.
func NewStakingTest(address common.Address, backend bind.ContractBackend) (*StakingTest, error) {
	contract, err := bindStakingTest(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &StakingTest{StakingTestCaller: StakingTestCaller{contract: contract}, StakingTestTransactor: StakingTestTransactor{contract: contract}, StakingTestFilterer: StakingTestFilterer{contract: contract}}, nil
}

// NewStakingTestCaller creates a new read-only instance of StakingTest, bound to a specific deployed contract.
func NewStakingTestCaller(address common.Address, caller bind.ContractCaller) (*StakingTestCaller, error) {
	contract, err := bindStakingTest(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &StakingTestCaller{contract: contract}, nil
}

// NewStakingTestTransactor creates a new write-only instance of StakingTest, bound to a specific deployed contract.
func NewStakingTestTransactor(address common.Address, transactor bind.ContractTransactor) (*StakingTestTransactor, error) {
	contract, err := bindStakingTest(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &StakingTestTransactor{contract: contract}, nil
}

// NewStakingTestFilterer creates a new log filterer instance of StakingTest, bound to a specific deployed contract.
func NewStakingTestFilterer(address common.Address, filterer bind.ContractFilterer) (*StakingTestFilterer, error) {
	contract, err := bindStakingTest(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &StakingTestFilterer{contract: contract}, nil
}

// bindStakingTest binds a generic wrapper to an already deployed contract.
func bindStakingTest(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := StakingTestMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_StakingTest *StakingTestRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _StakingTest.Contract.StakingTestCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_StakingTest *StakingTestRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _StakingTest.Contract.StakingTestTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_StakingTest *StakingTestRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _StakingTest.Contract.StakingTestTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_StakingTest *StakingTestCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _StakingTest.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_StakingTest *StakingTestTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _StakingTest.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_StakingTest *StakingTestTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _StakingTest.Contract.contract.Transact(opts, method, params...)
}

// Delegation is a free data retrieval call binding the contract method 0x241774e6.
//
// Solidity: function delegation(address delegatorAddress, string validatorAddress) view returns(uint256 shares, (uint256,string) balance)
func (_StakingTest *StakingTestCaller) Delegation(opts *bind.CallOpts, delegatorAddress common.Address, validatorAddress string) (struct {
	Shares  *big.Int
	Balance CosmosCoin
}, error) {
	var out []interface{}
	err := _StakingTest.contract.Call(opts, &out, "delegation", delegatorAddress, validatorAddress)

	outstruct := new(struct {
		Shares  *big.Int
		Balance CosmosCoin
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Shares = *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)
	outstruct.Balance = *abi.ConvertType(out[1], new(CosmosCoin)).(*CosmosCoin)

	return *outstruct, err

}

// Delegation is a free data retrieval call binding the contract method 0x241774e6.
//
// Solidity: function delegation(address delegatorAddress, string validatorAddress) view returns(uint256 shares, (uint256,string) balance)
func (_StakingTest *StakingTestSession) Delegation(delegatorAddress common.Address, validatorAddress string) (struct {
	Shares  *big.Int
	Balance CosmosCoin
}, error) {
	return _StakingTest.Contract.Delegation(&_StakingTest.CallOpts, delegatorAddress, validatorAddress)
}

// Delegation is a free data retrieval call binding the contract method 0x241774e6.
//
// Solidity: function delegation(address delegatorAddress, string validatorAddress) view returns(uint256 shares, (uint256,string) balance)
func (_StakingTest *StakingTestCallerSession) Delegation(delegatorAddress common.Address, validatorAddress string) (struct {
	Shares  *big.Int
	Balance CosmosCoin
}, error) {
	return _StakingTest.Contract.Delegation(&_StakingTest.CallOpts, delegatorAddress, validatorAddress)
}

// Delegate is a paid mutator transaction binding the contract method 0x03f24de1.
//
// Solidity: function delegate(string validatorAddress, uint256 amount) returns(bool)
func (_StakingTest *StakingTestTransactor) Delegate(opts *bind.TransactOpts, validatorAddress string, amount *big.Int) (*types.Transaction, error) {
	return _StakingTest.contract.Transact(opts, "delegate", validatorAddress, amount)
}

// Delegate is a paid mutator transaction binding the contract method 0x03f24de1.
//
// Solidity: function delegate(string validatorAddress, uint256 amount) returns(bool)
func (_StakingTest *StakingTestSession) Delegate(validatorAddress string, amount *big.Int) (*types.Transaction, error) {
	return _StakingTest.Contract.Delegate(&_StakingTest.TransactOpts, validatorAddress, amount)
}

// Delegate is a paid mutator transaction binding the contract method 0x03f24de1.
//
// Solidity: function delegate(string validatorAddress, uint256 amount) returns(bool)
func (_StakingTest *StakingTestTransactorSession) Delegate(validatorAddress string, amount *big.Int) (*types.Transaction, error) {
	return _StakingTest.Contract.Delegate(&_StakingTest.TransactOpts, validatorAddress, amount)
}

// Redelegate is a paid mutator transaction binding the contract method 0x7dd0209d.
//
// Solidity: function redelegate(string validatorSrc, string validatorDst, uint256 amount) returns(bool)
func (_StakingTest *StakingTestTransactor) Redelegate(opts *bind.TransactOpts, validatorSrc string, validatorDst string, amount *big.Int) (*types.Transaction, error) {
	return _StakingTest.contract.Transact(opts, "redelegate", validatorSrc, validatorDst, amount)
}

// Redelegate is a paid mutator transaction binding the contract method 0x7dd0209d.
//
// Solidity: function redelegate(string validatorSrc, string validatorDst, uint256 amount) returns(bool)
func (_StakingTest *StakingTestSession) Redelegate(validatorSrc string, validatorDst string, amount *big.Int) (*types.Transaction, error) {
	return _StakingTest.Contract.Redelegate(&_StakingTest.TransactOpts, validatorSrc, validatorDst, amount)
}

// Redelegate is a paid mutator transaction binding the contract method 0x7dd0209d.
//
// Solidity: function redelegate(string validatorSrc, string validatorDst, uint256 amount) returns(bool)
func (_StakingTest *StakingTestTransactorSession) Redelegate(validatorSrc string, validatorDst string, amount *big.Int) (*types.Transaction, error) {
	return _StakingTest.Contract.Redelegate(&_StakingTest.TransactOpts, validatorSrc, validatorDst, amount)
}

// Undelegate is a paid mutator transaction binding the contract method 0x8dfc8897.
//
// Solidity: function undelegate(string validatorAddress, uint256 amount) returns(bool)
func (_StakingTest *StakingTestTransactor) Undelegate(opts *bind.TransactOpts, validatorAddress string, amount *big.Int) (*types.Transaction, error) {
	return _StakingTest.contract.Transact(opts, "undelegate", validatorAddress, amount)
}

// Undelegate is a paid mutator transaction binding the contract method 0x8dfc8897.
//
// Solidity: function undelegate(string validatorAddress, uint256 amount) returns(bool)
func (_StakingTest *StakingTestSession) Undelegate(validatorAddress string, amount *big.Int) (*types.Transaction, error) {
	return _StakingTest.Contract.Undelegate(&_StakingTest.TransactOpts, validatorAddress, amount)
}

// Undelegate is a paid mutator transaction binding the contract method 0x8dfc8897.
//
// Solidity: function undelegate(string validatorAddress, uint256 amount) returns(bool)
func (_StakingTest *StakingTestTransactorSession) Undelegate(validatorAddress string, amount *big.Int) (*types.Transaction, error) {
	return _StakingTest.Contract.Undelegate(&_StakingTest.TransactOpts, validatorAddress, amount)
}

// WithdrawDelegatorRewards is a paid mutator transaction binding the contract method 0x6636125e.
//
// Solidity: function withdrawDelegatorRewards(string validatorAddress) returns((uint256,string)[] amount)
func (_StakingTest *StakingTestTransactor) WithdrawDelegatorRewards(opts *bind.TransactOpts, validatorAddress string) (*types.Transaction, error) {
	return _StakingTest.contract.Transact(opts, "withdrawDelegatorRewards", validatorAddress)
}

// WithdrawDelegatorRewards is a paid mutator transaction binding the contract method 0x6636125e.
//
// Solidity: function withdrawDelegatorRewards(string validatorAddress) returns((uint256,string)[] amount)
func (_StakingTest *StakingTestSession) WithdrawDelegatorRewards(validatorAddress string) (*types.Transaction, error) {
	return _StakingTest.Contract.WithdrawDelegatorRewards(&_StakingTest.TransactOpts, validatorAddress)
}

// WithdrawDelegatorRewards is a paid mutator transaction binding the contract method 0x6636125e.
//
// Solidity: function withdrawDelegatorRewards(string validatorAddress) returns((uint256,string)[] amount)
func (_StakingTest *StakingTestTransactorSession) WithdrawDelegatorRewards(validatorAddress string) (*types.Transaction, error) {
	return _StakingTest.Contract.WithdrawDelegatorRewards(&_StakingTest.TransactOpts, validatorAddress)
}
