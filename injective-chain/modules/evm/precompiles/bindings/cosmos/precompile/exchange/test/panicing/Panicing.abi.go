// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package panicing

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

// PanicingMetaData contains all meta data concerning the Panicing contract.
var PanicingMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_exchangeContract\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"string\",\"name\":\"reason\",\"type\":\"string\"}],\"name\":\"UserRevert\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"exchangeContract\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"panicDurningCall\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"string\",\"name\":\"reason\",\"type\":\"string\"}],\"name\":\"userRevert\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561000f575f5ffd5b506040516107ed3803806107ed833981810160405281019061003191906100d4565b805f5f6101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550506100ff565b5f5ffd5b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f6100a38261007a565b9050919050565b6100b381610099565b81146100bd575f5ffd5b50565b5f815190506100ce816100aa565b92915050565b5f602082840312156100e9576100e8610076565b5b5f6100f6848285016100c0565b91505092915050565b6106e18061010c5f395ff3fe608060405234801561000f575f5ffd5b506004361061003f575f3560e01c8063165717b7146100435780633f0a07971461005f578063adfb0e901461007d575b5f5ffd5b61005d600480360381019061005891906103a6565b610087565b005b610067610110565b604051610074919061042c565b60405180910390f35b610085610134565b005b3373ffffffffffffffffffffffffffffffffffffffff167f45e85bd60b827366cbad1c6c014ae1de96ff92338f7e2141da3e7efd7048bc8d826040516100cd91906104a5565b60405180910390a26040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161010790610535565b60405180910390fd5b5f5f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b5f5f5f5f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1660405161017990610580565b5f604051808303815f865af19150503d805f81146101b2576040519150601f19603f3d011682016040523d82523d5f602084013e6101b7565b606091505b509150915081156101fd576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016101f4906105de565b60405180910390fd5b8060405160200161020e919061068a565b6040516020818303038152906040526040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161025091906104a5565b60405180910390fd5b5f604051905090565b5f5ffd5b5f5ffd5b5f5ffd5b5f5ffd5b5f601f19601f8301169050919050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52604160045260245ffd5b6102b882610272565b810181811067ffffffffffffffff821117156102d7576102d6610282565b5b80604052505050565b5f6102e9610259565b90506102f582826102af565b919050565b5f67ffffffffffffffff82111561031457610313610282565b5b61031d82610272565b9050602081019050919050565b828183375f83830152505050565b5f61034a610345846102fa565b6102e0565b9050828152602081018484840111156103665761036561026e565b5b61037184828561032a565b509392505050565b5f82601f83011261038d5761038c61026a565b5b813561039d848260208601610338565b91505092915050565b5f602082840312156103bb576103ba610262565b5b5f82013567ffffffffffffffff8111156103d8576103d7610266565b5b6103e484828501610379565b91505092915050565b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f610416826103ed565b9050919050565b6104268161040c565b82525050565b5f60208201905061043f5f83018461041d565b92915050565b5f81519050919050565b5f82825260208201905092915050565b8281835e5f83830152505050565b5f61047782610445565b610481818561044f565b935061049181856020860161045f565b61049a81610272565b840191505092915050565b5f6020820190508181035f8301526104bd818461046d565b905092915050565b7f75736572207265717565737465642061207265766572743b20736565206576655f8201527f6e7420666f722064657461696c73000000000000000000000000000000000000602082015250565b5f61051f602e8361044f565b915061052a826104c5565b604082019050919050565b5f6020820190508181035f83015261054c81610513565b9050919050565b5f81905092915050565b50565b5f61056b5f83610553565b91506105768261055d565b5f82019050919050565b5f61058a82610560565b9150819050919050565b7f63616c6c2073686f756c642068617665206661696c65640000000000000000005f82015250565b5f6105c860178361044f565b91506105d382610594565b602082019050919050565b5f6020820190508181035f8301526105f5816105bc565b9050919050565b5f81905092915050565b7f70616e696320647572696e672063616c6c0000000000000000000000000000005f82015250565b5f61063a6011836105fc565b915061064582610606565b601182019050919050565b5f81519050919050565b5f61066482610650565b61066e8185610553565b935061067e81856020860161045f565b80840191505092915050565b5f6106948261062e565b91506106a0828461065a565b91508190509291505056fea2646970667358221220b152e92530bb33d5f05a7ff8cd6fbe9f827eb520de2099c43cdcda274746124e64736f6c634300081d0033",
}

