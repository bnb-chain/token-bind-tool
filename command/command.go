package command

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/binance-chain/token-bind-tool/contracts/erc721"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/accounts/usbwallet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/binance-chain/token-bind-tool/config"
	constValue "github.com/binance-chain/token-bind-tool/const"
	"github.com/binance-chain/token-bind-tool/contracts/bep20"
	"github.com/binance-chain/token-bind-tool/contracts/ownable"
	"github.com/binance-chain/token-bind-tool/contracts/tokenhub"
	tokenmanager "github.com/binance-chain/token-bind-tool/contracts/tokenmanger"
	"github.com/binance-chain/token-bind-tool/utils"
)

var (
	ledgerBasePath = accounts.DerivationPath{0x80000000 + 44, 0x80000000 + 60, 0x80000000 + 0, 0, 0}
)

func generateOrGetTempAccount(keystorePath string, chainId *big.Int) (*keystore.KeyStore, accounts.Account, error) {
	path, err := os.Getwd()
	if err != nil {
		return nil, accounts.Account{}, err
	}
	keyStore := keystore.NewKeyStore(keystorePath, keystore.StandardScryptN, keystore.StandardScryptP)
	if len(keyStore.Accounts()) == 0 {
		newAccount, err := keyStore.NewAccount(constValue.Passwd)
		if err != nil {
			return nil, accounts.Account{}, err
		}
		err = keyStore.Unlock(newAccount, constValue.Passwd)
		if err != nil {
			return nil, accounts.Account{}, err
		}
		return keyStore, newAccount, nil
	} else if len(keyStore.Accounts()) == 1 {
		accountList := keyStore.Accounts()
		if len(accountList) != 1 {
			return nil, accounts.Account{}, err
		}
		account := accountList[0]
		err = keyStore.Unlock(account, constValue.Passwd)
		if err != nil {
			return nil, accounts.Account{}, err
		}
		return keyStore, account, nil
	} else {
		return nil, accounts.Account{}, fmt.Errorf("expect only one or zero keystore file in %s", filepath.Join(path, constValue.BindKeystore))
	}
}

func openLedger(index uint32) (accounts.Wallet, accounts.Account, error) {
	ledgerHub, err := usbwallet.NewLedgerHub()
	if err != nil {
		return nil, accounts.Account{}, fmt.Errorf("failed to start Ledger hub, disabling: %v", err)
	}
	wallets := ledgerHub.Wallets()
	if len(wallets) == 0 {
		return nil, accounts.Account{}, fmt.Errorf("empty ledger wallet")
	}
	wallet := wallets[0]
	err = wallet.Close()
	if err != nil {
		fmt.Println(err.Error())
	}

	err = wallet.Open("")
	if err != nil {
		return nil, accounts.Account{}, fmt.Errorf("failed to start Ledger hub, disabling: %v", err)
	}

	walletStatus, err := wallet.Status()
	if err != nil {
		return nil, accounts.Account{}, fmt.Errorf("failed to start Ledger hub, disabling: %v", err)
	}
	fmt.Println(walletStatus)
	//fmt.Println(wallet.URL())

	ledgerPath := make(accounts.DerivationPath, len(ledgerBasePath))
	copy(ledgerPath, ledgerBasePath)
	ledgerPath[2] = ledgerPath[2] + index
	ledgerAccount, err := wallet.Derive(ledgerPath, true)
	if err != nil {
		return nil, accounts.Account{}, fmt.Errorf("failed to derive account from ledger: %v", err)
	}
	return wallet, ledgerAccount, nil
}

func getEnv() (*ethclient.Client, *big.Int, error) {
	networkType := viper.GetString(constValue.NetworkType)
	if networkType != constValue.TestNet && networkType != constValue.Mainnet {
		return nil, nil, fmt.Errorf("unsupported network type")
	}
	var rpcClient *rpc.Client
	var err error
	var chainId *big.Int
	if networkType == constValue.Mainnet {
		rpcClient, err = rpc.DialContext(context.Background(), constValue.MainnnetRPC)
		if err != nil {
			return nil, chainId, err
		}
		chainId = big.NewInt(constValue.MainnetChainID)
	} else {
		rpcClient, err = rpc.DialContext(context.Background(), constValue.TestnetRPC)
		if err != nil {
			return nil, chainId, err
		}
		chainId = big.NewInt(constValue.TestnetChainID)
	}
	return ethclient.NewClient(rpcClient), chainId, nil

}

func InitKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "initKey",
		Short: "Init temp key store",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, chainId, err := getEnv()
			if err != nil {
				return err
			}
			keystorePath := viper.GetString(constValue.KeystorePath)
			_, acc, err := generateOrGetTempAccount(keystorePath, chainId)
			if err != nil {
				return err
			}
			fmt.Println(acc.Address.String())
			return err
		},
	}
	cmd.Flags().String(constValue.KeystorePath, constValue.BindKeystore, "keystore path")
	return cmd
}

func DeployContractCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deployContract --config-path {config-path}",
		Short: "Deploy a contract",
		RunE: func(cmd *cobra.Command, args []string) error {
			ethClient, chainId, err := getEnv()
			if err != nil {
				return err
			}
			configPath := viper.GetString(constValue.ConfigPath)
			configData, err := config.ReadConfigData(configPath)
			if err != nil {
				return err
			}
			keystorePath := viper.GetString(constValue.KeystorePath)
			keyStore, tempAccount, err := generateOrGetTempAccount(keystorePath, chainId)
			if err != nil {
				return err
			}
			contractAddr, err := DeployContractFromTempAccount(ethClient, keyStore, tempAccount, configData.ContractData, chainId)
			if err != nil {
				return err
			}
			fmt.Println(contractAddr.String())
			return nil
		},
	}
	cmd.Flags().String(constValue.KeystorePath, constValue.BindKeystore, "keystore path")
	cmd.Flags().String(constValue.ConfigPath, "", "config file path")

	return cmd
}

func DeployCanonicalContractCmd() *cobra.Command {
	const (
		flagCanonicalImplAddr = "canonical-impl-addr"
		flagName              = "name"
		flagSymbol            = "symbol"
		flagDecimals          = "decimals"
		flagTotalSupply       = "total-supply"
		flagMintable          = "mintable"
		flagOwner             = "owner"
		flagProxyAdmin        = "proxy-admin"
	)
	cmd := &cobra.Command{
		Use:   "deployCanonicalProxyContract --config-path {config-path}",
		Short: "Deploy a proxy contract to a canonical bep20 implementation",
		RunE: func(cmd *cobra.Command, args []string) error {
			ethClient, chainId, err := getEnv()
			if err != nil {
				return err
			}

			canonicalImplAddr := viper.GetString(flagCanonicalImplAddr)
			err = utils.ValidateBSCAddr(canonicalImplAddr)
			if err != nil {
				return err
			}

			name := viper.GetString(flagName)
			symbol := viper.GetString(flagSymbol)
			if len(name) == 0 || len(symbol) == 0 {
				return fmt.Errorf("missing token name or symbol")
			}

			decimals := viper.GetInt(flagDecimals)
			if decimals < 0 {
				return fmt.Errorf("decimals must not be negative")
			}

			totalSupplyStr := viper.GetString(flagTotalSupply)
			if len(totalSupplyStr) == 0 {
				return fmt.Errorf("missing total supply")
			}
			totalSupply := big.NewInt(0)
			totalSupply.SetString(totalSupplyStr, 10)
			totalSupply = utils.ConvertToBEP20Amount(totalSupply, int64(decimals))

			mintable := viper.GetBool(flagMintable)

			owner := viper.GetString(flagOwner)
			err = utils.ValidateBSCAddr(owner)
			if err != nil {
				return err
			}

			proxyAdmin := viper.GetString(flagProxyAdmin)
			err = utils.ValidateBSCAddr(proxyAdmin)
			if err != nil {
				return err
			}

			keystorePath := viper.GetString(constValue.KeystorePath)
			keyStore, tempAccount, err := generateOrGetTempAccount(keystorePath, chainId)
			if err != nil {
				return err
			}

			canonicalUpgradeableBEP20, err := abi.JSON(strings.NewReader(constValue.CanonicalUpgradeableBEP20))
			if err != nil {
				return err
			}

			abiEncodingInitialize, err := canonicalUpgradeableBEP20.Pack("initialize", name, symbol, uint8(decimals), totalSupply, mintable, common.HexToAddress(owner))
			if err != nil {
				return err
			}

			upgradeableProxyABI, err := abi.JSON(strings.NewReader(constValue.UpgradeableProxyABI))
			if err != nil {
				return err
			}

			abiEncodingConstructor, err := upgradeableProxyABI.Pack("", common.HexToAddress(canonicalImplAddr), common.HexToAddress(proxyAdmin), abiEncodingInitialize)
			if err != nil {
				return err
			}
			abiEncodingConstructorStr := hex.EncodeToString(abiEncodingConstructor)

			contractAddr, err := DeployContractFromTempAccount(ethClient, keyStore, tempAccount, constValue.CanonicalUpgradeableBEP20BytesCode+abiEncodingConstructorStr, chainId)
			if err != nil {
				return err
			}
			fmt.Println(contractAddr.String())
			return nil
		},
	}
	cmd.Flags().String(constValue.KeystorePath, constValue.BindKeystore, "keystore path")
	cmd.Flags().String(flagCanonicalImplAddr, "0x8feCC1762561eE3D1b2ea003E1d78B71c5581BcE", "canonical implementation address")
	cmd.Flags().String(flagName, "", "token name")
	cmd.Flags().String(flagSymbol, "", "token symbol")
	cmd.Flags().Int(flagDecimals, 18, "token decimals")
	cmd.Flags().String(flagTotalSupply, "", "total supply")
	cmd.Flags().Bool(flagMintable, true, "mintable")
	cmd.Flags().String(flagOwner, "", "bep20 token owner")
	cmd.Flags().String(flagProxyAdmin, "", "proxy admin")

	return cmd
}

func ApproveBindAndTransferOwnershipCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "approveBindAndTransferOwnership --config-path {config path}",
		Short: "Use temp account in the keystore to approveBind. Then transfer bep20 ownership and rest bep20 balance to the ledger account(specified in the config file). This command supposes that the temp account in the keystore is the owner of the bep20 token",
		RunE: func(cmd *cobra.Command, args []string) error {
			if viper.GetString(constValue.NetworkType) == constValue.TestNet && viper.GetString(constValue.PeggyAmount) == "" {
				return fmt.Errorf("on testnet, you must specify peggy amount manually")
			}
			ethClient, chainId, err := getEnv()
			if err != nil {
				return err
			}
			bep20ContractAddr := viper.GetString(constValue.BEP20ContractAddr)
			if !strings.HasPrefix(bep20ContractAddr, "0x") || len(bep20ContractAddr) != constValue.BSCAddrLength {
				return fmt.Errorf("invalid bep20 contract address")
			}
			bep20Owner := viper.GetString(constValue.BEP20Owner)
			if utils.ValidateBSCAddr(bep20Owner) != nil {
				return err
			}
			bep2Symbol := viper.GetString(constValue.BEP2Symbol)
			if len(bep2Symbol) == 0 {
				return fmt.Errorf("missing bep2 symbol")
			}

			keystorePath := viper.GetString(constValue.KeystorePath)
			keyStore, tempAccount, err := generateOrGetTempAccount(keystorePath, chainId)
			if err != nil {
				return err
			}
			var peggyAmount *big.Int
			if viper.GetString(constValue.NetworkType) == constValue.TestNet {
				peggyAmount = big.NewInt(0)
				peggyAmount.SetString(viper.GetString(constValue.PeggyAmount), 10)
			}
			return ApproveBindAndTransferOwnershipAndRestBalanceBackToLedgerAccount(ethClient, keyStore, tempAccount, common.HexToAddress(bep20ContractAddr), peggyAmount, bep2Symbol, common.HexToAddress(bep20Owner), chainId)
		},
	}
	cmd.Flags().String(constValue.KeystorePath, constValue.BindKeystore, "keystore path")
	cmd.Flags().String(constValue.BEP20ContractAddr, "", "bep20 contract address")
	cmd.Flags().String(constValue.BEP20Owner, "", "bep20 token owner")
	cmd.Flags().String(constValue.BEP2Symbol, "", "bep2 token symbol")
	cmd.Flags().String(constValue.PeggyAmount, "", "peggy amount, which is identical to the peggy amount in bind transaction")
	return cmd
}

