// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package exchange

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

// ExchangeTestMetaData contains all meta data concerning the ExchangeTest contract.
var ExchangeTestMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[],\"name\":\"counter\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"subaccountID\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"denom\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"delegateDeposit\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"subaccountID\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"denom\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"deposit\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"subaccountID\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"denom\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"depositAndRevert\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"subaccountID\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"denom\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"depositTest1\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"subaccountID\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"denom\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"depositTest2\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"subaccountID\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"denom\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"withdraw\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x608060405260655f806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555034801561004f575f80fd5b50610bfe8061005d5f395ff3fe608060405234801561000f575f80fd5b506004361061007b575f3560e01c806361bc221a1161005957806361bc221a1461010f578063ba73b8181461012d578063e653ab541461015d578063f81a095f1461018d5761007b565b8063222ac0551461007f5780634864c6ab146100af57806354c25b6d146100df575b5f80fd5b610099600480360381019061009491906107e6565b6101bd565b6040516100a69190610888565b60405180910390f35b6100c960048036038101906100c491906107e6565b610266565b6040516100d69190610888565b60405180910390f35b6100f960048036038101906100f491906107e6565b61036d565b6040516101069190610888565b60405180910390f35b610117610449565b60405161012491906108b0565b60405180910390f35b610147600480360381019061014291906107e6565b61044f565b6040516101549190610888565b60405180910390f35b610177600480360381019061017291906107e6565b6104f8565b6040516101849190610888565b60405180910390f35b6101a760048036038101906101a291906107e6565b61058a565b6040516101b49190610888565b60405180910390f35b5f805f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1663e441dec9308686866040518563ffffffff1660e01b815260040161021d9493929190610982565b6020604051808303815f875af1158015610239573d5f803e3d5ffd5b505050506040513d601f19601f8201168201806040525081019061025d91906109fd565b90509392505050565b5f80606573ffffffffffffffffffffffffffffffffffffffff16308686866040516024016102979493929190610982565b6040516020818303038152906040527fb24bd3f9000000000000000000000000000000000000000000000000000000007bffffffffffffffffffffffffffffffffffffffffffffffffffffffff19166020820180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff83818316178352505050506040516103219190610a6c565b5f60405180830381855af49150503d805f8114610359576040519150601f19603f3d011682016040523d82523d5f602084013e61035e565b606091505b50509050809150509392505050565b5f60015f81548092919061038090610aaf565b91905055505f805f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1663e441dec9338787876040518563ffffffff1660e01b81526004016103e59493929190610982565b6020604051808303815f875af1158015610401573d5f803e3d5ffd5b505050506040513d601f19601f8201168201806040525081019061042591906109fd565b905060015f81548092919061043990610af6565b9190505550809150509392505050565b60015481565b5f805f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1663eb28c205308686866040518563ffffffff1660e01b81526004016104af9493929190610982565b6020604051808303815f875af11580156104cb573d5f803e3d5ffd5b505050506040513d601f19601f820116820180604052508101906104ef91906109fd565b90509392505050565b5f3073ffffffffffffffffffffffffffffffffffffffff1663f81a095f8585856040518463ffffffff1660e01b815260040161053693929190610b1d565b6020604051808303815f875af192505050801561057157506040513d601f19601f8201168201806040525081019061056e91906109fd565b60015b1561057f5780915050610583565b5f90505b9392505050565b5f805f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1663e441dec9338686866040518563ffffffff1660e01b81526004016105ea9493929190610982565b6020604051808303815f875af1158015610606573d5f803e3d5ffd5b505050506040513d601f19601f8201168201806040525081019061062a91906109fd565b506040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161065d90610baa565b60405180910390fd5b5f604051905090565b5f80fd5b5f80fd5b5f80fd5b5f80fd5b5f601f19601f8301169050919050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52604160045260245ffd5b6106c58261067f565b810181811067ffffffffffffffff821117156106e4576106e361068f565b5b80604052505050565b5f6106f6610666565b905061070282826106bc565b919050565b5f67ffffffffffffffff8211156107215761072061068f565b5b61072a8261067f565b9050602081019050919050565b828183375f83830152505050565b5f61075761075284610707565b6106ed565b9050828152602081018484840111156107735761077261067b565b5b61077e848285610737565b509392505050565b5f82601f83011261079a57610799610677565b5b81356107aa848260208601610745565b91505092915050565b5f819050919050565b6107c5816107b3565b81146107cf575f80fd5b50565b5f813590506107e0816107bc565b92915050565b5f805f606084860312156107fd576107fc61066f565b5b5f84013567ffffffffffffffff81111561081a57610819610673565b5b61082686828701610786565b935050602084013567ffffffffffffffff81111561084757610846610673565b5b61085386828701610786565b9250506040610864868287016107d2565b9150509250925092565b5f8115159050919050565b6108828161086e565b82525050565b5f60208201905061089b5f830184610879565b92915050565b6108aa816107b3565b82525050565b5f6020820190506108c35f8301846108a1565b92915050565b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f6108f2826108c9565b9050919050565b610902816108e8565b82525050565b5f81519050919050565b5f82825260208201905092915050565b5f5b8381101561093f578082015181840152602081019050610924565b5f8484015250505050565b5f61095482610908565b61095e8185610912565b935061096e818560208601610922565b6109778161067f565b840191505092915050565b5f6080820190506109955f8301876108f9565b81810360208301526109a7818661094a565b905081810360408301526109bb818561094a565b90506109ca60608301846108a1565b95945050505050565b6109dc8161086e565b81146109e6575f80fd5b50565b5f815190506109f7816109d3565b92915050565b5f60208284031215610a1257610a1161066f565b5b5f610a1f848285016109e9565b91505092915050565b5f81519050919050565b5f81905092915050565b5f610a4682610a28565b610a508185610a32565b9350610a60818560208601610922565b80840191505092915050565b5f610a778284610a3c565b915081905092915050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f610ab9826107b3565b91507fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8203610aeb57610aea610a82565b5b600182019050919050565b5f610b00826107b3565b91505f8203610b1257610b11610a82565b5b600182039050919050565b5f6060820190508181035f830152610b35818661094a565b90508181036020830152610b49818561094a565b9050610b5860408301846108a1565b949350505050565b7f74657374696e67000000000000000000000000000000000000000000000000005f82015250565b5f610b94600783610912565b9150610b9f82610b60565b602082019050919050565b5f6020820190508181035f830152610bc181610b88565b905091905056fea26469706673582212202df3eb7d74ed4579f8b01da1bccf13a945f0a6b17377ab1b3334ea503783305164736f6c63430008180033",
}

