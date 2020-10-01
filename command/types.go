package command

import "github.com/shopspring/decimal"

type TokenStatsInfo struct {
	Name                   string
	Symbol                 string
	Decimals               int64
	TotalSupply            decimal.Decimal
	TokenHubBalance        decimal.Decimal
	WithdrawAddrBalance    decimal.Decimal
	LedgerAccount19Balance decimal.Decimal
	LedgerAccount20Balance decimal.Decimal
	BSCCirculation         decimal.Decimal
	ScatteredBalance       decimal.Decimal

	Price                   decimal.Decimal
	ScatteredMarketCapacity decimal.Decimal
}

type TokenPrice struct {
	Mins  int    `json:"mins"`
	Price string `json:"price"`
}

var StableCoin = make(map[string]bool)