func DeployBEP20ContractTransferTotalSupplyAndOwnershipCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deployBEP20ContractTransferTotalSupplyAndOwnership --config-path {config path}",
		Short: "Deploy a bep20 contract, and transfer total balance and ownership to the account(specified in the config file)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ethClient, chainId, err := getEnv()
			if err != nil {
				return err
			}
			configPath := viper.GetString(constValue.ConfigPath)
			config, err := config.ReadConfigData(configPath)
			if err != nil {
				return err
			}
			keystorePath := viper.GetString(constValue.KeystorePath)
			keyStore, tempAccount, err := generateOrGetTempAccount(keystorePath, chainId)
			if err != nil {
				return err
			}
			bep20Owner := viper.GetString(constValue.BEP20Owner)
			if utils.ValidateBSCAddr(bep20Owner) != nil {
				return err
			}
			contractAddr, err := DeployContractFromTempAccount(ethClient, keyStore, tempAccount, config.ContractData, chainId)
			if err != nil {
				return err
			}
			return TransferTokenAndOwnership(ethClient, keyStore, tempAccount, common.HexToAddress(bep20Owner), contractAddr, chainId)
		},
	}
	cmd.Flags().String(constValue.KeystorePath, constValue.BindKeystore, "keystore path")
	cmd.Flags().String(constValue.ConfigPath, "", "config file path")
	cmd.Flags().String(constValue.BEP20Owner, "", "bep20 contract address")
	return cmd
}

func ApproveBindFromLedgerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "approveBindFromLedger",
		Short: "Call tokenManager contract to approve bind with a bep2 token. Users should firstly send bind transaction on Binance Chain, and wait for 30 second",
		RunE: func(cmd *cobra.Command, args []string) error {
			if viper.GetString(constValue.NetworkType) == constValue.TestNet && viper.GetString(constValue.PeggyAmount) == "" {
				return fmt.Errorf("on testnet, you must specify peggy amount manually")
			}
			ethClient, chainId, err := getEnv()
			if err != nil {
				return err
			}

			ledgerAccountIndex := viper.GetInt32(constValue.LedgerAccountIndex)

			bep20ContractAddr := viper.GetString(constValue.BEP20ContractAddr)
			if !strings.HasPrefix(bep20ContractAddr, "0x") || len(bep20ContractAddr) != constValue.BSCAddrLength {
				return fmt.Errorf("Invalid bep20 contract address")
			}
			bep2Symbol := viper.GetString(constValue.BEP2Symbol)
			if len(bep2Symbol) == 0 {
				return fmt.Errorf("missing bep2 symbol")
			}
			ledgerWallet, ledgerAccount, err := openLedger(uint32(ledgerAccountIndex))
			if err != nil {
				return err
			}
			var peggyAmount *big.Int
			if viper.GetString(constValue.NetworkType) == constValue.TestNet {
				peggyAmount = big.NewInt(0)
				peggyAmount.SetString(viper.GetString(constValue.PeggyAmount), 10)
			}
			return ApproveBind(ethClient, ledgerWallet, ledgerAccount, bep2Symbol, common.HexToAddress(bep20ContractAddr), peggyAmount, chainId)
		},
	}
	cmd.Flags().String(constValue.BEP20ContractAddr, "", "bep20 contract address")
	cmd.Flags().String(constValue.BEP2Symbol, "", "bep2 token symbol")
	cmd.Flags().Int64(constValue.LedgerAccountIndex, 0, "ledger account index")
	cmd.Flags().String(constValue.PeggyAmount, "", "peggy amount, which is identical to the peggy amount in bind transaction")
	return cmd
}

func RefundRestBNBCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "refundRestBNB",
		Short: "Refund all rest BNB of the temp account to to user specified bsc address",
		RunE: func(cmd *cobra.Command, args []string) error {
			ethClient, chainId, err := getEnv()
			if err != nil {
				return err
			}
			recipientStr := viper.GetString(constValue.Recipient)
			if !strings.HasPrefix(recipientStr, "0x") || len(recipientStr) != constValue.BSCAddrLength {
				return fmt.Errorf("Invalid refund address")
			}
			keystorePath := viper.GetString(constValue.KeystorePath)
			keyStore, tempAccount, err := generateOrGetTempAccount(keystorePath, chainId)
			if err != nil {
				return err
			}
			return RefundRestBNB(ethClient, keyStore, tempAccount, common.HexToAddress(recipientStr), chainId)
		},
	}
	cmd.Flags().String(constValue.KeystorePath, constValue.BindKeystore, "keystore path")
	cmd.Flags().String(constValue.Recipient, "", "recipient, bsc address")
	return cmd
}

