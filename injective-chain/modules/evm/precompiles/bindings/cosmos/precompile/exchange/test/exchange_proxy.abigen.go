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
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"string\",\"name\":\"marketID\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"subaccountID\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"feeRecipient\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"price\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quantity\",\"type\":\"uint256\"},{\"internalType\":\"string\",\"name\":\"cid\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"orderType\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"margin\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"triggerPrice\",\"type\":\"uint256\"}],\"internalType\":\"structIExchangeModule.DerivativeOrder\",\"name\":\"order\",\"type\":\"tuple\"}],\"name\":\"createDerivativeLimitOrder\",\"outputs\":[{\"components\":[{\"internalType\":\"string\",\"name\":\"orderHash\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"cid\",\"type\":\"string\"}],\"internalType\":\"structIExchangeModule.CreateDerivativeLimitOrderResponse\",\"name\":\"response\",\"type\":\"tuple\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"grantee\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"granter\",\"type\":\"address\"},{\"internalType\":\"ExchangeTypes.MsgType\",\"name\":\"msgType\",\"type\":\"uint8\"}],\"name\":\"queryAllowance\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"allowed\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x608060405260655f806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555034801561004f575f80fd5b50610b678061005d5f395ff3fe608060405234801561000f575f80fd5b5060043610610034575f3560e01c806320c69837146100385780635ce4a73114610068575b5f80fd5b610052600480360381019061004d919061030b565b610098565b60405161005f9190610430565b60405180910390f35b610082600480360381019061007d9190610486565b610181565b60405161008f91906104f0565b60405180910390f35b6100a0610263565b5f8054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166320c6983784846040518363ffffffff1660e01b81526004016100fa929190610748565b5f604051808303815f875af192505050801561013857506040513d5f823e3d601f19601f820116820180604052508101906101359190610921565b60015b610177576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161016e906109e8565b60405180910390fd5b8091505092915050565b5f805f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166327be18a88585856040518463ffffffff1660e01b81526004016101df93929190610a3f565b602060405180830381865afa92505050801561021957506040513d601f19601f820116820180604052508101906102169190610a9e565b60015b610258576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161024f90610b13565b60405180910390fd5b809150509392505050565b604051806040016040528060608152602001606081525090565b5f604051905090565b5f80fd5b5f80fd5b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f6102b78261028e565b9050919050565b6102c7816102ad565b81146102d1575f80fd5b50565b5f813590506102e2816102be565b92915050565b5f80fd5b5f6101208284031215610302576103016102e8565b5b81905092915050565b5f806040838503121561032157610320610286565b5b5f61032e858286016102d4565b925050602083013567ffffffffffffffff81111561034f5761034e61028a565b5b61035b858286016102ec565b9150509250929050565b5f81519050919050565b5f82825260208201905092915050565b5f5b8381101561039c578082015181840152602081019050610381565b5f8484015250505050565b5f601f19601f8301169050919050565b5f6103c182610365565b6103cb818561036f565b93506103db81856020860161037f565b6103e4816103a7565b840191505092915050565b5f604083015f8301518482035f86015261040982826103b7565b9150506020830151848203602086015261042382826103b7565b9150508091505092915050565b5f6020820190508181035f83015261044881846103ef565b905092915050565b5f60ff82169050919050565b61046581610450565b811461046f575f80fd5b50565b5f813590506104808161045c565b92915050565b5f805f6060848603121561049d5761049c610286565b5b5f6104aa868287016102d4565b93505060206104bb868287016102d4565b92505060406104cc86828701610472565b9150509250925092565b5f8115159050919050565b6104ea816104d6565b82525050565b5f6020820190506105035f8301846104e1565b92915050565b610512816102ad565b82525050565b5f80fd5b5f80fd5b5f80fd5b5f80833560016020038436030381126105405761053f610520565b5b83810192508235915060208301925067ffffffffffffffff82111561056857610567610518565b5b60018202360383131561057e5761057d61051c565b5b509250929050565b828183375f83830152505050565b5f61059f838561036f565b93506105ac838584610586565b6105b5836103a7565b840190509392505050565b5f819050919050565b6105d2816105c0565b81146105dc575f80fd5b50565b5f813590506105ed816105c9565b92915050565b5f61060160208401846105df565b905092915050565b610612816105c0565b82525050565b5f610120830161062a5f840184610524565b8583035f87015261063c838284610594565b9250505061064d6020840184610524565b8583036020870152610660838284610594565b925050506106716040840184610524565b8583036040870152610684838284610594565b9250505061069560608401846105f3565b6106a26060860182610609565b506106b060808401846105f3565b6106bd6080860182610609565b506106cb60a0840184610524565b85830360a08701526106de838284610594565b925050506106ef60c0840184610524565b85830360c0870152610702838284610594565b9250505061071360e08401846105f3565b61072060e0860182610609565b5061072f6101008401846105f3565b61073d610100860182610609565b508091505092915050565b5f60408201905061075b5f830185610509565b818103602083015261076d8184610618565b90509392505050565b5f80fd5b7f4e487b71000000000000000000000000000000000000000000000000000000005f52604160045260245ffd5b6107b0826103a7565b810181811067ffffffffffffffff821117156107cf576107ce61077a565b5b80604052505050565b5f6107e161027d565b90506107ed82826107a7565b919050565b5f80fd5b5f80fd5b5f80fd5b5f67ffffffffffffffff8211156108185761081761077a565b5b610821826103a7565b9050602081019050919050565b5f61084061083b846107fe565b6107d8565b90508281526020810184848401111561085c5761085b6107fa565b5b61086784828561037f565b509392505050565b5f82601f830112610883576108826107f6565b5b815161089384826020860161082e565b91505092915050565b5f604082840312156108b1576108b0610776565b5b6108bb60406107d8565b90505f82015167ffffffffffffffff8111156108da576108d96107f2565b5b6108e68482850161086f565b5f83015250602082015167ffffffffffffffff811115610909576109086107f2565b5b6109158482850161086f565b60208301525092915050565b5f6020828403121561093657610935610286565b5b5f82015167ffffffffffffffff8111156109535761095261028a565b5b61095f8482850161089c565b91505092915050565b5f82825260208201905092915050565b7f6572726f72206372656174696e672064657269766174697665206c696d6974205f8201527f6f72646572000000000000000000000000000000000000000000000000000000602082015250565b5f6109d2602583610968565b91506109dd82610978565b604082019050919050565b5f6020820190508181035f8301526109ff816109c6565b9050919050565b5f819050919050565b5f610a29610a24610a1f84610450565b610a06565b610450565b9050919050565b610a3981610a0f565b82525050565b5f606082019050610a525f830186610509565b610a5f6020830185610509565b610a6c6040830184610a30565b949350505050565b610a7d816104d6565b8114610a87575f80fd5b50565b5f81519050610a9881610a74565b92915050565b5f60208284031215610ab357610ab2610286565b5b5f610ac084828501610a8a565b91505092915050565b7f6572726f72207175657279696e6720616c6c6f77616e636500000000000000005f82015250565b5f610afd601883610968565b9150610b0882610ac9565b602082019050919050565b5f6020820190508181035f830152610b2a81610af1565b905091905056fea26469706673582212203c224a442d22a35d282845a872fb59b7513b10794a16c09f61b2aa162339186364736f6c63430008180033",
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
