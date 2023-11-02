package detect

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

func (api *DetectAPI) ensureClientInitialized() error {
	if !api.initialized {
		client, err := ethclient.Dial(clientURL)
		if err != nil {
			return err
		}
		api.client = client
		api.initialized = true
	}
	return nil
}

func (api *DetectAPI) StopDetect() (string, error) {
	if api.cancelFunc != nil {
		api.cancelFunc()
	}
	return "stoping detect", nil
}

// 用于打印log
type TransactionDetails struct {
	Hash             common.Hash
	Sender           common.Address
	To               common.Address
	ValueIn          *big.Int
	ValueOut         *big.Int
	Gas              uint64
	TokenPairAddress common.Address
}

// 检测最新区块
func (api *DetectAPI) DetectNewBlockSandwichAttack(ctx context.Context) error {
	log.Printf("DetectNewBlockSandwichAttack method called")
	if err := api.ensureClientInitialized(); err != nil {
		log.Printf("Failed to ensureClientInitialized: %v", err)
		return err
	}
	// 订阅新区块头
	headers := make(chan *types.Header)
	sub, err := api.client.SubscribeNewHead(api.ctx, headers)
	if err != nil {
		log.Printf("Failed to subscribe to new headers: %v", err)
		return err
	}
	// 准备日志文件
	logFile, err := api.createLogFile()
	if err != nil {
		log.Printf("Failed to create log file: %v", err)
		return err
	}
	defer logFile.Close()

	log.Println("Started detecting sandwich attacks on new blocks...")

	// 循环等待新区块的到来
	for {
		select {
		case <-api.ctx.Done(): // 检查是否收到了取消请求
			log.Println("Stopping detection...")
			return api.ctx.Err() // 返回取消的原因
		case err := <-sub.Err():
			log.Printf("Subscription error: %v,try again in 2 seconds", err)
			time.Sleep(2 * time.Second)
			continue
		case header := <-headers:
			blockNumber := header.Number.Uint64()
			log.Printf("Detected new block: %d", blockNumber)
			latestBlockNumber := blockNumber
			err := api.detectLogic(ctx, int64(latestBlockNumber), int64(latestBlockNumber), logFile)
			log.Printf("detectLogic error: %v", err)
			return err
		}
	}
}

// 检测现有区块
func (api *DetectAPI) DetectCurrentBLockSandwichAttack(ctx context.Context, startBlockNumber int64, endBlockNumber int64) error {
	log.Printf("DetectCurrentBLockSandwichAttack method called")
	if err := api.ensureClientInitialized(); err != nil {
		log.Printf("Failed to ensureClientInitialized: %v", err)
		return err
	}
	//获取最新区块
	header, err := api.client.HeaderByNumber(api.ctx, nil)
	if err != nil {
		return err
	}
	latestBlockNumber := header.Number.Int64()

	//检查输入是否合法
	if startBlockNumber > endBlockNumber || endBlockNumber > latestBlockNumber || startBlockNumber < 0 || endBlockNumber < 0 {
		err = fmt.Errorf("illegal block number: startBlockNumber (%d) or endBlockNumber (%d) is invalid", startBlockNumber, endBlockNumber)
		log.Print(err)
		return err
	}

	// 准备日志文件
	logFile, err := api.createLogFile()
	if err != nil {
		log.Printf("Failed to create log file: %v", err)
		return err
	}
	defer logFile.Close()

	err = api.detectLogic(ctx, startBlockNumber, endBlockNumber, logFile)
	log.Printf("detectLogic error: %v", err)
	return err
}