func QueryERC721TotalSupply() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "queryERC721TotalSupply",
		Short: "Query erc721 totalSupply",
		RunE: func(cmd *cobra.Command, args []string) error {
			ethClient, _, err := getEnv()
			if err != nil {
				return err
			}
			erc721AddrStr := viper.GetString(constValue.ERC721Addr)
			if !strings.HasPrefix(erc721AddrStr, "0x") || len(erc721AddrStr) != constValue.BSCAddrLength {
				return fmt.Errorf("Invalid erc721 contract address")
			}

			erc721Inst, err := erc721.NewErc721(common.HexToAddress(erc721AddrStr), ethClient)
			if err != nil {
				return err
			}
			totalSupply, err := erc721Inst.TotalSupply(utils.GetCallOpts())
			if err != nil {
				return err
			}
			fmt.Println(fmt.Sprintf("totalSupply of %s is %s", erc721AddrStr, totalSupply.String()))
			return nil
		},
	}
	cmd.Flags().String(constValue.ERC721Addr, "", "keystore path")
	return cmd
}

func DeployContractFromTempAccount(ethClient *ethclient.Client, keyStore *keystore.KeyStore, tempAccount accounts.Account, contractByteCodeStr string, chainId *big.Int) (common.Address, error) {
	contractByteCode, err := hex.DecodeString(contractByteCodeStr)
	if err != nil {
		return common.Address{}, err
	}
	txHash, err := utils.DeployContract(ethClient, keyStore, tempAccount, contractByteCode, chainId)
	if err != nil {
		return common.Address{}, err
	}
	time.Sleep(10 * time.Second)

	txRecipient, err := ethClient.TransactionReceipt(context.Background(), txHash)
	if err != nil {
		return common.Address{}, err
	}
	contractAddr := txRecipient.ContractAddress
	return contractAddr, nil
}

