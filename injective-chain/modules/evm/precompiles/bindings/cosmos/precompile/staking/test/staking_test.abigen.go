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
	ABI: "[{\"type\":\"function\",\"name\":\"delegate\",\"inputs\":[{\"name\":\"validatorAddress\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"delegation\",\"inputs\":[{\"name\":\"delegatorAddress\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"validatorAddress\",\"type\":\"string\",\"internalType\":\"string\"}],\"outputs\":[{\"name\":\"shares\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"balance\",\"type\":\"tuple\",\"internalType\":\"structCosmos.Coin\",\"components\":[{\"name\":\"amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"denom\",\"type\":\"string\",\"internalType\":\"string\"}]}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"redelegate\",\"inputs\":[{\"name\":\"validatorSrc\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"validatorDst\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"undelegate\",\"inputs\":[{\"name\":\"validatorAddress\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"withdrawDelegatorRewards\",\"inputs\":[{\"name\":\"validatorAddress\",\"type\":\"string\",\"internalType\":\"string\"}],\"outputs\":[{\"name\":\"amount\",\"type\":\"tuple[]\",\"internalType\":\"structCosmos.Coin[]\",\"components\":[{\"name\":\"amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"denom\",\"type\":\"string\",\"internalType\":\"string\"}]}],\"stateMutability\":\"nonpayable\"}]",
	Bin: "0x608060405260665f5f6101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550348015604e575f5ffd5b50610e648061005c5f395ff3fe608060405234801561000f575f5ffd5b5060043610610055575f3560e01c806303f24de114610059578063241774e6146100895780636636125e146100ba5780637dd0209d146100ea5780638dfc88971461011a575b5f5ffd5b610073600480360381019061006e9190610628565b61014a565b604051610080919061069c565b60405180910390f35b6100a3600480360381019061009e919061070f565b6101ee565b6040516100b192919061083b565b60405180910390f35b6100d460048036038101906100cf9190610869565b61029f565b6040516100e191906109a5565b60405180910390f35b61010460048036038101906100ff91906109c5565b610344565b604051610111919061069c565b60405180910390f35b610134600480360381019061012f9190610628565b6103eb565b604051610141919061069c565b60405180910390f35b5f5f5f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166303f24de184846040518363ffffffff1660e01b81526004016101a6929190610a95565b6020604051808303815f875af11580156101c2573d5f5f3e3d5ffd5b505050506040513d601f19601f820116820180604052508101906101e69190610aed565b905092915050565b5f6101f761048f565b5f5f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1663241774e685856040518363ffffffff1660e01b8152600401610252929190610b27565b5f60405180830381865afa15801561026c573d5f5f3e3d5ffd5b505050506040513d5f823e3d601f19601f820116820180604052508101906102949190610c48565b915091509250929050565b60605f5f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16636636125e836040518263ffffffff1660e01b81526004016102fa9190610ca2565b5f604051808303815f875af1158015610315573d5f5f3e3d5ffd5b505050506040513d5f823e3d601f19601f8201168201806040525081019061033d9190610da4565b9050919050565b5f5f5f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16637dd0209d8585856040518463ffffffff1660e01b81526004016103a293929190610deb565b6020604051808303815f875af11580156103be573d5f5f3e3d5ffd5b505050506040513d601f19601f820116820180604052508101906103e29190610aed565b90509392505050565b5f5f5f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16638dfc889784846040518363ffffffff1660e01b8152600401610447929190610a95565b6020604051808303815f875af1158015610463573d5f5f3e3d5ffd5b505050506040513d601f19601f820116820180604052508101906104879190610aed565b905092915050565b60405180604001604052805f8152602001606081525090565b5f604051905090565b5f5ffd5b5f5ffd5b5f5ffd5b5f5ffd5b5f601f19601f8301169050919050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52604160045260245ffd5b610507826104c1565b810181811067ffffffffffffffff82111715610526576105256104d1565b5b80604052505050565b5f6105386104a8565b905061054482826104fe565b919050565b5f67ffffffffffffffff821115610563576105626104d1565b5b61056c826104c1565b9050602081019050919050565b828183375f83830152505050565b5f61059961059484610549565b61052f565b9050828152602081018484840111156105b5576105b46104bd565b5b6105c0848285610579565b509392505050565b5f82601f8301126105dc576105db6104b9565b5b81356105ec848260208601610587565b91505092915050565b5f819050919050565b610607816105f5565b8114610611575f5ffd5b50565b5f81359050610622816105fe565b92915050565b5f5f6040838503121561063e5761063d6104b1565b5b5f83013567ffffffffffffffff81111561065b5761065a6104b5565b5b610667858286016105c8565b925050602061067885828601610614565b9150509250929050565b5f8115159050919050565b61069681610682565b82525050565b5f6020820190506106af5f83018461068d565b92915050565b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f6106de826106b5565b9050919050565b6106ee816106d4565b81146106f8575f5ffd5b50565b5f81359050610709816106e5565b92915050565b5f5f60408385031215610725576107246104b1565b5b5f610732858286016106fb565b925050602083013567ffffffffffffffff811115610753576107526104b5565b5b61075f858286016105c8565b9150509250929050565b610772816105f5565b82525050565b610781816105f5565b82525050565b5f81519050919050565b5f82825260208201905092915050565b5f5b838110156107be5780820151818401526020810190506107a3565b5f8484015250505050565b5f6107d382610787565b6107dd8185610791565b93506107ed8185602086016107a1565b6107f6816104c1565b840191505092915050565b5f604083015f8301516108165f860182610778565b506020830151848203602086015261082e82826107c9565b9150508091505092915050565b5f60408201905061084e5f830185610769565b81810360208301526108608184610801565b90509392505050565b5f6020828403121561087e5761087d6104b1565b5b5f82013567ffffffffffffffff81111561089b5761089a6104b5565b5b6108a7848285016105c8565b91505092915050565b5f81519050919050565b5f82825260208201905092915050565b5f819050602082019050919050565b5f604083015f8301516108ee5f860182610778565b506020830151848203602086015261090682826107c9565b9150508091505092915050565b5f61091e83836108d9565b905092915050565b5f602082019050919050565b5f61093c826108b0565b61094681856108ba565b935083602082028501610958856108ca565b805f5b8581101561099357848403895281516109748582610913565b945061097f83610926565b925060208a0199505060018101905061095b565b50829750879550505050505092915050565b5f6020820190508181035f8301526109bd8184610932565b905092915050565b5f5f5f606084860312156109dc576109db6104b1565b5b5f84013567ffffffffffffffff8111156109f9576109f86104b5565b5b610a05868287016105c8565b935050602084013567ffffffffffffffff811115610a2657610a256104b5565b5b610a32868287016105c8565b9250506040610a4386828701610614565b9150509250925092565b5f82825260208201905092915050565b5f610a6782610787565b610a718185610a4d565b9350610a818185602086016107a1565b610a8a816104c1565b840191505092915050565b5f6040820190508181035f830152610aad8185610a5d565b9050610abc6020830184610769565b9392505050565b610acc81610682565b8114610ad6575f5ffd5b50565b5f81519050610ae781610ac3565b92915050565b5f60208284031215610b0257610b016104b1565b5b5f610b0f84828501610ad9565b91505092915050565b610b21816106d4565b82525050565b5f604082019050610b3a5f830185610b18565b8181036020830152610b4c8184610a5d565b90509392505050565b5f81519050610b63816105fe565b92915050565b5f5ffd5b5f5ffd5b5f610b83610b7e84610549565b61052f565b905082815260208101848484011115610b9f57610b9e6104bd565b5b610baa8482856107a1565b509392505050565b5f82601f830112610bc657610bc56104b9565b5b8151610bd6848260208601610b71565b91505092915050565b5f60408284031215610bf457610bf3610b69565b5b610bfe604061052f565b90505f610c0d84828501610b55565b5f83015250602082015167ffffffffffffffff811115610c3057610c2f610b6d565b5b610c3c84828501610bb2565b60208301525092915050565b5f5f60408385031215610c5e57610c5d6104b1565b5b5f610c6b85828601610b55565b925050602083015167ffffffffffffffff811115610c8c57610c8b6104b5565b5b610c9885828601610bdf565b9150509250929050565b5f6020820190508181035f830152610cba8184610a5d565b905092915050565b5f67ffffffffffffffff821115610cdc57610cdb6104d1565b5b602082029050602081019050919050565b5f5ffd5b5f610d03610cfe84610cc2565b61052f565b90508083825260208201905060208402830185811115610d2657610d25610ced565b5b835b81811015610d6d57805167ffffffffffffffff811115610d4b57610d4a6104b9565b5b808601610d588982610bdf565b85526020850194505050602081019050610d28565b5050509392505050565b5f82601f830112610d8b57610d8a6104b9565b5b8151610d9b848260208601610cf1565b91505092915050565b5f60208284031215610db957610db86104b1565b5b5f82015167ffffffffffffffff811115610dd657610dd56104b5565b5b610de284828501610d77565b91505092915050565b5f6060820190508181035f830152610e038186610a5d565b90508181036020830152610e178185610a5d565b9050610e266040830184610769565b94935050505056fea26469706673582212203dea92cf95d6cdd519aa628811af90db4bd6abbbdd1d2501251b44c68b2856aa64736f6c634300081b0033",
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
