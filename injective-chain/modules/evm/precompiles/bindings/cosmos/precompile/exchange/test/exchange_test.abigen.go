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
	ABI: "[{\"inputs\":[{\"internalType\":\"ExchangeTypes.MsgType\",\"name\":\"msgType\",\"type\":\"uint8\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"string\",\"name\":\"denom\",\"type\":\"string\"}],\"internalType\":\"structCosmos.Coin[]\",\"name\":\"spendLimit\",\"type\":\"tuple[]\"},{\"internalType\":\"uint256\",\"name\":\"duration\",\"type\":\"uint256\"}],\"name\":\"approve\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"counter\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"subaccountID\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"denom\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"delegateDeposit\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"subaccountID\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"denom\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"deposit\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"subaccountID\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"denom\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"depositAndRevert\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"subaccountID\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"denom\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"depositTest1\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"subaccountID\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"denom\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"depositTest2\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"subaccountID\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"denom\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"withdraw\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x608060405260655f806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555034801561004f575f80fd5b506112c18061005d5f395ff3fe608060405234801561000f575f80fd5b5060043610610086575f3560e01c806361bc221a1161005957806361bc221a1461014a578063ba73b81814610168578063e653ab5414610198578063f81a095f146101c857610086565b8063222ac0551461008a5780632df0f577146100ba5780634864c6ab146100ea57806354c25b6d1461011a575b5f80fd5b6100a4600480360381019061009f919061097b565b6101f8565b6040516100b19190610a1d565b60405180910390f35b6100d460048036038101906100cf9190610bbf565b6102a1565b6040516100e19190610a1d565b60405180910390f35b61010460048036038101906100ff919061097b565b6103fb565b6040516101119190610a1d565b60405180910390f35b610134600480360381019061012f919061097b565b610502565b6040516101419190610a1d565b60405180910390f35b6101526105de565b60405161015f9190610c3a565b60405180910390f35b610182600480360381019061017d919061097b565b6105e4565b60405161018f9190610a1d565b60405180910390f35b6101b260048036038101906101ad919061097b565b61068d565b6040516101bf9190610a1d565b60405180910390f35b6101e260048036038101906101dd919061097b565b61071f565b6040516101ef9190610a1d565b60405180910390f35b5f805f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1663e441dec9308686866040518563ffffffff1660e01b81526004016102589493929190610d0c565b6020604051808303815f875af1158015610274573d5f803e3d5ffd5b505050506040513d601f19601f820116820180604052508101906102989190610d87565b90509392505050565b5f80600167ffffffffffffffff8111156102be576102bd610824565b5b6040519080825280602002602001820160405280156102ec5781602001602082028036833780820191505090505b50905084815f8151811061030357610302610db2565b5b602002602001019060ff16908160ff16815250505f8054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166328e06289308387876040518563ffffffff1660e01b8152600401610375949392919061100c565b6020604051808303815f875af19250505080156103b057506040513d601f19601f820116820180604052508101906103ad9190610d87565b60015b6103ef576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016103e6906110cd565b60405180910390fd5b80925050509392505050565b5f80606573ffffffffffffffffffffffffffffffffffffffff163086868660405160240161042c9493929190610d0c565b6040516020818303038152906040527fb24bd3f9000000000000000000000000000000000000000000000000000000007bffffffffffffffffffffffffffffffffffffffffffffffffffffffff19166020820180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff83818316178352505050506040516104b6919061112f565b5f60405180830381855af49150503d805f81146104ee576040519150601f19603f3d011682016040523d82523d5f602084013e6104f3565b606091505b50509050809150509392505050565b5f60015f81548092919061051590611172565b91905055505f805f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1663e441dec9338787876040518563ffffffff1660e01b815260040161057a9493929190610d0c565b6020604051808303815f875af1158015610596573d5f803e3d5ffd5b505050506040513d601f19601f820116820180604052508101906105ba9190610d87565b905060015f8154809291906105ce906111b9565b9190505550809150509392505050565b60015481565b5f805f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1663eb28c205308686866040518563ffffffff1660e01b81526004016106449493929190610d0c565b6020604051808303815f875af1158015610660573d5f803e3d5ffd5b505050506040513d601f19601f820116820180604052508101906106849190610d87565b90509392505050565b5f3073ffffffffffffffffffffffffffffffffffffffff1663f81a095f8585856040518463ffffffff1660e01b81526004016106cb939291906111e0565b6020604051808303815f875af192505050801561070657506040513d601f19601f820116820180604052508101906107039190610d87565b60015b156107145780915050610718565b5f90505b9392505050565b5f805f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1663e441dec9338686866040518563ffffffff1660e01b815260040161077f9493929190610d0c565b6020604051808303815f875af115801561079b573d5f803e3d5ffd5b505050506040513d601f19601f820116820180604052508101906107bf9190610d87565b506040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016107f29061126d565b60405180910390fd5b5f604051905090565b5f80fd5b5f80fd5b5f80fd5b5f80fd5b5f601f19601f8301169050919050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52604160045260245ffd5b61085a82610814565b810181811067ffffffffffffffff8211171561087957610878610824565b5b80604052505050565b5f61088b6107fb565b90506108978282610851565b919050565b5f67ffffffffffffffff8211156108b6576108b5610824565b5b6108bf82610814565b9050602081019050919050565b828183375f83830152505050565b5f6108ec6108e78461089c565b610882565b90508281526020810184848401111561090857610907610810565b5b6109138482856108cc565b509392505050565b5f82601f83011261092f5761092e61080c565b5b813561093f8482602086016108da565b91505092915050565b5f819050919050565b61095a81610948565b8114610964575f80fd5b50565b5f8135905061097581610951565b92915050565b5f805f6060848603121561099257610991610804565b5b5f84013567ffffffffffffffff8111156109af576109ae610808565b5b6109bb8682870161091b565b935050602084013567ffffffffffffffff8111156109dc576109db610808565b5b6109e88682870161091b565b92505060406109f986828701610967565b9150509250925092565b5f8115159050919050565b610a1781610a03565b82525050565b5f602082019050610a305f830184610a0e565b92915050565b5f60ff82169050919050565b610a4b81610a36565b8114610a55575f80fd5b50565b5f81359050610a6681610a42565b92915050565b5f67ffffffffffffffff821115610a8657610a85610824565b5b602082029050602081019050919050565b5f80fd5b5f80fd5b5f80fd5b5f60408284031215610ab857610ab7610a9b565b5b610ac26040610882565b90505f610ad184828501610967565b5f83015250602082013567ffffffffffffffff811115610af457610af3610a9f565b5b610b008482850161091b565b60208301525092915050565b5f610b1e610b1984610a6c565b610882565b90508083825260208201905060208402830185811115610b4157610b40610a97565b5b835b81811015610b8857803567ffffffffffffffff811115610b6657610b6561080c565b5b808601610b738982610aa3565b85526020850194505050602081019050610b43565b5050509392505050565b5f82601f830112610ba657610ba561080c565b5b8135610bb6848260208601610b0c565b91505092915050565b5f805f60608486031215610bd657610bd5610804565b5b5f610be386828701610a58565b935050602084013567ffffffffffffffff811115610c0457610c03610808565b5b610c1086828701610b92565b9250506040610c2186828701610967565b9150509250925092565b610c3481610948565b82525050565b5f602082019050610c4d5f830184610c2b565b92915050565b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f610c7c82610c53565b9050919050565b610c8c81610c72565b82525050565b5f81519050919050565b5f82825260208201905092915050565b5f5b83811015610cc9578082015181840152602081019050610cae565b5f8484015250505050565b5f610cde82610c92565b610ce88185610c9c565b9350610cf8818560208601610cac565b610d0181610814565b840191505092915050565b5f608082019050610d1f5f830187610c83565b8181036020830152610d318186610cd4565b90508181036040830152610d458185610cd4565b9050610d546060830184610c2b565b95945050505050565b610d6681610a03565b8114610d70575f80fd5b50565b5f81519050610d8181610d5d565b92915050565b5f60208284031215610d9c57610d9b610804565b5b5f610da984828501610d73565b91505092915050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52603260045260245ffd5b5f81519050919050565b5f82825260208201905092915050565b5f819050602082019050919050565b5f819050919050565b5f610e2b610e26610e2184610a36565b610e08565b610a36565b9050919050565b610e3b81610e11565b82525050565b5f610e4c8383610e32565b60208301905092915050565b5f602082019050919050565b5f610e6e82610ddf565b610e788185610de9565b9350610e8383610df9565b805f5b83811015610eb3578151610e9a8882610e41565b9750610ea583610e58565b925050600181019050610e86565b5085935050505092915050565b5f81519050919050565b5f82825260208201905092915050565b5f819050602082019050919050565b610ef281610948565b82525050565b5f82825260208201905092915050565b5f610f1282610c92565b610f1c8185610ef8565b9350610f2c818560208601610cac565b610f3581610814565b840191505092915050565b5f604083015f830151610f555f860182610ee9565b5060208301518482036020860152610f6d8282610f08565b9150508091505092915050565b5f610f858383610f40565b905092915050565b5f602082019050919050565b5f610fa382610ec0565b610fad8185610eca565b935083602082028501610fbf85610eda565b805f5b85811015610ffa5784840389528151610fdb8582610f7a565b9450610fe683610f8d565b925060208a01995050600181019050610fc2565b50829750879550505050505092915050565b5f60808201905061101f5f830187610c83565b81810360208301526110318186610e64565b905081810360408301526110458185610f99565b90506110546060830184610c2b565b95945050505050565b7f6572726f7220617070726f76696e67206d73672077697468207370656e64206c5f8201527f696d697400000000000000000000000000000000000000000000000000000000602082015250565b5f6110b7602483610c9c565b91506110c28261105d565b604082019050919050565b5f6020820190508181035f8301526110e4816110ab565b9050919050565b5f81519050919050565b5f81905092915050565b5f611109826110eb565b61111381856110f5565b9350611123818560208601610cac565b80840191505092915050565b5f61113a82846110ff565b915081905092915050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f61117c82610948565b91507fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff82036111ae576111ad611145565b5b600182019050919050565b5f6111c382610948565b91505f82036111d5576111d4611145565b5b600182039050919050565b5f6060820190508181035f8301526111f88186610cd4565b9050818103602083015261120c8185610cd4565b905061121b6040830184610c2b565b949350505050565b7f74657374696e67000000000000000000000000000000000000000000000000005f82015250565b5f611257600783610c9c565b915061126282611223565b602082019050919050565b5f6020820190508181035f8301526112848161124b565b905091905056fea26469706673582212205322684ec8c0bcea297c085fb1afb0bf304ff7aa8f7ee900f9e372b2f0be0d8564736f6c63430008180033",
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

