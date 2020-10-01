package command

import (
	"encoding/json"
	"fmt"
	constValue "github.com/binance-chain/token-bind-tool/const"
	"github.com/binance-chain/token-bind-tool/contracts/bep20"
	"github.com/binance-chain/token-bind-tool/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/shopspring/decimal"
	"io/ioutil"
	"net/http"
)

func buildTokenStatsInfo(ethClient *ethclient.Client, contractAddr common.Address, excludeAddressArray []string, bnbPrice decimal.Decimal) (TokenStatsInfo, error) {
	bep20Instance, err := bep20.NewBep20(contractAddr, ethClient)
	if err != nil {
		return TokenStatsInfo{}, err
	}
	name, err := bep20Instance.Name(utils.GetCallOpts())
	if err != nil {
		return TokenStatsInfo{}, err
	}
	symbol, err := bep20Instance.Symbol(utils.GetCallOpts())
	if err != nil {
		return TokenStatsInfo{}, err
	}
	decimals, err := bep20Instance.Decimals(utils.GetCallOpts())
	if err != nil {
		return TokenStatsInfo{}, err
	}
	fmt.Print(fmt.Sprintf("%s ", symbol))
	totalSupply, err := bep20Instance.TotalSupply(utils.GetCallOpts())
	if err != nil {
		return TokenStatsInfo{}, err
	}
	owner, err := bep20Instance.GetOwner(utils.GetCallOpts())
	if err != nil {
		return TokenStatsInfo{}, err
	}
	tokenHubBalance, err := bep20Instance.BalanceOf(utils.GetCallOpts(), constValue.TokenHubContractAddr)
	if err != nil {
		return TokenStatsInfo{}, err
	}
	superOwnerAddr1Balance, err := bep20Instance.BalanceOf(utils.GetCallOpts(), constValue.SuperOwnerAddr1)
	if err != nil {
		return TokenStatsInfo{}, err
	}
	superOwnerAddr2Balance, err := bep20Instance.BalanceOf(utils.GetCallOpts(), constValue.SuperOwnerAddr2)
	if err != nil {
		return TokenStatsInfo{}, err
	}
	withdrawAddrBalance, err := bep20Instance.BalanceOf(utils.GetCallOpts(), constValue.WithdrawAddr)
	if err != nil {
		return TokenStatsInfo{}, err
	}

	excludeBalance := utils.BigIntAdd(tokenHubBalance, superOwnerAddr1Balance, superOwnerAddr2Balance, withdrawAddrBalance)
	if owner.String() != constValue.SuperOwnerAddr1.String() && owner.String() != constValue.SuperOwnerAddr2.String() {
		ownerBalance, err := bep20Instance.BalanceOf(utils.GetCallOpts(), owner)
		if err != nil {
			return TokenStatsInfo{}, err
		}
		excludeBalance = utils.BigIntAdd(excludeBalance, ownerBalance)
	}

	for _, addr := range excludeAddressArray {
		excludeAddressBalance, err := bep20Instance.BalanceOf(utils.GetCallOpts(), common.HexToAddress(addr))
		if err != nil {
			return TokenStatsInfo{}, err
		}
		excludeBalance = utils.BigIntAdd(excludeBalance, excludeAddressBalance)
	}
	scatteredBalance := utils.BigIntSub(totalSupply, excludeBalance)

	var price decimal.Decimal
	if StableCoin[symbol] {
		price, _ = decimal.NewFromString("1")
	} else {
		price, _ = getPrice(symbol, bnbPrice)
	}

	tokenStatsInfo := TokenStatsInfo{
		Name:                    name,
		Symbol:                  symbol,
		Decimals:                decimals.Int64(),
		TotalSupply:             decimal.NewFromBigInt(totalSupply, -int32(decimals.Int64())),
		TokenHubBalance:         decimal.NewFromBigInt(tokenHubBalance, -int32(decimals.Int64())),
		WithdrawAddrBalance:     decimal.NewFromBigInt(withdrawAddrBalance, -int32(decimals.Int64())),
		LedgerAccount19Balance:  decimal.NewFromBigInt(superOwnerAddr1Balance, -int32(decimals.Int64())),
		LedgerAccount20Balance:  decimal.NewFromBigInt(superOwnerAddr2Balance, -int32(decimals.Int64())),
		BSCCirculation:          decimal.NewFromBigInt(utils.BigIntSub(totalSupply, tokenHubBalance), -int32(decimals.Int64())),
		ScatteredBalance:        decimal.NewFromBigInt(scatteredBalance, -int32(decimals.Int64())),
		Price:                   price,
		ScatteredMarketCapacity: decimal.NewFromBigInt(scatteredBalance, -int32(decimals.Int64())).Mul(price),
	}

	return tokenStatsInfo, nil
}

