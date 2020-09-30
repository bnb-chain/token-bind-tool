package _const

import (
	"github.com/ethereum/go-ethereum/common"
)

const (
	Passwd              = "12345678"

	NetworkType         = "network-type"
	KeystorePath        = "keystore-path"
	ConfigPath          = "config-path"
	Operation           = "operation"
	BEP20ContractAddr   = "bep20-contract-addr"
	BEP20Owner          = "bep20-owner"
	BEP2Symbol          = "bep2-symbol"
	Recipient           = "recipient"
	PeggyAmount         = "peggy-amount"
	LedgerAccountIndex  = "ledger-account-index"

	Mainnet = "mainnet"
	TestNet = "testnet"

	BindKeystore = "bind_keystore"

	TestnetRPC     = "https://data-seed-prebsc-1-s1.binance.org:8545"
	TestnetChainID = 97

	MainnnetRPC    = "https://bsc-dataseed1.binance.org:443"
	MainnetChainID = 56

	DefaultGasPrice = 20000000000
	DefaultGasLimit = 4700000

	MainnetExplorerTxUrl = "%s: https://bscscan.com/tx/%s"
	TestnetExplorerTxUrl = "%s: https://testnet.bscscan.com/tx/%s"

	MainnetExplorerAddressUrl = "%s: https://bscscan.com/address/%s"
	TestnetExplorerAddressUrl = "%s: https://testnet.bscscan.com/address/%s"

	BSCAddrLength = 42
)

var (
	TokenHubContractAddr     = common.HexToAddress("0x0000000000000000000000000000000000001004")
	TokenManagerContractAddr = common.HexToAddress("0x0000000000000000000000000000000000001008")
	SuperOwnerAddr1 = common.HexToAddress("0xF68a4b64162906efF0fF6aE34E2bB1Cd42FEf62d")
	SuperOwnerAddr2 = common.HexToAddress("0xd2f93484f2d319194cba95c5171b18c1d8cfd6c4")
	WithdrawAddr = common.HexToAddress("0x631fc1ea2270e98fbd9d92658ece0f5a269aa161")
)
