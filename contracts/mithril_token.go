// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package contracts

import (
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// MithrilTokenABI is the input ABI used to generate the binding from.
const MithrilTokenABI = "[{\"constant\":true,\"inputs\":[],\"name\":\"name\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"totalSupply\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"from\",\"type\":\"address\"},{\"name\":\"to\",\"type\":\"address\"},{\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"transferFrom\",\"outputs\":[{\"name\":\"success\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"decimals\",\"outputs\":[{\"name\":\"\",\"type\":\"uint8\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_supply\",\"type\":\"uint256\"},{\"name\":\"_vault\",\"type\":\"address\"},{\"name\":\"_wallet\",\"type\":\"address\"}],\"name\":\"init\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"wallet\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"spender\",\"type\":\"address\"},{\"name\":\"value\",\"type\":\"uint256\"},{\"name\":\"extraData\",\"type\":\"bytes\"}],\"name\":\"approve\",\"outputs\":[{\"name\":\"success\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"address\"}],\"name\":\"balanceOf\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"symbol\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"to\",\"type\":\"address\"},{\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"transfer\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"address\"},{\"name\":\"\",\"type\":\"address\"}],\"name\":\"allowance\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"vault\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"payable\":true,\"stateMutability\":\"payable\",\"type\":\"fallback\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Transfer\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"from\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"value\",\"type\":\"uint256\"},{\"indexed\":true,\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"extraData\",\"type\":\"bytes\"}],\"name\":\"Approval\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"owner\",\"type\":\"address\"}],\"name\":\"OwnershipTransfered\",\"type\":\"event\"}]"

// MithrilTokenBin is the compiled bytecode used for deploying new contracts.
const MithrilTokenBin = `6060604052341561000f57600080fd5b60008054600160a060020a033316600160a060020a03199091161790556108da8061003b6000396000f3006060604052600436106100cf5763ffffffff7c010000000000000000000000000000000000000000000000000000000060003504166306fdde03811461010557806318160ddd1461018f57806323b872dd146101b4578063313ce567146101f05780634557b4bb14610219578063521eb273146102415780635c17f9f41461027057806370a08231146102d55780638da5cb5b146102f457806395d89b4114610307578063a9059cbb1461031a578063dd62ed3e1461033c578063f2fde38b14610361578063fbfa77cf14610380575b600554600160a060020a03163480156108fc0290604051600060405180830381858888f19350505050151561010357600080fd5b005b341561011057600080fd5b610118610393565b60405160208082528190810183818151815260200191508051906020019080838360005b8381101561015457808201518382015260200161013c565b50505050905090810190601f1680156101815780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b341561019a57600080fd5b6101a26103ca565b60405190815260200160405180910390f35b34156101bf57600080fd5b6101dc600160a060020a03600435811690602435166044356103d0565b604051901515815260200160405180910390f35b34156101fb57600080fd5b610203610478565b60405160ff909116815260200160405180910390f35b341561022457600080fd5b610103600435600160a060020a036024358116906044351661047d565b341561024c57600080fd5b610254610518565b604051600160a060020a03909116815260200160405180910390f35b341561027b57600080fd5b6101dc60048035600160a060020a03169060248035919060649060443590810190830135806020601f8201819004810201604051908101604052818152929190602084018383808284375094965061052795505050505050565b34156102e057600080fd5b6101a2600160a060020a0360043516610602565b34156102ff57600080fd5b610254610614565b341561031257600080fd5b610118610623565b341561032557600080fd5b610103600160a060020a036004351660243561065a565b341561034757600080fd5b6101a2600160a060020a0360043581169060243516610669565b341561036c57600080fd5b610103600160a060020a0360043516610686565b341561038b57600080fd5b6102546106ff565b60408051908101604052600d81527f4d69746872696c20546f6b656e00000000000000000000000000000000000000602082015281565b60015481565b600160a060020a0380841660009081526003602090815260408083203390941683529290529081205482111561040557600080fd5b600160a060020a038085166000908152600360209081526040808320339094168352929052205461043c908363ffffffff61070e16565b600160a060020a038086166000908152600360209081526040808320339094168352929052205561046e848484610720565b5060019392505050565b601281565b60005433600160a060020a0390811691161461049857600080fd5b600454600160a060020a0316156104ae57600080fd5b600160a060020a03821615156104c357600080fd5b60018390556004805473ffffffffffffffffffffffffffffffffffffffff19908116600160a060020a039485161791829055600580549091169284169290921790915516600090815260026020526040902055565b600554600160a060020a031681565b600160a060020a03338116600081815260036020908152604080832094881680845294909152808220869055909291907f4f2ccab30e52b306d3db2a1a0de078b7086c50ed233ea398995eaf7d64ac63be90869086905182815260406020820181815290820183818151815260200191508051906020019080838360005b838110156105bd5780820151838201526020016105a5565b50505050905090810190601f1680156105ea5780820380516001836020036101000a031916815260200191505b50935050505060405180910390a35060019392505050565b60026020526000908152604090205481565b600054600160a060020a031681565b60408051908101604052600481527f4d49544800000000000000000000000000000000000000000000000000000000602082015281565b610665338383610720565b5050565b600360209081526000928352604080842090915290825290205481565b60005433600160a060020a039081169116146106a157600080fd5b6000805473ffffffffffffffffffffffffffffffffffffffff1916600160a060020a038381169190911791829055167f9736aeb40a8f30a5c076a9897428fdf7ec0e909c96dce63533664c9b5c835da660405160405180910390a250565b600454600160a060020a031681565b60008282111561071a57fe5b50900390565b600160a060020a0383166000908152600260205260408120548290101561074657600080fd5b600160a060020a0383166000908152600260205260409020548281011161076c57600080fd5b600160a060020a0380841660009081526002602052604080822054928716825290205461079e9163ffffffff61089816565b600160a060020a0385166000908152600260205260409020549091506107ca908363ffffffff61070e16565b600160a060020a0380861660009081526002602052604080822093909355908516815220546107ff908363ffffffff61089816565b600160a060020a03808516600081815260026020526040908190209390935591908616907fddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef9085905190815260200160405180910390a3600160a060020a03808416600090815260026020526040808220549287168252902054829161088b919063ffffffff61089816565b1461089257fe5b50505050565b6000828201838110156108a757fe5b93925050505600a165627a7a723058200d7f5317d5e41aa77a059b8313413d5b9a537ebd88241e40f417377216a435430029`

// DeployMithrilToken deploys a new Ethereum contract, binding an instance of MithrilToken to it.
func DeployMithrilToken(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *MithrilToken, error) {
	parsed, err := abi.JSON(strings.NewReader(MithrilTokenABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(MithrilTokenBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &MithrilToken{MithrilTokenCaller: MithrilTokenCaller{contract: contract}, MithrilTokenTransactor: MithrilTokenTransactor{contract: contract}, MithrilTokenFilterer: MithrilTokenFilterer{contract: contract}}, nil
}

// MithrilToken is an auto generated Go binding around an Ethereum contract.
type MithrilToken struct {
	MithrilTokenCaller     // Read-only binding to the contract
	MithrilTokenTransactor // Write-only binding to the contract
	MithrilTokenFilterer   // Log filterer for contract events
}

// MithrilTokenCaller is an auto generated read-only Go binding around an Ethereum contract.
type MithrilTokenCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MithrilTokenTransactor is an auto generated write-only Go binding around an Ethereum contract.
type MithrilTokenTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MithrilTokenFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type MithrilTokenFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// MithrilTokenSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type MithrilTokenSession struct {
	Contract     *MithrilToken     // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// MithrilTokenCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type MithrilTokenCallerSession struct {
	Contract *MithrilTokenCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts       // Call options to use throughout this session
}

// MithrilTokenTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type MithrilTokenTransactorSession struct {
	Contract     *MithrilTokenTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts       // Transaction auth options to use throughout this session
}

// MithrilTokenRaw is an auto generated low-level Go binding around an Ethereum contract.
type MithrilTokenRaw struct {
	Contract *MithrilToken // Generic contract binding to access the raw methods on
}

// MithrilTokenCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type MithrilTokenCallerRaw struct {
	Contract *MithrilTokenCaller // Generic read-only contract binding to access the raw methods on
}

// MithrilTokenTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type MithrilTokenTransactorRaw struct {
	Contract *MithrilTokenTransactor // Generic write-only contract binding to access the raw methods on
}