// PanicingABI is the input ABI used to generate the binding from.
// Deprecated: Use PanicingMetaData.ABI instead.
var PanicingABI = PanicingMetaData.ABI

// PanicingBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use PanicingMetaData.Bin instead.
var PanicingBin = PanicingMetaData.Bin

// DeployPanicing deploys a new Ethereum contract, binding an instance of Panicing to it.
func DeployPanicing(auth *bind.TransactOpts, backend bind.ContractBackend, _exchangeContract common.Address) (common.Address, *types.Transaction, *Panicing, error) {
	parsed, err := PanicingMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(PanicingBin), backend, _exchangeContract)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Panicing{PanicingCaller: PanicingCaller{contract: contract}, PanicingTransactor: PanicingTransactor{contract: contract}, PanicingFilterer: PanicingFilterer{contract: contract}}, nil
}

// Panicing is an auto generated Go binding around an Ethereum contract.
type Panicing struct {
	PanicingCaller     // Read-only binding to the contract
	PanicingTransactor // Write-only binding to the contract
	PanicingFilterer   // Log filterer for contract events
}

// PanicingCaller is an auto generated read-only Go binding around an Ethereum contract.
type PanicingCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PanicingTransactor is an auto generated write-only Go binding around an Ethereum contract.
type PanicingTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PanicingFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type PanicingFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PanicingSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type PanicingSession struct {
	Contract     *Panicing         // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// PanicingCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type PanicingCallerSession struct {
	Contract *PanicingCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts   // Call options to use throughout this session
}

// PanicingTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type PanicingTransactorSession struct {
	Contract     *PanicingTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// PanicingRaw is an auto generated low-level Go binding around an Ethereum contract.
type PanicingRaw struct {
	Contract *Panicing // Generic contract binding to access the raw methods on
}

// PanicingCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type PanicingCallerRaw struct {
	Contract *PanicingCaller // Generic read-only contract binding to access the raw methods on
}

// PanicingTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type PanicingTransactorRaw struct {
	Contract *PanicingTransactor // Generic write-only contract binding to access the raw methods on
}

