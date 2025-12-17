// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package wrappers

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

// InjERC20MetaData contains all meta data concerning the InjERC20 contract.
var InjERC20MetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"string\",\"name\":\"name_\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol_\",\"type\":\"string\"},{\"internalType\":\"uint8\",\"name\":\"decimals_\",\"type\":\"uint8\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Approval\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Transfer\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"}],\"name\":\"allowance\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"approve\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"balanceOf\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"burn\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"decimals\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"subtractedValue\",\"type\":\"uint256\"}],\"name\":\"decreaseAllowance\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"addedValue\",\"type\":\"uint256\"}],\"name\":\"increaseAllowance\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"mint\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"name\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"symbol\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"totalSupply\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"transfer\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"sender\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"recipient\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"transferFrom\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x60a060405234801561000f575f5ffd5b50604051612418380380612418833981810160405281019061003191906102a1565b828281600390816100429190610539565b5080600490816100529190610539565b5050505f61006461011760201b60201c565b90508060055f6101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055508073ffffffffffffffffffffffffffffffffffffffff165f73ffffffffffffffffffffffffffffffffffffffff167f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e060405160405180910390a3508060ff1660808160ff1681525050505050610608565b5f33905090565b5f604051905090565b5f5ffd5b5f5ffd5b5f5ffd5b5f5ffd5b5f601f19601f8301169050919050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52604160045260245ffd5b61017d82610137565b810181811067ffffffffffffffff8211171561019c5761019b610147565b5b80604052505050565b5f6101ae61011e565b90506101ba8282610174565b919050565b5f67ffffffffffffffff8211156101d9576101d8610147565b5b6101e282610137565b9050602081019050919050565b8281835e5f83830152505050565b5f61020f61020a846101bf565b6101a5565b90508281526020810184848401111561022b5761022a610133565b5b6102368482856101ef565b509392505050565b5f82601f8301126102525761025161012f565b5b81516102628482602086016101fd565b91505092915050565b5f60ff82169050919050565b6102808161026b565b811461028a575f5ffd5b50565b5f8151905061029b81610277565b92915050565b5f5f5f606084860312156102b8576102b7610127565b5b5f84015167ffffffffffffffff8111156102d5576102d461012b565b5b6102e18682870161023e565b935050602084015167ffffffffffffffff8111156103025761030161012b565b5b61030e8682870161023e565b925050604061031f8682870161028d565b9150509250925092565b5f81519050919050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52602260045260245ffd5b5f600282049050600182168061037757607f821691505b60208210810361038a57610389610333565b5b50919050565b5f819050815f5260205f209050919050565b5f6020601f8301049050919050565b5f82821b905092915050565b5f600883026103ec7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff826103b1565b6103f686836103b1565b95508019841693508086168417925050509392505050565b5f819050919050565b5f819050919050565b5f61043a6104356104308461040e565b610417565b61040e565b9050919050565b5f819050919050565b61045383610420565b61046761045f82610441565b8484546103bd565b825550505050565b5f5f905090565b61047e61046f565b61048981848461044a565b505050565b5b818110156104ac576104a15f82610476565b60018101905061048f565b5050565b601f8211156104f1576104c281610390565b6104cb846103a2565b810160208510156104da578190505b6104ee6104e6856103a2565b83018261048e565b50505b505050565b5f82821c905092915050565b5f6105115f19846008026104f6565b1980831691505092915050565b5f6105298383610502565b9150826002028217905092915050565b61054282610329565b67ffffffffffffffff81111561055b5761055a610147565b5b6105658254610360565b6105708282856104b0565b5f60209050601f8311600181146105a1575f841561058f578287015190505b610599858261051e565b865550610600565b601f1984166105af86610390565b5f5b828110156105d6578489015182556001820191506020850194506020810190506105b1565b868310156105f357848901516105ef601f891682610502565b8355505b6001600288020188555050505b505050505050565b608051611df86106205f395f6104fa0152611df85ff3fe608060405234801561000f575f5ffd5b50600436106100fe575f3560e01c8063715018a611610095578063a457c2d711610064578063a457c2d71461029a578063a9059cbb146102ca578063dd62ed3e146102fa578063f2fde38b1461032a576100fe565b8063715018a6146102385780638da5cb5b1461024257806395d89b41146102605780639dc29fac1461027e576100fe565b8063313ce567116100d1578063313ce5671461019e57806339509351146101bc57806340c10f19146101ec57806370a0823114610208576100fe565b806306fdde0314610102578063095ea7b31461012057806318160ddd1461015057806323b872dd1461016e575b5f5ffd5b61010a610346565b6040516101179190611417565b60405180910390f35b61013a600480360381019061013591906114c8565b6103d6565b6040516101479190611520565b60405180910390f35b6101586103f3565b6040516101659190611548565b60405180910390f35b61018860048036038101906101839190611561565b6103fc565b6040516101959190611520565b60405180910390f35b6101a66104f7565b6040516101b391906115cc565b60405180910390f35b6101d660048036038101906101d191906114c8565b61051e565b6040516101e39190611520565b60405180910390f35b610206600480360381019061020191906114c8565b6105c5565b005b610222600480360381019061021d91906115e5565b61064f565b60405161022f9190611548565b60405180910390f35b610240610694565b005b61024a61071a565b604051610257919061161f565b60405180910390f35b610268610742565b6040516102759190611417565b60405180910390f35b610298600480360381019061029391906114c8565b6107d2565b005b6102b460048036038101906102af91906114c8565b61085c565b6040516102c19190611520565b60405180910390f35b6102e460048036038101906102df91906114c8565b61094b565b6040516102f19190611520565b60405180910390f35b610314600480360381019061030f9190611638565b610968565b6040516103219190611548565b60405180910390f35b610344600480360381019061033f91906115e5565b6109ea565b005b606060038054610355906116a3565b80601f0160208091040260200160405190810160405280929190818152602001828054610381906116a3565b80156103cc5780601f106103a3576101008083540402835291602001916103cc565b820191905f5260205f20905b8154815290600101906020018083116103af57829003601f168201915b5050505050905090565b5f6103e96103e2610b92565b8484610b99565b6001905092915050565b5f600254905090565b5f610408848484610d5c565b5f60015f8673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f61044f610b92565b73ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f20549050828110156104ce576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016104c590611743565b60405180910390fd5b6104eb856104da610b92565b85846104e6919061178e565b610b99565b60019150509392505050565b5f7f0000000000000000000000000000000000000000000000000000000000000000905090565b5f6105bb61052a610b92565b848460015f610537610b92565b73ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f8873ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f20546105b691906117c1565b610b99565b6001905092915050565b6105cd610b92565b73ffffffffffffffffffffffffffffffffffffffff166105eb61071a565b73ffffffffffffffffffffffffffffffffffffffff1614610641576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016106389061183e565b60405180910390fd5b61064b8282610fcf565b5050565b5f5f5f8373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f20549050919050565b61069c610b92565b73ffffffffffffffffffffffffffffffffffffffff166106ba61071a565b73ffffffffffffffffffffffffffffffffffffffff1614610710576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016107079061183e565b60405180910390fd5b61071861111b565b565b5f60055f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff16905090565b606060048054610751906116a3565b80601f016020809104026020016040519081016040528092919081815260200182805461077d906116a3565b80156107c85780601f1061079f576101008083540402835291602001916107c8565b820191905f5260205f20905b8154815290600101906020018083116107ab57829003601f168201915b5050505050905090565b6107da610b92565b73ffffffffffffffffffffffffffffffffffffffff166107f861071a565b73ffffffffffffffffffffffffffffffffffffffff161461084e576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016108459061183e565b60405180910390fd5b61085882826111d8565b5050565b5f5f60015f610869610b92565b73ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f8573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f2054905082811015610923576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161091a906118cc565b60405180910390fd5b61094061092e610b92565b85858461093b919061178e565b610b99565b600191505092915050565b5f61095e610957610b92565b8484610d5c565b6001905092915050565b5f60015f8473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f8373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f2054905092915050565b6109f2610b92565b73ffffffffffffffffffffffffffffffffffffffff16610a1061071a565b73ffffffffffffffffffffffffffffffffffffffff1614610a66576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610a5d9061183e565b60405180910390fd5b5f73ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff1603610ad4576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610acb9061195a565b60405180910390fd5b8073ffffffffffffffffffffffffffffffffffffffff1660055f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff167f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e060405160405180910390a38060055f6101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555050565b5f33905090565b5f73ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff1603610c07576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610bfe906119e8565b60405180910390fd5b5f73ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff1603610c75576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610c6c90611a76565b60405180910390fd5b8060015f8573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f8473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f20819055508173ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff167f8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b92583604051610d4f9190611548565b60405180910390a3505050565b5f73ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff1603610dca576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610dc190611b04565b60405180910390fd5b5f73ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff1603610e38576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610e2f90611b92565b60405180910390fd5b610e438383836113a2565b5f5f5f8573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f2054905081811015610ec6576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610ebd90611c20565b60405180910390fd5b8181610ed2919061178e565b5f5f8673ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f2081905550815f5f8573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f828254610f5d91906117c1565b925050819055508273ffffffffffffffffffffffffffffffffffffffff168473ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef84604051610fc19190611548565b60405180910390a350505050565b5f73ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff160361103d576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161103490611c88565b60405180910390fd5b6110485f83836113a2565b8060025f82825461105991906117c1565b92505081905550805f5f8473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f205f8282546110ab91906117c1565b925050819055508173ffffffffffffffffffffffffffffffffffffffff165f73ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef8360405161110f9190611548565b60405180910390a35050565b5f73ffffffffffffffffffffffffffffffffffffffff1660055f9054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff167f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e060405160405180910390a35f60055f6101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff160217905550565b5f73ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff1603611246576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161123d90611d16565b60405180910390fd5b611251825f836113a2565b5f5f5f8473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f20549050818110156112d4576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016112cb90611da4565b60405180910390fd5b81816112e0919061178e565b5f5f8573ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020015f20819055508160025f828254611331919061178e565b925050819055505f73ffffffffffffffffffffffffffffffffffffffff168373ffffffffffffffffffffffffffffffffffffffff167fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef846040516113959190611548565b60405180910390a3505050565b505050565b5f81519050919050565b5f82825260208201905092915050565b8281835e5f83830152505050565b5f601f19601f8301169050919050565b5f6113e9826113a7565b6113f381856113b1565b93506114038185602086016113c1565b61140c816113cf565b840191505092915050565b5f6020820190508181035f83015261142f81846113df565b905092915050565b5f5ffd5b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f6114648261143b565b9050919050565b6114748161145a565b811461147e575f5ffd5b50565b5f8135905061148f8161146b565b92915050565b5f819050919050565b6114a781611495565b81146114b1575f5ffd5b50565b5f813590506114c28161149e565b92915050565b5f5f604083850312156114de576114dd611437565b5b5f6114eb85828601611481565b92505060206114fc858286016114b4565b9150509250929050565b5f8115159050919050565b61151a81611506565b82525050565b5f6020820190506115335f830184611511565b92915050565b61154281611495565b82525050565b5f60208201905061155b5f830184611539565b92915050565b5f5f5f6060848603121561157857611577611437565b5b5f61158586828701611481565b935050602061159686828701611481565b92505060406115a7868287016114b4565b9150509250925092565b5f60ff82169050919050565b6115c6816115b1565b82525050565b5f6020820190506115df5f8301846115bd565b92915050565b5f602082840312156115fa576115f9611437565b5b5f61160784828501611481565b91505092915050565b6116198161145a565b82525050565b5f6020820190506116325f830184611610565b92915050565b5f5f6040838503121561164e5761164d611437565b5b5f61165b85828601611481565b925050602061166c85828601611481565b9150509250929050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52602260045260245ffd5b5f60028204905060018216806116ba57607f821691505b6020821081036116cd576116cc611676565b5b50919050565b7f45524332303a207472616e7366657220616d6f756e74206578636565647320615f8201527f6c6c6f77616e6365000000000000000000000000000000000000000000000000602082015250565b5f61172d6028836113b1565b9150611738826116d3565b604082019050919050565b5f6020820190508181035f83015261175a81611721565b9050919050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f61179882611495565b91506117a383611495565b92508282039050818111156117bb576117ba611761565b5b92915050565b5f6117cb82611495565b91506117d683611495565b92508282019050808211156117ee576117ed611761565b5b92915050565b7f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e65725f82015250565b5f6118286020836113b1565b9150611833826117f4565b602082019050919050565b5f6020820190508181035f8301526118558161181c565b9050919050565b7f45524332303a2064656372656173656420616c6c6f77616e63652062656c6f775f8201527f207a65726f000000000000000000000000000000000000000000000000000000602082015250565b5f6118b66025836113b1565b91506118c18261185c565b604082019050919050565b5f6020820190508181035f8301526118e3816118aa565b9050919050565b7f4f776e61626c653a206e6577206f776e657220697320746865207a65726f20615f8201527f6464726573730000000000000000000000000000000000000000000000000000602082015250565b5f6119446026836113b1565b915061194f826118ea565b604082019050919050565b5f6020820190508181035f83015261197181611938565b9050919050565b7f45524332303a20617070726f76652066726f6d20746865207a65726f206164645f8201527f7265737300000000000000000000000000000000000000000000000000000000602082015250565b5f6119d26024836113b1565b91506119dd82611978565b604082019050919050565b5f6020820190508181035f8301526119ff816119c6565b9050919050565b7f45524332303a20617070726f766520746f20746865207a65726f2061646472655f8201527f7373000000000000000000000000000000000000000000000000000000000000602082015250565b5f611a606022836113b1565b9150611a6b82611a06565b604082019050919050565b5f6020820190508181035f830152611a8d81611a54565b9050919050565b7f45524332303a207472616e736665722066726f6d20746865207a65726f2061645f8201527f6472657373000000000000000000000000000000000000000000000000000000602082015250565b5f611aee6025836113b1565b9150611af982611a94565b604082019050919050565b5f6020820190508181035f830152611b1b81611ae2565b9050919050565b7f45524332303a207472616e7366657220746f20746865207a65726f20616464725f8201527f6573730000000000000000000000000000000000000000000000000000000000602082015250565b5f611b7c6023836113b1565b9150611b8782611b22565b604082019050919050565b5f6020820190508181035f830152611ba981611b70565b9050919050565b7f45524332303a207472616e7366657220616d6f756e74206578636565647320625f8201527f616c616e63650000000000000000000000000000000000000000000000000000602082015250565b5f611c0a6026836113b1565b9150611c1582611bb0565b604082019050919050565b5f6020820190508181035f830152611c3781611bfe565b9050919050565b7f45524332303a206d696e7420746f20746865207a65726f2061646472657373005f82015250565b5f611c72601f836113b1565b9150611c7d82611c3e565b602082019050919050565b5f6020820190508181035f830152611c9f81611c66565b9050919050565b7f45524332303a206275726e2066726f6d20746865207a65726f206164647265735f8201527f7300000000000000000000000000000000000000000000000000000000000000602082015250565b5f611d006021836113b1565b9150611d0b82611ca6565b604082019050919050565b5f6020820190508181035f830152611d2d81611cf4565b9050919050565b7f45524332303a206275726e20616d6f756e7420657863656564732062616c616e5f8201527f6365000000000000000000000000000000000000000000000000000000000000602082015250565b5f611d8e6022836113b1565b9150611d9982611d34565b604082019050919050565b5f6020820190508181035f830152611dbb81611d82565b905091905056fea2646970667358221220f1b5bd9bcfc825d0867c73cd613e56dd6e314ccf00dcaa9b1ef2b753cec0a75764736f6c634300081d0033",
}

