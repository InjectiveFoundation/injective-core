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

// CosmosCoin is an auto generated low-level Go binding around an user-defined struct.
type CosmosCoin struct {
	Amount *big.Int
	Denom  string
}

// IExchangeModuleCreateDerivativeLimitOrderResponse is an auto generated low-level Go binding around an user-defined struct.
type IExchangeModuleCreateDerivativeLimitOrderResponse struct {
	OrderHash string
	Cid       string
}

// IExchangeModuleDerivativeOrder is an auto generated low-level Go binding around an user-defined struct.
type IExchangeModuleDerivativeOrder struct {
	MarketID     string
	SubaccountID string
	FeeRecipient string
	Price        *big.Int
	Quantity     *big.Int
	Cid          string
	OrderType    string
	Margin       *big.Int
	TriggerPrice *big.Int
}

// ExchangeProxyMetaData contains all meta data concerning the ExchangeProxy contract.
var ExchangeProxyMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"ExchangeTypes.MsgType\",\"name\":\"msgType\",\"type\":\"uint8\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"},{\"internalType\":\"string\",\"name\":\"denom\",\"type\":\"string\"}],\"internalType\":\"structCosmos.Coin[]\",\"name\":\"spendLimit\",\"type\":\"tuple[]\"},{\"internalType\":\"uint256\",\"name\":\"duration\",\"type\":\"uint256\"}],\"name\":\"approve\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"string\",\"name\":\"marketID\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"subaccountID\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"feeRecipient\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"price\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quantity\",\"type\":\"uint256\"},{\"internalType\":\"string\",\"name\":\"cid\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"orderType\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"margin\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"triggerPrice\",\"type\":\"uint256\"}],\"internalType\":\"structIExchangeModule.DerivativeOrder\",\"name\":\"order\",\"type\":\"tuple\"}],\"name\":\"createDerivativeLimitOrder\",\"outputs\":[{\"components\":[{\"internalType\":\"string\",\"name\":\"orderHash\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"cid\",\"type\":\"string\"}],\"internalType\":\"structIExchangeModule.CreateDerivativeLimitOrderResponse\",\"name\":\"response\",\"type\":\"tuple\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"grantee\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"granter\",\"type\":\"address\"},{\"internalType\":\"ExchangeTypes.MsgType\",\"name\":\"msgType\",\"type\":\"uint8\"}],\"name\":\"queryAllowance\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"allowed\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"ExchangeTypes.MsgType\",\"name\":\"msgType\",\"type\":\"uint8\"}],\"name\":\"revoke\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x608060405260655f806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555034801561004f575f80fd5b506114388061005d5f395ff3fe608060405234801561000f575f80fd5b506004361061004a575f3560e01c806320c698371461004e5780632df0f5771461007e5780635ce4a731146100ae578063d9a06a90146100de575b5f80fd5b6100686004803603810190610063919061062f565b61010e565b6040516100759190610754565b60405180910390f35b61009860048036038101906100939190610a5c565b6101f7565b6040516100a59190610ae2565b60405180910390f35b6100c860048036038101906100c39190610afb565b610351565b6040516100d59190610ae2565b60405180910390f35b6100f860048036038101906100f39190610b4b565b610433565b6040516101059190610ae2565b60405180910390f35b610116610587565b5f8054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166320c6983784846040518363ffffffff1660e01b8152600401610170929190610d74565b5f604051808303815f875af19250505080156101ae57506040513d5f823e3d601f19601f820116820180604052508101906101ab9190610e95565b60015b6101ed576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016101e490610f5c565b60405180910390fd5b8091505092915050565b5f80600167ffffffffffffffff811115610214576102136107ae565b5b6040519080825280602002602001820160405280156102425781602001602082028036833780820191505090505b50905084815f8151811061025957610258610f7a565b5b602002602001019060ff16908160ff16815250505f8054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166328e06289308387876040518563ffffffff1660e01b81526004016102cb949392919061118c565b6020604051808303815f875af192505050801561030657506040513d601f19601f820116820180604052508101906103039190611207565b60015b610345576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161033c906112a2565b60405180910390fd5b80925050509392505050565b5f805f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166327be18a88585856040518463ffffffff1660e01b81526004016103af939291906112cf565b602060405180830381865afa9250505080156103e957506040513d601f19601f820116820180604052508101906103e69190611207565b60015b610428576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161041f9061134e565b60405180910390fd5b809150509392505050565b5f80600167ffffffffffffffff8111156104505761044f6107ae565b5b60405190808252806020026020018201604052801561047e5781602001602082028036833780820191505090505b50905082815f8151811061049557610494610f7a565b5b602002602001019060ff16908160ff16815250505f8054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16633f79e2b130836040518363ffffffff1660e01b815260040161050392919061136c565b6020604051808303815f875af192505050801561053e57506040513d601f19601f8201168201806040525081019061053b9190611207565b60015b61057d576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610574906113e4565b60405180910390fd5b8092505050919050565b604051806040016040528060608152602001606081525090565b5f604051905090565b5f80fd5b5f80fd5b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f6105db826105b2565b9050919050565b6105eb816105d1565b81146105f5575f80fd5b50565b5f81359050610606816105e2565b92915050565b5f80fd5b5f61012082840312156106265761062561060c565b5b81905092915050565b5f8060408385031215610645576106446105aa565b5b5f610652858286016105f8565b925050602083013567ffffffffffffffff811115610673576106726105ae565b5b61067f85828601610610565b9150509250929050565b5f81519050919050565b5f82825260208201905092915050565b5f5b838110156106c05780820151818401526020810190506106a5565b5f8484015250505050565b5f601f19601f8301169050919050565b5f6106e582610689565b6106ef8185610693565b93506106ff8185602086016106a3565b610708816106cb565b840191505092915050565b5f604083015f8301518482035f86015261072d82826106db565b9150506020830151848203602086015261074782826106db565b9150508091505092915050565b5f6020820190508181035f83015261076c8184610713565b905092915050565b5f60ff82169050919050565b61078981610774565b8114610793575f80fd5b50565b5f813590506107a481610780565b92915050565b5f80fd5b7f4e487b71000000000000000000000000000000000000000000000000000000005f52604160045260245ffd5b6107e4826106cb565b810181811067ffffffffffffffff82111715610803576108026107ae565b5b80604052505050565b5f6108156105a1565b905061082182826107db565b919050565b5f67ffffffffffffffff8211156108405761083f6107ae565b5b602082029050602081019050919050565b5f80fd5b5f80fd5b5f80fd5b5f819050919050565b61086f8161085d565b8114610879575f80fd5b50565b5f8135905061088a81610866565b92915050565b5f80fd5b5f67ffffffffffffffff8211156108ae576108ad6107ae565b5b6108b7826106cb565b9050602081019050919050565b828183375f83830152505050565b5f6108e46108df84610894565b61080c565b905082815260208101848484011115610900576108ff610890565b5b61090b8482856108c4565b509392505050565b5f82601f830112610927576109266107aa565b5b81356109378482602086016108d2565b91505092915050565b5f6040828403121561095557610954610855565b5b61095f604061080c565b90505f61096e8482850161087c565b5f83015250602082013567ffffffffffffffff81111561099157610990610859565b5b61099d84828501610913565b60208301525092915050565b5f6109bb6109b684610826565b61080c565b905080838252602082019050602084028301858111156109de576109dd610851565b5b835b81811015610a2557803567ffffffffffffffff811115610a0357610a026107aa565b5b808601610a108982610940565b855260208501945050506020810190506109e0565b5050509392505050565b5f82601f830112610a4357610a426107aa565b5b8135610a538482602086016109a9565b91505092915050565b5f805f60608486031215610a7357610a726105aa565b5b5f610a8086828701610796565b935050602084013567ffffffffffffffff811115610aa157610aa06105ae565b5b610aad86828701610a2f565b9250506040610abe8682870161087c565b9150509250925092565b5f8115159050919050565b610adc81610ac8565b82525050565b5f602082019050610af55f830184610ad3565b92915050565b5f805f60608486031215610b1257610b116105aa565b5b5f610b1f868287016105f8565b9350506020610b30868287016105f8565b9250506040610b4186828701610796565b9150509250925092565b5f60208284031215610b6057610b5f6105aa565b5b5f610b6d84828501610796565b91505092915050565b610b7f816105d1565b82525050565b5f80fd5b5f80fd5b5f80fd5b5f8083356001602003843603038112610bad57610bac610b8d565b5b83810192508235915060208301925067ffffffffffffffff821115610bd557610bd4610b85565b5b600182023603831315610beb57610bea610b89565b5b509250929050565b5f610bfe8385610693565b9350610c0b8385846108c4565b610c14836106cb565b840190509392505050565b5f610c2d602084018461087c565b905092915050565b610c3e8161085d565b82525050565b5f6101208301610c565f840184610b91565b8583035f870152610c68838284610bf3565b92505050610c796020840184610b91565b8583036020870152610c8c838284610bf3565b92505050610c9d6040840184610b91565b8583036040870152610cb0838284610bf3565b92505050610cc16060840184610c1f565b610cce6060860182610c35565b50610cdc6080840184610c1f565b610ce96080860182610c35565b50610cf760a0840184610b91565b85830360a0870152610d0a838284610bf3565b92505050610d1b60c0840184610b91565b85830360c0870152610d2e838284610bf3565b92505050610d3f60e0840184610c1f565b610d4c60e0860182610c35565b50610d5b610100840184610c1f565b610d69610100860182610c35565b508091505092915050565b5f604082019050610d875f830185610b76565b8181036020830152610d998184610c44565b90509392505050565b5f610db4610daf84610894565b61080c565b905082815260208101848484011115610dd057610dcf610890565b5b610ddb8482856106a3565b509392505050565b5f82601f830112610df757610df66107aa565b5b8151610e07848260208601610da2565b91505092915050565b5f60408284031215610e2557610e24610855565b5b610e2f604061080c565b90505f82015167ffffffffffffffff811115610e4e57610e4d610859565b5b610e5a84828501610de3565b5f83015250602082015167ffffffffffffffff811115610e7d57610e7c610859565b5b610e8984828501610de3565b60208301525092915050565b5f60208284031215610eaa57610ea96105aa565b5b5f82015167ffffffffffffffff811115610ec757610ec66105ae565b5b610ed384828501610e10565b91505092915050565b5f82825260208201905092915050565b7f6572726f72206372656174696e672064657269766174697665206c696d6974205f8201527f6f72646572000000000000000000000000000000000000000000000000000000602082015250565b5f610f46602583610edc565b9150610f5182610eec565b604082019050919050565b5f6020820190508181035f830152610f7381610f3a565b9050919050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52603260045260245ffd5b5f81519050919050565b5f82825260208201905092915050565b5f819050602082019050919050565b5f819050919050565b5f610ff3610fee610fe984610774565b610fd0565b610774565b9050919050565b61100381610fd9565b82525050565b5f6110148383610ffa565b60208301905092915050565b5f602082019050919050565b5f61103682610fa7565b6110408185610fb1565b935061104b83610fc1565b805f5b8381101561107b5781516110628882611009565b975061106d83611020565b92505060018101905061104e565b5085935050505092915050565b5f81519050919050565b5f82825260208201905092915050565b5f819050602082019050919050565b5f604083015f8301516110c65f860182610c35565b50602083015184820360208601526110de82826106db565b9150508091505092915050565b5f6110f683836110b1565b905092915050565b5f602082019050919050565b5f61111482611088565b61111e8185611092565b935083602082028501611130856110a2565b805f5b8581101561116b578484038952815161114c85826110eb565b9450611157836110fe565b925060208a01995050600181019050611133565b50829750879550505050505092915050565b6111868161085d565b82525050565b5f60808201905061119f5f830187610b76565b81810360208301526111b1818661102c565b905081810360408301526111c5818561110a565b90506111d4606083018461117d565b95945050505050565b6111e681610ac8565b81146111f0575f80fd5b50565b5f81519050611201816111dd565b92915050565b5f6020828403121561121c5761121b6105aa565b5b5f611229848285016111f3565b91505092915050565b7f6572726f7220617070726f76696e67206d73672077697468207370656e64206c5f8201527f696d697400000000000000000000000000000000000000000000000000000000602082015250565b5f61128c602483610edc565b915061129782611232565b604082019050919050565b5f6020820190508181035f8301526112b981611280565b9050919050565b6112c981610fd9565b82525050565b5f6060820190506112e25f830186610b76565b6112ef6020830185610b76565b6112fc60408301846112c0565b949350505050565b7f6572726f72207175657279696e6720616c6c6f77616e636500000000000000005f82015250565b5f611338601883610edc565b915061134382611304565b602082019050919050565b5f6020820190508181035f8301526113658161132c565b9050919050565b5f60408201905061137f5f830185610b76565b8181036020830152611391818461102c565b90509392505050565b7f6572726f72207265766f6b696e67206d6574686f6400000000000000000000005f82015250565b5f6113ce601583610edc565b91506113d98261139a565b602082019050919050565b5f6020820190508181035f8301526113fb816113c2565b905091905056fea26469706673582212202ea81aea83e83488bd90800722ea23368e41bef9d058ebc34d3e852b79dc959664736f6c63430008180033",
}

