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
	ABI: "[{\"type\":\"constructor\",\"inputs\":[{\"name\":\"name_\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"symbol_\",\"type\":\"string\",\"internalType\":\"string\"},{\"name\":\"decimals_\",\"type\":\"uint8\",\"internalType\":\"uint8\"},{\"name\":\"initial_supply_\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"payable\"},{\"type\":\"function\",\"name\":\"allowance\",\"inputs\":[{\"name\":\"owner\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"spender\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"approve\",\"inputs\":[{\"name\":\"spender\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"value\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"balanceOf\",\"inputs\":[{\"name\":\"account\",\"type\":\"address\",\"internalType\":\"address\"}],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"decimals\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint8\",\"internalType\":\"uint8\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"name\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"symbol\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"string\",\"internalType\":\"string\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"totalSupply\",\"inputs\":[],\"outputs\":[{\"name\":\"\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"stateMutability\":\"view\"},{\"type\":\"function\",\"name\":\"transfer\",\"inputs\":[{\"name\":\"to\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"value\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"function\",\"name\":\"transferFrom\",\"inputs\":[{\"name\":\"from\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"to\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"value\",\"type\":\"uint256\",\"internalType\":\"uint256\"}],\"outputs\":[{\"name\":\"\",\"type\":\"bool\",\"internalType\":\"bool\"}],\"stateMutability\":\"nonpayable\"},{\"type\":\"event\",\"name\":\"Approval\",\"inputs\":[{\"name\":\"owner\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"spender\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"value\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"event\",\"name\":\"Transfer\",\"inputs\":[{\"name\":\"from\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"to\",\"type\":\"address\",\"indexed\":true,\"internalType\":\"address\"},{\"name\":\"value\",\"type\":\"uint256\",\"indexed\":false,\"internalType\":\"uint256\"}],\"anonymous\":false},{\"type\":\"error\",\"name\":\"ERC20InsufficientAllowance\",\"inputs\":[{\"name\":\"spender\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"allowance\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"needed\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"type\":\"error\",\"name\":\"ERC20InsufficientBalance\",\"inputs\":[{\"name\":\"sender\",\"type\":\"address\",\"internalType\":\"address\"},{\"name\":\"balance\",\"type\":\"uint256\",\"internalType\":\"uint256\"},{\"name\":\"needed\",\"type\":\"uint256\",\"internalType\":\"uint256\"}]},{\"type\":\"error\",\"name\":\"ERC20InvalidApprover\",\"inputs\":[{\"name\":\"approver\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"ERC20InvalidReceiver\",\"inputs\":[{\"name\":\"receiver\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"ERC20InvalidSender\",\"inputs\":[{\"name\":\"sender\",\"type\":\"address\",\"internalType\":\"address\"}]},{\"type\":\"error\",\"name\":\"ERC20InvalidSpender\",\"inputs\":[{\"name\":\"spender\",\"type\":\"address\",\"internalType\":\"address\"}]}]",
	Bin: "0x6080604052606460055f6101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550604051611e83380380611e8383398181016040528101906100669190610677565b83838360405180602001604052805f81525060405180602001604052805f81525081600390816100969190610917565b5080600490816100a69190610917565b50505060055f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166337d2c2f48484846040518463ffffffff1660e01b815260040161010793929190610a3d565b6020604051808303815f875af1158015610123573d5f803e3d5ffd5b505050506040513d601f19601f820116820180604052508101906101479190610ab5565b505050505f81111561016457610163338261016d60201b60201c565b5b50505050610bbc565b5f73ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff16036101dd575f6040517fec442f050000000000000000000000000000000000000000000000000000000081526004016101d49190610b1f565b60405180910390fd5b6101ee5f83836101f260201b60201c565b5050565b5f73ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff16036102c75760055f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166340c10f1983836040518363ffffffff1660e01b8152600401610281929190610b47565b6020604051808303815f875af115801561029d573d5f803e3d5ffd5b505050506040513d601f19601f820116820180604052508101906102c19190610ab5565b5061043d565b5f73ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff160361039c5760055f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16639dc29fac84836040518363ffffffff1660e01b8152600401610356929190610b47565b6020604051808303815f875af1158015610372573d5f803e3d5ffd5b505050506040513d601f19601f820116820180604052508101906103969190610ab5565b5061043c565b60055f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1663beabacc88484846040518463ffffffff1660e01b81526004016103fa93929190610b6e565b6020604051808303815f875af1158015610416573d5f803e3d5ffd5b505050506040513d601f19601f8201168201806040525081019061043a9190610ab5565b505b5b8173ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef8360405161049a9190610ba3565b60405180910390a3505050565b5f604051905090565b5f80fd5b5f80fd5b5f80fd5b5f80fd5b5f601f19601f8301169050919050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52604160045260245ffd5b610506826104c0565b810181811067ffffffffffffffff82111715610525576105246104d0565b5b80604052505050565b5f6105376104a7565b905061054382826104fd565b919050565b5f67ffffffffffffffff821115610562576105616104d0565b5b61056b826104c0565b9050602081019050919050565b5f5b8381101561059557808201518184015260208101905061057a565b5f8484015250505050565b5f6105b26105ad84610548565b61052e565b9050828152602081018484840111156105ce576105cd6104bc565b5b6105d9848285610578565b509392505050565b5f82601f8301126105f5576105f46104b8565b5b81516106058482602086016105a0565b91505092915050565b5f60ff82169050919050565b6106238161060e565b811461062d575f80fd5b50565b5f8151905061063e8161061a565b92915050565b5f819050919050565b61065681610644565b8114610660575f80fd5b50565b5f815190506106718161064d565b92915050565b5f805f806080858703121561068f5761068e6104b0565b5b5f85015167ffffffffffffffff8111156106ac576106ab6104b4565b5b6106b8878288016105e1565b945050602085015167ffffffffffffffff8111156106d9576106d86104b4565b5b6106e5878288016105e1565b93505060406106f687828801610630565b925050606061070787828801610663565b91505092959194509250565b5f81519050919050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52602260045260245ffd5b5f600282049050600182168061076157607f821691505b6020821081036107745761077361071d565b5b50919050565b5f819050815f5260205f209050919050565b5f6020601f8301049050919050565b5f82821b905092915050565b5f600883026107d67fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8261079b565b6107e0868361079b565b95508019841693508086168417925050509392505050565b5f819050919050565b5f61081b61081661081184610644565b6107f8565b610644565b9050919050565b5f819050919050565b61083483610801565b61084861084082610822565b8484546107a7565b825550505050565b5f90565b61085c610850565b61086781848461082b565b505050565b5b8181101561088a5761087f5f82610854565b60018101905061086d565b5050565b601f8211156108cf576108a08161077a565b6108a98461078c565b810160208510156108b8578190505b6108cc6108c48561078c565b83018261086c565b50505b505050565b5f82821c905092915050565b5f6108ef5f19846008026108d4565b1980831691505092915050565b5f61090783836108e0565b9150826002028217905092915050565b61092082610713565b67ffffffffffffffff811115610939576109386104d0565b5b610943825461074a565b61094e82828561088e565b5f60209050601f83116001811461097f575f841561096d578287015190505b61097785826108fc565b8655506109de565b601f19841661098d8661077a565b5f5b828110156109b45784890151825560018201915060208501945060208101905061098f565b868310156109d157848901516109cd601f8916826108e0565b8355505b6001600288020188555050505b505050505050565b5f82825260208201905092915050565b5f610a0082610713565b610a0a81856109e6565b9350610a1a818560208601610578565b610a23816104c0565b840191505092915050565b610a378161060e565b82525050565b5f6060820190508181035f830152610a5581866109f6565b90508181036020830152610a6981856109f6565b9050610a786040830184610a2e565b949350505050565b5f8115159050919050565b610a9481610a80565b8114610a9e575f80fd5b50565b5f81519050610aaf81610a8b565b92915050565b5f60208284031215610aca57610ac96104b0565b5b5f610ad784828501610aa1565b91505092915050565b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f610b0982610ae0565b9050919050565b610b1981610aff565b82525050565b5f602082019050610b325f830184610b10565b92915050565b610b4181610644565b82525050565b5f604082019050610b5a5f830185610b10565b610b676020830184610b38565b9392505050565b5f606082019050610b815f830186610b10565b610b8e6020830185610b10565b610b9b6040830184610b38565b949350505050565b5f602082019050610bb65f830184610b38565b92915050565b6112ba80610bc95f395ff3fe608060405234801561000f575f80fd5b5060043610610091575f3560e01c8063313ce56711610064578063313ce5671461013157806370a082311461014f57806395d89b411461017f578063a9059cbb1461019d578063dd62ed3e146101cd57610091565b806306fdde0314610095578063095ea7b3146100b357806318160ddd146100e357806323b872dd14610101575b5f80fd5b61009d6101fd565b6040516100aa9190610cfc565b60405180910390f35b6100cd60048036038101906100c89190610dba565b6102aa565b6040516100da9190610e12565b60405180910390f35b6100eb6102cc565b6040516100f89190610e3a565b60405180910390f35b61011b60048036038101906101169190610e53565b61036b565b6040516101289190610e12565b60405180910390f35b610139610399565b6040516101469190610ebe565b60405180910390f35b61016960048036038101906101649190610ed7565b610447565b6040516101769190610e3a565b60405180910390f35b6101876104ea565b6040516101949190610cfc565b60405180910390f35b6101b760048036038101906101b29190610dba565b610500565b6040516101c49190610e12565b60405180910390f35b6101e760048036038101906101e29190610f02565b610522565b6040516101f49190610e3a565b60405180910390f35b60608060055f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16632ba21572306040518263ffffffff1660e01b815260040161025a9190610f4f565b5f60405180830381865afa158015610274573d5f803e3d5ffd5b505050506040513d5f823e3d601f19601f8201168201806040525081019061029c91906110b0565b905050809150508091505090565b5f806102b46105a4565b90506102c18185856105ab565b600191505092915050565b5f60055f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1663e4dc2aa4306040518263ffffffff1660e01b81526004016103279190610f4f565b602060405180830381865afa158015610342573d5f803e3d5ffd5b505050506040513d601f19601f82011682018060405250810190610366919061114c565b905090565b5f806103756105a4565b90506103828582856105bd565b61038d858585610650565b60019150509392505050565b5f8060055f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16632ba21572306040518263ffffffff1660e01b81526004016103f59190610f4f565b5f60405180830381865afa15801561040f573d5f803e3d5ffd5b505050506040513d5f823e3d601f19601f8201168201806040525081019061043791906110b0565b9091509050809150508091505090565b5f60055f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1663f7888aec30846040518363ffffffff1660e01b81526004016104a4929190611177565b602060405180830381865afa1580156104bf573d5f803e3d5ffd5b505050506040513d601f19601f820116820180604052508101906104e3919061114c565b9050919050565b60605b60016104ed576104fb610740565b905090565b5f8061050a6105a4565b9050610517818585610650565b600191505092915050565b5f60015f8473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f8373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f2054905092915050565b5f33905090565b6105b883838360016107ee565b505050565b5f6105c88484610522565b90507fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff81101561064a578181101561063b578281836040517ffb8f41b20000000000000000000000000000000000000000000000000000000081526004016106329392919061119e565b60405180910390fd5b61064984848484035f6107ee565b5b50505050565b5f73ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff16036106c0575f6040517f96c6fd1e0000000000000000000000000000000000000000000000000000000081526004016106b79190610f4f565b60405180910390fd5b5f73ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff1603610730575f6040517fec442f050000000000000000000000000000000000000000000000000000000081526004016107279190610f4f565b60405180910390fd5b61073b8383836109bd565b505050565b60608060055f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16632ba21572306040518263ffffffff1660e01b815260040161079d9190610f4f565b5f60405180830381865afa1580156107b7573d5f803e3d5ffd5b505050506040513d5f823e3d601f19601f820116820180604052508101906107df91906110b0565b90915050809150508091505090565b5f73ffffffffffffffffffffffffffffffffffffffff168473ffffffffffffffffffffffffffffffffffffffff160361085e575f6040517fe602df050000000000000000000000000000000000000000000000000000000081526004016108559190610f4f565b60405180910390fd5b5f73ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff16036108ce575f6040517f94280d620000000000000000000000000000000000000000000000000000000081526004016108c59190610f4f565b60405180910390fd5b8160015f8673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f8573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f208190555080156109b7578273ffffffffffffffffffffffffffffffffffffffff168473ffffffffffffffffffffffffffffffffffffffff167f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925846040516109ae9190610e3a565b60405180910390a35b50505050565b5f73ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff1603610a925760055f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166340c10f1983836040518363ffffffff1660e01b8152600401610a4c9291906111d3565b6020604051808303815f875af1158015610a68573d5f803e3d5ffd5b505050506040513d601f19601f82011682018060405250810190610a8c9190611224565b50610c08565b5f73ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff1603610b675760055f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16639dc29fac84836040518363ffffffff1660e01b8152600401610b219291906111d3565b6020604051808303815f875af1158015610b3d573d5f803e3d5ffd5b505050506040513d601f19601f82011682018060405250810190610b619190611224565b50610c07565b60055f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1663beabacc88484846040518463ffffffff1660e01b8152600401610bc59392919061124f565b6020604051808303815f875af1158015610be1573d5f803e3d5ffd5b505050506040513d601f19601f82011682018060405250810190610c059190611224565b505b5b8173ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef83604051610c659190610e3a565b60405180910390a3505050565b5f81519050919050565b5f82825260208201905092915050565b5f5b83811015610ca9578082015181840152602081019050610c8e565b5f8484015250505050565b5f601f19601f8301169050919050565b5f610cce82610c72565b610cd88185610c7c565b9350610ce8818560208601610c8c565b610cf181610cb4565b840191505092915050565b5f6020820190508181035f830152610d148184610cc4565b905092915050565b5f604051905090565b5f80fd5b5f80fd5b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f610d5682610d2d565b9050919050565b610d6681610d4c565b8114610d70575f80fd5b50565b5f81359050610d8181610d5d565b92915050565b5f819050919050565b610d9981610d87565b8114610da3575f80fd5b50565b5f81359050610db481610d90565b92915050565b5f8060408385031215610dd057610dcf610d25565b5b5f610ddd85828601610d73565b9250506020610dee85828601610da6565b9150509250929050565b5f8115159050919050565b610e0c81610df8565b82525050565b5f602082019050610e255f830184610e03565b92915050565b610e3481610d87565b82525050565b5f602082019050610e4d5f830184610e2b565b92915050565b5f805f60608486031215610e6a57610e69610d25565b5b5f610e7786828701610d73565b9350506020610e8886828701610d73565b9250506040610e9986828701610da6565b9150509250925092565b5f60ff82169050919050565b610eb881610ea3565b82525050565b5f602082019050610ed15f830184610eaf565b92915050565b5f60208284031215610eec57610eeb610d25565b5b5f610ef984828501610d73565b91505092915050565b5f8060408385031215610f1857610f17610d25565b5b5f610f2585828601610d73565b9250506020610f3685828601610d73565b9150509250929050565b610f4981610d4c565b82525050565b5f602082019050610f625f830184610f40565b92915050565b5f80fd5b5f80fd5b7f4e487b71000000000000000000000000000000000000000000000000000000005f52604160045260245ffd5b610fa682610cb4565b810181811067ffffffffffffffff82111715610fc557610fc4610f70565b5b80604052505050565b5f610fd7610d1c565b9050610fe38282610f9d565b919050565b5f67ffffffffffffffff82111561100257611001610f70565b5b61100b82610cb4565b9050602081019050919050565b5f61102a61102584610fe8565b610fce565b90508281526020810184848401111561104657611045610f6c565b5b611051848285610c8c565b509392505050565b5f82601f83011261106d5761106c610f68565b5b815161107d848260208601611018565b91505092915050565b61108f81610ea3565b8114611099575f80fd5b50565b5f815190506110aa81611086565b92915050565b5f805f606084860312156110c7576110c6610d25565b5b5f84015167ffffffffffffffff8111156110e4576110e3610d29565b5b6110f086828701611059565b935050602084015167ffffffffffffffff81111561111157611110610d29565b5b61111d86828701611059565b925050604061112e8682870161109c565b9150509250925092565b5f8151905061114681610d90565b92915050565b5f6020828403121561116157611160610d25565b5b5f61116e84828501611138565b91505092915050565b5f60408201905061118a5f830185610f40565b6111976020830184610f40565b9392505050565b5f6060820190506111b15f830186610f40565b6111be6020830185610e2b565b6111cb6040830184610e2b565b949350505050565b5f6040820190506111e65f830185610f40565b6111f36020830184610e2b565b9392505050565b61120381610df8565b811461120d575f80fd5b50565b5f8151905061121e816111fa565b92915050565b5f6020828403121561123957611238610d25565b5b5f61124684828501611210565b91505092915050565b5f6060820190506112625f830186610f40565b61126f6020830185610f40565b61127c6040830184610e2b565b94935050505056fea2646970667358221220fad6557bd52d1e1caa5ab5099cd627b252303c5ddd6e1c625343e1b987d29cf864736f6c634300081a0033",
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
