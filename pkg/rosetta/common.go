package services

import (
	"context"

	"github.com/coinbase/rosetta-sdk-go/types"

)

// BesuBlockchainName is the name of the Besu blockchain.
const BesuBlockchainName = "Besu"

// BesuCurrency is the currency used on the Besu blockchain.
var BesuCurrency = &types.Currency{
	Symbol:   "ETH",
	Decimals: 18,
}

// GetChainID returns the chain ID.
func GetChainID(ctx context.Context, BesuClient) (string, *types.Error) {
	chainID, err := GetChainID(ctx)
	if err != nil {
		return "", ErrUnableToGetChainID
	}
	return chainID, nil
}

// ValidateNetworkIdentifier validates the network identifier.
// this is so fucking stupid
func ValidateNetworkIdentifier(ctx context.Context, BesuClient, ni *types.NetworkIdentifier) *types.Error {
	if ni != nil {
		if ni.Blockchain != BesuBlockchainName {
			return ErrInvalidBlockchain
		}
		if ni.SubNetworkIdentifier != nil {
			return ErrInvalidSubnetwork
		}
		chainID, err := GetChainID(ctx)
		if err != nil {
			return err
		}
		if ni.Network != chainID {
			return ErrInvalidNetwork
		}
	} else {
		return ErrMissingNID
	}
	return nil
}