// ExchangeProxyABI is the input ABI used to generate the binding from.
// Deprecated: Use ExchangeProxyMetaData.ABI instead.
var ExchangeProxyABI = ExchangeProxyMetaData.ABI

// ExchangeProxyBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use ExchangeProxyMetaData.Bin instead.
var ExchangeProxyBin = ExchangeProxyMetaData.Bin

// DeployExchangeProxy deploys a new Ethereum contract, binding an instance of ExchangeProxy to it.
func DeployExchangeProxy(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *ExchangeProxy, error) {
	parsed, err := ExchangeProxyMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(ExchangeProxyBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &ExchangeProxy{ExchangeProxyCaller: ExchangeProxyCaller{contract: contract}, ExchangeProxyTransactor: ExchangeProxyTransactor{contract: contract}, ExchangeProxyFilterer: ExchangeProxyFilterer{contract: contract}}, nil
}

// ExchangeProxy is an auto generated Go binding around an Ethereum contract.
type ExchangeProxy struct {
	ExchangeProxyCaller     // Read-only binding to the contract
	ExchangeProxyTransactor // Write-only binding to the contract
	ExchangeProxyFilterer   // Log filterer for contract events
}

// ExchangeProxyCaller is an auto generated read-only Go binding around an Ethereum contract.
type ExchangeProxyCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ExchangeProxyTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ExchangeProxyTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ExchangeProxyFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ExchangeProxyFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ExchangeProxySession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ExchangeProxySession struct {
	Contract     *ExchangeProxy    // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ExchangeProxyCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ExchangeProxyCallerSession struct {
	Contract *ExchangeProxyCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts        // Call options to use throughout this session
}

// ExchangeProxyTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ExchangeProxyTransactorSession struct {
	Contract     *ExchangeProxyTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// ExchangeProxyRaw is an auto generated low-level Go binding around an Ethereum contract.
type ExchangeProxyRaw struct {
	Contract *ExchangeProxy // Generic contract binding to access the raw methods on
}

// ExchangeProxyCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ExchangeProxyCallerRaw struct {
	Contract *ExchangeProxyCaller // Generic read-only contract binding to access the raw methods on
}

// ExchangeProxyTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ExchangeProxyTransactorRaw struct {
	Contract *ExchangeProxyTransactor // Generic write-only contract binding to access the raw methods on
}

// NewExchangeProxy creates a new instance of ExchangeProxy, bound to a specific deployed contract.
func NewExchangeProxy(address common.Address, backend bind.ContractBackend) (*ExchangeProxy, error) {
	contract, err := bindExchangeProxy(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ExchangeProxy{ExchangeProxyCaller: ExchangeProxyCaller{contract: contract}, ExchangeProxyTransactor: ExchangeProxyTransactor{contract: contract}, ExchangeProxyFilterer: ExchangeProxyFilterer{contract: contract}}, nil
}

// NewExchangeProxyCaller creates a new read-only instance of ExchangeProxy, bound to a specific deployed contract.
func NewExchangeProxyCaller(address common.Address, caller bind.ContractCaller) (*ExchangeProxyCaller, error) {
	contract, err := bindExchangeProxy(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ExchangeProxyCaller{contract: contract}, nil
}

// NewExchangeProxyTransactor creates a new write-only instance of ExchangeProxy, bound to a specific deployed contract.
func NewExchangeProxyTransactor(address common.Address, transactor bind.ContractTransactor) (*ExchangeProxyTransactor, error) {
	contract, err := bindExchangeProxy(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ExchangeProxyTransactor{contract: contract}, nil
}

// NewExchangeProxyFilterer creates a new log filterer instance of ExchangeProxy, bound to a specific deployed contract.
func NewExchangeProxyFilterer(address common.Address, filterer bind.ContractFilterer) (*ExchangeProxyFilterer, error) {
	contract, err := bindExchangeProxy(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ExchangeProxyFilterer{contract: contract}, nil
}

// bindExchangeProxy binds a generic wrapper to an already deployed contract.
func bindExchangeProxy(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ExchangeProxyMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ExchangeProxy *ExchangeProxyRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ExchangeProxy.Contract.ExchangeProxyCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ExchangeProxy *ExchangeProxyRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ExchangeProxy.Contract.ExchangeProxyTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ExchangeProxy *ExchangeProxyRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ExchangeProxy.Contract.ExchangeProxyTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ExchangeProxy *ExchangeProxyCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ExchangeProxy.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ExchangeProxy *ExchangeProxyTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ExchangeProxy.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ExchangeProxy *ExchangeProxyTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ExchangeProxy.Contract.contract.Transact(opts, method, params...)
}

// QueryAllowance is a free data retrieval call binding the contract method 0x5ce4a731.
//
// Solidity: function queryAllowance(address grantee, address granter, uint8 msgType) view returns(bool allowed)
func (_ExchangeProxy *ExchangeProxyCaller) QueryAllowance(opts *bind.CallOpts, grantee common.Address, granter common.Address, msgType uint8) (bool, error) {
	var out []interface{}
	err := _ExchangeProxy.contract.Call(opts, &out, "queryAllowance", grantee, granter, msgType)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// QueryAllowance is a free data retrieval call binding the contract method 0x5ce4a731.
//
// Solidity: function queryAllowance(address grantee, address granter, uint8 msgType) view returns(bool allowed)
func (_ExchangeProxy *ExchangeProxySession) QueryAllowance(grantee common.Address, granter common.Address, msgType uint8) (bool, error) {
	return _ExchangeProxy.Contract.QueryAllowance(&_ExchangeProxy.CallOpts, grantee, granter, msgType)
}

// QueryAllowance is a free data retrieval call binding the contract method 0x5ce4a731.
//
// Solidity: function queryAllowance(address grantee, address granter, uint8 msgType) view returns(bool allowed)
func (_ExchangeProxy *ExchangeProxyCallerSession) QueryAllowance(grantee common.Address, granter common.Address, msgType uint8) (bool, error) {
	return _ExchangeProxy.Contract.QueryAllowance(&_ExchangeProxy.CallOpts, grantee, granter, msgType)
}

// Approve is a paid mutator transaction binding the contract method 0x2df0f577.
//
// Solidity: function approve(uint8 msgType, (uint256,string)[] spendLimit, uint256 duration) returns(bool success)
func (_ExchangeProxy *ExchangeProxyTransactor) Approve(opts *bind.TransactOpts, msgType uint8, spendLimit []CosmosCoin, duration *big.Int) (*types.Transaction, error) {
	return _ExchangeProxy.contract.Transact(opts, "approve", msgType, spendLimit, duration)
}

// Approve is a paid mutator transaction binding the contract method 0x2df0f577.
//
// Solidity: function approve(uint8 msgType, (uint256,string)[] spendLimit, uint256 duration) returns(bool success)
func (_ExchangeProxy *ExchangeProxySession) Approve(msgType uint8, spendLimit []CosmosCoin, duration *big.Int) (*types.Transaction, error) {
	return _ExchangeProxy.Contract.Approve(&_ExchangeProxy.TransactOpts, msgType, spendLimit, duration)
}

// Approve is a paid mutator transaction binding the contract method 0x2df0f577.
//
// Solidity: function approve(uint8 msgType, (uint256,string)[] spendLimit, uint256 duration) returns(bool success)
func (_ExchangeProxy *ExchangeProxyTransactorSession) Approve(msgType uint8, spendLimit []CosmosCoin, duration *big.Int) (*types.Transaction, error) {
	return _ExchangeProxy.Contract.Approve(&_ExchangeProxy.TransactOpts, msgType, spendLimit, duration)
}

// CreateDerivativeLimitOrder is a paid mutator transaction binding the contract method 0x20c69837.
//
// Solidity: function createDerivativeLimitOrder(address sender, (string,string,string,uint256,uint256,string,string,uint256,uint256) order) returns((string,string) response)
func (_ExchangeProxy *ExchangeProxyTransactor) CreateDerivativeLimitOrder(opts *bind.TransactOpts, sender common.Address, order IExchangeModuleDerivativeOrder) (*types.Transaction, error) {
	return _ExchangeProxy.contract.Transact(opts, "createDerivativeLimitOrder", sender, order)
}

// CreateDerivativeLimitOrder is a paid mutator transaction binding the contract method 0x20c69837.
//
// Solidity: function createDerivativeLimitOrder(address sender, (string,string,string,uint256,uint256,string,string,uint256,uint256) order) returns((string,string) response)
func (_ExchangeProxy *ExchangeProxySession) CreateDerivativeLimitOrder(sender common.Address, order IExchangeModuleDerivativeOrder) (*types.Transaction, error) {
	return _ExchangeProxy.Contract.CreateDerivativeLimitOrder(&_ExchangeProxy.TransactOpts, sender, order)
}

// CreateDerivativeLimitOrder is a paid mutator transaction binding the contract method 0x20c69837.
//
// Solidity: function createDerivativeLimitOrder(address sender, (string,string,string,uint256,uint256,string,string,uint256,uint256) order) returns((string,string) response)
func (_ExchangeProxy *ExchangeProxyTransactorSession) CreateDerivativeLimitOrder(sender common.Address, order IExchangeModuleDerivativeOrder) (*types.Transaction, error) {
	return _ExchangeProxy.Contract.CreateDerivativeLimitOrder(&_ExchangeProxy.TransactOpts, sender, order)
}

// Revoke is a paid mutator transaction binding the contract method 0xd9a06a90.
//
// Solidity: function revoke(uint8 msgType) returns(bool success)
func (_ExchangeProxy *ExchangeProxyTransactor) Revoke(opts *bind.TransactOpts, msgType uint8) (*types.Transaction, error) {
	return _ExchangeProxy.contract.Transact(opts, "revoke", msgType)
}

// Revoke is a paid mutator transaction binding the contract method 0xd9a06a90.
//
// Solidity: function revoke(uint8 msgType) returns(bool success)
func (_ExchangeProxy *ExchangeProxySession) Revoke(msgType uint8) (*types.Transaction, error) {
	return _ExchangeProxy.Contract.Revoke(&_ExchangeProxy.TransactOpts, msgType)
}

// Revoke is a paid mutator transaction binding the contract method 0xd9a06a90.
//
// Solidity: function revoke(uint8 msgType) returns(bool success)
func (_ExchangeProxy *ExchangeProxyTransactorSession) Revoke(msgType uint8) (*types.Transaction, error) {
	return _ExchangeProxy.Contract.Revoke(&_ExchangeProxy.TransactOpts, msgType)
}
