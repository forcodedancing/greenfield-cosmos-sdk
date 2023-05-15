package tx

import (
	errorsmod "cosmossdk.io/errors"
	"encoding/json"
	"fmt"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	types "github.com/cosmos/cosmos-sdk/types/tx"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"math/big"
)

// signModeEip712MultiHandler defines the SIGN_MODE_DIRECT SignModeHandler
type signModeEip712MultiHandler struct{}

var _ signing.SignModeHandler = signModeEip712MultiHandler{}

// DefaultMode implements SignModeHandler.DefaultMode
func (signModeEip712MultiHandler) DefaultMode() signingtypes.SignMode {
	return signingtypes.SignMode_SIGN_MODE_EIP_712
}

// Modes implements SignModeHandler.Modes
func (signModeEip712MultiHandler) Modes() []signingtypes.SignMode {
	return []signingtypes.SignMode{signingtypes.SignMode_SIGN_MODE_EIP_712}
}

// GetSignBytes implements SignModeHandler.GetSignBytes
func (signModeEip712MultiHandler) GetSignBytes(mode signingtypes.SignMode, signerData signing.SignerData, tx sdk.Tx) ([]byte, error) {
	if mode != signingtypes.SignMode_SIGN_MODE_EIP_712 {
		return nil, fmt.Errorf("expected %s, got %s", signingtypes.SignMode_SIGN_MODE_EIP_712, mode)
	}

	// get the EIP155 chainID from the signerData
	chainID, err := sdk.ParseChainID(signerData.ChainID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse chainID: %s", signerData.ChainID)
	}

	// get the EIP712 types and signDoc from the tx
	msgTypes, signDoc, err := GetMultiMsgTypes(signerData, tx, chainID)
	if err != nil {
		return nil, errorsmod.Wrapf(err, "failed to get msg types")
	}

	// pack the tx data in EIP712 object
	typedData, err := WrapMultiTxToTypedData(chainID.Uint64(), signDoc, msgTypes)
	if err != nil {
		return nil, errorsmod.Wrapf(err, "failed to pack tx data in EIP712 object")
	}

	// compute the hash
	sigHash, err := ComputeTypedDataHash(typedData)
	if err != nil {
		return nil, err
	}

	return sigHash, nil
}

func GetMultiMsgTypes(signerData signing.SignerData, tx sdk.Tx, typedChainID *big.Int) ([]apitypes.Types, *types.SignDocEip712Multi, error) {
	protoTx, ok := tx.(*wrapper)
	if !ok {
		return nil, nil, fmt.Errorf("can only handle a protobuf Tx, got %T", tx)
	}

	var msgTypes = make([]apitypes.Types, 0)
	var signDocs = new(types.SignDocEip712Multi)

	// construct the signDoc
	for _, msg := range protoTx.GetMsgs() {
		msgAny, _ := codectypes.NewAnyWithValue(msg)
		signDoc := &types.SignDocEip712{
			AccountNumber: signerData.AccountNumber,
			Sequence:      signerData.Sequence,
			ChainId:       typedChainID.Uint64(),
			TimeoutHeight: protoTx.GetTimeoutHeight(),
			Fee: types.Fee{
				Amount:   protoTx.GetFee(),
				GasLimit: protoTx.GetGas(),
				Payer:    protoTx.FeePayer().String(),
				Granter:  protoTx.FeeGranter().String(),
			},
			Memo: protoTx.GetMemo(),
			Tip:  protoTx.GetTip(),
			Msg:  msgAny,
		}

		// extract the msg types
		tmpMsgTypes, err := extractMsgTypes(msg)
		if err != nil {
			return nil, nil, err
		}

		// patch the msg types to include `Tip` if it's not empty
		if signDoc.Tip != nil {
			tmpMsgTypes["Tx"] = []apitypes.Type{
				{Name: "account_number", Type: "uint256"},
				{Name: "chain_id", Type: "uint256"},
				{Name: "fee", Type: "Fee"},
				{Name: "memo", Type: "string"},
				{Name: "msg", Type: "Msg"},
				{Name: "sequence", Type: "uint256"},
				{Name: "timeout_height", Type: "uint256"},
				{Name: "tip", Type: "Tip"},
			}
			tmpMsgTypes["Tip"] = []apitypes.Type{
				{Name: "amount", Type: "Coin[]"},
				{Name: "tipper", Type: "string"},
			}
		}
		msgTypes = append(msgTypes, tmpMsgTypes)
		signDocs.Docs = append(signDocs.Docs, signDoc)
	}

	return msgTypes, signDocs, nil
}

func WrapMultiTxToTypedData(
	chainID uint64,
	signDoc *types.SignDocEip712Multi,
	msgTypes []apitypes.Types,
) (apitypes.TypedData, error) {
	ttypes := apitypes.Types{
		"EIP712Domain": {
			{
				Name: "name",
				Type: "string",
			},
			{
				Name: "version",
				Type: "string",
			},
			{
				Name: "chainId",
				Type: "uint256",
			},
			{
				Name: "verifyingContract",
				Type: "string",
			},
			{
				Name: "salt",
				Type: "string",
			},
		},
		"Tx": {},
	}
	messages := apitypes.TypedDataMessage{}
	var d apitypes.TypedDataDomain
	for i, msgType := range msgTypes {
		typedData, err := WrapTxToTypedData(chainID, signDoc.Docs[i], msgType)
		if err != nil {
			return apitypes.TypedData{}, err
		}

		bz, err := json.Marshal(typedData.Message)
		if err != nil {
			panic(err)
		}

		tt := apitypes.Type{}
		tt.Name = fmt.Sprintf("content%d", i+1)
		tt.Type = "string"
		ttypes["Tx"] = []apitypes.Type{tt}
		messages[fmt.Sprintf("content%d", i+1)] = string(bz)
		d = typedData.Domain
	}

	result := apitypes.TypedData{
		Types:       ttypes,
		PrimaryType: "Tx",
		Domain:      d,
		Message:     messages,
	}
	fmt.Printf("typedData %v \n", result)
	return result, nil
}