func getPrice(symbol string, bnbPrice decimal.Decimal) (decimal.Decimal, error) {
	if symbol == "BTCB" {
		symbol = "BTC"
	}
	price, err := getTokenPrice1(symbol, bnbPrice)
	if err == nil {
		return price, nil
	}
	price, err = getTokenPrice2(symbol, bnbPrice)
	if err == nil {
		return price, nil
	}
	return getTokenPrice3(symbol)
}

func getBNBPrice() (decimal.Decimal, error) {
	resp, err := http.Get("https://api.binance.com/api/v3/avgPrice?symbol=BNBUSDT")
	if err != nil {
		return decimal.Decimal{}, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return decimal.Decimal{}, err
	}
	tokenPrice := TokenPrice{}
	err = json.Unmarshal(body, &tokenPrice)
	if err != nil {
		return decimal.Decimal{}, err
	}
	price, err := decimal.NewFromString(tokenPrice.Price)
	if err != nil {
		return decimal.Decimal{}, err
	}
	return price, nil
}

func getTokenPrice1(symbol string, bnbPrice decimal.Decimal) (decimal.Decimal, error) {
	resp, err := http.Get(fmt.Sprintf("https://api.binance.com/api/v3/avgPrice?symbol=%sBNB", symbol))
	if err != nil {
		return decimal.Decimal{}, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return decimal.Decimal{}, err
	}
	tokenPrice := TokenPrice{}
	err = json.Unmarshal(body, &tokenPrice)
	if err != nil {
		return decimal.Decimal{}, err
	}
	price, err := decimal.NewFromString(tokenPrice.Price)
	if err != nil {
		return decimal.Decimal{}, fmt.Errorf("price %s, err: %s", string(body), err.Error())
	}
	return price.Mul(bnbPrice), nil
}
func getTokenPrice2(symbol string, bnbPrice decimal.Decimal) (decimal.Decimal, error) {
	resp, err := http.Get(fmt.Sprintf("https://api.binance.com/api/v3/avgPrice?symbol=BNB%s", symbol))
	if err != nil {
		return decimal.Decimal{}, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return decimal.Decimal{}, err
	}
	tokenPrice := TokenPrice{}
	err = json.Unmarshal(body, &tokenPrice)
	if err != nil {
		return decimal.Decimal{}, err
	}
	price, err := decimal.NewFromString(tokenPrice.Price)
	if err != nil {
		return decimal.Decimal{}, fmt.Errorf("price %s, err: %s", string(body), err.Error())
	}
	return bnbPrice.Div(price), nil
}
func getTokenPrice3(symbol string) (decimal.Decimal, error) {
	resp, err := http.Get(fmt.Sprintf("https://api.binance.com/api/v3/avgPrice?symbol=%sUSDT", symbol))
	if err != nil {
		return decimal.Decimal{}, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return decimal.Decimal{}, err
	}
	tokenPrice := TokenPrice{}
	err = json.Unmarshal(body, &tokenPrice)
	if err != nil {
		return decimal.Decimal{}, err
	}
	price, err := decimal.NewFromString(tokenPrice.Price)
	if err != nil {
		return decimal.Decimal{}, fmt.Errorf("price %s, err: %s", string(body), err.Error())
	}
	return price, nil
}