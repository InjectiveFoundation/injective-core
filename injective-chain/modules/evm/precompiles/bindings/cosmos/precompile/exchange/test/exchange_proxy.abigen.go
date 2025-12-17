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

// IExchangeModuleBatchCreateDerivativeLimitOrdersResponse is an auto generated low-level Go binding around an user-defined struct.
type IExchangeModuleBatchCreateDerivativeLimitOrdersResponse struct {
	OrderHashes       []string
	CreatedOrdersCids []string
	FailedOrdersCids  []string
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
	ABI: "[{\"type\":\"function\",\"name\":\"batchCreateDerivativeLimitOrders\",\"inputs\":[{\"name\":\"sender\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"orders\",\"type\":\"tuple[]\",\"internalType\":\"structIExchangeModule.DerivativeOrder[]\",\"components\":[{\"name\":\"marketID\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"subaccountID\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"feeRecipient\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"price\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"quantity\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"cid\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"orderType\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"margin\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"triggerPrice\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]}],\"outputs\":[{\"name\":\"response\",\"type\":\"tuple\",\"internalType\":\"structIExchangeModule.BatchCreateDerivativeLimitOrdersResponse\",\"components\":[{\"name\":\"orderHashes\",\"type\":\"string[]\",\"internalType\":\"string[]\"},{\"name\":\"createdOrdersCids\",\"type\":\"string[]\",\"internalType\":\"string[]\"},{\"name\":\"failedOrdersCids\",\"type\":\"string[]\",\"internalType\":\"string[]\"}]}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"createDerivativeLimitOrder\",\"inputs\":[{\"name\":\"sender\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"order\",\"type\":\"tuple\",\"internalType\":\"structIExchangeModule.DerivativeOrder\",\"components\":[{\"name\":\"marketID\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"subaccountID\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"feeRecipient\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"price\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"quantity\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"cid\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"orderType\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"margin\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"triggerPrice\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]}],\"outputs\":[{\"name\":\"response\",\"type\":\"tuple\",\"internalType\":\"structIExchangeModule.CreateDerivativeLimitOrderResponse\",\"components\":[{\"name\":\"orderHash\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"cid\",\"type\":\"string\",\"internalType\":\"string\"}]}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"queryAllowance\",\"inputs\":[{\"name\":\"grantee\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"granter\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"msgType\",\"type\":\"uint8\",\"internalType\":\"ExchangeTypes.MsgType\"}],\"outputs\":[{\"name\":\"allowed\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"view\"}]",
	Bin: "0x608060405260655f5f6101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550348015604e575f5ffd5b5061133b8061005c5f395ff3fe608060405234801561000f575f5ffd5b506004361061003f575f3560e01c806320c69837146100435780635ce4a7311461007357806379374eab146100a3575b5f5ffd5b61005d60048036038101906100589190610455565b6100d3565b60405161006a919061057a565b60405180910390f35b61008d600480360381019061008891906105d0565b6101bd565b60405161009a919061063a565b60405180910390f35b6100bd60048036038101906100b891906106b4565b61029f565b6040516100ca9190610827565b60405180910390f35b6100db61038c565b5f5f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166320c6983784846040518363ffffffff1660e01b8152600401610136929190610a86565b5f604051808303815f875af192505050801561017457506040513d5f823e3d601f19601f820116820180604052508101906101719190610c5b565b60015b6101b3576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016101aa90610d22565b60405180910390fd5b8091505092915050565b5f5f5f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166327be18a88585856040518463ffffffff1660e01b815260040161021b93929190610d79565b602060405180830381865afa92505050801561025557506040513d601f19601f820116820180604052508101906102529190610dd8565b60015b610294576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161028b90610e4d565b60405180910390fd5b809150509392505050565b6102a76103a6565b5f5f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166379374eab8585856040518463ffffffff1660e01b81526004016103049392919061106d565b5f604051808303815f875af192505050801561034257506040513d5f823e3d601f19601f8201168201806040525081019061033f9190611230565b60015b610381576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610378906112e7565b60405180910390fd5b809150509392505050565b604051806040016040528060608152602001606081525090565b60405180606001604052806060815260200160608152602001606081525090565b5f604051905090565b5f5ffd5b5f5ffd5b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f610401826103d8565b9050919050565b610411816103f7565b811461041b575f5ffd5b50565b5f8135905061042c81610408565b92915050565b5f5ffd5b5f610120828403121561044c5761044b610432565b5b81905092915050565b5f5f6040838503121561046b5761046a6103d0565b5b5f6104788582860161041e565b925050602083013567ffffffffffffffff811115610499576104986103d4565b5b6104a585828601610436565b9150509250929050565b5f81519050919050565b5f82825260208201905092915050565b5f5b838110156104e65780820151818401526020810190506104cb565b5f8484015250505050565b5f601f19601f8301169050919050565b5f61050b826104af565b61051581856104b9565b93506105258185602086016104c9565b61052e816104f1565b840191505092915050565b5f604083015f8301518482035f8601526105538282610501565b9150506020830151848203602086015261056d8282610501565b9150508091505092915050565b5f6020820190508181035f8301526105928184610539565b905092915050565b5f60ff82169050919050565b6105af8161059a565b81146105b9575f5ffd5b50565b5f813590506105ca816105a6565b92915050565b5f5f5f606084860312156105e7576105e66103d0565b5b5f6105f48682870161041e565b93505060206106058682870161041e565b9250506040610616868287016105bc565b9150509250925092565b5f8115159050919050565b61063481610620565b82525050565b5f60208201905061064d5f83018461062b565b92915050565b5f5ffd5b5f5ffd5b5f5ffd5b5f5f83601f84011261067457610673610653565b5b8235905067ffffffffffffffff81111561069157610690610657565b5b6020830191508360208202830111156106ad576106ac61065b565b5b9250929050565b5f5f5f604084860312156106cb576106ca6103d0565b5b5f6106d88682870161041e565b935050602084013567ffffffffffffffff8111156106f9576106f86103d4565b5b6107058682870161065f565b92509250509250925092565b5f81519050919050565b5f82825260208201905092915050565b5f819050602082019050919050565b5f6107458383610501565b905092915050565b5f602082019050919050565b5f61076382610711565b61076d818561071b565b93508360208202850161077f8561072b565b805f5b858110156107ba578484038952815161079b858261073a565b94506107a68361074d565b925060208a01995050600181019050610782565b50829750879550505050505092915050565b5f606083015f8301518482035f8601526107e68282610759565b915050602083015184820360208601526108008282610759565b9150506040830151848203604086015261081a8282610759565b9150508091505092915050565b5f6020820190508181035f83015261083f81846107cc565b905092915050565b610850816103f7565b82525050565b5f5ffd5b5f5ffd5b5f5ffd5b5f5f8335600160200384360303811261087e5761087d61085e565b5b83810192508235915060208301925067ffffffffffffffff8211156108a6576108a5610856565b5b6001820236038313156108bc576108bb61085a565b5b509250929050565b828183375f83830152505050565b5f6108dd83856104b9565b93506108ea8385846108c4565b6108f3836104f1565b840190509392505050565b5f819050919050565b610910816108fe565b811461091a575f5ffd5b50565b5f8135905061092b81610907565b92915050565b5f61093f602084018461091d565b905092915050565b610950816108fe565b82525050565b5f61012083016109685f840184610862565b8583035f87015261097a8382846108d2565b9250505061098b6020840184610862565b858303602087015261099e8382846108d2565b925050506109af6040840184610862565b85830360408701526109c28382846108d2565b925050506109d36060840184610931565b6109e06060860182610947565b506109ee6080840184610931565b6109fb6080860182610947565b50610a0960a0840184610862565b85830360a0870152610a1c8382846108d2565b92505050610a2d60c0840184610862565b85830360c0870152610a408382846108d2565b92505050610a5160e0840184610931565b610a5e60e0860182610947565b50610a6d610100840184610931565b610a7b610100860182610947565b508091505092915050565b5f604082019050610a995f830185610847565b8181036020830152610aab8184610956565b90509392505050565b5f5ffd5b7f4e487b71000000000000000000000000000000000000000000000000000000005f52604160045260245ffd5b610aee826104f1565b810181811067ffffffffffffffff82111715610b0d57610b0c610ab8565b5b80604052505050565b5f610b1f6103c7565b9050610b2b8282610ae5565b919050565b5f5ffd5b5f5ffd5b5f67ffffffffffffffff821115610b5257610b51610ab8565b5b610b5b826104f1565b9050602081019050919050565b5f610b7a610b7584610b38565b610b16565b905082815260208101848484011115610b9657610b95610b34565b5b610ba18482856104c9565b509392505050565b5f82601f830112610bbd57610bbc610653565b5b8151610bcd848260208601610b68565b91505092915050565b5f60408284031215610beb57610bea610ab4565b5b610bf56040610b16565b90505f82015167ffffffffffffffff811115610c1457610c13610b30565b5b610c2084828501610ba9565b5f83015250602082015167ffffffffffffffff811115610c4357610c42610b30565b5b610c4f84828501610ba9565b60208301525092915050565b5f60208284031215610c7057610c6f6103d0565b5b5f82015167ffffffffffffffff811115610c8d57610c8c6103d4565b5b610c9984828501610bd6565b91505092915050565b5f82825260208201905092915050565b7f6572726f72206372656174696e672064657269766174697665206c696d6974205f8201527f6f72646572000000000000000000000000000000000000000000000000000000602082015250565b5f610d0c602583610ca2565b9150610d1782610cb2565b604082019050919050565b5f6020820190508181035f830152610d3981610d00565b9050919050565b5f819050919050565b5f610d63610d5e610d598461059a565b610d40565b61059a565b9050919050565b610d7381610d49565b82525050565b5f606082019050610d8c5f830186610847565b610d996020830185610847565b610da66040830184610d6a565b949350505050565b610db781610620565b8114610dc1575f5ffd5b50565b5f81519050610dd281610dae565b92915050565b5f60208284031215610ded57610dec6103d0565b5b5f610dfa84828501610dc4565b91505092915050565b7f6572726f72207175657279696e6720616c6c6f77616e636500000000000000005f82015250565b5f610e37601883610ca2565b9150610e4282610e03565b602082019050919050565b5f6020820190508181035f830152610e6481610e2b565b9050919050565b5f82825260208201905092915050565b5f819050919050565b5f6101208301610e965f840184610862565b8583035f870152610ea88382846108d2565b92505050610eb96020840184610862565b8583036020870152610ecc8382846108d2565b92505050610edd6040840184610862565b8583036040870152610ef08382846108d2565b92505050610f016060840184610931565b610f0e6060860182610947565b50610f1c6080840184610931565b610f296080860182610947565b50610f3760a0840184610862565b85830360a0870152610f4a8382846108d2565b92505050610f5b60c0840184610862565b85830360c0870152610f6e8382846108d2565b92505050610f7f60e0840184610931565b610f8c60e0860182610947565b50610f9b610100840184610931565b610fa9610100860182610947565b508091505092915050565b5f610fbf8383610e84565b905092915050565b5f8235600161012003833603038112610fe357610fe261085e565b5b82810191505092915050565b5f602082019050919050565b5f6110068385610e6b565b93508360208402850161101884610e7b565b805f5b8781101561105b5784840389526110328284610fc7565b61103c8582610fb4565b945061104783610fef565b925060208a0199505060018101905061101b565b50829750879450505050509392505050565b5f6040820190506110805f830186610847565b8181036020830152611093818486610ffb565b9050949350505050565b5f67ffffffffffffffff8211156110b7576110b6610ab8565b5b602082029050602081019050919050565b5f6110da6110d58461109d565b610b16565b905080838252602082019050602084028301858111156110fd576110fc61065b565b5b835b8181101561114457805167ffffffffffffffff81111561112257611121610653565b5b80860161112f8982610ba9565b855260208501945050506020810190506110ff565b5050509392505050565b5f82601f83011261116257611161610653565b5b81516111728482602086016110c8565b91505092915050565b5f606082840312156111905761118f610ab4565b5b61119a6060610b16565b90505f82015167ffffffffffffffff8111156111b9576111b8610b30565b5b6111c58482850161114e565b5f83015250602082015167ffffffffffffffff8111156111e8576111e7610b30565b5b6111f48482850161114e565b602083015250604082015167ffffffffffffffff81111561121857611217610b30565b5b6112248482850161114e565b60408301525092915050565b5f60208284031215611245576112446103d0565b5b5f82015167ffffffffffffffff811115611262576112616103d4565b5b61126e8482850161117b565b91505092915050565b7f6572726f72206372656174696e672064657269766174697665206c696d6974205f8201527f6f726465727320696e2062617463680000000000000000000000000000000000602082015250565b5f6112d1602f83610ca2565b91506112dc82611277565b604082019050919050565b5f6020820190508181035f8301526112fe816112c5565b905091905056fea2646970667358221220e04db468f7f799b596e29f1f32d98eb487b41134a107d216d512c0ab8b3d2bb064736f6c634300081b0033",
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

// BatchCreateDerivativeLimitOrders is a paid mutator transaction binding the contract method 0x79374eab.
//
// Solidity: function batchCreateDerivativeLimitOrders(address sender, (string,string,string,uint256,uint256,string,string,uint256,uint256)[] orders) returns((string[],string[],string[]) response)
func (_ExchangeProxy *ExchangeProxyTransactor) BatchCreateDerivativeLimitOrders(opts *bind.TransactOpts, sender common.Address, orders []IExchangeModuleDerivativeOrder) (*types.Transaction, error) {
	return _ExchangeProxy.contract.Transact(opts, "batchCreateDerivativeLimitOrders", sender, orders)
}

// BatchCreateDerivativeLimitOrders is a paid mutator transaction binding the contract method 0x79374eab.
//
// Solidity: function batchCreateDerivativeLimitOrders(address sender, (string,string,string,uint256,uint256,string,string,uint256,uint256)[] orders) returns((string[],string[],string[]) response)
func (_ExchangeProxy *ExchangeProxySession) BatchCreateDerivativeLimitOrders(sender common.Address, orders []IExchangeModuleDerivativeOrder) (*types.Transaction, error) {
	return _ExchangeProxy.Contract.BatchCreateDerivativeLimitOrders(&_ExchangeProxy.TransactOpts, sender, orders)
}

// BatchCreateDerivativeLimitOrders is a paid mutator transaction binding the contract method 0x79374eab.
//
// Solidity: function batchCreateDerivativeLimitOrders(address sender, (string,string,string,uint256,uint256,string,string,uint256,uint256)[] orders) returns((string[],string[],string[]) response)
func (_ExchangeProxy *ExchangeProxyTransactorSession) BatchCreateDerivativeLimitOrders(sender common.Address, orders []IExchangeModuleDerivativeOrder) (*types.Transaction, error) {
	return _ExchangeProxy.Contract.BatchCreateDerivativeLimitOrders(&_ExchangeProxy.TransactOpts, sender, orders)
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
