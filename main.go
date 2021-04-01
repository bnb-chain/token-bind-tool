package main

import (
	"os"

	"github.com/binance-chain/token-bind-tool/command"
	constvalue "github.com/binance-chain/token-bind-tool/const"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Bind all flags and read the config into viper
func bindFlagsLoadViper(cmd *cobra.Command, args []string) error {
	// cmd.Flags() includes flags from this command and all persistent flags from the parent
	if err := viper.BindPFlags(cmd.Flags()); err != nil {
		return err
	}
	return nil
}

type cobraCmdFunc func(cmd *cobra.Command, args []string) error

func concatCobraCmdFuncs(fs ...cobraCmdFunc) cobraCmdFunc {
	return func(cmd *cobra.Command, args []string) error {
		for _, f := range fs {
			if f != nil {
				if err := f(cmd, args); err != nil {
					return err
				}
			}
		}
		return nil
	}
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "token-bind-tool",
		Short: "Command line interface for deploy bep20 contract and bind with bep2 token",
	}
	rootCmd.PersistentFlags().String(constvalue.NetworkType, constvalue.Mainnet, "mainnet or testnet")
	rootCmd.AddCommand(
		command.InitKeyCmd(),
		command.DeployContractCmd(),
		command.DeployCanonicalContractCmd(),
		command.ApproveBindAndTransferOwnershipCmd(),
		command.DeployBEP20ContractTransferTotalSupplyAndOwnershipCmd(),
		command.ApproveBindFromLedgerCmd(),
		command.RefundRestBNBCmd(),
		command.QueryERC721TotalSupply(),
	)
	// prepare and add flags
	rootCmd.PersistentPreRunE = concatCobraCmdFuncs(bindFlagsLoadViper, rootCmd.PersistentPreRunE)
	rootCmd.SilenceUsage = true
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