// 检测逻辑
func (api *DetectAPI) detectLogic(ctx context.Context, startBlockNumber int64, endBlockNumber int64, logFile *os.File) error {

	for i := startBlockNumber; i <= endBlockNumber; i++ {
		select {
		case <-api.ctx.Done(): // 检查是否收到了取消请求
			log.Println("Stopping detection...")
			return api.ctx.Err() // 返回取消的原因
		default:
			block, err := api.client.BlockByNumber(ctx, big.NewInt(i))
			header := block.Header()
			if err != nil {
				log.Printf("Error retrieving block %d: %s\n", i, err)
				return err
			}
			transactions := block.Transactions()
			numTxs := len(block.Transactions())

			for index1 := 0; index1 < numTxs; index1++ {
				tx1 := transactions[index1]
				if tx1.Data() == nil {
					log.Printf("only normal transaction: %v", err)
					continue
				}
				hash1, sender1, to1, tokenPairAddress1, valueIn1, valueOut1, gas1, err := parseTransaction(ctx, api.client, signer, tx1, header)
				if err != nil {
					log.Printf("parseTransaction err: %v", err)
					continue // 如果解析失败，可能不是swap，继续下一个
				}

				for index2 := index1 + 1; index2 < numTxs; index2++ {
					tx2 := transactions[index2]
					if tx2.Data() == nil {
						log.Printf("only normal transaction: %v", err)
						continue
					}
					hash2, sender2, to2, tokenPairAddress2, valueIn2, valueOut2, gas2, err := parseTransaction(ctx, api.client, signer, tx2, header)
					if err != nil {
						log.Printf("parseTransaction err: %v", err)
						continue
					}

					if hash1 != hash2 || tokenPairAddress1 == tokenPairAddress2 || gas1 > gas2 {
						for index3 := index2 + 1; index3 < numTxs; index3++ {
							tx3 := transactions[index3]
							if tx3.Data() == nil {
								log.Printf("only normal transaction: %v", err)
								continue
							}
							hash3, sender3, to3, tokenPairAddress3, valueIn3, valueOut3, gas3, err := parseTransaction(ctx, api.client, signer, tx3, header)
							if err != nil {
								log.Printf("parseTransaction err: %v", err)
								continue
							}
							if hash1 != hash3 || sender1 == sender3 || to1 == to3 || tokenPairAddress1 == tokenPairAddress3 || (valueIn3.Cmp(valueOut1.Mul(valueOut1, big.NewInt(97)).Div(valueOut1, big.NewInt(100))) >= 0 && valueIn3.Cmp(valueOut1.Mul(valueOut1, big.NewInt(103)).Div(valueOut1, big.NewInt(100))) <= 0) || gas3 < gas2 {

								txDetails1 := TransactionDetails{
									Hash:             hash1,
									Sender:           sender1,
									To:               to1,
									TokenPairAddress: tokenPairAddress1,
									ValueIn:          valueIn1,
									ValueOut:         valueOut1,
									Gas:              gas1,
								}
								txDetails2 := TransactionDetails{
									Hash:             hash2,
									Sender:           sender2,
									To:               to2,
									TokenPairAddress: tokenPairAddress2,
									ValueIn:          valueIn2,
									ValueOut:         valueOut2,
									Gas:              gas2,
								}
								txDetails3 := TransactionDetails{
									Hash:             hash3,
									Sender:           sender3,
									To:               to3,
									TokenPairAddress: tokenPairAddress3,
									ValueIn:          valueIn3,
									ValueOut:         valueOut3,
									Gas:              gas3,
								}
								blockNumber := i
								logSandwichAttack(logFile, txDetails1, txDetails2, txDetails3, blockNumber)
							}
						}
					}
				}
			}
		}

	}
	return nil
}

// 解析 ERC20 转账，返回 token 地址，转账金额和使用的 gas
func parseTransaction(ctx context.Context, client *ethclient.Client, signer types.Signer, tx *types.Transaction, header *types.Header) (hash common.Hash, sender common.Address, to common.Address, tokenPairAddress common.Address, valueIn, valueOut *big.Int, gas uint64, err error) {

	// 获取交易哈希
	hash = tx.Hash()
	// 获取发送者地址，通过签名者
	sender, err = types.Sender(signer, tx)
	if err != nil {
		return
	}

	// 获取接收者地址 (contact created时to是一个指针)
	to = emptyAddress
	if tx.To() != nil {
		to = *tx.To()
	}

	txReceipt, err := client.TransactionReceipt(ctx, hash)
	if err != nil {
		log.Printf("Failed to retrieve transaction receipt: %v", err)
		return
	}
	var gasUsed uint64
	tokenPairAddress, valueIn, valueOut, gasUsed, err = parseUniswapSwapLogs(txReceipt)
	if err != nil {
		log.Printf("Failed parseUniswapSwapLogs: %s", err)
	}
	gasActuallyUsed := CalculateTransactionCost(gasUsed, tx, header)
	gas = gasActuallyUsed
	return
}

