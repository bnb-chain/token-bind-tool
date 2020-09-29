package command

import (
	"context"
	"math/big"
	"testing"

	constValue "github.com/binance-chain/token-bind-tool/const"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

func TestBlockGasLimit(t *testing.T) {
	rpcClient, err := rpc.DialContext(context.Background(), constValue.MainnnetRPC)
	require.NoError(t, err)
	ethClient := ethclient.NewClient(rpcClient)

	block, err := ethClient.BlockByNumber(context.Background(), big.NewInt(476174))
	require.NoError(t, err)

	gasSum := uint64(0)
	for _, tx := range block.Transactions() {
		t.Log(tx.Hash().String())
		t.Log(tx.Gas())
		gasSum += tx.Gas()
		t.Log(gasSum)
	}
}

func TestBlockGasLimit1(t *testing.T) {
	rpcClient, err := rpc.DialContext(context.Background(), constValue.MainnnetRPC)
	require.NoError(t, err)
	ethClient := ethclient.NewClient(rpcClient)

	block, err := ethClient.BlockByNumber(context.Background(), big.NewInt(476574))
	require.NoError(t, err)

	gasSum := uint64(0)
	for _, tx := range block.Transactions() {
		t.Log(tx.Hash().String())
		t.Log(tx.Gas())
		gasSum += tx.Gas()
		t.Log(gasSum)
	}
}

func TestBlockGasLimit2(t *testing.T) {
	rpcClient, err := rpc.DialContext(context.Background(), constValue.MainnnetRPC)
	require.NoError(t, err)
	ethClient := ethclient.NewClient(rpcClient)

	block, err := ethClient.BlockByNumber(context.Background(), big.NewInt(476574))
	require.NoError(t, err)

	gasSum := uint64(0)
	for _, tx := range block.Transactions() {
		txRecipient, err := ethClient.TransactionReceipt(context.Background(), tx.Hash())
		require.NoError(t, err)
		t.Log(tx.Hash().String())
		t.Log(tx.Gas())
		t.Log(txRecipient.GasUsed)
		t.Log(float64(txRecipient.GasUsed)/float64(tx.Gas()))
		gasSum += txRecipient.GasUsed
		t.Log(gasSum)
		t.Log("---------------")
	}
}