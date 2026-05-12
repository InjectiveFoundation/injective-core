// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package evm

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


// TransfersHookWithSideEffectMetaData contains all meta data concerning the TransfersHookWithSideEffect contract.
var TransfersHookWithSideEffectMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"function\",\"name\":\"counter\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"postTransfer\",\"inputs\":[{\"name\":\"from\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"to\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"amount\",\"type\":\"tuple\",\"internalType\":\"structCosmos.Coin\",\"components\":[{\"name\":\"amount\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"denom\",\"type\":\"string\",\"internalType\":\"string\"}]}],\"outputs\":[],\"stateMutability\":\"nonpayable\"},{\"type\":\"event\",\"name\":\"CountChanged\",\"inputs\":[{\"name\":\"newValue\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"},{\"name\":\"action\",\"type\":\"string\",\"indexed\":false,\"internalType\":\"string\"}],\"anonymous\":false}]",
	Bin: "0x6080604052348015600e575f5ffd5b506103098061001c5f395ff3fe608060405234801561000f575f5ffd5b5060043610610034575f3560e01c806361bc221a146100385780636afb70c714610056575b5f5ffd5b610040610072565b60405161004d91906100e4565b60405180910390f35b610070600480360381019061006b9190610181565b610077565b005b5f5481565b60015f5f828254610088919061021a565b925050819055507f902cdecf278c2651239a65e4506b50f6ee0ed8d23056e0fa9e21c7ad99dfe8c45f546040516100bf91906102a7565b60405180910390a1505050565b5f819050919050565b6100de816100cc565b82525050565b5f6020820190506100f75f8301846100d5565b92915050565b5f5ffd5b5f5ffd5b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f61012e82610105565b9050919050565b61013e81610124565b8114610148575f5ffd5b50565b5f8135905061015981610135565b92915050565b5f5ffd5b5f604082840312156101785761017761015f565b5b81905092915050565b5f5f5f60608486031215610198576101976100fd565b5b5f6101a58682870161014b565b93505060206101b68682870161014b565b925050604084013567ffffffffffffffff8111156101d7576101d6610101565b5b6101e386828701610163565b9150509250925092565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f610224826100cc565b915061022f836100cc565b9250828201905080821115610247576102466101ed565b5b92915050565b5f82825260208201905092915050565b7f696e6372656d656e7400000000000000000000000000000000000000000000005f82015250565b5f61029160098361024d565b915061029c8261025d565b602082019050919050565b5f6040820190506102ba5f8301846100d5565b81810360208301526102cb81610285565b90509291505056fea26469706673582212205662172e409d83cc4f942a98ef6464027df6734481dfc09d11955f9994db91cf64736f6c634300081e0033",
}

// TransfersHookWithSideEffectABI is the input ABI used to generate the binding from.
// Deprecated: Use TransfersHookWithSideEffectMetaData.ABI instead.
var TransfersHookWithSideEffectABI = TransfersHookWithSideEffectMetaData.ABI

// TransfersHookWithSideEffectBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use TransfersHookWithSideEffectMetaData.Bin instead.
var TransfersHookWithSideEffectBin = TransfersHookWithSideEffectMetaData.Bin

// DeployTransfersHookWithSideEffect deploys a new Ethereum contract, binding an instance of TransfersHookWithSideEffect to it.
func DeployTransfersHookWithSideEffect(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *TransfersHookWithSideEffect, error) {
	parsed, err := TransfersHookWithSideEffectMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(TransfersHookWithSideEffectBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &TransfersHookWithSideEffect{TransfersHookWithSideEffectCaller: TransfersHookWithSideEffectCaller{contract: contract}, TransfersHookWithSideEffectTransactor: TransfersHookWithSideEffectTransactor{contract: contract}, TransfersHookWithSideEffectFilterer: TransfersHookWithSideEffectFilterer{contract: contract}}, nil
}

// TransfersHookWithSideEffect is an auto generated Go binding around an Ethereum contract.
type TransfersHookWithSideEffect struct {
	TransfersHookWithSideEffectCaller     // Read-only binding to the contract
	TransfersHookWithSideEffectTransactor // Write-only binding to the contract
	TransfersHookWithSideEffectFilterer   // Log filterer for contract events
}

// TransfersHookWithSideEffectCaller is an auto generated read-only Go binding around an Ethereum contract.
type TransfersHookWithSideEffectCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TransfersHookWithSideEffectTransactor is an auto generated write-only Go binding around an Ethereum contract.
type TransfersHookWithSideEffectTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TransfersHookWithSideEffectFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type TransfersHookWithSideEffectFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// TransfersHookWithSideEffectSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type TransfersHookWithSideEffectSession struct {
	Contract     *TransfersHookWithSideEffect // Generic contract binding to set the session for
	CallOpts     bind.CallOpts                // Call options to use throughout this session
	TransactOpts bind.TransactOpts            // Transaction auth options to use throughout this session
}

// TransfersHookWithSideEffectCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type TransfersHookWithSideEffectCallerSession struct {
	Contract *TransfersHookWithSideEffectCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                      // Call options to use throughout this session
}

// TransfersHookWithSideEffectTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type TransfersHookWithSideEffectTransactorSession struct {
	Contract     *TransfersHookWithSideEffectTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                      // Transaction auth options to use throughout this session
}

// TransfersHookWithSideEffectRaw is an auto generated low-level Go binding around an Ethereum contract.
type TransfersHookWithSideEffectRaw struct {
	Contract *TransfersHookWithSideEffect // Generic contract binding to access the raw methods on
}

// TransfersHookWithSideEffectCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type TransfersHookWithSideEffectCallerRaw struct {
	Contract *TransfersHookWithSideEffectCaller // Generic read-only contract binding to access the raw methods on
}

// TransfersHookWithSideEffectTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type TransfersHookWithSideEffectTransactorRaw struct {
	Contract *TransfersHookWithSideEffectTransactor // Generic write-only contract binding to access the raw methods on
}

// NewTransfersHookWithSideEffect creates a new instance of TransfersHookWithSideEffect, bound to a specific deployed contract.
func NewTransfersHookWithSideEffect(address common.Address, backend bind.ContractBackend) (*TransfersHookWithSideEffect, error) {
	contract, err := bindTransfersHookWithSideEffect(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &TransfersHookWithSideEffect{TransfersHookWithSideEffectCaller: TransfersHookWithSideEffectCaller{contract: contract}, TransfersHookWithSideEffectTransactor: TransfersHookWithSideEffectTransactor{contract: contract}, TransfersHookWithSideEffectFilterer: TransfersHookWithSideEffectFilterer{contract: contract}}, nil
}

// NewTransfersHookWithSideEffectCaller creates a new read-only instance of TransfersHookWithSideEffect, bound to a specific deployed contract.
func NewTransfersHookWithSideEffectCaller(address common.Address, caller bind.ContractCaller) (*TransfersHookWithSideEffectCaller, error) {
	contract, err := bindTransfersHookWithSideEffect(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &TransfersHookWithSideEffectCaller{contract: contract}, nil
}

// NewTransfersHookWithSideEffectTransactor creates a new write-only instance of TransfersHookWithSideEffect, bound to a specific deployed contract.
func NewTransfersHookWithSideEffectTransactor(address common.Address, transactor bind.ContractTransactor) (*TransfersHookWithSideEffectTransactor, error) {
	contract, err := bindTransfersHookWithSideEffect(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &TransfersHookWithSideEffectTransactor{contract: contract}, nil
}

// NewTransfersHookWithSideEffectFilterer creates a new log filterer instance of TransfersHookWithSideEffect, bound to a specific deployed contract.
func NewTransfersHookWithSideEffectFilterer(address common.Address, filterer bind.ContractFilterer) (*TransfersHookWithSideEffectFilterer, error) {
	contract, err := bindTransfersHookWithSideEffect(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &TransfersHookWithSideEffectFilterer{contract: contract}, nil
}

// bindTransfersHookWithSideEffect binds a generic wrapper to an already deployed contract.
func bindTransfersHookWithSideEffect(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := TransfersHookWithSideEffectMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_TransfersHookWithSideEffect *TransfersHookWithSideEffectRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _TransfersHookWithSideEffect.Contract.TransfersHookWithSideEffectCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_TransfersHookWithSideEffect *TransfersHookWithSideEffectRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TransfersHookWithSideEffect.Contract.TransfersHookWithSideEffectTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_TransfersHookWithSideEffect *TransfersHookWithSideEffectRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _TransfersHookWithSideEffect.Contract.TransfersHookWithSideEffectTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_TransfersHookWithSideEffect *TransfersHookWithSideEffectCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _TransfersHookWithSideEffect.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_TransfersHookWithSideEffect *TransfersHookWithSideEffectTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _TransfersHookWithSideEffect.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_TransfersHookWithSideEffect *TransfersHookWithSideEffectTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _TransfersHookWithSideEffect.Contract.contract.Transact(opts, method, params...)
}

// Counter is a free data retrieval call binding the contract method 0x61bc221a.
//
// Solidity: function counter() view returns(uint256)
func (_TransfersHookWithSideEffect *TransfersHookWithSideEffectCaller) Counter(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _TransfersHookWithSideEffect.contract.Call(opts, &out, "counter")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Counter is a free data retrieval call binding the contract method 0x61bc221a.
//
// Solidity: function counter() view returns(uint256)
func (_TransfersHookWithSideEffect *TransfersHookWithSideEffectSession) Counter() (*big.Int, error) {
	return _TransfersHookWithSideEffect.Contract.Counter(&_TransfersHookWithSideEffect.CallOpts)
}

// Counter is a free data retrieval call binding the contract method 0x61bc221a.
//
// Solidity: function counter() view returns(uint256)
func (_TransfersHookWithSideEffect *TransfersHookWithSideEffectCallerSession) Counter() (*big.Int, error) {
	return _TransfersHookWithSideEffect.Contract.Counter(&_TransfersHookWithSideEffect.CallOpts)
}

// PostTransfer is a paid mutator transaction binding the contract method 0x6afb70c7.
//
// Solidity: function postTransfer(address from, address to, (uint256,string) amount) returns()
func (_TransfersHookWithSideEffect *TransfersHookWithSideEffectTransactor) PostTransfer(opts *bind.TransactOpts, from common.Address, to common.Address, amount CosmosCoin) (*types.Transaction, error) {
	return _TransfersHookWithSideEffect.contract.Transact(opts, "postTransfer", from, to, amount)
}

// PostTransfer is a paid mutator transaction binding the contract method 0x6afb70c7.
//
// Solidity: function postTransfer(address from, address to, (uint256,string) amount) returns()
func (_TransfersHookWithSideEffect *TransfersHookWithSideEffectSession) PostTransfer(from common.Address, to common.Address, amount CosmosCoin) (*types.Transaction, error) {
	return _TransfersHookWithSideEffect.Contract.PostTransfer(&_TransfersHookWithSideEffect.TransactOpts, from, to, amount)
}

// PostTransfer is a paid mutator transaction binding the contract method 0x6afb70c7.
//
// Solidity: function postTransfer(address from, address to, (uint256,string) amount) returns()
func (_TransfersHookWithSideEffect *TransfersHookWithSideEffectTransactorSession) PostTransfer(from common.Address, to common.Address, amount CosmosCoin) (*types.Transaction, error) {
	return _TransfersHookWithSideEffect.Contract.PostTransfer(&_TransfersHookWithSideEffect.TransactOpts, from, to, amount)
}

// TransfersHookWithSideEffectCountChangedIterator is returned from FilterCountChanged and is used to iterate over the raw logs and unpacked data for CountChanged events raised by the TransfersHookWithSideEffect contract.
type TransfersHookWithSideEffectCountChangedIterator struct {
	Event *TransfersHookWithSideEffectCountChanged // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *TransfersHookWithSideEffectCountChangedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(TransfersHookWithSideEffectCountChanged)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(TransfersHookWithSideEffectCountChanged)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *TransfersHookWithSideEffectCountChangedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *TransfersHookWithSideEffectCountChangedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// TransfersHookWithSideEffectCountChanged represents a CountChanged event raised by the TransfersHookWithSideEffect contract.
type TransfersHookWithSideEffectCountChanged struct {
	NewValue *big.Int
	Action   string
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterCountChanged is a free log retrieval operation binding the contract event 0x902cdecf278c2651239a65e4506b50f6ee0ed8d23056e0fa9e21c7ad99dfe8c4.
//
// Solidity: event CountChanged(uint256 newValue, string action)
func (_TransfersHookWithSideEffect *TransfersHookWithSideEffectFilterer) FilterCountChanged(opts *bind.FilterOpts) (*TransfersHookWithSideEffectCountChangedIterator, error) {

	logs, sub, err := _TransfersHookWithSideEffect.contract.FilterLogs(opts, "CountChanged")
	if err != nil {
		return nil, err
	}
	return &TransfersHookWithSideEffectCountChangedIterator{contract: _TransfersHookWithSideEffect.contract, event: "CountChanged", logs: logs, sub: sub}, nil
}

// WatchCountChanged is a free log subscription operation binding the contract event 0x902cdecf278c2651239a65e4506b50f6ee0ed8d23056e0fa9e21c7ad99dfe8c4.
//
// Solidity: event CountChanged(uint256 newValue, string action)
func (_TransfersHookWithSideEffect *TransfersHookWithSideEffectFilterer) WatchCountChanged(opts *bind.WatchOpts, sink chan<- *TransfersHookWithSideEffectCountChanged) (event.Subscription, error) {

	logs, sub, err := _TransfersHookWithSideEffect.contract.WatchLogs(opts, "CountChanged")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(TransfersHookWithSideEffectCountChanged)
				if err := _TransfersHookWithSideEffect.contract.UnpackLog(event, "CountChanged", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseCountChanged is a log parse operation binding the contract event 0x902cdecf278c2651239a65e4506b50f6ee0ed8d23056e0fa9e21c7ad99dfe8c4.
//
// Solidity: event CountChanged(uint256 newValue, string action)
func (_TransfersHookWithSideEffect *TransfersHookWithSideEffectFilterer) ParseCountChanged(log types.Log) (*TransfersHookWithSideEffectCountChanged, error) {
	event := new(TransfersHookWithSideEffectCountChanged)
	if err := _TransfersHookWithSideEffect.contract.UnpackLog(event, "CountChanged", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
