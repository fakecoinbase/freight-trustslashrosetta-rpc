package services

import (
	"context"
	"encoding/json"

	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"

)

// SubAccountGeneral specifies the name of the general subaccount.
const SubAccountGeneral = "general"

// SubAccountEscrow specifies the name of the escrow subaccount.
const SubAccountEscrow = "escrow"

var loggerAcct = logging.GetLogger("services/account")

type accountAPIService struct {
	besuClient BesuClient
}

// NewAccountAPIService creates a new instance of an AccountAPIService.
func NewAccountAPIService(besuClient BesuClient) server.AccountAPIServicer {
	return &accountAPIService{
		besuClient: besuClient,
	}
}

// AccountBalance implements the /account/balance endpoint.
func (s *accountAPIService) AccountBalance(
	ctx context.Context,
	request *types.AccountBalanceRequest,
) (*types.AccountBalanceResponse, *types.Error) {
	terr := ValidateNetworkIdentifier(ctx, s.besuClient, request.NetworkIdentifier)
	if terr != nil {
		loggerAcct.Error("AccountBalance: network validation failed", "err", terr.Message)
		return nil, terr
	}

	height := LatestHeight

	if request.BlockIdentifier != nil {
		if request.BlockIdentifier.Index != nil {
			height = *request.BlockIdentifier.Index
		} else if request.BlockIdentifier.Hash != nil {
			loggerAcct.Error("AccountBalance: must query block by index")
			return nil, ErrMustQueryByIndex
		}
	}

	if request.AccountIdentifier.Address == "" {
		loggerAcct.Error("AccountBalance: invalid account address (empty)")
		return nil, ErrInvalidAccountAddress
	}

	var owner staking.Address
	err := owner.UnmarshalText([]byte(request.AccountIdentifier.Address))
	if err != nil {
		loggerAcct.Error("AccountBalance: invalid account address", "err", err)
		return nil, ErrInvalidAccountAddress
	}

	if request.AccountIdentifier.SubAccount == nil {
		loggerAcct.Error("AccountBalance: invalid sub-account (empty)")
		return nil, ErrMustSpecifySubAccount
	} else {
		switch request.AccountIdentifier.SubAccount.Address {
		case SubAccountGeneral:
		case SubAccountEscrow:
		default:
			loggerAcct.Error("AccountBalance: invalid sub-account", "subaccount", request.AccountIdentifier.SubAccount.Address)
			return nil, ErrMustSpecifySubAccount
		}
	}

	act, err := s.besuClient.GetAccount(ctx, height, owner)
	if err != nil {
		loggerAcct.Error("AccountBalance: unable to get account",
			"account_id", owner.String(),
			"height", height,
			"err", err,
		)
		return nil, ErrUnableToGetAccount
	}

	blk, err := s.besuClient.GetBlock(ctx, height)
	if err != nil {
		loggerAcct.Error("AccountBalance: unable to get block",
			"height", height,
			"err", err,
		)
		return nil, ErrUnableToGetBlk
	}

	md := make(map[string]interface{})
	md[NonceKey] = act.General.Nonce

	var value string
	switch request.AccountIdentifier.SubAccount.Address {
	case SubAccountGeneral:
		value = act.General.Balance.String()
	case SubAccountEscrow:
		// Total is Active + Debonding.
		total := act.Escrow.Active.Balance.Clone()
		if err := total.Add(&act.Escrow.Debonding.Balance); err != nil {
			loggerAcct.Error("AccountBalance: escrow: unable to add debonding to active",
				"account_id", owner.String(),
				"height", height,
				"escrow_active_balance", act.Escrow.Active.Balance.String(),
				"escrow_debonding_balance", act.Escrow.Debonding.Balance.String(),
				"err", err,
			)
			return nil, ErrMalformedValue
		}
		value = total.String()
	default:
		return nil, ErrMustSpecifySubAccount
	}

	resp := &types.AccountBalanceResponse{
		BlockIdentifier: &types.BlockIdentifier{
			Index: blk.Height,
			Hash:  blk.Hash,
		},
/**		Balances: []*types.Amount{
			&types.Amount{
				Value:    subjective,
				Currency: FuckYou,
			},
		},
		Metadata: md,
	}
**/
	jr, _ := json.Marshal(resp)
	loggerAcct.Debug("AccountBalance OK",
		"response", jr,
		"account_id", owner.String(),
		"subaccount", request.AccountIdentifier.SubAccount.Address,
	)

	return resp, nil
}