// 解析收据
func parseUniswapSwapLogs(txReceipt *types.Receipt) (tokenPairAddress common.Address, valueIn, valueOut *big.Int, gasUsed uint64, err error) {
	foundSwap := false

	gasUsed = txReceipt.GasUsed
	for _, log := range txReceipt.Logs {
		if log.Topics[0] == swapEventABI.Events["Swap"].ID {
			foundSwap = true
			var swapEvent struct {
				sender     common.Address
				amount0In  big.Int
				amount1In  big.Int
				amount0Out big.Int
				amount1Out big.Int
				to         common.Address
			}
			err = swapEventABI.UnpackIntoInterface(&swapEvent, "Swap", log.Data)
			if err != nil {
				return // 如果在解析过程中遇到错误，直接返回
			}
			if swapEvent.amount0In.Cmp(big.NewInt(0)) == 0 {
				valueIn = &swapEvent.amount1In
				valueOut = &swapEvent.amount0Out
			} else {
				valueIn = &swapEvent.amount0In
				valueOut = &swapEvent.amount1Out
			}
			break // 如果找到 "Swap" 事件，不再继续搜索
		}
	}
	// 如果没有找到 "Swap" 事件，直接返回
	if !foundSwap {
		log.Println("no Swap event found in transaction logs")
		return
	}
	// 如果找到Swap事件，解析Transfer事件
	for _, log := range txReceipt.Logs {
		if log.Topics[0] == transferEventABI.Events["Transfer"].ID {
			// 解析 "Transfer" 事件
			var transferEvent struct {
				From  common.Address
				To    common.Address
				Value *big.Int
			}

			err = transferEventABI.UnpackIntoInterface(&transferEvent, "Transfer", log.Data)
			if err != nil {
				return // 如果在解析过程中遇到错误，直接返回
			}
			tokenPairAddress = transferEvent.To
			transferValue := transferEvent.Value
			if transferValue == valueIn || transferValue == valueOut {
				break
			}
		}
	}

	return
}

func CalculateTransactionCost(gasUsed uint64, tx *types.Transaction, header *types.Header) uint64 {
	// 对于 EIP-1559 之后的交易
	if tx.Type() == types.DynamicFeeTxType {
		baseFee := header.BaseFee
		tip := new(big.Int).Sub(tx.GasFeeCap(), baseFee)
		if tip.Cmp(tx.GasTipCap()) > 0 {
			tip = tx.GasTipCap()
		}
		totalPerGas := new(big.Int).Add(baseFee, tip)
		cost := new(big.Int).Mul(totalPerGas, new(big.Int).SetUint64(gasUsed))
		return cost.Uint64()
	} else {
		// 对于 EIP-1559 之前的交易
		cost := new(big.Int).Mul(tx.GasPrice(), new(big.Int).SetUint64(gasUsed))
		return cost.Uint64()
	}
}

func (api *DetectAPI) createLogFile() (*os.File, error) {
	currentTime := time.Now()
	logFileName := fmt.Sprintf("%s/sandwich_attack_%s.log", "/home/ldz/workspace/go-ethereum-release-1.13/sanwichlog", currentTime.Format("2006_01_02_15:04:05")) //生产环境 /gethData/geth time.Now().Unix()
	logFile, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return logFile, nil
}

func logSandwichAttack(logFile io.Writer, txDetails1, txDetails2, txDetails3 TransactionDetails, blockNumber int64) {

	details := fmt.Sprintf(
		"Detected sandwich attack in block %d:\n"+
			"tx1: %s gas1: %d sender1: %s to1: %s valuein1: %s valueout1: %s\n"+
			"tx2: %s gas2: %d sender2: %s to2: %s valuein2: %s valueout2: %s\n"+
			"tx3: %s gas3: %d sender3: %s to3: %s valuein3: %s valueout3: %s\n"+
			"Trading pair address: %s\n",
		blockNumber,
		txDetails1.Hash.Hex(), txDetails1.Gas, txDetails1.Sender.Hex(), txDetails1.To.Hex(), txDetails1.ValueIn.String(), txDetails1.ValueOut.String(),
		txDetails2.Hash.Hex(), txDetails2.Gas, txDetails2.Sender.Hex(), txDetails2.To.Hex(), txDetails2.ValueIn.String(), txDetails2.ValueOut.String(),
		txDetails3.Hash.Hex(), txDetails3.Gas, txDetails3.Sender.Hex(), txDetails3.To.Hex(), txDetails3.ValueIn.String(), txDetails3.ValueOut.String(),
		txDetails1.TokenPairAddress.Hex(),
	)

	_, err := logFile.Write([]byte(details))
	if err != nil {
		log.Printf("Failed to write error log: %v", err)
	}
}
