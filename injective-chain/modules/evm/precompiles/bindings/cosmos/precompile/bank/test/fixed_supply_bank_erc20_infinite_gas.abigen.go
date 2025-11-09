// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package bank

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

// FixedSupplyBankERC20InfiniteGasMetaData contains all meta data concerning the FixedSupplyBankERC20InfiniteGas contract.
var FixedSupplyBankERC20InfiniteGasMetaData = &bind.MetaData{
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"name_\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"symbol_\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"decimals_\",\"type\":\"uint8\",\"internalType\":\"uint8\"},{\"name\":\"initial_supply_\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"allowance\",\"inputs\":[{\"name\":\"owner\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"spender\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"approve\",\"inputs\":[{\"name\":\"spender\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"value\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"balanceOf\",\"inputs\":[{\"name\":\"account\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"decimals\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint8\",\"internalType\":\"uint8\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"name\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"symbol\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"totalSupply\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"transfer\",\"inputs\":[{\"name\":\"to\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"value\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"transferFrom\",\"inputs\":[{\"name\":\"from\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"to\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"value\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"event\",\"name\":\"Approval\",\"inputs\":[{\"name\":\"owner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"spender\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"value\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Transfer\",\"inputs\":[{\"name\":\"from\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"to\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"value\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"error\",\"name\":\"ERC20InsufficientAllowance\",\"inputs\":[{\"name\":\"spender\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"allowance\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"needed\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"type\":\"error\",\"name\":\"ERC20InsufficientBalance\",\"inputs\":[{\"name\":\"sender\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"balance\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"needed\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"type\":\"error\",\"name\":\"ERC20InvalidApprover\",\"inputs\":[{\"name\":\"approver\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"ERC20InvalidReceiver\",\"inputs\":[{\"name\":\"receiver\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"ERC20InvalidSender\",\"inputs\":[{\"name\":\"sender\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"ERC20InvalidSpender\",\"inputs\":[{\"name\":\"spender\",\"type\":\"address\",\"internalType\":\"address\"}]}]",
	Bin: "0x6080604052606460055f6101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550348015610050575f5ffd5b50604051611e58380380611e5883398181016040528101906100729190610671565b8383838383838360405180602001604052805f81525060405180602001604052805f81525081600390816100a69190610914565b5080600490816100b69190610914565b50505060055f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166337d2c2f48484846040518463ffffffff1660e01b815260040161011793929190610a3a565b6020604051808303815f875af1158015610133573d5f5f3e3d5ffd5b505050506040513d601f19601f820116820180604052508101906101579190610ab2565b505050505f81111561017457610173338261018160201b60201c565b5b5050505050505050610bb9565b5f73ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff16036101f1575f6040517fec442f050000000000000000000000000000000000000000000000000000000081526004016101e89190610b1c565b60405180910390fd5b6102025f838361020660201b60201c565b5050565b5f73ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff16036102db5760055f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166340c10f1983836040518363ffffffff1660e01b8152600401610295929190610b44565b6020604051808303815f875af11580156102b1573d5f5f3e3d5ffd5b505050506040513d601f19601f820116820180604052508101906102d59190610ab2565b50610451565b5f73ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff16036103b05760055f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16639dc29fac84836040518363ffffffff1660e01b815260040161036a929190610b44565b6020604051808303815f875af1158015610386573d5f5f3e3d5ffd5b505050506040513d601f19601f820116820180604052508101906103aa9190610ab2565b50610450565b60055f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1663beabacc88484846040518463ffffffff1660e01b815260040161040e93929190610b6b565b6020604051808303815f875af115801561042a573d5f5f3e3d5ffd5b505050506040513d601f19601f8201168201806040525081019061044e9190610ab2565b505b5b8173ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef836040516104ae9190610ba0565b60405180910390a3505050565b5f604051905090565b5f5ffd5b5f5ffd5b5f5ffd5b5f5ffd5b5f601f19601f8301169050919050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52604160045260245ffd5b61051a826104d4565b810181811067ffffffffffffffff82111715610539576105386104e4565b5b80604052505050565b5f61054b6104bb565b90506105578282610511565b919050565b5f67ffffffffffffffff821115610576576105756104e4565b5b61057f826104d4565b9050602081019050919050565b8281835e5f83830152505050565b5f6105ac6105a78461055c565b610542565b9050828152602081018484840111156105c8576105c76104d0565b5b6105d384828561058c565b509392505050565b5f82601f8301126105ef576105ee6104cc565b5b81516105ff84826020860161059a565b91505092915050565b5f60ff82169050919050565b61061d81610608565b8114610627575f5ffd5b50565b5f8151905061063881610614565b92915050565b5f819050919050565b6106508161063e565b811461065a575f5ffd5b50565b5f8151905061066b81610647565b92915050565b5f5f5f5f60808587031215610689576106886104c4565b5b5f85015167ffffffffffffffff8111156106a6576106a56104c8565b5b6106b2878288016105db565b945050602085015167ffffffffffffffff8111156106d3576106d26104c8565b5b6106df878288016105db565b93505060406106f08782880161062a565b92505060606107018782880161065d565b91505092959194509250565b5f81519050919050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52602260045260245ffd5b5f600282049050600182168061075b57607f821691505b60208210810361076e5761076d610717565b5b50919050565b5f819050815f5260205f209050919050565b5f6020601f8301049050919050565b5f82821b905092915050565b5f600883026107d07fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff82610795565b6107da8683610795565b95508019841693508086168417925050509392505050565b5f819050919050565b5f61081561081061080b8461063e565b6107f2565b61063e565b9050919050565b5f819050919050565b61082e836107fb565b61084261083a8261081c565b8484546107a1565b825550505050565b5f5f905090565b61085961084a565b610864818484610825565b505050565b5b818110156108875761087c5f82610851565b60018101905061086a565b5050565b601f8211156108cc5761089d81610774565b6108a684610786565b810160208510156108b5578190505b6108c96108c185610786565b830182610869565b50505b505050565b5f82821c905092915050565b5f6108ec5f19846008026108d1565b1980831691505092915050565b5f61090483836108dd565b9150826002028217905092915050565b61091d8261070d565b67ffffffffffffffff811115610936576109356104e4565b5b6109408254610744565b61094b82828561088b565b5f60209050601f83116001811461097c575f841561096a578287015190505b61097485826108f9565b8655506109db565b601f19841661098a86610774565b5f5b828110156109b15784890151825560018201915060208501945060208101905061098c565b868310156109ce57848901516109ca601f8916826108dd565b8355505b6001600288020188555050505b505050505050565b5f82825260208201905092915050565b5f6109fd8261070d565b610a0781856109e3565b9350610a1781856020860161058c565b610a20816104d4565b840191505092915050565b610a3481610608565b82525050565b5f6060820190508181035f830152610a5281866109f3565b90508181036020830152610a6681856109f3565b9050610a756040830184610a2b565b949350505050565b5f8115159050919050565b610a9181610a7d565b8114610a9b575f5ffd5b50565b5f81519050610aac81610a88565b92915050565b5f60208284031215610ac757610ac66104c4565b5b5f610ad484828501610a9e565b91505092915050565b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f610b0682610add565b9050919050565b610b1681610afc565b82525050565b5f602082019050610b2f5f830184610b0d565b92915050565b610b3e8161063e565b82525050565b5f604082019050610b575f830185610b0d565b610b646020830184610b35565b9392505050565b5f606082019050610b7e5f830186610b0d565b610b8b6020830185610b0d565b610b986040830184610b35565b949350505050565b5f602082019050610bb35f830184610b35565b92915050565b61129280610bc65f395ff3fe608060405234801561000f575f5ffd5b5060043610610091575f3560e01c8063313ce56711610064578063313ce5671461013157806370a082311461014f57806395d89b411461017f578063a9059cbb1461019d578063dd62ed3e146101cd57610091565b806306fdde0314610095578063095ea7b3146100b357806318160ddd146100e357806323b872dd14610101575b5f5ffd5b61009d6101fd565b6040516100aa9190610cd4565b60405180910390f35b6100cd60048036038101906100c89190610d92565b6102aa565b6040516100da9190610dea565b60405180910390f35b6100eb6102cc565b6040516100f89190610e12565b60405180910390f35b61011b60048036038101906101169190610e2b565b61036b565b6040516101289190610dea565b60405180910390f35b610139610399565b6040516101469190610e96565b60405180910390f35b61016960048036038101906101649190610eaf565b610447565b6040516101769190610e12565b60405180910390f35b6101876104ea565b6040516101949190610cd4565b60405180910390f35b6101b760048036038101906101b29190610d92565b6105a0565b6040516101c49190610dea565b60405180910390f35b6101e760048036038101906101e29190610eda565b6105c2565b6040516101f49190610e12565b60405180910390f35b60608060055f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16632ba21572306040518263ffffffff1660e01b815260040161025a9190610f27565b5f60405180830381865afa158015610274573d5f5f3e3d5ffd5b505050506040513d5f823e3d601f19601f8201168201806040525081019061029c9190611088565b905050809150508091505090565b5f5f6102b4610644565b90506102c181858561064b565b600191505092915050565b5f60055f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1663e4dc2aa4306040518263ffffffff1660e01b81526004016103279190610f27565b602060405180830381865afa158015610342573d5f5f3e3d5ffd5b505050506040513d601f19601f820116820180604052508101906103669190611124565b905090565b5f5f610375610644565b905061038285828561065d565b61038d8585856106f0565b60019150509392505050565b5f5f60055f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16632ba21572306040518263ffffffff1660e01b81526004016103f59190610f27565b5f60405180830381865afa15801561040f573d5f5f3e3d5ffd5b505050506040513d5f823e3d601f19601f820116820180604052508101906104379190611088565b9091509050809150508091505090565b5f60055f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1663f7888aec30846040518363ffffffff1660e01b81526004016104a492919061114f565b602060405180830381865afa1580156104bf573d5f5f3e3d5ffd5b505050506040513d601f19601f820116820180604052508101906104e39190611124565b9050919050565b60605b60016104ed57606060055f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16632ba21572306040518263ffffffff1660e01b815260040161054f9190610f27565b5f60405180830381865afa158015610569573d5f5f3e3d5ffd5b505050506040513d5f823e3d601f19601f820116820180604052508101906105919190611088565b90915050809150508091505090565b5f5f6105aa610644565b90506105b78185856106f0565b600191505092915050565b5f60015f8473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f8373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f2054905092915050565b5f33905090565b61065883838360016107e0565b505050565b5f61066884846105c2565b90507fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8110156106ea57818110156106db578281836040517ffb8f41b20000000000000000000000000000000000000000000000000000000081526004016106d293929190611176565b60405180910390fd5b6106e984848484035f6107e0565b5b50505050565b5f73ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff1603610760575f6040517f96c6fd1e0000000000000000000000000000000000000000000000000000000081526004016107579190610f27565b60405180910390fd5b5f73ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff16036107d0575f6040517fec442f050000000000000000000000000000000000000000000000000000000081526004016107c79190610f27565b60405180910390fd5b6107db8383836109af565b505050565b5f73ffffffffffffffffffffffffffffffffffffffff168473ffffffffffffffffffffffffffffffffffffffff1603610850575f6040517fe602df050000000000000000000000000000000000000000000000000000000081526004016108479190610f27565b60405180910390fd5b5f73ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff16036108c0575f6040517f94280d620000000000000000000000000000000000000000000000000000000081526004016108b79190610f27565b60405180910390fd5b8160015f8673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f8573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f208190555080156109a9578273ffffffffffffffffffffffffffffffffffffffff168473ffffffffffffffffffffffffffffffffffffffff167f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925846040516109a09190610e12565b60405180910390a35b50505050565b5f73ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff1603610a845760055f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166340c10f1983836040518363ffffffff1660e01b8152600401610a3e9291906111ab565b6020604051808303815f875af1158015610a5a573d5f5f3e3d5ffd5b505050506040513d601f19601f82011682018060405250810190610a7e91906111fc565b50610bfa565b5f73ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff1603610b595760055f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16639dc29fac84836040518363ffffffff1660e01b8152600401610b139291906111ab565b6020604051808303815f875af1158015610b2f573d5f5f3e3d5ffd5b505050506040513d601f19601f82011682018060405250810190610b5391906111fc565b50610bf9565b60055f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1663beabacc88484846040518463ffffffff1660e01b8152600401610bb793929190611227565b6020604051808303815f875af1158015610bd3573d5f5f3e3d5ffd5b505050506040513d601f19601f82011682018060405250810190610bf791906111fc565b505b5b8173ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef83604051610c579190610e12565b60405180910390a3505050565b5f81519050919050565b5f82825260208201905092915050565b8281835e5f83830152505050565b5f601f19601f8301169050919050565b5f610ca682610c64565b610cb08185610c6e565b9350610cc0818560208601610c7e565b610cc981610c8c565b840191505092915050565b5f6020820190508181035f830152610cec8184610c9c565b905092915050565b5f604051905090565b5f5ffd5b5f5ffd5b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f610d2e82610d05565b9050919050565b610d3e81610d24565b8114610d48575f5ffd5b50565b5f81359050610d5981610d35565b92915050565b5f819050919050565b610d7181610d5f565b8114610d7b575f5ffd5b50565b5f81359050610d8c81610d68565b92915050565b5f5f60408385031215610da857610da7610cfd565b5b5f610db585828601610d4b565b9250506020610dc685828601610d7e565b9150509250929050565b5f8115159050919050565b610de481610dd0565b82525050565b5f602082019050610dfd5f830184610ddb565b92915050565b610e0c81610d5f565b82525050565b5f602082019050610e255f830184610e03565b92915050565b5f5f5f60608486031215610e4257610e41610cfd565b5b5f610e4f86828701610d4b565b9350506020610e6086828701610d4b565b9250506040610e7186828701610d7e565b9150509250925092565b5f60ff82169050919050565b610e9081610e7b565b82525050565b5f602082019050610ea95f830184610e87565b92915050565b5f60208284031215610ec457610ec3610cfd565b5b5f610ed184828501610d4b565b91505092915050565b5f5f60408385031215610ef057610eef610cfd565b5b5f610efd85828601610d4b565b9250506020610f0e85828601610d4b565b9150509250929050565b610f2181610d24565b82525050565b5f602082019050610f3a5f830184610f18565b92915050565b5f5ffd5b5f5ffd5b7f4e487b71000000000000000000000000000000000000000000000000000000005f52604160045260245ffd5b610f7e82610c8c565b810181811067ffffffffffffffff82111715610f9d57610f9c610f48565b5b80604052505050565b5f610faf610cf4565b9050610fbb8282610f75565b919050565b5f67ffffffffffffffff821115610fda57610fd9610f48565b5b610fe382610c8c565b9050602081019050919050565b5f611002610ffd84610fc0565b610fa6565b90508281526020810184848401111561101e5761101d610f44565b5b611029848285610c7e565b509392505050565b5f82601f83011261104557611044610f40565b5b8151611055848260208601610ff0565b91505092915050565b61106781610e7b565b8114611071575f5ffd5b50565b5f815190506110828161105e565b92915050565b5f5f5f6060848603121561109f5761109e610cfd565b5b5f84015167ffffffffffffffff8111156110bc576110bb610d01565b5b6110c886828701611031565b935050602084015167ffffffffffffffff8111156110e9576110e8610d01565b5b6110f586828701611031565b925050604061110686828701611074565b9150509250925092565b5f8151905061111e81610d68565b92915050565b5f6020828403121561113957611138610cfd565b5b5f61114684828501611110565b91505092915050565b5f6040820190506111625f830185610f18565b61116f6020830184610f18565b9392505050565b5f6060820190506111895f830186610f18565b6111966020830185610e03565b6111a36040830184610e03565b949350505050565b5f6040820190506111be5f830185610f18565b6111cb6020830184610e03565b9392505050565b6111db81610dd0565b81146111e5575f5ffd5b50565b5f815190506111f6816111d2565b92915050565b5f6020828403121561121157611210610cfd565b5b5f61121e848285016111e8565b91505092915050565b5f60608201905061123a5f830186610f18565b6112476020830185610f18565b6112546040830184610e03565b94935050505056fea2646970667358221220bff04d9c56866abda93d734471751ffb8265bf7be4c83f7016a795664a969b6964736f6c634300081b0033",
}

