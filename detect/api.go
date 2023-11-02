package detect

import (
	"context"
	"log"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

type DetectAPI struct {
	client      *ethclient.Client  // 以太坊客户端，用于查询区块链数据
	cancelFunc  context.CancelFunc // 取消函数
	ctx         context.Context
	initialized bool
}

func NewDetectAPI() *DetectAPI { //
	ctx, cancelFunc := context.WithCancel(context.Background())
	log.Println("Initializing DetectAPI")
	return &DetectAPI{
		cancelFunc:  cancelFunc,
		ctx:         ctx,
		initialized: false,
	}
}

func (api *DetectAPI) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "detect",
			Version:   "1.0",
			Service:   api,
			Public:    true,
		},
	}
}