// ExchangeTestABI is the input ABI used to generate the binding from.
// Deprecated: Use ExchangeTestMetaData.ABI instead.
var ExchangeTestABI = ExchangeTestMetaData.ABI

// ExchangeTestBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use ExchangeTestMetaData.Bin instead.
var ExchangeTestBin = ExchangeTestMetaData.Bin

// DeployExchangeTest deploys a new Ethereum contract, binding an instance of ExchangeTest to it.
func DeployExchangeTest(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *ExchangeTest, error) {
	parsed, err := ExchangeTestMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(ExchangeTestBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &ExchangeTest{ExchangeTestCaller: ExchangeTestCaller{contract: contract}, ExchangeTestTransactor: ExchangeTestTransactor{contract: contract}, ExchangeTestFilterer: ExchangeTestFilterer{contract: contract}}, nil
}

// ExchangeTest is an auto generated Go binding around an Ethereum contract.
type ExchangeTest struct {
	ExchangeTestCaller     // Read-only binding to the contract
	ExchangeTestTransactor // Write-only binding to the contract
	ExchangeTestFilterer   // Log filterer for contract events
}

// ExchangeTestCaller is an auto generated read-only Go binding around an Ethereum contract.
type ExchangeTestCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ExchangeTestTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ExchangeTestTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ExchangeTestFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ExchangeTestFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ExchangeTestSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ExchangeTestSession struct {
	Contract     *ExchangeTest     // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ExchangeTestCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ExchangeTestCallerSession struct {
	Contract *ExchangeTestCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts       // Call options to use throughout this session
}

// ExchangeTestTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ExchangeTestTransactorSession struct {
	Contract     *ExchangeTestTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts       // Transaction auth options to use throughout this session
}

// ExchangeTestRaw is an auto generated low-level Go binding around an Ethereum contract.
type ExchangeTestRaw struct {
	Contract *ExchangeTest // Generic contract binding to access the raw methods on
}

// ExchangeTestCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ExchangeTestCallerRaw struct {
	Contract *ExchangeTestCaller // Generic read-only contract binding to access the raw methods on
}

// ExchangeTestTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ExchangeTestTransactorRaw struct {
	Contract *ExchangeTestTransactor // Generic write-only contract binding to access the raw methods on
}

// NewExchangeTest creates a new instance of ExchangeTest, bound to a specific deployed contract.
func NewExchangeTest(address common.Address, backend bind.ContractBackend) (*ExchangeTest, error) {
	contract, err := bindExchangeTest(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ExchangeTest{ExchangeTestCaller: ExchangeTestCaller{contract: contract}, ExchangeTestTransactor: ExchangeTestTransactor{contract: contract}, ExchangeTestFilterer: ExchangeTestFilterer{contract: contract}}, nil
}

// NewExchangeTestCaller creates a new read-only instance of ExchangeTest, bound to a specific deployed contract.
func NewExchangeTestCaller(address common.Address, caller bind.ContractCaller) (*ExchangeTestCaller, error) {
	contract, err := bindExchangeTest(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ExchangeTestCaller{contract: contract}, nil
}

// NewExchangeTestTransactor creates a new write-only instance of ExchangeTest, bound to a specific deployed contract.
func NewExchangeTestTransactor(address common.Address, transactor bind.ContractTransactor) (*ExchangeTestTransactor, error) {
	contract, err := bindExchangeTest(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ExchangeTestTransactor{contract: contract}, nil
}

// NewExchangeTestFilterer creates a new log filterer instance of ExchangeTest, bound to a specific deployed contract.
func NewExchangeTestFilterer(address common.Address, filterer bind.ContractFilterer) (*ExchangeTestFilterer, error) {
	contract, err := bindExchangeTest(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ExchangeTestFilterer{contract: contract}, nil
}

// bindExchangeTest binds a generic wrapper to an already deployed contract.
func bindExchangeTest(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ExchangeTestMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ExchangeTest *ExchangeTestRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ExchangeTest.Contract.ExchangeTestCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ExchangeTest *ExchangeTestRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ExchangeTest.Contract.ExchangeTestTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ExchangeTest *ExchangeTestRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ExchangeTest.Contract.ExchangeTestTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ExchangeTest *ExchangeTestCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ExchangeTest.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ExchangeTest *ExchangeTestTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ExchangeTest.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ExchangeTest *ExchangeTestTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ExchangeTest.Contract.contract.Transact(opts, method, params...)
}

// Counter is a free data retrieval call binding the contract method 0x61bc221a.
//
// Solidity: function counter() view returns(uint256)
func (_ExchangeTest *ExchangeTestCaller) Counter(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _ExchangeTest.contract.Call(opts, &out, "counter")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Counter is a free data retrieval call binding the contract method 0x61bc221a.
//
// Solidity: function counter() view returns(uint256)
func (_ExchangeTest *ExchangeTestSession) Counter() (*big.Int, error) {
	return _ExchangeTest.Contract.Counter(&_ExchangeTest.CallOpts)
}

// Counter is a free data retrieval call binding the contract method 0x61bc221a.
//
// Solidity: function counter() view returns(uint256)
func (_ExchangeTest *ExchangeTestCallerSession) Counter() (*big.Int, error) {
	return _ExchangeTest.Contract.Counter(&_ExchangeTest.CallOpts)
}

// DelegateDeposit is a paid mutator transaction binding the contract method 0x4864c6ab.
//
// Solidity: function delegateDeposit(string subaccountID, string denom, uint256 amount) returns(bool)
func (_ExchangeTest *ExchangeTestTransactor) DelegateDeposit(opts *bind.TransactOpts, subaccountID string, denom string, amount *big.Int) (*types.Transaction, error) {
	return _ExchangeTest.contract.Transact(opts, "delegateDeposit", subaccountID, denom, amount)
}

// DelegateDeposit is a paid mutator transaction binding the contract method 0x4864c6ab.
//
// Solidity: function delegateDeposit(string subaccountID, string denom, uint256 amount) returns(bool)
func (_ExchangeTest *ExchangeTestSession) DelegateDeposit(subaccountID string, denom string, amount *big.Int) (*types.Transaction, error) {
	return _ExchangeTest.Contract.DelegateDeposit(&_ExchangeTest.TransactOpts, subaccountID, denom, amount)
}

// DelegateDeposit is a paid mutator transaction binding the contract method 0x4864c6ab.
//
// Solidity: function delegateDeposit(string subaccountID, string denom, uint256 amount) returns(bool)
func (_ExchangeTest *ExchangeTestTransactorSession) DelegateDeposit(subaccountID string, denom string, amount *big.Int) (*types.Transaction, error) {
	return _ExchangeTest.Contract.DelegateDeposit(&_ExchangeTest.TransactOpts, subaccountID, denom, amount)
}

// Deposit is a paid mutator transaction binding the contract method 0x222ac055.
//
// Solidity: function deposit(string subaccountID, string denom, uint256 amount) returns(bool)
func (_ExchangeTest *ExchangeTestTransactor) Deposit(opts *bind.TransactOpts, subaccountID string, denom string, amount *big.Int) (*types.Transaction, error) {
	return _ExchangeTest.contract.Transact(opts, "deposit", subaccountID, denom, amount)
}

// Deposit is a paid mutator transaction binding the contract method 0x222ac055.
//
// Solidity: function deposit(string subaccountID, string denom, uint256 amount) returns(bool)
func (_ExchangeTest *ExchangeTestSession) Deposit(subaccountID string, denom string, amount *big.Int) (*types.Transaction, error) {
	return _ExchangeTest.Contract.Deposit(&_ExchangeTest.TransactOpts, subaccountID, denom, amount)
}

// Deposit is a paid mutator transaction binding the contract method 0x222ac055.
//
// Solidity: function deposit(string subaccountID, string denom, uint256 amount) returns(bool)
func (_ExchangeTest *ExchangeTestTransactorSession) Deposit(subaccountID string, denom string, amount *big.Int) (*types.Transaction, error) {
	return _ExchangeTest.Contract.Deposit(&_ExchangeTest.TransactOpts, subaccountID, denom, amount)
}

// DepositAndRevert is a paid mutator transaction binding the contract method 0xf81a095f.
//
// Solidity: function depositAndRevert(string subaccountID, string denom, uint256 amount) returns(bool)
func (_ExchangeTest *ExchangeTestTransactor) DepositAndRevert(opts *bind.TransactOpts, subaccountID string, denom string, amount *big.Int) (*types.Transaction, error) {
	return _ExchangeTest.contract.Transact(opts, "depositAndRevert", subaccountID, denom, amount)
}

// DepositAndRevert is a paid mutator transaction binding the contract method 0xf81a095f.
//
// Solidity: function depositAndRevert(string subaccountID, string denom, uint256 amount) returns(bool)
func (_ExchangeTest *ExchangeTestSession) DepositAndRevert(subaccountID string, denom string, amount *big.Int) (*types.Transaction, error) {
	return _ExchangeTest.Contract.DepositAndRevert(&_ExchangeTest.TransactOpts, subaccountID, denom, amount)
}

// DepositAndRevert is a paid mutator transaction binding the contract method 0xf81a095f.
//
// Solidity: function depositAndRevert(string subaccountID, string denom, uint256 amount) returns(bool)
func (_ExchangeTest *ExchangeTestTransactorSession) DepositAndRevert(subaccountID string, denom string, amount *big.Int) (*types.Transaction, error) {
	return _ExchangeTest.Contract.DepositAndRevert(&_ExchangeTest.TransactOpts, subaccountID, denom, amount)
}

// DepositTest1 is a paid mutator transaction binding the contract method 0x54c25b6d.
//
// Solidity: function depositTest1(string subaccountID, string denom, uint256 amount) returns(bool)
func (_ExchangeTest *ExchangeTestTransactor) DepositTest1(opts *bind.TransactOpts, subaccountID string, denom string, amount *big.Int) (*types.Transaction, error) {
	return _ExchangeTest.contract.Transact(opts, "depositTest1", subaccountID, denom, amount)
}

// DepositTest1 is a paid mutator transaction binding the contract method 0x54c25b6d.
//
// Solidity: function depositTest1(string subaccountID, string denom, uint256 amount) returns(bool)
func (_ExchangeTest *ExchangeTestSession) DepositTest1(subaccountID string, denom string, amount *big.Int) (*types.Transaction, error) {
	return _ExchangeTest.Contract.DepositTest1(&_ExchangeTest.TransactOpts, subaccountID, denom, amount)
}

// DepositTest1 is a paid mutator transaction binding the contract method 0x54c25b6d.
//
// Solidity: function depositTest1(string subaccountID, string denom, uint256 amount) returns(bool)
func (_ExchangeTest *ExchangeTestTransactorSession) DepositTest1(subaccountID string, denom string, amount *big.Int) (*types.Transaction, error) {
	return _ExchangeTest.Contract.DepositTest1(&_ExchangeTest.TransactOpts, subaccountID, denom, amount)
}

// DepositTest2 is a paid mutator transaction binding the contract method 0xe653ab54.
//
// Solidity: function depositTest2(string subaccountID, string denom, uint256 amount) returns(bool)
func (_ExchangeTest *ExchangeTestTransactor) DepositTest2(opts *bind.TransactOpts, subaccountID string, denom string, amount *big.Int) (*types.Transaction, error) {
	return _ExchangeTest.contract.Transact(opts, "depositTest2", subaccountID, denom, amount)
}

// DepositTest2 is a paid mutator transaction binding the contract method 0xe653ab54.
//
// Solidity: function depositTest2(string subaccountID, string denom, uint256 amount) returns(bool)
func (_ExchangeTest *ExchangeTestSession) DepositTest2(subaccountID string, denom string, amount *big.Int) (*types.Transaction, error) {
	return _ExchangeTest.Contract.DepositTest2(&_ExchangeTest.TransactOpts, subaccountID, denom, amount)
}

// DepositTest2 is a paid mutator transaction binding the contract method 0xe653ab54.
//
// Solidity: function depositTest2(string subaccountID, string denom, uint256 amount) returns(bool)
func (_ExchangeTest *ExchangeTestTransactorSession) DepositTest2(subaccountID string, denom string, amount *big.Int) (*types.Transaction, error) {
	return _ExchangeTest.Contract.DepositTest2(&_ExchangeTest.TransactOpts, subaccountID, denom, amount)
}

// PanicDurningDeposit is a paid mutator transaction binding the contract method 0xab85e0bf.
//
// Solidity: function panicDurningDeposit() returns()
func (_ExchangeTest *ExchangeTestTransactor) PanicDurningDeposit(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ExchangeTest.contract.Transact(opts, "panicDurningDeposit")
}

// PanicDurningDeposit is a paid mutator transaction binding the contract method 0xab85e0bf.
//
// Solidity: function panicDurningDeposit() returns()
func (_ExchangeTest *ExchangeTestSession) PanicDurningDeposit() (*types.Transaction, error) {
	return _ExchangeTest.Contract.PanicDurningDeposit(&_ExchangeTest.TransactOpts)
}

// PanicDurningDeposit is a paid mutator transaction binding the contract method 0xab85e0bf.
//
// Solidity: function panicDurningDeposit() returns()
func (_ExchangeTest *ExchangeTestTransactorSession) PanicDurningDeposit() (*types.Transaction, error) {
	return _ExchangeTest.Contract.PanicDurningDeposit(&_ExchangeTest.TransactOpts)
}

// Withdraw is a paid mutator transaction binding the contract method 0xba73b818.
//
// Solidity: function withdraw(string subaccountID, string denom, uint256 amount) returns(bool)
func (_ExchangeTest *ExchangeTestTransactor) Withdraw(opts *bind.TransactOpts, subaccountID string, denom string, amount *big.Int) (*types.Transaction, error) {
	return _ExchangeTest.contract.Transact(opts, "withdraw", subaccountID, denom, amount)
}

// Withdraw is a paid mutator transaction binding the contract method 0xba73b818.
//
// Solidity: function withdraw(string subaccountID, string denom, uint256 amount) returns(bool)
func (_ExchangeTest *ExchangeTestSession) Withdraw(subaccountID string, denom string, amount *big.Int) (*types.Transaction, error) {
	return _ExchangeTest.Contract.Withdraw(&_ExchangeTest.TransactOpts, subaccountID, denom, amount)
}

// Withdraw is a paid mutator transaction binding the contract method 0xba73b818.
//
// Solidity: function withdraw(string subaccountID, string denom, uint256 amount) returns(bool)
func (_ExchangeTest *ExchangeTestTransactorSession) Withdraw(subaccountID string, denom string, amount *big.Int) (*types.Transaction, error) {
	return _ExchangeTest.Contract.Withdraw(&_ExchangeTest.TransactOpts, subaccountID, denom, amount)
}