// InjERC20ABI is the input ABI used to generate the binding from.
// Deprecated: Use InjERC20MetaData.ABI instead.
var InjERC20ABI = InjERC20MetaData.ABI

// InjERC20Bin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use InjERC20MetaData.Bin instead.
var InjERC20Bin = InjERC20MetaData.Bin

// DeployInjERC20 deploys a new Ethereum contract, binding an instance of InjERC20 to it.
func DeployInjERC20(auth *bind.TransactOpts, backend bind.ContractBackend, name_ string, symbol_ string, decimals_ uint8) (common.Address, *types.Transaction, *InjERC20, error) {
	parsed, err := InjERC20MetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(InjERC20Bin), backend, name_, symbol_, decimals_)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &InjERC20{InjERC20Caller: InjERC20Caller{contract: contract}, InjERC20Transactor: InjERC20Transactor{contract: contract}, InjERC20Filterer: InjERC20Filterer{contract: contract}}, nil
}

// InjERC20 is an auto generated Go binding around an Ethereum contract.
type InjERC20 struct {
	InjERC20Caller     // Read-only binding to the contract
	InjERC20Transactor // Write-only binding to the contract
	InjERC20Filterer   // Log filterer for contract events
}

// InjERC20Caller is an auto generated read-only Go binding around an Ethereum contract.
type InjERC20Caller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// InjERC20Transactor is an auto generated write-only Go binding around an Ethereum contract.
type InjERC20Transactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// InjERC20Filterer is an auto generated log filtering Go binding around an Ethereum contract events.
type InjERC20Filterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// InjERC20Session is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type InjERC20Session struct {
	Contract     *InjERC20         // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// InjERC20CallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type InjERC20CallerSession struct {
	Contract *InjERC20Caller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts   // Call options to use throughout this session
}