func ApproveBindAndTransferOwnershipAndRestBalanceBackToLedgerAccount(ethClient *ethclient.Client, keyStore *keystore.KeyStore, tempAccount accounts.Account, bep20ContractAddr common.Address, peggyAmount *big.Int, bep2Symbol string, bep20Owner common.Address, chainId *big.Int) error {
	bep20Instance, err := bep20.NewBep20(bep20ContractAddr, ethClient)
	if err != nil {
		return err
	}
	tokenManagerInstance, err := tokenmanager.NewTokenmanager(constValue.TokenManagerContractAddr, ethClient)
	if err != nil {
		return err
	}
	var lockAmount *big.Int
	if peggyAmount == nil {
		lockAmount, err = tokenManagerInstance.QueryRequiredLockAmountForBind(utils.GetCallOpts(), bep2Symbol)
		if err != nil {
			return err
		}
	} else {
		totalSupply, err := bep20Instance.TotalSupply(utils.GetCallOpts())
		if err != nil {
			return err
		}
		decimals, err := bep20Instance.Decimals(utils.GetCallOpts())
		if err != nil {
			return err
		}

		lockAmount = big.NewInt(1).Sub(totalSupply, utils.ConvertToBEP20Amount(peggyAmount, decimals.Int64()))
		if lockAmount.Cmp(big.NewInt(0)) < 0 {
			return fmt.Errorf("peggy amount is large than total supply")
		}
	}

	fmt.Println(fmt.Sprintf("Approve %s:%s to TokenManager from %s", lockAmount.String(), bep2Symbol, tempAccount.Address.String()))
	approveTxHash, err := bep20Instance.Approve(utils.GetTransactor(ethClient, keyStore, tempAccount, big.NewInt(0)), constValue.TokenManagerContractAddr, lockAmount)
	if err != nil {
		return err
	}
	utils.PrintTxExplorerUrl("Approve token to tokenManagerContractAddr txHash", approveTxHash.Hash().String(), chainId)

	utils.Sleep(10)

	tokenhubInstance, err := tokenhub.NewTokenhub(constValue.TokenHubContractAddr, ethClient)
	if err != nil {
		return err
	}
	miniRelayerFee, err := tokenhubInstance.GetMiniRelayFee(utils.GetCallOpts())
	if err != nil {
		return err
	}

	approveBindTx, err := tokenManagerInstance.ApproveBind(utils.GetTransactor(ethClient, keyStore, tempAccount, miniRelayerFee), bep20ContractAddr, bep2Symbol)
	if err != nil {
		return err
	}
	utils.PrintTxExplorerUrl("ApproveBind txHash", approveBindTx.Hash().String(), chainId)

	utils.Sleep(10)

	approveBindTxRecipient, err := ethClient.TransactionReceipt(context.Background(), approveBindTx.Hash())
	if err != nil {
		return err
	}
	fmt.Println("Track approveBind Tx status")
	if approveBindTxRecipient.Status != 1 {
		fmt.Println("Approve Bind is failed")
		rejectBindTx, err := tokenManagerInstance.RejectBind(utils.GetTransactor(ethClient, keyStore, tempAccount, miniRelayerFee), bep20ContractAddr, bep2Symbol)
		if err != nil {
			return err
		}
		utils.PrintTxExplorerUrl("RejectBind txHash", rejectBindTx.Hash().String(), chainId)
		utils.Sleep(10)
		fmt.Println("Track rejectBind Tx status")
		rejectBindTxRecipient, err := ethClient.TransactionReceipt(context.Background(), rejectBindTx.Hash())
		if err != nil {
			return err
		}
		fmt.Println(fmt.Sprintf("reject bind tx recipient status %d", rejectBindTxRecipient.Status))
		return nil
	} else {
		fmt.Println("Approve Bind is successful")
	}

	restBEP20Balance, err := bep20Instance.BalanceOf(utils.GetCallOpts(), tempAccount.Address)
	if err != nil {
		return err
	}
	if restBEP20Balance.Cmp(big.NewInt(0)) > 0 {
		fmt.Println(fmt.Sprintf("Refund rest BEP20 balance %s to %s", restBEP20Balance.String(), bep20Owner.String()))
		refundRestBEP20BalanceTxHash, err := bep20Instance.Transfer(utils.GetTransactor(ethClient, keyStore, tempAccount, big.NewInt(0)), bep20Owner, restBEP20Balance)
		if err != nil {
			return err
		}
		utils.PrintTxExplorerUrl("Refund rest BEP20 balance txHash", refundRestBEP20BalanceTxHash.Hash().String(), chainId)
	}

	utils.Sleep(10)
	ownershipInstance, err := ownable.NewOwnable(bep20ContractAddr, ethClient)
	if err != nil {
		return err
	}
	fmt.Println(fmt.Sprintf("Transfer ownership to %s", tempAccount.Address.String()))
	transferOwnerShipTxHash, err := ownershipInstance.TransferOwnership(utils.GetTransactor(ethClient, keyStore, tempAccount, big.NewInt(0)), bep20Owner)
	if err != nil {
		return err
	}
	utils.PrintTxExplorerUrl("Transfer ownership txHash", transferOwnerShipTxHash.Hash().String(), chainId)
	fmt.Println("--------------------------------------------------------------------------------------------------------------------------------")
	return nil
}

