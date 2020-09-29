package utils

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	bindconst "github.com/binance-chain/token-bind-tool/const"
	bindtypes "github.com/binance-chain/token-bind-tool/types"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

func Sleep(second int64) {
	fmt.Println(fmt.Sprintf("Sleep %d second", second))
	time.Sleep(time.Duration(second) * time.Second)
}

func GetTransactor(ethClient *ethclient.Client, keyStore *keystore.KeyStore, account accounts.Account, value *big.Int) *bind.TransactOpts {
	nonce, _ := ethClient.PendingNonceAt(context.Background(), account.Address)
	txOpts, _ := bind.NewKeyStoreTransactor(keyStore, account)
	txOpts.Nonce = big.NewInt(int64(nonce))
	txOpts.Value = value
	txOpts.GasLimit = bindconst.DefaultGasLimit
	txOpts.GasPrice = big.NewInt(bindconst.DefaultGasPrice)
	return txOpts
}

func GetCallOpts() *bind.CallOpts {
	callOpts := &bind.CallOpts{
		Pending: true,
		Context: context.Background(),
	}
	return callOpts
}

func DeployContract(ethClient *ethclient.Client, wallet *keystore.KeyStore, account accounts.Account, contractData hexutil.Bytes, chainId *big.Int) (common.Hash, error) {
	gasLimit := hexutil.Uint64(bindconst.DefaultGasLimit)
	nonce, err := ethClient.PendingNonceAt(context.Background(), account.Address)
	if err != nil {
		return common.Hash{}, err
	}
	gasPrice := hexutil.Big(*big.NewInt(bindconst.DefaultGasPrice))
	nonceUint64 := hexutil.Uint64(nonce)
	sendTxArgs := &bindtypes.SendTxArgs{
		From:     account.Address,
		Data:     &contractData,
		Gas:      &gasLimit,
		GasPrice: &gasPrice,
		Nonce:    &nonceUint64,
	}
	tx := toTransaction(sendTxArgs)

	signTx, err := wallet.SignTx(account, tx, chainId)
	if err != nil {
		return common.Hash{}, err
	}

	return signTx.Hash(), ethClient.SendTransaction(context.Background(), signTx)
}

func SendBNBToTempAccount(rpcClient *ethclient.Client, wallet accounts.Wallet, account accounts.Account, recipient common.Address, amount *big.Int, chainId *big.Int) error {
	gasLimit := hexutil.Uint64(bindconst.DefaultGasLimit)
	nonce, err := rpcClient.PendingNonceAt(context.Background(), account.Address)
	if err != nil {
		return err
	}
	gasPrice := hexutil.Big(*big.NewInt(bindconst.DefaultGasPrice))
	amountBig := hexutil.Big(*amount)
	nonceUint64 := hexutil.Uint64(nonce)
	sendTxArgs := &bindtypes.SendTxArgs{
		From:     account.Address,
		To:       &recipient,
		Gas:      &gasLimit,
		GasPrice: &gasPrice,
		Value:    &amountBig,
		Nonce:    &nonceUint64,
	}
	tx := toTransaction(sendTxArgs)

	signTx, err := wallet.SignTx(account, tx, chainId)
	if err != nil {
		return err
	}
	return rpcClient.SendTransaction(context.Background(), signTx)
}

func SendAllRestBNB(ethClient *ethclient.Client, wallet *keystore.KeyStore, account accounts.Account, recipient common.Address, chainId *big.Int) (common.Hash, error) {
	restBalance, _ := ethClient.BalanceAt(context.Background(), account.Address, nil)
	txFee := big.NewInt(1).Mul(big.NewInt(21000), big.NewInt(bindconst.DefaultGasPrice))
	if restBalance.Cmp(txFee) < 0 {
		return common.Hash{}, fmt.Errorf("rest BNB %s is less than minimum transfer transaction fee %s", restBalance.String(), txFee.String())
	}
	amount := big.NewInt(1).Sub(restBalance, txFee)
	fmt.Println(fmt.Sprintf("rest balance %s, transfer BNB tx fee %s, transfer %s back to %s", restBalance.String(), txFee.String(), amount.String(), recipient.String()))
	gasLimit := hexutil.Uint64(21000)
	nonce, err := ethClient.PendingNonceAt(context.Background(), account.Address)
	if err != nil {
		return common.Hash{}, err
	}
	gasPrice := hexutil.Big(*big.NewInt(bindconst.DefaultGasPrice))
	amountBig := hexutil.Big(*amount)
	nonceUint64 := hexutil.Uint64(nonce)
	sendTxArgs := &bindtypes.SendTxArgs{
		From:     account.Address,
		To:       &recipient,
		Gas:      &gasLimit,
		GasPrice: &gasPrice,
		Value:    &amountBig,
		Nonce:    &nonceUint64,
	}
	tx := toTransaction(sendTxArgs)

	signTx, err := wallet.SignTx(account, tx, chainId)
	if err != nil {
		return common.Hash{}, err
	}
	return signTx.Hash(), ethClient.SendTransaction(context.Background(), signTx)
}