// Approve is a paid mutator transaction binding the contract method 0x2df0f577.
//
// Solidity: function approve(uint8 msgType, (uint256,string)[] spendLimit, uint256 duration) returns(bool success)
func (_ExchangeTest *ExchangeTestTransactor) Approve(opts *bind.TransactOpts, msgType uint8, spendLimit []CosmosCoin, duration *big.Int) (*types.Transaction, error) {
	return _ExchangeTest.contract.Transact(opts, "approve", msgType, spendLimit, duration)
}

// Approve is a paid mutator transaction binding the contract method 0x2df0f577.
//
// Solidity: function approve(uint8 msgType, (uint256,string)[] spendLimit, uint256 duration) returns(bool success)
func (_ExchangeTest *ExchangeTestSession) Approve(msgType uint8, spendLimit []CosmosCoin, duration *big.Int) (*types.Transaction, error) {
	return _ExchangeTest.Contract.Approve(&_ExchangeTest.TransactOpts, msgType, spendLimit, duration)
}

// Approve is a paid mutator transaction binding the contract method 0x2df0f577.
//
// Solidity: function approve(uint8 msgType, (uint256,string)[] spendLimit, uint256 duration) returns(bool success)
func (_ExchangeTest *ExchangeTestTransactorSession) Approve(msgType uint8, spendLimit []CosmosCoin, duration *big.Int) (*types.Transaction, error) {
	return _ExchangeTest.Contract.Approve(&_ExchangeTest.TransactOpts, msgType, spendLimit, duration)
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
