// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package contracts

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

// CounterMetaData contains all meta data concerning the Counter contract.
var CounterMetaData = &bind.MetaData{
	ABI: "[{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"reason\",\"type\":\"string\"}],\"name\":\"UserRevert\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"newValue\",\"type\":\"uint256\"}],\"name\":\"ValueSet\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"increment\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"number\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"newNumber\",\"type\":\"uint256\"}],\"name\":\"setNumber\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"reason\",\"type\":\"string\"}],\"name\":\"userRevert\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f5ffd5b506105cd8061001c5f395ff3fe608060405234801561000f575f5ffd5b506004361061004a575f3560e01c8063165717b71461004e5780633fb5c1cb1461006a5780638381f58a14610086578063d09de08a146100a4575b5f5ffd5b61006860048036038101906100639190610348565b6100ae565b005b610084600480360381019061007f91906103c2565b610137565b005b61008e61018f565b60405161009b91906103fc565b60405180910390f35b6100ac610194565b005b3373ffffffffffffffffffffffffffffffffffffffff167f45e85bd60b827366cbad1c6c014ae1de96ff92338f7e2141da3e7efd7048bc8d826040516100f49190610475565b60405180910390a26040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161012e90610505565b60405180910390fd5b805f819055503373ffffffffffffffffffffffffffffffffffffffff167ff3f57717dff9f5f10af315efdbfadc60c42152c11fc0c3c413bbfbdc661f143c5f5460405161018491906103fc565b60405180910390a250565b5f5481565b5f5f8154809291906101a590610550565b91905055503373ffffffffffffffffffffffffffffffffffffffff167ff3f57717dff9f5f10af315efdbfadc60c42152c11fc0c3c413bbfbdc661f143c5f546040516101f191906103fc565b60405180910390a2565b5f604051905090565b5f5ffd5b5f5ffd5b5f5ffd5b5f5ffd5b5f601f19601f8301169050919050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52604160045260245ffd5b61025a82610214565b810181811067ffffffffffffffff8211171561027957610278610224565b5b80604052505050565b5f61028b6101fb565b90506102978282610251565b919050565b5f67ffffffffffffffff8211156102b6576102b5610224565b5b6102bf82610214565b9050602081019050919050565b828183375f83830152505050565b5f6102ec6102e78461029c565b610282565b90508281526020810184848401111561030857610307610210565b5b6103138482856102cc565b509392505050565b5f82601f83011261032f5761032e61020c565b5b813561033f8482602086016102da565b91505092915050565b5f6020828403121561035d5761035c610204565b5b5f82013567ffffffffffffffff81111561037a57610379610208565b5b6103868482850161031b565b91505092915050565b5f819050919050565b6103a18161038f565b81146103ab575f5ffd5b50565b5f813590506103bc81610398565b92915050565b5f602082840312156103d7576103d6610204565b5b5f6103e4848285016103ae565b91505092915050565b6103f68161038f565b82525050565b5f60208201905061040f5f8301846103ed565b92915050565b5f81519050919050565b5f82825260208201905092915050565b8281835e5f83830152505050565b5f61044782610415565b610451818561041f565b935061046181856020860161042f565b61046a81610214565b840191505092915050565b5f6020820190508181035f83015261048d818461043d565b905092915050565b7f75736572207265717565737465642061207265766572743b20736565206576655f8201527f6e7420666f722064657461696c73000000000000000000000000000000000000602082015250565b5f6104ef602e8361041f565b91506104fa82610495565b604082019050919050565b5f6020820190508181035f83015261051c816104e3565b9050919050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f61055a8261038f565b91507fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff820361058c5761058b610523565b5b60018201905091905056fea2646970667358221220314a611b2d1879d7e25b387ef1eae3ad545d7529c3117bcab2feac4b295ecd0164736f6c634300081b0033",
}

// CounterABI is the input ABI used to generate the binding from.
// Deprecated: Use CounterMetaData.ABI instead.
var CounterABI = CounterMetaData.ABI

// CounterBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use CounterMetaData.Bin instead.
var CounterBin = CounterMetaData.Bin

// DeployCounter deploys a new Ethereum contract, binding an instance of Counter to it.
func DeployCounter(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Counter, error) {
	parsed, err := CounterMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(CounterBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Counter{CounterCaller: CounterCaller{contract: contract}, CounterTransactor: CounterTransactor{contract: contract}, CounterFilterer: CounterFilterer{contract: contract}}, nil
}

// Counter is an auto generated Go binding around an Ethereum contract.
type Counter struct {
	CounterCaller     // Read-only binding to the contract
	CounterTransactor // Write-only binding to the contract
	CounterFilterer   // Log filterer for contract events
}

// CounterCaller is an auto generated read-only Go binding around an Ethereum contract.
type CounterCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CounterTransactor is an auto generated write-only Go binding around an Ethereum contract.
type CounterTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CounterFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type CounterFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// CounterSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type CounterSession struct {
	Contract     *Counter          // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// CounterCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type CounterCallerSession struct {
	Contract *CounterCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts  // Call options to use throughout this session
}

// CounterTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type CounterTransactorSession struct {
	Contract     *CounterTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// CounterRaw is an auto generated low-level Go binding around an Ethereum contract.
type CounterRaw struct {
	Contract *Counter // Generic contract binding to access the raw methods on
}

// CounterCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type CounterCallerRaw struct {
	Contract *CounterCaller // Generic read-only contract binding to access the raw methods on
}

// CounterTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type CounterTransactorRaw struct {
	Contract *CounterTransactor // Generic write-only contract binding to access the raw methods on
}

// NewCounter creates a new instance of Counter, bound to a specific deployed contract.
func NewCounter(address common.Address, backend bind.ContractBackend) (*Counter, error) {
	contract, err := bindCounter(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Counter{CounterCaller: CounterCaller{contract: contract}, CounterTransactor: CounterTransactor{contract: contract}, CounterFilterer: CounterFilterer{contract: contract}}, nil
}

// NewCounterCaller creates a new read-only instance of Counter, bound to a specific deployed contract.
func NewCounterCaller(address common.Address, caller bind.ContractCaller) (*CounterCaller, error) {
	contract, err := bindCounter(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &CounterCaller{contract: contract}, nil
}

// NewCounterTransactor creates a new write-only instance of Counter, bound to a specific deployed contract.
func NewCounterTransactor(address common.Address, transactor bind.ContractTransactor) (*CounterTransactor, error) {
	contract, err := bindCounter(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &CounterTransactor{contract: contract}, nil
}

// NewCounterFilterer creates a new log filterer instance of Counter, bound to a specific deployed contract.
func NewCounterFilterer(address common.Address, filterer bind.ContractFilterer) (*CounterFilterer, error) {
	contract, err := bindCounter(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &CounterFilterer{contract: contract}, nil
}

// bindCounter binds a generic wrapper to an already deployed contract.
func bindCounter(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := CounterMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Counter *CounterRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Counter.Contract.CounterCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Counter *CounterRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Counter.Contract.CounterTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Counter *CounterRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Counter.Contract.CounterTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Counter *CounterCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Counter.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Counter *CounterTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Counter.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Counter *CounterTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Counter.Contract.contract.Transact(opts, method, params...)
}

// Number is a free data retrieval call binding the contract method 0x8381f58a.
//
// Solidity: function number() view returns(uint256)
func (_Counter *CounterCaller) Number(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Counter.contract.Call(opts, &out, "number")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Number is a free data retrieval call binding the contract method 0x8381f58a.
//
// Solidity: function number() view returns(uint256)
func (_Counter *CounterSession) Number() (*big.Int, error) {
	return _Counter.Contract.Number(&_Counter.CallOpts)
}

// Number is a free data retrieval call binding the contract method 0x8381f58a.
//
// Solidity: function number() view returns(uint256)
func (_Counter *CounterCallerSession) Number() (*big.Int, error) {
	return _Counter.Contract.Number(&_Counter.CallOpts)
}

// Increment is a paid mutator transaction binding the contract method 0xd09de08a.
//
// Solidity: function increment() returns()
func (_Counter *CounterTransactor) Increment(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Counter.contract.Transact(opts, "increment")
}

// Increment is a paid mutator transaction binding the contract method 0xd09de08a.
//
// Solidity: function increment() returns()
func (_Counter *CounterSession) Increment() (*types.Transaction, error) {
	return _Counter.Contract.Increment(&_Counter.TransactOpts)
}

// Increment is a paid mutator transaction binding the contract method 0xd09de08a.
//
// Solidity: function increment() returns()
func (_Counter *CounterTransactorSession) Increment() (*types.Transaction, error) {
	return _Counter.Contract.Increment(&_Counter.TransactOpts)
}

// SetNumber is a paid mutator transaction binding the contract method 0x3fb5c1cb.
//
// Solidity: function setNumber(uint256 newNumber) returns()
func (_Counter *CounterTransactor) SetNumber(opts *bind.TransactOpts, newNumber *big.Int) (*types.Transaction, error) {
	return _Counter.contract.Transact(opts, "setNumber", newNumber)
}

// SetNumber is a paid mutator transaction binding the contract method 0x3fb5c1cb.
//
// Solidity: function setNumber(uint256 newNumber) returns()
func (_Counter *CounterSession) SetNumber(newNumber *big.Int) (*types.Transaction, error) {
	return _Counter.Contract.SetNumber(&_Counter.TransactOpts, newNumber)
}

// SetNumber is a paid mutator transaction binding the contract method 0x3fb5c1cb.
//
// Solidity: function setNumber(uint256 newNumber) returns()
func (_Counter *CounterTransactorSession) SetNumber(newNumber *big.Int) (*types.Transaction, error) {
	return _Counter.Contract.SetNumber(&_Counter.TransactOpts, newNumber)
}

// UserRevert is a paid mutator transaction binding the contract method 0x165717b7.
//
// Solidity: function userRevert(string reason) returns()
func (_Counter *CounterTransactor) UserRevert(opts *bind.TransactOpts, reason string) (*types.Transaction, error) {
	return _Counter.contract.Transact(opts, "userRevert", reason)
}

// UserRevert is a paid mutator transaction binding the contract method 0x165717b7.
//
// Solidity: function userRevert(string reason) returns()
func (_Counter *CounterSession) UserRevert(reason string) (*types.Transaction, error) {
	return _Counter.Contract.UserRevert(&_Counter.TransactOpts, reason)
}

// UserRevert is a paid mutator transaction binding the contract method 0x165717b7.
//
// Solidity: function userRevert(string reason) returns()
func (_Counter *CounterTransactorSession) UserRevert(reason string) (*types.Transaction, error) {
	return _Counter.Contract.UserRevert(&_Counter.TransactOpts, reason)
}

// CounterUserRevertIterator is returned from FilterUserRevert and is used to iterate over the raw logs and unpacked data for UserRevert events raised by the Counter contract.
type CounterUserRevertIterator struct {
	Event *CounterUserRevert // Event containing the contract specifics and raw log

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
func (it *CounterUserRevertIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CounterUserRevert)
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
		it.Event = new(CounterUserRevert)
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
func (it *CounterUserRevertIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CounterUserRevertIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CounterUserRevert represents a UserRevert event raised by the Counter contract.
type CounterUserRevert struct {
	Sender common.Address
	Reason string
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterUserRevert is a free log retrieval operation binding the contract event 0x45e85bd60b827366cbad1c6c014ae1de96ff92338f7e2141da3e7efd7048bc8d.
//
// Solidity: event UserRevert(address indexed sender, string reason)
func (_Counter *CounterFilterer) FilterUserRevert(opts *bind.FilterOpts, sender []common.Address) (*CounterUserRevertIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _Counter.contract.FilterLogs(opts, "UserRevert", senderRule)
	if err != nil {
		return nil, err
	}
	return &CounterUserRevertIterator{contract: _Counter.contract, event: "UserRevert", logs: logs, sub: sub}, nil
}

// WatchUserRevert is a free log subscription operation binding the contract event 0x45e85bd60b827366cbad1c6c014ae1de96ff92338f7e2141da3e7efd7048bc8d.
//
// Solidity: event UserRevert(address indexed sender, string reason)
func (_Counter *CounterFilterer) WatchUserRevert(opts *bind.WatchOpts, sink chan<- *CounterUserRevert, sender []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _Counter.contract.WatchLogs(opts, "UserRevert", senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CounterUserRevert)
				if err := _Counter.contract.UnpackLog(event, "UserRevert", log); err != nil {
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

// ParseUserRevert is a log parse operation binding the contract event 0x45e85bd60b827366cbad1c6c014ae1de96ff92338f7e2141da3e7efd7048bc8d.
//
// Solidity: event UserRevert(address indexed sender, string reason)
func (_Counter *CounterFilterer) ParseUserRevert(log types.Log) (*CounterUserRevert, error) {
	event := new(CounterUserRevert)
	if err := _Counter.contract.UnpackLog(event, "UserRevert", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// CounterValueSetIterator is returned from FilterValueSet and is used to iterate over the raw logs and unpacked data for ValueSet events raised by the Counter contract.
type CounterValueSetIterator struct {
	Event *CounterValueSet // Event containing the contract specifics and raw log

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
func (it *CounterValueSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(CounterValueSet)
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
		it.Event = new(CounterValueSet)
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
func (it *CounterValueSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *CounterValueSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// CounterValueSet represents a ValueSet event raised by the Counter contract.
type CounterValueSet struct {
	Sender   common.Address
	NewValue *big.Int
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterValueSet is a free log retrieval operation binding the contract event 0xf3f57717dff9f5f10af315efdbfadc60c42152c11fc0c3c413bbfbdc661f143c.
//
// Solidity: event ValueSet(address indexed sender, uint256 newValue)
func (_Counter *CounterFilterer) FilterValueSet(opts *bind.FilterOpts, sender []common.Address) (*CounterValueSetIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _Counter.contract.FilterLogs(opts, "ValueSet", senderRule)
	if err != nil {
		return nil, err
	}
	return &CounterValueSetIterator{contract: _Counter.contract, event: "ValueSet", logs: logs, sub: sub}, nil
}

// WatchValueSet is a free log subscription operation binding the contract event 0xf3f57717dff9f5f10af315efdbfadc60c42152c11fc0c3c413bbfbdc661f143c.
//
// Solidity: event ValueSet(address indexed sender, uint256 newValue)
func (_Counter *CounterFilterer) WatchValueSet(opts *bind.WatchOpts, sink chan<- *CounterValueSet, sender []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _Counter.contract.WatchLogs(opts, "ValueSet", senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(CounterValueSet)
				if err := _Counter.contract.UnpackLog(event, "ValueSet", log); err != nil {
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

// ParseValueSet is a log parse operation binding the contract event 0xf3f57717dff9f5f10af315efdbfadc60c42152c11fc0c3c413bbfbdc661f143c.
//
// Solidity: event ValueSet(address indexed sender, uint256 newValue)
func (_Counter *CounterFilterer) ParseValueSet(log types.Log) (*CounterValueSet, error) {
	event := new(CounterValueSet)
	if err := _Counter.contract.UnpackLog(event, "ValueSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