func toTransaction(args *bindtypes.SendTxArgs) *types.Transaction {
	var input []byte
	if args.Input != nil {
		input = *args.Input
	} else if args.Data != nil {
		input = *args.Data
	}
	if args.To == nil {
		return types.NewContractCreation(uint64(*args.Nonce), (*big.Int)(args.Value), uint64(*args.Gas), (*big.Int)(args.GasPrice), input)
	}
	return types.NewTransaction(uint64(*args.Nonce), *args.To, (*big.Int)(args.Value), uint64(*args.Gas), (*big.Int)(args.GasPrice), input)
}

func PrintTxExplorerUrl(msg, txHash string, chainID *big.Int) {
	if chainID.Cmp(big.NewInt(bindconst.MainnetChainID)) == 0 {
		fmt.Println(fmt.Sprintf(bindconst.MainnetExplorerTxUrl, msg, txHash))
	} else {
		fmt.Println(fmt.Sprintf(bindconst.TestnetExplorerTxUrl, msg, txHash))
	}
}

func PrintAddrExplorerUrl(msg, address string, chainID *big.Int) {
	if chainID.Cmp(big.NewInt(bindconst.MainnetChainID)) == 0 {
		fmt.Println(fmt.Sprintf(bindconst.MainnetExplorerAddressUrl, msg, address))
	} else {
		fmt.Println(fmt.Sprintf(bindconst.TestnetExplorerAddressUrl, msg, address))
	}
}

func SendTransactionFromLedger(rpcClient *ethclient.Client, wallet accounts.Wallet, account accounts.Account, recipient common.Address, value *big.Int, data *hexutil.Bytes, chainId *big.Int) (*types.Transaction, error) {
	gasLimit := hexutil.Uint64(bindconst.DefaultGasLimit)
	nonce, err := rpcClient.PendingNonceAt(context.Background(), account.Address)
	if err != nil {
		return nil, err
	}
	gasPrice := hexutil.Big(*big.NewInt(bindconst.DefaultGasPrice))
	valueBig := hexutil.Big(*value)
	nonceUint64 := hexutil.Uint64(nonce)
	sendTxArgs := &bindtypes.SendTxArgs{
		From:     account.Address,
		To:       &recipient,
		Data:     data,
		Gas:      &gasLimit,
		GasPrice: &gasPrice,
		Value:    &valueBig,
		Nonce:    &nonceUint64,
	}
	tx := toTransaction(sendTxArgs)

	signTx, err := wallet.SignTx(account, tx, chainId)
	if err != nil {
		return nil, err
	}
	return signTx, rpcClient.SendTransaction(context.Background(), signTx)
}

func ValidateBSCAddr(addr string) error {
	if !strings.HasPrefix(addr, "0x") || len(addr) != 42 {
		return fmt.Errorf("invalid BEP20 owner account, expect bsc address, like 0x4E656459ed25bF986Eea1196Bc1B00665401645d")
	}
	return nil
}

func ConvertToBEP20Amount(amount *big.Int, decimals int64) *big.Int {
	if decimals >= 8 {
		precision := 1
		for idx := int64(0); idx < decimals - 8 ; idx ++ {
			precision *= 10
		}
		return big.NewInt(1).Mul(amount, big.NewInt(int64(precision)))
	} else {
		precision := 1
		for idx := int64(0); idx < 8 - decimals ; idx ++ {
			precision *= 10
		}
		return big.NewInt(1).Div(amount, big.NewInt(int64(precision)))
	}
}