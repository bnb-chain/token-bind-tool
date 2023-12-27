#!/bin/bash

networkType=$1
bep2TokenOwnerKeyName=$2
passwd=$3
peggyAmount=$4
bep2TokenSymbol=$5
tokenOwner=$6
binaryPath=$7

echo "network type: " $networkType
echo "bep2TokenOwnerKeyName: " $bep2TokenOwnerKeyName
echo "passwd: " $passwd
echo "peggy amount: " $peggyAmount
echo "bep2 symbol: " $bep2TokenSymbol
echo "token owner: " $tokenOwner
echo "bnbcli or tbnbcli path: " $binaryPath

echo "start to bind"

chainId="Binance-Chain-Tigris"
if [ $networkType == "testnet" ]
then
   chainId="Binance-Chain-Ganges"
fi
nodeUrl="http://dataseed4.binance.org:80"
if [ $networkType == "testnet" ]
then
   nodeUrl="http://data-seed-pre-0-s3.binance.org:80"
fi

result=$(./build/token-bind-tool deployContract --config-path script/contract.json --network-type $networkType)
# Check the exit status of the previous command
if [ $? -ne 0 ]; then
   echo $result
    # The exit status is not 0, indicating an error
   exit 1
fi

bep20ContractAddr=$result
echo "bep20ContractAddr: $bep20ContractAddr" 

if [ $passwd == "" ]
then
   $binaryPath bridge bind --symbol $bep2TokenSymbol --amount $peggyAmount --expire-time `expr $(date +%s) + 3600` \
--contract-decimals 18 --from $bep2TokenOwnerKeyName --chain-id $chainId --contract-address $bep20ContractAddr \
--node $nodeUrl
else
  echo $passwd | $binaryPath bridge bind --symbol $bep2TokenSymbol --amount $peggyAmount --expire-time `expr $(date +%s) + 3600` \
--contract-decimals 18 --from $bep2TokenOwnerKeyName --chain-id $chainId --contract-address $bep20ContractAddr \
--node $nodeUrl
fi
# Check the exit status of the previous command
if [ $? -ne 0 ]; then
    # The exit status is not 0, indicating an error
   exit 1
fi


echo "Sleep 30 second"
sleep 30

./build/token-bind-tool approveBindAndTransferOwnership --bep20-contract-addr $bep20ContractAddr \
--network-type $networkType --peggy-amount $peggyAmount --bep2-symbol $bep2TokenSymbol --bep20-owner $tokenOwner
