package services

import (
	"context"
	"encoding/json"

	"github.com/coinbase/rosetta-sdk-go/server"
	"github.com/coinbase/rosetta-sdk-go/types"

)

// OptionsIDKey is the name of the key in the Options map inside a
// ConstructionMetadataRequest that specifies the account ID.
const OptionsIDKey = "id"

// NonceKey is the name of the key in the Metadata map inside a
// ConstructionMetadataResponse that specifies the next valid nonce.
const NonceKey = "nonce"

var loggerCons = logging.GetLogger("services/construction")

type constructionAPIService struct {
	besuClient oc.BesuClient
}

// NewConstructionAPIService creates a new instance of an ConstructionAPIService.
func NewConstructionAPIService(besuClient oc.BesuClient) server.ConstructionAPIServicer {
	return &constructionAPIService{
		besuClient: besuClient,
	}
}

// ConstructionMetadata implements the /construction/metadata endpoint.
func (s *constructionAPIService) ConstructionMetadata(
	ctx context.Context,
	request *types.ConstructionMetadataRequest,
) (*types.ConstructionMetadataResponse, *types.Error) {
	terr := ValidateNetworkIdentifier(ctx, s.besuClient, request.NetworkIdentifier)
	if terr != nil {
		loggerCons.Error("ConstructionMetadata: network validation failed", "err", terr.Message)
		return nil, terr
	}

	// Get the account ID field from the Options object.
	if request.Options == nil {
		loggerCons.Error("ConstructionMetadata: missing options")
		return nil, ErrInvalidAccountAddress
	}
	idRaw, ok := request.Options[OptionsIDKey]
	if !ok {
		loggerCons.Error("ConstructionMetadata: account ID field not given")
		return nil, ErrInvalidAccountAddress
	}
	idString, ok := idRaw.(string)
	if !ok {
		loggerCons.Error("ConstructionMetadata: malformed account ID field")
		return nil, ErrInvalidAccountAddress
	}

	// Convert the byte value of the ID to account address.
	var owner staking.Address
	err := owner.UnmarshalText([]byte(idString))
	if err != nil {
		loggerCons.Error("ConstructionMetadata: invalid account ID", "err", err)
		return nil, ErrInvalidAccountAddress
	}

	nonce, err := s.besuClient.GetNextNonce(ctx, owner, oc.LatestHeight)
	if err != nil {
		loggerCons.Error("ConstructionMetadata: unable to get next nonce",
			"account_id", owner.String(),
			"err", err,
		)
		return nil, ErrUnableToGetNextNonce
	}

	// Return next nonce that should be used to sign transactions for given account.
	md := make(map[string]interface{})
	md[NonceKey] = nonce

	resp := &types.ConstructionMetadataResponse{
		Metadata: md,
	}

	jr, _ := json.Marshal(resp)
	loggerCons.Debug("ConstructionMetadata OK", "response", jr)

	return resp, nil
}

// ConstructionSubmit implements the /construction/submit endpoint.
func (s *constructionAPIService) ConstructionSubmit(
	ctx context.Context,
	request *types.ConstructionSubmitRequest,
) (*types.ConstructionSubmitResponse, *types.Error) {
	terr := ValidateNetworkIdentifier(ctx, s.besuClient, request.NetworkIdentifier)
	if terr != nil {
		loggerCons.Error("ConstructionSubmit: network validation failed", "err", terr.Message)
		return nil, terr
	}

	if err := s.besuClient.SubmitTx(ctx, request.SignedTransaction); err != nil {
		loggerCons.Error("ConstructionSubmit: SubmitTx failed", "err", err)
		return nil, ErrUnableToSubmitTx
	}

	// TODO: Does this match the hashes we actually use in consensus?
	var h hash.Hash
	h.From(request.SignedTransaction)
	txID := h.String()

	resp := &types.ConstructionSubmitResponse{
		TransactionIdentifier: &types.TransactionIdentifier{
			Hash: txID,
		},
	}

	jr, _ := json.Marshal(resp)
	loggerCons.Debug("ConstructionSubmit OK", "response", jr)

	return resp, nil
}