// FixedSupplyBankERC20InfiniteGasABI is the input ABI used to generate the binding from.
// Deprecated: Use FixedSupplyBankERC20InfiniteGasMetaData.ABI instead.
var FixedSupplyBankERC20InfiniteGasABI = FixedSupplyBankERC20InfiniteGasMetaData.ABI

// FixedSupplyBankERC20InfiniteGasBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use FixedSupplyBankERC20InfiniteGasMetaData.Bin instead.
var FixedSupplyBankERC20InfiniteGasBin = FixedSupplyBankERC20InfiniteGasMetaData.Bin

// DeployFixedSupplyBankERC20InfiniteGas deploys a new Ethereum contract, binding an instance of FixedSupplyBankERC20InfiniteGas to it.
func DeployFixedSupplyBankERC20InfiniteGas(auth *bind.TransactOpts, backend bind.ContractBackend, name_ string, symbol_ string, decimals_ uint8, initial_supply_ *big.Int) (common.Address, *types.Transaction, *FixedSupplyBankERC20InfiniteGas, error) {
	parsed, err := FixedSupplyBankERC20InfiniteGasMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(FixedSupplyBankERC20InfiniteGasBin), backend, name_, symbol_, decimals_, initial_supply_)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &FixedSupplyBankERC20InfiniteGas{FixedSupplyBankERC20InfiniteGasCaller: FixedSupplyBankERC20InfiniteGasCaller{contract: contract}, FixedSupplyBankERC20InfiniteGasTransactor: FixedSupplyBankERC20InfiniteGasTransactor{contract: contract}, FixedSupplyBankERC20InfiniteGasFilterer: FixedSupplyBankERC20InfiniteGasFilterer{contract: contract}}, nil
}

