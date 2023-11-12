package detect

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// "Transfer" 事件的 ABI
const erc20TransferEventABI = `[{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"from","type":"address"},{"indexed":true,"internalType":"address","name":"to","type":"address"},{"indexed":false,"internalType":"uint256","name":"value","type":"uint256"}],"name":"Transfer","type":"event"}]`

// Uniswap "Swap" 事件的 ABI
const uniswapSwapEventABI = `[{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"sender","type":"address"},{"indexed":false,"internalType":"uint256","name":"amount0In","type":"uint256"},{"indexed":false,"internalType":"uint256","name":"amount1In","type":"uint256"},{"indexed":false,"internalType":"uint256","name":"amount0Out","type":"uint256"},{"indexed":false,"internalType":"uint256","name":"amount1Out","type":"uint256"},{"indexed":true,"internalType":"address","name":"to","type":"address"}],"name":"Swap","type":"event"}]`

var emptyAddress = common.HexToAddress("0x0000000000000000000000000000000000000000") //空地址，用于合约创建时的to地址

var swapEventABI abi.ABI
var transferEventABI abi.ABI
var TransferEventSignatureHash common.Hash
var SwapEventABIEventSignatureHash common.Hash

var signer types.Signer

// 本地运行
var clientURL = "ws://localhost:8546"

// 在开始时初始化
func init() {
	// var err error
	// swapEventABI, err = abi.JSON(strings.NewReader(uniswapSwapEventABI))
	// if err != nil {
	// 	log.Printf("Error parsing Uniswap Swap event ABI: %v", err)
	// 	panic("failed to parse Uniswap Swap event ABI11")
	// }

	// transferEventABI, err = abi.JSON(strings.NewReader(erc20TransferEventABI))
	// if err != nil {
	// 	log.Printf("Error parsing ERC20 Transfer event ABI: %v", err)
	// 	panic("failed to parse ERC20 Transfer event ABI")
	// }

	chainID := big.NewInt(1) // 主网
	//signer = types.NewEIP155Signer(chainID)
	signer = types.NewLondonSigner(chainID)
	TransferEventSignatureHash = getTransferEventSignatureHash()
	SwapEventABIEventSignatureHash = getSwapEventABIEventSignatureHash()
}

func getTransferEventSignatureHash() common.Hash {
	TransferEventSignatureHash := crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))
	return TransferEventSignatureHash
}
func getSwapEventABIEventSignatureHash() common.Hash {
	SwapEventABIEventSignatureHash := crypto.Keccak256Hash([]byte("Swap(address,uint256,uint256,uint256,uint256,address)"))
	return SwapEventABIEventSignatureHash
}
