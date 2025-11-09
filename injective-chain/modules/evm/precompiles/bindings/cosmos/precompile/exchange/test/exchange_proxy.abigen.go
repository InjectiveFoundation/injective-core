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
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"string\",\"name\":\"marketID\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"subaccountID\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"feeRecipient\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"price\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quantity\",\"type\":\"uint256\"},{\"internalType\":\"string\",\"name\":\"cid\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"orderType\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"margin\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"triggerPrice\",\"type\":\"uint256\"}],\"internalType\":\"structIExchangeModule.DerivativeOrder[]\",\"name\":\"orders\",\"type\":\"tuple[]\"}],\"name\":\"batchCreateDerivativeLimitOrders\",\"outputs\":[{\"components\":[{\"internalType\":\"string[]\",\"name\":\"orderHashes\",\"type\":\"string[]\"},{\"internalType\":\"string[]\",\"name\":\"createdOrdersCids\",\"type\":\"string[]\"},{\"internalType\":\"string[]\",\"name\":\"failedOrdersCids\",\"type\":\"string[]\"}],\"internalType\":\"structIExchangeModule.BatchCreateDerivativeLimitOrdersResponse\",\"name\":\"response\",\"type\":\"tuple\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"string\",\"name\":\"marketID\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"subaccountID\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"feeRecipient\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"price\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quantity\",\"type\":\"uint256\"},{\"internalType\":\"string\",\"name\":\"cid\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"orderType\",\"type\":\"string\"},{\"internalType\":\"uint256\",\"name\":\"margin\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"triggerPrice\",\"type\":\"uint256\"}],\"internalType\":\"structIExchangeModule.DerivativeOrder\",\"name\":\"order\",\"type\":\"tuple\"}],\"name\":\"createDerivativeLimitOrder\",\"outputs\":[{\"components\":[{\"internalType\":\"string\",\"name\":\"orderHash\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"cid\",\"type\":\"string\"}],\"internalType\":\"structIExchangeModule.CreateDerivativeLimitOrderResponse\",\"name\":\"response\",\"type\":\"tuple\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"grantee\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"granter\",\"type\":\"address\"},{\"internalType\":\"ExchangeTypes.MsgType\",\"name\":\"msgType\",\"type\":\"uint8\"}],\"name\":\"queryAllowance\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"allowed\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x608060405260655f806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555034801561004f575f80fd5b506113398061005d5f395ff3fe608060405234801561000f575f80fd5b506004361061003f575f3560e01c806320c69837146100435780635ce4a7311461007357806379374eab146100a3575b5f80fd5b61005d60048036038101906100589190610453565b6100d3565b60405161006a9190610578565b60405180910390f35b61008d600480360381019061008891906105ce565b6101bc565b60405161009a9190610638565b60405180910390f35b6100bd60048036038101906100b891906106b2565b61029e565b6040516100ca9190610825565b60405180910390f35b6100db61038a565b5f8054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166320c6983784846040518363ffffffff1660e01b8152600401610135929190610a84565b5f604051808303815f875af192505050801561017357506040513d5f823e3d601f19601f820116820180604052508101906101709190610c59565b60015b6101b2576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016101a990610d20565b60405180910390fd5b8091505092915050565b5f805f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166327be18a88585856040518463ffffffff1660e01b815260040161021a93929190610d77565b602060405180830381865afa92505050801561025457506040513d601f19601f820116820180604052508101906102519190610dd6565b60015b610293576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161028a90610e4b565b60405180910390fd5b809150509392505050565b6102a66103a4565b5f8054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166379374eab8585856040518463ffffffff1660e01b81526004016103029392919061106b565b5f604051808303815f875af192505050801561034057506040513d5f823e3d601f19601f8201168201806040525081019061033d919061122e565b60015b61037f576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610376906112e5565b60405180910390fd5b809150509392505050565b604051806040016040528060608152602001606081525090565b60405180606001604052806060815260200160608152602001606081525090565b5f604051905090565b5f80fd5b5f80fd5b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f6103ff826103d6565b9050919050565b61040f816103f5565b8114610419575f80fd5b50565b5f8135905061042a81610406565b92915050565b5f80fd5b5f610120828403121561044a57610449610430565b5b81905092915050565b5f8060408385031215610469576104686103ce565b5b5f6104768582860161041c565b925050602083013567ffffffffffffffff811115610497576104966103d2565b5b6104a385828601610434565b9150509250929050565b5f81519050919050565b5f82825260208201905092915050565b5f5b838110156104e45780820151818401526020810190506104c9565b5f8484015250505050565b5f601f19601f8301169050919050565b5f610509826104ad565b61051381856104b7565b93506105238185602086016104c7565b61052c816104ef565b840191505092915050565b5f604083015f8301518482035f86015261055182826104ff565b9150506020830151848203602086015261056b82826104ff565b9150508091505092915050565b5f6020820190508181035f8301526105908184610537565b905092915050565b5f60ff82169050919050565b6105ad81610598565b81146105b7575f80fd5b50565b5f813590506105c8816105a4565b92915050565b5f805f606084860312156105e5576105e46103ce565b5b5f6105f28682870161041c565b93505060206106038682870161041c565b9250506040610614868287016105ba565b9150509250925092565b5f8115159050919050565b6106328161061e565b82525050565b5f60208201905061064b5f830184610629565b92915050565b5f80fd5b5f80fd5b5f80fd5b5f8083601f84011261067257610671610651565b5b8235905067ffffffffffffffff81111561068f5761068e610655565b5b6020830191508360208202830111156106ab576106aa610659565b5b9250929050565b5f805f604084860312156106c9576106c86103ce565b5b5f6106d68682870161041c565b935050602084013567ffffffffffffffff8111156106f7576106f66103d2565b5b6107038682870161065d565b92509250509250925092565b5f81519050919050565b5f82825260208201905092915050565b5f819050602082019050919050565b5f61074383836104ff565b905092915050565b5f602082019050919050565b5f6107618261070f565b61076b8185610719565b93508360208202850161077d85610729565b805f5b858110156107b857848403895281516107998582610738565b94506107a48361074b565b925060208a01995050600181019050610780565b50829750879550505050505092915050565b5f606083015f8301518482035f8601526107e48282610757565b915050602083015184820360208601526107fe8282610757565b915050604083015184820360408601526108188282610757565b9150508091505092915050565b5f6020820190508181035f83015261083d81846107ca565b905092915050565b61084e816103f5565b82525050565b5f80fd5b5f80fd5b5f80fd5b5f808335600160200384360303811261087c5761087b61085c565b5b83810192508235915060208301925067ffffffffffffffff8211156108a4576108a3610854565b5b6001820236038313156108ba576108b9610858565b5b509250929050565b828183375f83830152505050565b5f6108db83856104b7565b93506108e88385846108c2565b6108f1836104ef565b840190509392505050565b5f819050919050565b61090e816108fc565b8114610918575f80fd5b50565b5f8135905061092981610905565b92915050565b5f61093d602084018461091b565b905092915050565b61094e816108fc565b82525050565b5f61012083016109665f840184610860565b8583035f8701526109788382846108d0565b925050506109896020840184610860565b858303602087015261099c8382846108d0565b925050506109ad6040840184610860565b85830360408701526109c08382846108d0565b925050506109d1606084018461092f565b6109de6060860182610945565b506109ec608084018461092f565b6109f96080860182610945565b50610a0760a0840184610860565b85830360a0870152610a1a8382846108d0565b92505050610a2b60c0840184610860565b85830360c0870152610a3e8382846108d0565b92505050610a4f60e084018461092f565b610a5c60e0860182610945565b50610a6b61010084018461092f565b610a79610100860182610945565b508091505092915050565b5f604082019050610a975f830185610845565b8181036020830152610aa98184610954565b90509392505050565b5f80fd5b7f4e487b71000000000000000000000000000000000000000000000000000000005f52604160045260245ffd5b610aec826104ef565b810181811067ffffffffffffffff82111715610b0b57610b0a610ab6565b5b80604052505050565b5f610b1d6103c5565b9050610b298282610ae3565b919050565b5f80fd5b5f80fd5b5f67ffffffffffffffff821115610b5057610b4f610ab6565b5b610b59826104ef565b9050602081019050919050565b5f610b78610b7384610b36565b610b14565b905082815260208101848484011115610b9457610b93610b32565b5b610b9f8482856104c7565b509392505050565b5f82601f830112610bbb57610bba610651565b5b8151610bcb848260208601610b66565b91505092915050565b5f60408284031215610be957610be8610ab2565b5b610bf36040610b14565b90505f82015167ffffffffffffffff811115610c1257610c11610b2e565b5b610c1e84828501610ba7565b5f83015250602082015167ffffffffffffffff811115610c4157610c40610b2e565b5b610c4d84828501610ba7565b60208301525092915050565b5f60208284031215610c6e57610c6d6103ce565b5b5f82015167ffffffffffffffff811115610c8b57610c8a6103d2565b5b610c9784828501610bd4565b91505092915050565b5f82825260208201905092915050565b7f6572726f72206372656174696e672064657269766174697665206c696d6974205f8201527f6f72646572000000000000000000000000000000000000000000000000000000602082015250565b5f610d0a602583610ca0565b9150610d1582610cb0565b604082019050919050565b5f6020820190508181035f830152610d3781610cfe565b9050919050565b5f819050919050565b5f610d61610d5c610d5784610598565b610d3e565b610598565b9050919050565b610d7181610d47565b82525050565b5f606082019050610d8a5f830186610845565b610d976020830185610845565b610da46040830184610d68565b949350505050565b610db58161061e565b8114610dbf575f80fd5b50565b5f81519050610dd081610dac565b92915050565b5f60208284031215610deb57610dea6103ce565b5b5f610df884828501610dc2565b91505092915050565b7f6572726f72207175657279696e6720616c6c6f77616e636500000000000000005f82015250565b5f610e35601883610ca0565b9150610e4082610e01565b602082019050919050565b5f6020820190508181035f830152610e6281610e29565b9050919050565b5f82825260208201905092915050565b5f819050919050565b5f6101208301610e945f840184610860565b8583035f870152610ea68382846108d0565b92505050610eb76020840184610860565b8583036020870152610eca8382846108d0565b92505050610edb6040840184610860565b8583036040870152610eee8382846108d0565b92505050610eff606084018461092f565b610f0c6060860182610945565b50610f1a608084018461092f565b610f276080860182610945565b50610f3560a0840184610860565b85830360a0870152610f488382846108d0565b92505050610f5960c0840184610860565b85830360c0870152610f6c8382846108d0565b92505050610f7d60e084018461092f565b610f8a60e0860182610945565b50610f9961010084018461092f565b610fa7610100860182610945565b508091505092915050565b5f610fbd8383610e82565b905092915050565b5f8235600161012003833603038112610fe157610fe061085c565b5b82810191505092915050565b5f602082019050919050565b5f6110048385610e69565b93508360208402850161101684610e79565b805f5b878110156110595784840389526110308284610fc5565b61103a8582610fb2565b945061104583610fed565b925060208a01995050600181019050611019565b50829750879450505050509392505050565b5f60408201905061107e5f830186610845565b8181036020830152611091818486610ff9565b9050949350505050565b5f67ffffffffffffffff8211156110b5576110b4610ab6565b5b602082029050602081019050919050565b5f6110d86110d38461109b565b610b14565b905080838252602082019050602084028301858111156110fb576110fa610659565b5b835b8181101561114257805167ffffffffffffffff8111156111205761111f610651565b5b80860161112d8982610ba7565b855260208501945050506020810190506110fd565b5050509392505050565b5f82601f8301126111605761115f610651565b5b81516111708482602086016110c6565b91505092915050565b5f6060828403121561118e5761118d610ab2565b5b6111986060610b14565b90505f82015167ffffffffffffffff8111156111b7576111b6610b2e565b5b6111c38482850161114c565b5f83015250602082015167ffffffffffffffff8111156111e6576111e5610b2e565b5b6111f28482850161114c565b602083015250604082015167ffffffffffffffff81111561121657611215610b2e565b5b6112228482850161114c565b60408301525092915050565b5f60208284031215611243576112426103ce565b5b5f82015167ffffffffffffffff8111156112605761125f6103d2565b5b61126c84828501611179565b91505092915050565b7f6572726f72206372656174696e672064657269766174697665206c696d6974205f8201527f6f726465727320696e2062617463680000000000000000000000000000000000602082015250565b5f6112cf602f83610ca0565b91506112da82611275565b604082019050919050565b5f6020820190508181035f8301526112fc816112c3565b905091905056fea26469706673582212208c38a32ee36e1f6d62948823c951cea9a75c21005bb19306f6b3033605da90f964736f6c63430008180033",
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