// InjERC20TransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type InjERC20TransactorSession struct {
	Contract     *InjERC20Transactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// InjERC20Raw is an auto generated low-level Go binding around an Ethereum contract.
type InjERC20Raw struct {
	Contract *InjERC20 // Generic contract binding to access the raw methods on
}

// InjERC20CallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type InjERC20CallerRaw struct {
	Contract *InjERC20Caller // Generic read-only contract binding to access the raw methods on
}

// InjERC20TransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type InjERC20TransactorRaw struct {
	Contract *InjERC20Transactor // Generic write-only contract binding to access the raw methods on
}

// NewInjERC20 creates a new instance of InjERC20, bound to a specific deployed contract.
func NewInjERC20(address common.Address, backend bind.ContractBackend) (*InjERC20, error) {
	contract, err := bindInjERC20(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &InjERC20{InjERC20Caller: InjERC20Caller{contract: contract}, InjERC20Transactor: InjERC20Transactor{contract: contract}, InjERC20Filterer: InjERC20Filterer{contract: contract}}, nil
}

// NewInjERC20Caller creates a new read-only instance of InjERC20, bound to a specific deployed contract.
func NewInjERC20Caller(address common.Address, caller bind.ContractCaller) (*InjERC20Caller, error) {
	contract, err := bindInjERC20(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &InjERC20Caller{contract: contract}, nil
}

// NewInjERC20Transactor creates a new write-only instance of InjERC20, bound to a specific deployed contract.
func NewInjERC20Transactor(address common.Address, transactor bind.ContractTransactor) (*InjERC20Transactor, error) {
	contract, err := bindInjERC20(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &InjERC20Transactor{contract: contract}, nil
}

// NewInjERC20Filterer creates a new log filterer instance of InjERC20, bound to a specific deployed contract.
func NewInjERC20Filterer(address common.Address, filterer bind.ContractFilterer) (*InjERC20Filterer, error) {
	contract, err := bindInjERC20(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &InjERC20Filterer{contract: contract}, nil
}

// bindInjERC20 binds a generic wrapper to an already deployed contract.
func bindInjERC20(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := InjERC20MetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_InjERC20 *InjERC20Raw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _InjERC20.Contract.InjERC20Caller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_InjERC20 *InjERC20Raw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _InjERC20.Contract.InjERC20Transactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_InjERC20 *InjERC20Raw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _InjERC20.Contract.InjERC20Transactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_InjERC20 *InjERC20CallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _InjERC20.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_InjERC20 *InjERC20TransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _InjERC20.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_InjERC20 *InjERC20TransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _InjERC20.Contract.contract.Transact(opts, method, params...)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_InjERC20 *InjERC20Caller) Allowance(opts *bind.CallOpts, owner common.Address, spender common.Address) (*big.Int, error) {
	var out []interface{}
	err := _InjERC20.contract.Call(opts, &out, "allowance", owner, spender)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_InjERC20 *InjERC20Session) Allowance(owner common.Address, spender common.Address) (*big.Int, error) {
	return _InjERC20.Contract.Allowance(&_InjERC20.CallOpts, owner, spender)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance(address owner, address spender) view returns(uint256)
func (_InjERC20 *InjERC20CallerSession) Allowance(owner common.Address, spender common.Address) (*big.Int, error) {
	return _InjERC20.Contract.Allowance(&_InjERC20.CallOpts, owner, spender)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_InjERC20 *InjERC20Caller) BalanceOf(opts *bind.CallOpts, account common.Address) (*big.Int, error) {
	var out []interface{}
	err := _InjERC20.contract.Call(opts, &out, "balanceOf", account)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_InjERC20 *InjERC20Session) BalanceOf(account common.Address) (*big.Int, error) {
	return _InjERC20.Contract.BalanceOf(&_InjERC20.CallOpts, account)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address account) view returns(uint256)
func (_InjERC20 *InjERC20CallerSession) BalanceOf(account common.Address) (*big.Int, error) {
	return _InjERC20.Contract.BalanceOf(&_InjERC20.CallOpts, account)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_InjERC20 *InjERC20Caller) Decimals(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _InjERC20.contract.Call(opts, &out, "decimals")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_InjERC20 *InjERC20Session) Decimals() (uint8, error) {
	return _InjERC20.Contract.Decimals(&_InjERC20.CallOpts)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() view returns(uint8)
func (_InjERC20 *InjERC20CallerSession) Decimals() (uint8, error) {
	return _InjERC20.Contract.Decimals(&_InjERC20.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_InjERC20 *InjERC20Caller) Name(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _InjERC20.contract.Call(opts, &out, "name")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_InjERC20 *InjERC20Session) Name() (string, error) {
	return _InjERC20.Contract.Name(&_InjERC20.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() view returns(string)
func (_InjERC20 *InjERC20CallerSession) Name() (string, error) {
	return _InjERC20.Contract.Name(&_InjERC20.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_InjERC20 *InjERC20Caller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _InjERC20.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_InjERC20 *InjERC20Session) Owner() (common.Address, error) {
	return _InjERC20.Contract.Owner(&_InjERC20.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_InjERC20 *InjERC20CallerSession) Owner() (common.Address, error) {
	return _InjERC20.Contract.Owner(&_InjERC20.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_InjERC20 *InjERC20Caller) Symbol(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _InjERC20.contract.Call(opts, &out, "symbol")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_InjERC20 *InjERC20Session) Symbol() (string, error) {
	return _InjERC20.Contract.Symbol(&_InjERC20.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() view returns(string)
func (_InjERC20 *InjERC20CallerSession) Symbol() (string, error) {
	return _InjERC20.Contract.Symbol(&_InjERC20.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_InjERC20 *InjERC20Caller) TotalSupply(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _InjERC20.contract.Call(opts, &out, "totalSupply")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_InjERC20 *InjERC20Session) TotalSupply() (*big.Int, error) {
	return _InjERC20.Contract.TotalSupply(&_InjERC20.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() view returns(uint256)
func (_InjERC20 *InjERC20CallerSession) TotalSupply() (*big.Int, error) {
	return _InjERC20.Contract.TotalSupply(&_InjERC20.CallOpts)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 amount) returns(bool)
func (_InjERC20 *InjERC20Transactor) Approve(opts *bind.TransactOpts, spender common.Address, amount *big.Int) (*types.Transaction, error) {
	return _InjERC20.contract.Transact(opts, "approve", spender, amount)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 amount) returns(bool)
func (_InjERC20 *InjERC20Session) Approve(spender common.Address, amount *big.Int) (*types.Transaction, error) {
	return _InjERC20.Contract.Approve(&_InjERC20.TransactOpts, spender, amount)
}

// Approve is a paid mutator transaction binding the contract method 0x095ea7b3.
//
// Solidity: function approve(address spender, uint256 amount) returns(bool)
func (_InjERC20 *InjERC20TransactorSession) Approve(spender common.Address, amount *big.Int) (*types.Transaction, error) {
	return _InjERC20.Contract.Approve(&_InjERC20.TransactOpts, spender, amount)
}

// Burn is a paid mutator transaction binding the contract method 0x9dc29fac.
//
// Solidity: function burn(address account, uint256 amount) returns()
func (_InjERC20 *InjERC20Transactor) Burn(opts *bind.TransactOpts, account common.Address, amount *big.Int) (*types.Transaction, error) {
	return _InjERC20.contract.Transact(opts, "burn", account, amount)
}

// Burn is a paid mutator transaction binding the contract method 0x9dc29fac.
//
// Solidity: function burn(address account, uint256 amount) returns()
func (_InjERC20 *InjERC20Session) Burn(account common.Address, amount *big.Int) (*types.Transaction, error) {
	return _InjERC20.Contract.Burn(&_InjERC20.TransactOpts, account, amount)
}

// Burn is a paid mutator transaction binding the contract method 0x9dc29fac.
//
// Solidity: function burn(address account, uint256 amount) returns()
func (_InjERC20 *InjERC20TransactorSession) Burn(account common.Address, amount *big.Int) (*types.Transaction, error) {
	return _InjERC20.Contract.Burn(&_InjERC20.TransactOpts, account, amount)
}

// DecreaseAllowance is a paid mutator transaction binding the contract method 0xa457c2d7.
//
// Solidity: function decreaseAllowance(address spender, uint256 subtractedValue) returns(bool)
func (_InjERC20 *InjERC20Transactor) DecreaseAllowance(opts *bind.TransactOpts, spender common.Address, subtractedValue *big.Int) (*types.Transaction, error) {
	return _InjERC20.contract.Transact(opts, "decreaseAllowance", spender, subtractedValue)
}

// DecreaseAllowance is a paid mutator transaction binding the contract method 0xa457c2d7.
//
// Solidity: function decreaseAllowance(address spender, uint256 subtractedValue) returns(bool)
func (_InjERC20 *InjERC20Session) DecreaseAllowance(spender common.Address, subtractedValue *big.Int) (*types.Transaction, error) {
	return _InjERC20.Contract.DecreaseAllowance(&_InjERC20.TransactOpts, spender, subtractedValue)
}

// DecreaseAllowance is a paid mutator transaction binding the contract method 0xa457c2d7.
//
// Solidity: function decreaseAllowance(address spender, uint256 subtractedValue) returns(bool)
func (_InjERC20 *InjERC20TransactorSession) DecreaseAllowance(spender common.Address, subtractedValue *big.Int) (*types.Transaction, error) {
	return _InjERC20.Contract.DecreaseAllowance(&_InjERC20.TransactOpts, spender, subtractedValue)
}

// IncreaseAllowance is a paid mutator transaction binding the contract method 0x39509351.
//
// Solidity: function increaseAllowance(address spender, uint256 addedValue) returns(bool)
func (_InjERC20 *InjERC20Transactor) IncreaseAllowance(opts *bind.TransactOpts, spender common.Address, addedValue *big.Int) (*types.Transaction, error) {
	return _InjERC20.contract.Transact(opts, "increaseAllowance", spender, addedValue)
}

// IncreaseAllowance is a paid mutator transaction binding the contract method 0x39509351.
//
// Solidity: function increaseAllowance(address spender, uint256 addedValue) returns(bool)
func (_InjERC20 *InjERC20Session) IncreaseAllowance(spender common.Address, addedValue *big.Int) (*types.Transaction, error) {
	return _InjERC20.Contract.IncreaseAllowance(&_InjERC20.TransactOpts, spender, addedValue)
}

// IncreaseAllowance is a paid mutator transaction binding the contract method 0x39509351.
//
// Solidity: function increaseAllowance(address spender, uint256 addedValue) returns(bool)
func (_InjERC20 *InjERC20TransactorSession) IncreaseAllowance(spender common.Address, addedValue *big.Int) (*types.Transaction, error) {
	return _InjERC20.Contract.IncreaseAllowance(&_InjERC20.TransactOpts, spender, addedValue)
}

// Mint is a paid mutator transaction binding the contract method 0x40c10f19.
//
// Solidity: function mint(address account, uint256 amount) returns()
func (_InjERC20 *InjERC20Transactor) Mint(opts *bind.TransactOpts, account common.Address, amount *big.Int) (*types.Transaction, error) {
	return _InjERC20.contract.Transact(opts, "mint", account, amount)
}

// Mint is a paid mutator transaction binding the contract method 0x40c10f19.
//
// Solidity: function mint(address account, uint256 amount) returns()
func (_InjERC20 *InjERC20Session) Mint(account common.Address, amount *big.Int) (*types.Transaction, error) {
	return _InjERC20.Contract.Mint(&_InjERC20.TransactOpts, account, amount)
}

// Mint is a paid mutator transaction binding the contract method 0x40c10f19.
//
// Solidity: function mint(address account, uint256 amount) returns()
func (_InjERC20 *InjERC20TransactorSession) Mint(account common.Address, amount *big.Int) (*types.Transaction, error) {
	return _InjERC20.Contract.Mint(&_InjERC20.TransactOpts, account, amount)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_InjERC20 *InjERC20Transactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _InjERC20.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_InjERC20 *InjERC20Session) RenounceOwnership() (*types.Transaction, error) {
	return _InjERC20.Contract.RenounceOwnership(&_InjERC20.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_InjERC20 *InjERC20TransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _InjERC20.Contract.RenounceOwnership(&_InjERC20.TransactOpts)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address recipient, uint256 amount) returns(bool)
func (_InjERC20 *InjERC20Transactor) Transfer(opts *bind.TransactOpts, recipient common.Address, amount *big.Int) (*types.Transaction, error) {
	return _InjERC20.contract.Transact(opts, "transfer", recipient, amount)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address recipient, uint256 amount) returns(bool)
func (_InjERC20 *InjERC20Session) Transfer(recipient common.Address, amount *big.Int) (*types.Transaction, error) {
	return _InjERC20.Contract.Transfer(&_InjERC20.TransactOpts, recipient, amount)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(address recipient, uint256 amount) returns(bool)
func (_InjERC20 *InjERC20TransactorSession) Transfer(recipient common.Address, amount *big.Int) (*types.Transaction, error) {
	return _InjERC20.Contract.Transfer(&_InjERC20.TransactOpts, recipient, amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address sender, address recipient, uint256 amount) returns(bool)
func (_InjERC20 *InjERC20Transactor) TransferFrom(opts *bind.TransactOpts, sender common.Address, recipient common.Address, amount *big.Int) (*types.Transaction, error) {
	return _InjERC20.contract.Transact(opts, "transferFrom", sender, recipient, amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address sender, address recipient, uint256 amount) returns(bool)
func (_InjERC20 *InjERC20Session) TransferFrom(sender common.Address, recipient common.Address, amount *big.Int) (*types.Transaction, error) {
	return _InjERC20.Contract.TransferFrom(&_InjERC20.TransactOpts, sender, recipient, amount)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(address sender, address recipient, uint256 amount) returns(bool)
func (_InjERC20 *InjERC20TransactorSession) TransferFrom(sender common.Address, recipient common.Address, amount *big.Int) (*types.Transaction, error) {
	return _InjERC20.Contract.TransferFrom(&_InjERC20.TransactOpts, sender, recipient, amount)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_InjERC20 *InjERC20Transactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _InjERC20.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_InjERC20 *InjERC20Session) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _InjERC20.Contract.TransferOwnership(&_InjERC20.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_InjERC20 *InjERC20TransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _InjERC20.Contract.TransferOwnership(&_InjERC20.TransactOpts, newOwner)
}

// InjERC20ApprovalIterator is returned from FilterApproval and is used to iterate over the raw logs and unpacked data for Approval events raised by the InjERC20 contract.
type InjERC20ApprovalIterator struct {
	Event *InjERC20Approval // Event containing the contract specifics and raw log

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
func (it *InjERC20ApprovalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(InjERC20Approval)
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
		it.Event = new(InjERC20Approval)
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
func (it *InjERC20ApprovalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *InjERC20ApprovalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// InjERC20Approval represents a Approval event raised by the InjERC20 contract.
type InjERC20Approval struct {
	Owner   common.Address
	Spender common.Address
	Value   *big.Int
	Raw     types.Log // Blockchain specific contextual infos
}

// FilterApproval is a free log retrieval operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_InjERC20 *InjERC20Filterer) FilterApproval(opts *bind.FilterOpts, owner []common.Address, spender []common.Address) (*InjERC20ApprovalIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _InjERC20.contract.FilterLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return &InjERC20ApprovalIterator{contract: _InjERC20.contract, event: "Approval", logs: logs, sub: sub}, nil
}

// WatchApproval is a free log subscription operation binding the contract event 0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925.
//
// Solidity: event Approval(address indexed owner, address indexed spender, uint256 value)
func (_InjERC20 *InjERC20Filterer) WatchApproval(opts *bind.WatchOpts, sink chan<- *InjERC20Approval, owner []common.Address, spender []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}
	var spenderRule []interface{}
	for _, spenderItem := range spender {
		spenderRule = append(spenderRule, spenderItem)
	}

	logs, sub, err := _InjERC20.contract.WatchLogs(opts, "Approval", ownerRule, spenderRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(InjERC20Approval)
				if err := _InjERC20.contract.UnpackLog(event, "Approval", log); err != nil {
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
func (_InjERC20 *InjERC20Filterer) ParseApproval(log types.Log) (*InjERC20Approval, error) {
	event := new(InjERC20Approval)
	if err := _InjERC20.contract.UnpackLog(event, "Approval", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// InjERC20OwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the InjERC20 contract.
type InjERC20OwnershipTransferredIterator struct {
	Event *InjERC20OwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *InjERC20OwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(InjERC20OwnershipTransferred)
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
		it.Event = new(InjERC20OwnershipTransferred)
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
func (it *InjERC20OwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *InjERC20OwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// InjERC20OwnershipTransferred represents a OwnershipTransferred event raised by the InjERC20 contract.
type InjERC20OwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_InjERC20 *InjERC20Filterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*InjERC20OwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _InjERC20.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &InjERC20OwnershipTransferredIterator{contract: _InjERC20.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_InjERC20 *InjERC20Filterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *InjERC20OwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _InjERC20.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(InjERC20OwnershipTransferred)
				if err := _InjERC20.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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

// ParseOwnershipTransferred is a log parse operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_InjERC20 *InjERC20Filterer) ParseOwnershipTransferred(log types.Log) (*InjERC20OwnershipTransferred, error) {
	event := new(InjERC20OwnershipTransferred)
	if err := _InjERC20.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// InjERC20TransferIterator is returned from FilterTransfer and is used to iterate over the raw logs and unpacked data for Transfer events raised by the InjERC20 contract.
type InjERC20TransferIterator struct {
	Event *InjERC20Transfer // Event containing the contract specifics and raw log

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
func (it *InjERC20TransferIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(InjERC20Transfer)
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
		it.Event = new(InjERC20Transfer)
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
func (it *InjERC20TransferIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *InjERC20TransferIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// InjERC20Transfer represents a Transfer event raised by the InjERC20 contract.
type InjERC20Transfer struct {
	From  common.Address
	To    common.Address
	Value *big.Int
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterTransfer is a free log retrieval operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_InjERC20 *InjERC20Filterer) FilterTransfer(opts *bind.FilterOpts, from []common.Address, to []common.Address) (*InjERC20TransferIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _InjERC20.contract.FilterLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &InjERC20TransferIterator{contract: _InjERC20.contract, event: "Transfer", logs: logs, sub: sub}, nil
}

// WatchTransfer is a free log subscription operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(address indexed from, address indexed to, uint256 value)
func (_InjERC20 *InjERC20Filterer) WatchTransfer(opts *bind.WatchOpts, sink chan<- *InjERC20Transfer, from []common.Address, to []common.Address) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _InjERC20.contract.WatchLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(InjERC20Transfer)
				if err := _InjERC20.contract.UnpackLog(event, "Transfer", log); err != nil {
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
func (_InjERC20 *InjERC20Filterer) ParseTransfer(log types.Log) (*InjERC20Transfer, error) {
	event := new(InjERC20Transfer)
	if err := _InjERC20.contract.UnpackLog(event, "Transfer", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
