#!/bin/bash

networkType=$1
bep2TokenOwnerKeyName=$2
peggyAmount=$3
bep2TokenSymbol=$4
tokenOwner=$5
binaryPath=$6

echo "network type:" $networkType
echo "bep2TokenOwnerKeyName:" $bep2TokenOwnerKeyName
echo "token name:" $tokenName
echo "peggy amount:" $peggyAmount
echo "bep2 symbol:" $bep2TokenSymbol
echo "token owner" $tokenOwner

chmod +x ./binary/*

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

./build/token-bind-tool deployContract --config-path script/config.json --network-type $networkType

echo "Please input the new deploy bep20 contract address "
read bep20ContractAddr

$binaryPath bridge bind --symbol $bep2TokenSymbol --amount $peggyAmount --expire-time `expr $(date +%s) + 3600` \
--contract-decimals 18 --from $bep2TokenOwnerKeyName --chain-id $chainId --contract-address $bep20ContractAddr \
--node $nodeUrl

echo "Sleep 10 second"
sleep 10

./build/token-bind-tool approveBindAndTransferOwnership --config-path tokens/$tokenName/$tokenName.json --bep20-contract-addr $bep20ContractAddr --network-type $networkType --peggy-amount $peggyAmount