func ApproveBind(ethClient *ethclient.Client, ledgerWallet accounts.Wallet, ledgerAccount accounts.Account, bep2Symbol string, bep20ContractAddr common.Address, peggyAmount *big.Int, chainId *big.Int) error {
	tokenManagerInstance, err := tokenmanager.NewTokenmanager(constValue.TokenManagerContractAddr, ethClient)
	if err != nil {
		return err
	}
	var lockAmount *big.Int
	if peggyAmount == nil {
		lockAmount, err = tokenManagerInstance.QueryRequiredLockAmountForBind(utils.GetCallOpts(), bep2Symbol)
		if err != nil {
			return err
		}
	} else {
		bep20Instance, err := bep20.NewBep20(bep20ContractAddr, ethClient)
		if err != nil {
			return err
		}
		totalSupply, err := bep20Instance.TotalSupply(utils.GetCallOpts())
		if err != nil {
			return err
		}
		decimals, err := bep20Instance.Decimals(utils.GetCallOpts())
		if err != nil {
			return err
		}
		lockAmount = big.NewInt(1).Sub(totalSupply, utils.ConvertToBEP20Amount(peggyAmount, decimals.Int64()))
		if lockAmount.Cmp(big.NewInt(0)) < 0 {
			return fmt.Errorf("peggy amount is large than total supply")
		}
	}
	fmt.Println(fmt.Sprintf("Approve %s to TokenManager from %s", lockAmount.String(), ledgerAccount.Address.String()))
	bep20ABI, _ := abi.JSON(strings.NewReader(bep20.Bep20ABI))
	approveTxData, err := bep20ABI.Pack("approve", constValue.TokenManagerContractAddr, lockAmount)
	if err != nil {
		return err
	}
	hexApproveTxData := hexutil.Bytes(approveTxData)
	approveTx, err := utils.SendTransactionFromLedger(ethClient, ledgerWallet, ledgerAccount, bep20ContractAddr, big.NewInt(0), &hexApproveTxData, chainId)
	if err != nil {
		return err
	}
	utils.PrintTxExplorerUrl("Approve token to tokenManagerContractAddr txHash", approveTx.Hash().String(), chainId)

	utils.Sleep(10)

	tokenhubInstance, err := tokenhub.NewTokenhub(constValue.TokenHubContractAddr, ethClient)
	if err != nil {
		return err
	}
	miniRelayerFee, err := tokenhubInstance.GetMiniRelayFee(utils.GetCallOpts())
	if err != nil {
		return err
	}

	fmt.Println(fmt.Sprintf("ApproveBind from %s", ledgerAccount.Address.String()))
	tokenManagerABI, _ := abi.JSON(strings.NewReader(constValue.TokenManagerABI))
	approveBindTxData, err := tokenManagerABI.Pack("approveBind", bep20ContractAddr, bep2Symbol)
	if err != nil {
		return err
	}
	hexApproveBindTxData := hexutil.Bytes(approveBindTxData)
	approveBindTx, err := utils.SendTransactionFromLedger(ethClient, ledgerWallet, ledgerAccount, constValue.TokenManagerContractAddr, miniRelayerFee, &hexApproveBindTxData, chainId)
	if err != nil {
		return err
	}
	utils.PrintTxExplorerUrl("ApproveBind txHash", approveBindTx.Hash().String(), chainId)
	fmt.Println("--------------------------------------------------------------------------------------------------------------------------------")
	return nil
}

func TransferTokenAndOwnership(ethClient *ethclient.Client, keyStore *keystore.KeyStore, tempAccount accounts.Account, tokenOwner common.Address, bep20ContractAddr common.Address, chainId *big.Int) error {
	bep20Instance, err := bep20.NewBep20(bep20ContractAddr, ethClient)
	if err != nil {
		return err
	}
	totalSupply, err := bep20Instance.TotalSupply(utils.GetCallOpts())
	if err != nil {
		return err
	}
	fmt.Println(fmt.Sprintf("Total Supply %s", totalSupply.String()))

	fmt.Println(fmt.Sprintf("Transfer %s token to %s", totalSupply.String(), tokenOwner.String()))
	transferTxHash, err := bep20Instance.Transfer(utils.GetTransactor(ethClient, keyStore, tempAccount, big.NewInt(0)), tokenOwner, totalSupply)
	if err != nil {
		return err
	}
	utils.PrintTxExplorerUrl("Transfer token txHash", transferTxHash.Hash().String(), chainId)

	utils.Sleep(10)

	ownershipInstance, err := ownable.NewOwnable(bep20ContractAddr, ethClient)
	if err != nil {
		return err
	}
	fmt.Println(fmt.Sprintf("Transfer ownership to %s", tokenOwner.String()))
	transferOwnerShipTxHash, err := ownershipInstance.TransferOwnership(utils.GetTransactor(ethClient, keyStore, tempAccount, big.NewInt(0)), tokenOwner)
	if err != nil {
		return err
	}
	utils.PrintTxExplorerUrl("Transfer ownership txHash", transferOwnerShipTxHash.Hash().String(), chainId)
	fmt.Println("--------------------------------------------------------------------------------------------------------------------------------")
	return nil
}

func RefundRestBNB(ethClient *ethclient.Client, keyStore *keystore.KeyStore, tempAccount accounts.Account, refundAddr common.Address, chainId *big.Int) error {
	utils.Sleep(10)
	txHash, err := utils.SendAllRestBNB(ethClient, keyStore, tempAccount, refundAddr, chainId)
	if err != nil {
		return err
	}
	utils.PrintTxExplorerUrl("Refund txHash", txHash.String(), chainId)
	fmt.Println("--------------------------------------------------------------------------------------------------------------------------------")
	return nil
}