// NewPanicing creates a new instance of Panicing, bound to a specific deployed contract.
func NewPanicing(address common.Address, backend bind.ContractBackend) (*Panicing, error) {
	contract, err := bindPanicing(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Panicing{PanicingCaller: PanicingCaller{contract: contract}, PanicingTransactor: PanicingTransactor{contract: contract}, PanicingFilterer: PanicingFilterer{contract: contract}}, nil
}

// NewPanicingCaller creates a new read-only instance of Panicing, bound to a specific deployed contract.
func NewPanicingCaller(address common.Address, caller bind.ContractCaller) (*PanicingCaller, error) {
	contract, err := bindPanicing(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &PanicingCaller{contract: contract}, nil
}

// NewPanicingTransactor creates a new write-only instance of Panicing, bound to a specific deployed contract.
func NewPanicingTransactor(address common.Address, transactor bind.ContractTransactor) (*PanicingTransactor, error) {
	contract, err := bindPanicing(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &PanicingTransactor{contract: contract}, nil
}

// NewPanicingFilterer creates a new log filterer instance of Panicing, bound to a specific deployed contract.
func NewPanicingFilterer(address common.Address, filterer bind.ContractFilterer) (*PanicingFilterer, error) {
	contract, err := bindPanicing(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &PanicingFilterer{contract: contract}, nil
}

// bindPanicing binds a generic wrapper to an already deployed contract.
func bindPanicing(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := PanicingMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Panicing *PanicingRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Panicing.Contract.PanicingCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Panicing *PanicingRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Panicing.Contract.PanicingTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Panicing *PanicingRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Panicing.Contract.PanicingTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Panicing *PanicingCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Panicing.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Panicing *PanicingTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Panicing.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Panicing *PanicingTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Panicing.Contract.contract.Transact(opts, method, params...)
}

// ExchangeContract is a free data retrieval call binding the contract method 0x3f0a0797.
//
// Solidity: function exchangeContract() view returns(address)
func (_Panicing *PanicingCaller) ExchangeContract(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Panicing.contract.Call(opts, &out, "exchangeContract")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// ExchangeContract is a free data retrieval call binding the contract method 0x3f0a0797.
//
// Solidity: function exchangeContract() view returns(address)
func (_Panicing *PanicingSession) ExchangeContract() (common.Address, error) {
	return _Panicing.Contract.ExchangeContract(&_Panicing.CallOpts)
}

// ExchangeContract is a free data retrieval call binding the contract method 0x3f0a0797.
//
// Solidity: function exchangeContract() view returns(address)
func (_Panicing *PanicingCallerSession) ExchangeContract() (common.Address, error) {
	return _Panicing.Contract.ExchangeContract(&_Panicing.CallOpts)
}

// PanicDurningCall is a paid mutator transaction binding the contract method 0xadfb0e90.
//
// Solidity: function panicDurningCall() returns()
func (_Panicing *PanicingTransactor) PanicDurningCall(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Panicing.contract.Transact(opts, "panicDurningCall")
}

// PanicDurningCall is a paid mutator transaction binding the contract method 0xadfb0e90.
//
// Solidity: function panicDurningCall() returns()
func (_Panicing *PanicingSession) PanicDurningCall() (*types.Transaction, error) {
	return _Panicing.Contract.PanicDurningCall(&_Panicing.TransactOpts)
}

// PanicDurningCall is a paid mutator transaction binding the contract method 0xadfb0e90.
//
// Solidity: function panicDurningCall() returns()
func (_Panicing *PanicingTransactorSession) PanicDurningCall() (*types.Transaction, error) {
	return _Panicing.Contract.PanicDurningCall(&_Panicing.TransactOpts)
}

// UserRevert is a paid mutator transaction binding the contract method 0x165717b7.
//
// Solidity: function userRevert(string reason) returns()
func (_Panicing *PanicingTransactor) UserRevert(opts *bind.TransactOpts, reason string) (*types.Transaction, error) {
	return _Panicing.contract.Transact(opts, "userRevert", reason)
}

// UserRevert is a paid mutator transaction binding the contract method 0x165717b7.
//
// Solidity: function userRevert(string reason) returns()
func (_Panicing *PanicingSession) UserRevert(reason string) (*types.Transaction, error) {
	return _Panicing.Contract.UserRevert(&_Panicing.TransactOpts, reason)
}

// UserRevert is a paid mutator transaction binding the contract method 0x165717b7.
//
// Solidity: function userRevert(string reason) returns()
func (_Panicing *PanicingTransactorSession) UserRevert(reason string) (*types.Transaction, error) {
	return _Panicing.Contract.UserRevert(&_Panicing.TransactOpts, reason)
}

// PanicingUserRevertIterator is returned from FilterUserRevert and is used to iterate over the raw logs and unpacked data for UserRevert events raised by the Panicing contract.
type PanicingUserRevertIterator struct {
	Event *PanicingUserRevert // Event containing the contract specifics and raw log

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
func (it *PanicingUserRevertIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PanicingUserRevert)
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
		it.Event = new(PanicingUserRevert)
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
func (it *PanicingUserRevertIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PanicingUserRevertIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PanicingUserRevert represents a UserRevert event raised by the Panicing contract.
type PanicingUserRevert struct {
	Sender common.Address
	Reason string
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterUserRevert is a free log retrieval operation binding the contract event 0x45e85bd60b827366cbad1c6c014ae1de96ff92338f7e2141da3e7efd7048bc8d.
//
// Solidity: event UserRevert(address indexed sender, string reason)
func (_Panicing *PanicingFilterer) FilterUserRevert(opts *bind.FilterOpts, sender []common.Address) (*PanicingUserRevertIterator, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _Panicing.contract.FilterLogs(opts, "UserRevert", senderRule)
	if err != nil {
		return nil, err
	}
	return &PanicingUserRevertIterator{contract: _Panicing.contract, event: "UserRevert", logs: logs, sub: sub}, nil
}

// WatchUserRevert is a free log subscription operation binding the contract event 0x45e85bd60b827366cbad1c6c014ae1de96ff92338f7e2141da3e7efd7048bc8d.
//
// Solidity: event UserRevert(address indexed sender, string reason)
func (_Panicing *PanicingFilterer) WatchUserRevert(opts *bind.WatchOpts, sink chan<- *PanicingUserRevert, sender []common.Address) (event.Subscription, error) {

	var senderRule []interface{}
	for _, senderItem := range sender {
		senderRule = append(senderRule, senderItem)
	}

	logs, sub, err := _Panicing.contract.WatchLogs(opts, "UserRevert", senderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PanicingUserRevert)
				if err := _Panicing.contract.UnpackLog(event, "UserRevert", log); err != nil {
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
func (_Panicing *PanicingFilterer) ParseUserRevert(log types.Log) (*PanicingUserRevert, error) {
	event := new(PanicingUserRevert)
	if err := _Panicing.contract.UnpackLog(event, "UserRevert", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