// FixedSupplyBankERC20InfiniteGas is an auto generated Go binding around an Ethereum contract.
type FixedSupplyBankERC20InfiniteGas struct {
	FixedSupplyBankERC20InfiniteGasCaller     // Read-only binding to the contract
	FixedSupplyBankERC20InfiniteGasTransactor // Write-only binding to the contract
	FixedSupplyBankERC20InfiniteGasFilterer   // Log filterer for contract events
}

// FixedSupplyBankERC20InfiniteGasCaller is an auto generated read-only Go binding around an Ethereum contract.
type FixedSupplyBankERC20InfiniteGasCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// FixedSupplyBankERC20InfiniteGasTransactor is an auto generated write-only Go binding around an Ethereum contract.
type FixedSupplyBankERC20InfiniteGasTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// FixedSupplyBankERC20InfiniteGasFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type FixedSupplyBankERC20InfiniteGasFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// FixedSupplyBankERC20InfiniteGasSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type FixedSupplyBankERC20InfiniteGasSession struct {
	Contract     *FixedSupplyBankERC20InfiniteGas // Generic contract binding to set the session for
	CallOpts     bind.CallOpts                    // Call options to use throughout this session
	TransactOpts bind.TransactOpts                // Transaction auth options to use throughout this session
}

// FixedSupplyBankERC20InfiniteGasCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type FixedSupplyBankERC20InfiniteGasCallerSession struct {
	Contract *FixedSupplyBankERC20InfiniteGasCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                          // Call options to use throughout this session
}