// NewMithrilToken creates a new instance of MithrilToken, bound to a specific deployed contract.
func NewMithrilToken(address common.Address, backend bind.ContractBackend) (*MithrilToken, error) {
	contract, err := bindMithrilToken(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &MithrilToken{MithrilTokenCaller: MithrilTokenCaller{contract: contract}, MithrilTokenTransactor: MithrilTokenTransactor{contract: contract}, MithrilTokenFilterer: MithrilTokenFilterer{contract: contract}}, nil
}

// NewMithrilTokenCaller creates a new read-only instance of MithrilToken, bound to a specific deployed contract.
func NewMithrilTokenCaller(address common.Address, caller bind.ContractCaller) (*MithrilTokenCaller, error) {
	contract, err := bindMithrilToken(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &MithrilTokenCaller{contract: contract}, nil
}

// NewMithrilTokenTransactor creates a new write-only instance of MithrilToken, bound to a specific deployed contract.
func NewMithrilTokenTransactor(address common.Address, transactor bind.ContractTransactor) (*MithrilTokenTransactor, error) {
	contract, err := bindMithrilToken(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &MithrilTokenTransactor{contract: contract}, nil
}

// NewMithrilTokenFilterer creates a new log filterer instance of MithrilToken, bound to a specific deployed contract.
func NewMithrilTokenFilterer(address common.Address, filterer bind.ContractFilterer) (*MithrilTokenFilterer, error) {
	contract, err := bindMithrilToken(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &MithrilTokenFilterer{contract: contract}, nil
}

// bindMithrilToken binds a generic wrapper to an already deployed contract.
func bindMithrilToken(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(MithrilTokenABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MithrilToken *MithrilTokenRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _MithrilToken.Contract.MithrilTokenCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MithrilToken *MithrilTokenRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MithrilToken.Contract.MithrilTokenTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MithrilToken *MithrilTokenRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MithrilToken.Contract.MithrilTokenTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_MithrilToken *MithrilTokenCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _MithrilToken.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_MithrilToken *MithrilTokenTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _MithrilToken.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_MithrilToken *MithrilTokenTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _MithrilToken.Contract.contract.Transact(opts, method, params...)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance( address,  address) constant returns(uint256)
func (_MithrilToken *MithrilTokenCaller) Allowance(opts *bind.CallOpts, arg0 common.Address, arg1 common.Address) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _MithrilToken.contract.Call(opts, out, "allowance", arg0, arg1)
	return *ret0, err
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance( address,  address) constant returns(uint256)
func (_MithrilToken *MithrilTokenSession) Allowance(arg0 common.Address, arg1 common.Address) (*big.Int, error) {
	return _MithrilToken.Contract.Allowance(&_MithrilToken.CallOpts, arg0, arg1)
}

// Allowance is a free data retrieval call binding the contract method 0xdd62ed3e.
//
// Solidity: function allowance( address,  address) constant returns(uint256)
func (_MithrilToken *MithrilTokenCallerSession) Allowance(arg0 common.Address, arg1 common.Address) (*big.Int, error) {
	return _MithrilToken.Contract.Allowance(&_MithrilToken.CallOpts, arg0, arg1)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf( address) constant returns(uint256)
func (_MithrilToken *MithrilTokenCaller) BalanceOf(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _MithrilToken.contract.Call(opts, out, "balanceOf", arg0)
	return *ret0, err
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf( address) constant returns(uint256)
func (_MithrilToken *MithrilTokenSession) BalanceOf(arg0 common.Address) (*big.Int, error) {
	return _MithrilToken.Contract.BalanceOf(&_MithrilToken.CallOpts, arg0)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf( address) constant returns(uint256)
func (_MithrilToken *MithrilTokenCallerSession) BalanceOf(arg0 common.Address) (*big.Int, error) {
	return _MithrilToken.Contract.BalanceOf(&_MithrilToken.CallOpts, arg0)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() constant returns(uint8)
func (_MithrilToken *MithrilTokenCaller) Decimals(opts *bind.CallOpts) (uint8, error) {
	var (
		ret0 = new(uint8)
	)
	out := ret0
	err := _MithrilToken.contract.Call(opts, out, "decimals")
	return *ret0, err
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() constant returns(uint8)
func (_MithrilToken *MithrilTokenSession) Decimals() (uint8, error) {
	return _MithrilToken.Contract.Decimals(&_MithrilToken.CallOpts)
}

// Decimals is a free data retrieval call binding the contract method 0x313ce567.
//
// Solidity: function decimals() constant returns(uint8)
func (_MithrilToken *MithrilTokenCallerSession) Decimals() (uint8, error) {
	return _MithrilToken.Contract.Decimals(&_MithrilToken.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() constant returns(string)
func (_MithrilToken *MithrilTokenCaller) Name(opts *bind.CallOpts) (string, error) {
	var (
		ret0 = new(string)
	)
	out := ret0
	err := _MithrilToken.contract.Call(opts, out, "name")
	return *ret0, err
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() constant returns(string)
func (_MithrilToken *MithrilTokenSession) Name() (string, error) {
	return _MithrilToken.Contract.Name(&_MithrilToken.CallOpts)
}

// Name is a free data retrieval call binding the contract method 0x06fdde03.
//
// Solidity: function name() constant returns(string)
func (_MithrilToken *MithrilTokenCallerSession) Name() (string, error) {
	return _MithrilToken.Contract.Name(&_MithrilToken.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() constant returns(address)
func (_MithrilToken *MithrilTokenCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _MithrilToken.contract.Call(opts, out, "owner")
	return *ret0, err
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() constant returns(address)
func (_MithrilToken *MithrilTokenSession) Owner() (common.Address, error) {
	return _MithrilToken.Contract.Owner(&_MithrilToken.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() constant returns(address)
func (_MithrilToken *MithrilTokenCallerSession) Owner() (common.Address, error) {
	return _MithrilToken.Contract.Owner(&_MithrilToken.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() constant returns(string)
func (_MithrilToken *MithrilTokenCaller) Symbol(opts *bind.CallOpts) (string, error) {
	var (
		ret0 = new(string)
	)
	out := ret0
	err := _MithrilToken.contract.Call(opts, out, "symbol")
	return *ret0, err
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() constant returns(string)
func (_MithrilToken *MithrilTokenSession) Symbol() (string, error) {
	return _MithrilToken.Contract.Symbol(&_MithrilToken.CallOpts)
}

// Symbol is a free data retrieval call binding the contract method 0x95d89b41.
//
// Solidity: function symbol() constant returns(string)
func (_MithrilToken *MithrilTokenCallerSession) Symbol() (string, error) {
	return _MithrilToken.Contract.Symbol(&_MithrilToken.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() constant returns(uint256)
func (_MithrilToken *MithrilTokenCaller) TotalSupply(opts *bind.CallOpts) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _MithrilToken.contract.Call(opts, out, "totalSupply")
	return *ret0, err
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() constant returns(uint256)
func (_MithrilToken *MithrilTokenSession) TotalSupply() (*big.Int, error) {
	return _MithrilToken.Contract.TotalSupply(&_MithrilToken.CallOpts)
}

// TotalSupply is a free data retrieval call binding the contract method 0x18160ddd.
//
// Solidity: function totalSupply() constant returns(uint256)
func (_MithrilToken *MithrilTokenCallerSession) TotalSupply() (*big.Int, error) {
	return _MithrilToken.Contract.TotalSupply(&_MithrilToken.CallOpts)
}

// Vault is a free data retrieval call binding the contract method 0xfbfa77cf.
//
// Solidity: function vault() constant returns(address)
func (_MithrilToken *MithrilTokenCaller) Vault(opts *bind.CallOpts) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _MithrilToken.contract.Call(opts, out, "vault")
	return *ret0, err
}

// Vault is a free data retrieval call binding the contract method 0xfbfa77cf.
//
// Solidity: function vault() constant returns(address)
func (_MithrilToken *MithrilTokenSession) Vault() (common.Address, error) {
	return _MithrilToken.Contract.Vault(&_MithrilToken.CallOpts)
}

// Vault is a free data retrieval call binding the contract method 0xfbfa77cf.
//
// Solidity: function vault() constant returns(address)
func (_MithrilToken *MithrilTokenCallerSession) Vault() (common.Address, error) {
	return _MithrilToken.Contract.Vault(&_MithrilToken.CallOpts)
}

// Wallet is a free data retrieval call binding the contract method 0x521eb273.
//
// Solidity: function wallet() constant returns(address)
func (_MithrilToken *MithrilTokenCaller) Wallet(opts *bind.CallOpts) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _MithrilToken.contract.Call(opts, out, "wallet")
	return *ret0, err
}

// Wallet is a free data retrieval call binding the contract method 0x521eb273.
//
// Solidity: function wallet() constant returns(address)
func (_MithrilToken *MithrilTokenSession) Wallet() (common.Address, error) {
	return _MithrilToken.Contract.Wallet(&_MithrilToken.CallOpts)
}

// Wallet is a free data retrieval call binding the contract method 0x521eb273.
//
// Solidity: function wallet() constant returns(address)
func (_MithrilToken *MithrilTokenCallerSession) Wallet() (common.Address, error) {
	return _MithrilToken.Contract.Wallet(&_MithrilToken.CallOpts)
}

// Approve is a paid mutator transaction binding the contract method 0x5c17f9f4.
//
// Solidity: function approve(spender address, value uint256, extraData bytes) returns(success bool)
func (_MithrilToken *MithrilTokenTransactor) Approve(opts *bind.TransactOpts, spender common.Address, value *big.Int, extraData []byte) (*types.Transaction, error) {
	return _MithrilToken.contract.Transact(opts, "approve", spender, value, extraData)
}

// Approve is a paid mutator transaction binding the contract method 0x5c17f9f4.
//
// Solidity: function approve(spender address, value uint256, extraData bytes) returns(success bool)
func (_MithrilToken *MithrilTokenSession) Approve(spender common.Address, value *big.Int, extraData []byte) (*types.Transaction, error) {
	return _MithrilToken.Contract.Approve(&_MithrilToken.TransactOpts, spender, value, extraData)
}

// Approve is a paid mutator transaction binding the contract method 0x5c17f9f4.
//
// Solidity: function approve(spender address, value uint256, extraData bytes) returns(success bool)
func (_MithrilToken *MithrilTokenTransactorSession) Approve(spender common.Address, value *big.Int, extraData []byte) (*types.Transaction, error) {
	return _MithrilToken.Contract.Approve(&_MithrilToken.TransactOpts, spender, value, extraData)
}

// Init is a paid mutator transaction binding the contract method 0x4557b4bb.
//
// Solidity: function init(_supply uint256, _vault address, _wallet address) returns()
func (_MithrilToken *MithrilTokenTransactor) Init(opts *bind.TransactOpts, _supply *big.Int, _vault common.Address, _wallet common.Address) (*types.Transaction, error) {
	return _MithrilToken.contract.Transact(opts, "init", _supply, _vault, _wallet)
}

// Init is a paid mutator transaction binding the contract method 0x4557b4bb.
//
// Solidity: function init(_supply uint256, _vault address, _wallet address) returns()
func (_MithrilToken *MithrilTokenSession) Init(_supply *big.Int, _vault common.Address, _wallet common.Address) (*types.Transaction, error) {
	return _MithrilToken.Contract.Init(&_MithrilToken.TransactOpts, _supply, _vault, _wallet)
}

// Init is a paid mutator transaction binding the contract method 0x4557b4bb.
//
// Solidity: function init(_supply uint256, _vault address, _wallet address) returns()
func (_MithrilToken *MithrilTokenTransactorSession) Init(_supply *big.Int, _vault common.Address, _wallet common.Address) (*types.Transaction, error) {
	return _MithrilToken.Contract.Init(&_MithrilToken.TransactOpts, _supply, _vault, _wallet)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(to address, value uint256) returns()
func (_MithrilToken *MithrilTokenTransactor) Transfer(opts *bind.TransactOpts, to common.Address, value *big.Int) (*types.Transaction, error) {
	return _MithrilToken.contract.Transact(opts, "transfer", to, value)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(to address, value uint256) returns()
func (_MithrilToken *MithrilTokenSession) Transfer(to common.Address, value *big.Int) (*types.Transaction, error) {
	return _MithrilToken.Contract.Transfer(&_MithrilToken.TransactOpts, to, value)
}

// Transfer is a paid mutator transaction binding the contract method 0xa9059cbb.
//
// Solidity: function transfer(to address, value uint256) returns()
func (_MithrilToken *MithrilTokenTransactorSession) Transfer(to common.Address, value *big.Int) (*types.Transaction, error) {
	return _MithrilToken.Contract.Transfer(&_MithrilToken.TransactOpts, to, value)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(from address, to address, value uint256) returns(success bool)
func (_MithrilToken *MithrilTokenTransactor) TransferFrom(opts *bind.TransactOpts, from common.Address, to common.Address, value *big.Int) (*types.Transaction, error) {
	return _MithrilToken.contract.Transact(opts, "transferFrom", from, to, value)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(from address, to address, value uint256) returns(success bool)
func (_MithrilToken *MithrilTokenSession) TransferFrom(from common.Address, to common.Address, value *big.Int) (*types.Transaction, error) {
	return _MithrilToken.Contract.TransferFrom(&_MithrilToken.TransactOpts, from, to, value)
}

// TransferFrom is a paid mutator transaction binding the contract method 0x23b872dd.
//
// Solidity: function transferFrom(from address, to address, value uint256) returns(success bool)
func (_MithrilToken *MithrilTokenTransactorSession) TransferFrom(from common.Address, to common.Address, value *big.Int) (*types.Transaction, error) {
	return _MithrilToken.Contract.TransferFrom(&_MithrilToken.TransactOpts, from, to, value)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(newOwner address) returns()
func (_MithrilToken *MithrilTokenTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _MithrilToken.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(newOwner address) returns()
func (_MithrilToken *MithrilTokenSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _MithrilToken.Contract.TransferOwnership(&_MithrilToken.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(newOwner address) returns()
func (_MithrilToken *MithrilTokenTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _MithrilToken.Contract.TransferOwnership(&_MithrilToken.TransactOpts, newOwner)
}

// MithrilTokenApprovalIterator is returned from FilterApproval and is used to iterate over the raw logs and unpacked data for Approval events raised by the MithrilToken contract.
type MithrilTokenApprovalIterator struct {
	Event *MithrilTokenApproval // Event containing the contract specifics and raw log

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
func (it *MithrilTokenApprovalIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MithrilTokenApproval)
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
		it.Event = new(MithrilTokenApproval)
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
func (it *MithrilTokenApprovalIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MithrilTokenApprovalIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MithrilTokenApproval represents a Approval event raised by the MithrilToken contract.
type MithrilTokenApproval struct {
	From      common.Address
	Value     *big.Int
	To        common.Address
	ExtraData []byte
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterApproval is a free log retrieval operation binding the contract event 0x4f2ccab30e52b306d3db2a1a0de078b7086c50ed233ea398995eaf7d64ac63be.
//
// Solidity: event Approval(from indexed address, value uint256, to indexed address, extraData bytes)
func (_MithrilToken *MithrilTokenFilterer) FilterApproval(opts *bind.FilterOpts, from []common.Address, to []common.Address) (*MithrilTokenApprovalIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}

	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _MithrilToken.contract.FilterLogs(opts, "Approval", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &MithrilTokenApprovalIterator{contract: _MithrilToken.contract, event: "Approval", logs: logs, sub: sub}, nil
}

// WatchApproval is a free log subscription operation binding the contract event 0x4f2ccab30e52b306d3db2a1a0de078b7086c50ed233ea398995eaf7d64ac63be.
//
// Solidity: event Approval(from indexed address, value uint256, to indexed address, extraData bytes)
func (_MithrilToken *MithrilTokenFilterer) WatchApproval(opts *bind.WatchOpts, sink chan<- *MithrilTokenApproval, from []common.Address, to []common.Address) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}

	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _MithrilToken.contract.WatchLogs(opts, "Approval", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MithrilTokenApproval)
				if err := _MithrilToken.contract.UnpackLog(event, "Approval", log); err != nil {
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

// MithrilTokenOwnershipTransferedIterator is returned from FilterOwnershipTransfered and is used to iterate over the raw logs and unpacked data for OwnershipTransfered events raised by the MithrilToken contract.
type MithrilTokenOwnershipTransferedIterator struct {
	Event *MithrilTokenOwnershipTransfered // Event containing the contract specifics and raw log

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
func (it *MithrilTokenOwnershipTransferedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MithrilTokenOwnershipTransfered)
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
		it.Event = new(MithrilTokenOwnershipTransfered)
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
func (it *MithrilTokenOwnershipTransferedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MithrilTokenOwnershipTransferedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MithrilTokenOwnershipTransfered represents a OwnershipTransfered event raised by the MithrilToken contract.
type MithrilTokenOwnershipTransfered struct {
	Owner common.Address
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransfered is a free log retrieval operation binding the contract event 0x9736aeb40a8f30a5c076a9897428fdf7ec0e909c96dce63533664c9b5c835da6.
//
// Solidity: event OwnershipTransfered(owner indexed address)
func (_MithrilToken *MithrilTokenFilterer) FilterOwnershipTransfered(opts *bind.FilterOpts, owner []common.Address) (*MithrilTokenOwnershipTransferedIterator, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _MithrilToken.contract.FilterLogs(opts, "OwnershipTransfered", ownerRule)
	if err != nil {
		return nil, err
	}
	return &MithrilTokenOwnershipTransferedIterator{contract: _MithrilToken.contract, event: "OwnershipTransfered", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransfered is a free log subscription operation binding the contract event 0x9736aeb40a8f30a5c076a9897428fdf7ec0e909c96dce63533664c9b5c835da6.
//
// Solidity: event OwnershipTransfered(owner indexed address)
func (_MithrilToken *MithrilTokenFilterer) WatchOwnershipTransfered(opts *bind.WatchOpts, sink chan<- *MithrilTokenOwnershipTransfered, owner []common.Address) (event.Subscription, error) {

	var ownerRule []interface{}
	for _, ownerItem := range owner {
		ownerRule = append(ownerRule, ownerItem)
	}

	logs, sub, err := _MithrilToken.contract.WatchLogs(opts, "OwnershipTransfered", ownerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MithrilTokenOwnershipTransfered)
				if err := _MithrilToken.contract.UnpackLog(event, "OwnershipTransfered", log); err != nil {
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

// MithrilTokenTransferIterator is returned from FilterTransfer and is used to iterate over the raw logs and unpacked data for Transfer events raised by the MithrilToken contract.
type MithrilTokenTransferIterator struct {
	Event *MithrilTokenTransfer // Event containing the contract specifics and raw log

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
func (it *MithrilTokenTransferIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(MithrilTokenTransfer)
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
		it.Event = new(MithrilTokenTransfer)
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
func (it *MithrilTokenTransferIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *MithrilTokenTransferIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// MithrilTokenTransfer represents a Transfer event raised by the MithrilToken contract.
type MithrilTokenTransfer struct {
	From  common.Address
	To    common.Address
	Value *big.Int
	Raw   types.Log // Blockchain specific contextual infos
}

// FilterTransfer is a free log retrieval operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(from indexed address, to indexed address, value uint256)
func (_MithrilToken *MithrilTokenFilterer) FilterTransfer(opts *bind.FilterOpts, from []common.Address, to []common.Address) (*MithrilTokenTransferIterator, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _MithrilToken.contract.FilterLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return &MithrilTokenTransferIterator{contract: _MithrilToken.contract, event: "Transfer", logs: logs, sub: sub}, nil
}

// WatchTransfer is a free log subscription operation binding the contract event 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef.
//
// Solidity: event Transfer(from indexed address, to indexed address, value uint256)
func (_MithrilToken *MithrilTokenFilterer) WatchTransfer(opts *bind.WatchOpts, sink chan<- *MithrilTokenTransfer, from []common.Address, to []common.Address) (event.Subscription, error) {

	var fromRule []interface{}
	for _, fromItem := range from {
		fromRule = append(fromRule, fromItem)
	}
	var toRule []interface{}
	for _, toItem := range to {
		toRule = append(toRule, toItem)
	}

	logs, sub, err := _MithrilToken.contract.WatchLogs(opts, "Transfer", fromRule, toRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(MithrilTokenTransfer)
				if err := _MithrilToken.contract.UnpackLog(event, "Transfer", log); err != nil {
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