// FixedSupplyBankERC20InfiniteGasTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type FixedSupplyBankERC20InfiniteGasTransactorSession struct {
	Contract     *FixedSupplyBankERC20InfiniteGasTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                          // Transaction auth options to use throughout this session
}

// FixedSupplyBankERC20InfiniteGasRaw is an auto generated low-level Go binding around an Ethereum contract.
type FixedSupplyBankERC20InfiniteGasRaw struct {
	Contract *FixedSupplyBankERC20InfiniteGas // Generic contract binding to access the raw methods on
}

// FixedSupplyBankERC20InfiniteGasCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type FixedSupplyBankERC20InfiniteGasCallerRaw struct {
	Contract *FixedSupplyBankERC20InfiniteGasCaller // Generic read-only contract binding to access the raw methods on
}

// FixedSupplyBankERC20InfiniteGasTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type FixedSupplyBankERC20InfiniteGasTransactorRaw struct {
	Contract *FixedSupplyBankERC20InfiniteGasTransactor // Generic write-only contract binding to access the raw methods on
}

// NewFixedSupplyBankERC20InfiniteGas creates a new instance of FixedSupplyBankERC20InfiniteGas, bound to a specific deployed contract.
func NewFixedSupplyBankERC20InfiniteGas(address common.Address, backend bind.ContractBackend) (*FixedSupplyBankERC20InfiniteGas, error) {
	contract, err := bindFixedSupplyBankERC20InfiniteGas(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &FixedSupplyBankERC20InfiniteGas{FixedSupplyBankERC20InfiniteGasCaller: FixedSupplyBankERC20InfiniteGasCaller{contract: contract}, FixedSupplyBankERC20InfiniteGasTransactor: FixedSupplyBankERC20InfiniteGasTransactor{contract: contract}, FixedSupplyBankERC20InfiniteGasFilterer: FixedSupplyBankERC20InfiniteGasFilterer{contract: contract}}, nil
}

// NewFixedSupplyBankERC20InfiniteGasCaller creates a new read-only instance of FixedSupplyBankERC20InfiniteGas, bound to a specific deployed contract.
func NewFixedSupplyBankERC20InfiniteGasCaller(address common.Address, caller bind.ContractCaller) (*FixedSupplyBankERC20InfiniteGasCaller, error) {
	contract, err := bindFixedSupplyBankERC20InfiniteGas(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &FixedSupplyBankERC20InfiniteGasCaller{contract: contract}, nil
}

// NewFixedSupplyBankERC20InfiniteGasTransactor creates a new write-only instance of FixedSupplyBankERC20InfiniteGas, bound to a specific deployed contract.
func NewFixedSupplyBankERC20InfiniteGasTransactor(address common.Address, transactor bind.ContractTransactor) (*FixedSupplyBankERC20InfiniteGasTransactor, error) {
	contract, err := bindFixedSupplyBankERC20InfiniteGas(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &FixedSupplyBankERC20InfiniteGasTransactor{contract: contract}, nil
}

// NewFixedSupplyBankERC20InfiniteGasFilterer creates a new log filterer instance of FixedSupplyBankERC20InfiniteGas, bound to a specific deployed contract.
func NewFixedSupplyBankERC20InfiniteGasFilterer(address common.Address, filterer bind.ContractFilterer) (*FixedSupplyBankERC20InfiniteGasFilterer, error) {
	contract, err := bindFixedSupplyBankERC20InfiniteGas(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &FixedSupplyBankERC20InfiniteGasFilterer{contract: contract}, nil
}

// bindFixedSupplyBankERC20InfiniteGas binds a generic wrapper to an already deployed contract.
func bindFixedSupplyBankERC20InfiniteGas(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := FixedSupplyBankERC20InfiniteGasMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_FixedSupplyBankERC20InfiniteGas *FixedSupplyBankERC20InfiniteGasRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _FixedSupplyBankERC20InfiniteGas.Contract.FixedSupplyBankERC20InfiniteGasCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_FixedSupplyBankERC20InfiniteGas *FixedSupplyBankERC20InfiniteGasRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _FixedSupplyBankERC20InfiniteGas.Contract.FixedSupplyBankERC20InfiniteGasTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_FixedSupplyBankERC20InfiniteGas *FixedSupplyBankERC20InfiniteGasRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _FixedSupplyBankERC20InfiniteGas.Contract.FixedSupplyBankERC20InfiniteGasTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_FixedSupplyBankERC20InfiniteGas *FixedSupplyBankERC20InfiniteGasCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _FixedSupplyBankERC20InfiniteGas.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_FixedSupplyBankERC20InfiniteGas *FixedSupplyBankERC20InfiniteGasTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _FixedSupplyBankERC20InfiniteGas.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_FixedSupplyBankERC20InfiniteGas *FixedSupplyBankERC20InfiniteGasTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _FixedSupplyBankERC20InfiniteGas.Contract.contract.Transact(opts, method, params...)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_FixedSupplyBankERC20InfiniteGas *FixedSupplyBankERC20InfiniteGasCaller) Allowance(opts *bind.CallOpts, owner common.Address, spender common.Address) (*big.Int, error) {
	var out []interface{}
	err := _FixedSupplyBankERC20InfiniteGas.contract.Call(opts, &out, "allowance", owner, spender)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_FixedSupplyBankERC20InfiniteGas *FixedSupplyBankERC20InfiniteGasSession) Allowance(owner common.Address, spender common.Address) (*big.Int, error) {
	return _FixedSupplyBankERC20InfiniteGas.Contract.Allowance(&_FixedSupplyBankERC20InfiniteGas.CallOpts, owner, spender)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_FixedSupplyBankERC20InfiniteGas *FixedSupplyBankERC20InfiniteGasCallerSession) Allowance(owner common.Address, spender common.Address) (*big.Int, error) {
	return _FixedSupplyBankERC20InfiniteGas.Contract.Allowance(&_FixedSupplyBankERC20InfiniteGas.CallOpts, owner, spender)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_FixedSupplyBankERC20InfiniteGas *FixedSupplyBankERC20InfiniteGasCaller) BalanceOf(opts *bind.CallOpts, account common.Address) (*big.Int, error) {
	var out []interface{}
	err := _FixedSupplyBankERC20InfiniteGas.contract.Call(opts, &out, "balanceOf", account)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_FixedSupplyBankERC20InfiniteGas *FixedSupplyBankERC20InfiniteGasSession) BalanceOf(account common.Address) (*big.Int, error) {
	return _FixedSupplyBankERC20InfiniteGas.Contract.BalanceOf(&_FixedSupplyBankERC20InfiniteGas.CallOpts, account)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_FixedSupplyBankERC20InfiniteGas *FixedSupplyBankERC20InfiniteGasCallerSession) BalanceOf(account common.Address) (*big.Int, error) {
	return _FixedSupplyBankERC20InfiniteGas.Contract.BalanceOf(&_FixedSupplyBankERC20InfiniteGas.CallOpts, account)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_FixedSupplyBankERC20InfiniteGas *FixedSupplyBankERC20InfiniteGasCaller) Decimals(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _FixedSupplyBankERC20InfiniteGas.contract.Call(opts, &out, "decimals")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_FixedSupplyBankERC20InfiniteGas *FixedSupplyBankERC20InfiniteGasSession) Decimals() (uint8, error) {
	return _FixedSupplyBankERC20InfiniteGas.Contract.Decimals(&_FixedSupplyBankERC20InfiniteGas.CallOpts)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_FixedSupplyBankERC20InfiniteGas *FixedSupplyBankERC20InfiniteGasCallerSession) Decimals() (uint8, error) {
	return _FixedSupplyBankERC20InfiniteGas.Contract.Decimals(&_FixedSupplyBankERC20InfiniteGas.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_FixedSupplyBankERC20InfiniteGas *FixedSupplyBankERC20InfiniteGasCaller) Name(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _FixedSupplyBankERC20InfiniteGas.contract.Call(opts, &out, "name")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_FixedSupplyBankERC20InfiniteGas *FixedSupplyBankERC20InfiniteGasSession) Name() (string, error) {
	return _FixedSupplyBankERC20InfiniteGas.Contract.Name(&_FixedSupplyBankERC20InfiniteGas.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_FixedSupplyBankERC20InfiniteGas *FixedSupplyBankERC20InfiniteGasCallerSession) Name() (string, error) {
	return _FixedSupplyBankERC20InfiniteGas.Contract.Name(&_FixedSupplyBankERC20InfiniteGas.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_FixedSupplyBankERC20InfiniteGas *FixedSupplyBankERC20InfiniteGasCaller) Symbol(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _FixedSupplyBankERC20InfiniteGas.contract.Call(opts, &out, "symbol")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_FixedSupplyBankERC20InfiniteGas *FixedSupplyBankERC20InfiniteGasSession) Symbol() (string, error) {
	return _FixedSupplyBankERC20InfiniteGas.Contract.Symbol(&_FixedSupplyBankERC20InfiniteGas.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_FixedSupplyBankERC20InfiniteGas *FixedSupplyBankERC20InfiniteGasCallerSession) Symbol() (string, error) {
	return _FixedSupplyBankERC20InfiniteGas.Contract.Symbol(&_FixedSupplyBankERC20InfiniteGas.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_FixedSupplyBankERC20InfiniteGas *FixedSupplyBankERC20InfiniteGasCaller) TotalSupply(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _FixedSupplyBankERC20InfiniteGas.contract.Call(opts, &out, "totalSupply")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_FixedSupplyBankERC20InfiniteGas *FixedSupplyBankERC20InfiniteGasSession) TotalSupply() (*big.Int, error) {
	return _FixedSupplyBankERC20InfiniteGas.Contract.TotalSupply(&_FixedSupplyBankERC20InfiniteGas.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_FixedSupplyBankERC20InfiniteGas *FixedSupplyBankERC20InfiniteGasCallerSession) TotalSupply() (*big.Int, error) {
	return _FixedSupplyBankERC20InfiniteGas.Contract.TotalSupply(&_FixedSupplyBankERC20InfiniteGas.CallOpts)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 value) returns(bool)
func (_FixedSupplyBankERC20InfiniteGas *FixedSupplyBankERC20InfiniteGasTransactor) Approve(opts *bind.TransactOpts, spender common.Address, value *big.Int) (*types.Transaction, error) {
	return _FixedSupplyBankERC20InfiniteGas.contract.Transact(opts, "approve", spender, value)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 value) returns(bool)
func (_FixedSupplyBankERC20InfiniteGas *FixedSupplyBankERC20InfiniteGasSession) Approve(spender common.Address, value *big.Int) (*types.Transaction, error) {
	return _FixedSupplyBankERC20InfiniteGas.Contract.Approve(&_FixedSupplyBankERC20InfiniteGas.TransactOpts, spender, value)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 value) returns(bool)
func (_FixedSupplyBankERC20InfiniteGas *FixedSupplyBankERC20InfiniteGasTransactorSession) Approve(spender common.Address, value *big.Int) (*types.Transaction, error) {
	return _FixedSupplyBankERC20InfiniteGas.Contract.Approve(&_FixedSupplyBankERC20InfiniteGas.TransactOpts, spender, value)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 value) returns(bool)
func (_FixedSupplyBankERC20InfiniteGas *FixedSupplyBankERC20InfiniteGasTransactor) Transfer(opts *bind.TransactOpts, to common.Address, value *big.Int) (*types.Transaction, error) {
	return _FixedSupplyBankERC20InfiniteGas.contract.Transact(opts, "transfer", to, value)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 value) returns(bool)
func (_FixedSupplyBankERC20InfiniteGas *FixedSupplyBankERC20InfiniteGasSession) Transfer(to common.Address, value *big.Int) (*types.Transaction, error) {
	return _FixedSupplyBankERC20InfiniteGas.Contract.Transfer(&_FixedSupplyBankERC20InfiniteGas.TransactOpts, to, value)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address to, uint256 value) returns(bool)
func (_FixedSupplyBankERC20InfiniteGas *FixedSupplyBankERC20InfiniteGasTransactorSession) Transfer(to common.Address, value *big.Int) (*types.Transaction, error) {
	return _FixedSupplyBankERC20InfiniteGas.Contract.Transfer(&_FixedSupplyBankERC20InfiniteGas.TransactOpts, to, value)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 value) returns(bool)
func (_FixedSupplyBankERC20InfiniteGas *FixedSupplyBankERC20InfiniteGasTransactor) TransferFrom(opts *bind.TransactOpts, from common.Address, to common.Address, value *big.Int) (*types.Transaction, error) {
	return _FixedSupplyBankERC20InfiniteGas.contract.Transact(opts, "transferFrom", from, to, value)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 value) returns(bool)
func (_FixedSupplyBankERC20InfiniteGas *FixedSupplyBankERC20InfiniteGasSession) TransferFrom(from common.Address, to common.Address, value *big.Int) (*types.Transaction, error) {
	return _FixedSupplyBankERC20InfiniteGas.Contract.TransferFrom(&_FixedSupplyBankERC20InfiniteGas.TransactOpts, from, to, value)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address from, address to, uint256 value) returns(bool)
func (_FixedSupplyBankERC20InfiniteGas *FixedSupplyBankERC20InfiniteGasTransactorSession) TransferFrom(from common.Address, to common.Address, value *big.Int) (*types.Transaction, error) {
	return _FixedSupplyBankERC20InfiniteGas.Contract.TransferFrom(&_FixedSupplyBankERC20InfiniteGas.TransactOpts, from, to, value)
}

// FixedSupplyBankERC20InfiniteGasApprovalIterator is returned from FilterApproval and is used to iterate over the raw logs and unpacked data for Approval events raised by the FixedSupplyBankERC20InfiniteGas contract.
type FixedSupplyBankERC20InfiniteGasApprovalIterator struct {
	Event *FixedSupplyBankERC20InfiniteGasApproval // Event containing the contract specifics and raw log

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
func (it *FixedSupplyBankERC20InfiniteGasApprovalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FixedSupplyBankERC20InfiniteGasApproval)
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
		it.Event = new(FixedSupplyBankERC20InfiniteGasApproval)
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
func (it *FixedSupplyBankERC20InfiniteGasApprovalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FixedSupplyBankERC20InfiniteGasApprovalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FixedSupplyBankERC20InfiniteGasApproval represents a Approval event raised by the FixedSupplyBankERC20InfiniteGas contract.
type FixedSupplyBankERC20InfiniteGasApproval struct {
	Owner   common.Address
	Spender common.Address
	Value   *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterApproval is a free log retrieval operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_FixedSupplyBankERC20InfiniteGas *FixedSupplyBankERC20InfiniteGasFilterer) FilterApproval(opts *bind.FilterOpts, owner []common.Address, spender []common.Address) (*FixedSupplyBankERC20InfiniteGasApprovalIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _FixedSupplyBankERC20InfiniteGas.contract.FilterLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return &FixedSupplyBankERC20InfiniteGasApprovalIterator{contract: _FixedSupplyBankERC20InfiniteGas.contract, event: "Approval", logs: logs, sub: sub}, nil
}

// WatchApproval is a free log subscription operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_FixedSupplyBankERC20InfiniteGas *FixedSupplyBankERC20InfiniteGasFilterer) WatchApproval(opts *bind.WatchOpts, sink chan<- *FixedSupplyBankERC20InfiniteGasApproval, owner []common.Address, spender []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _FixedSupplyBankERC20InfiniteGas.contract.WatchLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FixedSupplyBankERC20InfiniteGasApproval)
				if err := _FixedSupplyBankERC20InfiniteGas.contract.UnpackLog(event, "Approval", log); err != nil {
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

// ParseApproval is a log parse operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_FixedSupplyBankERC20InfiniteGas *FixedSupplyBankERC20InfiniteGasFilterer) ParseApproval(log types.Log) (*FixedSupplyBankERC20InfiniteGasApproval, error) {
	event := new(FixedSupplyBankERC20InfiniteGasApproval)
	if err := _FixedSupplyBankERC20InfiniteGas.contract.UnpackLog(event, "Approval", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// FixedSupplyBankERC20InfiniteGasTransferIterator is returned from FilterTransfer and is used to iterate over the raw logs and unpacked data for Transfer events raised by the FixedSupplyBankERC20InfiniteGas contract.
type FixedSupplyBankERC20InfiniteGasTransferIterator struct {
	Event *FixedSupplyBankERC20InfiniteGasTransfer // Event containing the contract specifics and raw log

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
func (it *FixedSupplyBankERC20InfiniteGasTransferIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(FixedSupplyBankERC20InfiniteGasTransfer)
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
		it.Event = new(FixedSupplyBankERC20InfiniteGasTransfer)
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
func (it *FixedSupplyBankERC20InfiniteGasTransferIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *FixedSupplyBankERC20InfiniteGasTransferIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// FixedSupplyBankERC20InfiniteGasTransfer represents a Transfer event raised by the FixedSupplyBankERC20InfiniteGas contract.
type FixedSupplyBankERC20InfiniteGasTransfer struct {
	From  common.Address
	To    common.Address
	Value *big.Int
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterTransfer is a free log retrieval operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_FixedSupplyBankERC20InfiniteGas *FixedSupplyBankERC20InfiniteGasFilterer) FilterTransfer(opts *bind.FilterOpts, from []common.Address, to []common.Address) (*FixedSupplyBankERC20InfiniteGasTransferIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _FixedSupplyBankERC20InfiniteGas.contract.FilterLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &FixedSupplyBankERC20InfiniteGasTransferIterator{contract: _FixedSupplyBankERC20InfiniteGas.contract, event: "Transfer", logs: logs, sub: sub}, nil
}

// WatchTransfer is a free log subscription operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_FixedSupplyBankERC20InfiniteGas *FixedSupplyBankERC20InfiniteGasFilterer) WatchTransfer(opts *bind.WatchOpts, sink chan<- *FixedSupplyBankERC20InfiniteGasTransfer, from []common.Address, to []common.Address) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _FixedSupplyBankERC20InfiniteGas.contract.WatchLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(FixedSupplyBankERC20InfiniteGasTransfer)
				if err := _FixedSupplyBankERC20InfiniteGas.contract.UnpackLog(event, "Transfer", log); err != nil {
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

// ParseTransfer is a log parse operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_FixedSupplyBankERC20InfiniteGas *FixedSupplyBankERC20InfiniteGasFilterer) ParseTransfer(log types.Log) (*FixedSupplyBankERC20InfiniteGasTransfer, error) {
	event := new(FixedSupplyBankERC20InfiniteGasTransfer)
	if err := _FixedSupplyBankERC20InfiniteGas.contract.UnpackLog(event, "Transfer", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
