package tx

import (
	errorsmod "cosmossdk.io/errors"
	"fmt"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	types "github.com/cosmos/cosmos-sdk/types/tx"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/cosmos/gogoproto/jsonpb"
	"github.com/ethereum/go-ethereum/common/math"
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

	if len(tx.GetMsgs()) == 1 {
		handler := signModeEip712Handler{}
		return handler.GetSignBytes(mode, signerData, tx)
	}

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

func GetMultiMsgTypes(signerData signing.SignerData, tx sdk.Tx, typedChainID *big.Int) (apitypes.Types, *types.SignDocEip712Multi, error) {
	protoTx, ok := tx.(*wrapper)
	if !ok {
		return nil, nil, fmt.Errorf("can only handle a protobuf Tx, got %T", tx)
	}

	// construct the signDoc
	msgAnys := make([]*codectypes.Any, 0)
	for _, msg := range protoTx.GetMsgs() {
		msgAny, _ := codectypes.NewAnyWithValue(msg)
		msgAnys = append(msgAnys, msgAny)
	}

	signDoc := &types.SignDocEip712Multi{
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
		Msg:  msgAnys,
	}

	msgTypes := apitypes.Types{
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

	for i, _ := range protoTx.GetMsgs() {
		msgTypes["Tx"] = append(msgTypes["Tx"], apitypes.Type{
			Name: fmt.Sprintf("msg-%d", i+1),
			Type: "string",
		})
	}

	return msgTypes, signDoc, nil
}

func WrapMultiTxToTypedData(
	chainID uint64,
	signDoc *types.SignDocEip712Multi,
	msgTypes apitypes.Types,
) (apitypes.TypedData, error) {

	messages := apitypes.TypedDataMessage{}
	for i, msg := range signDoc.GetMsg() {
		msgCodec := jsonpb.Marshaler{
			EmitDefaults: true,
			OrigName:     true,
		}
		bz, err := msgCodec.MarshalToString(msg)
		if err != nil {
			return apitypes.TypedData{}, errorsmod.Wrap(err, "failed to JSON marshal data")
		}
		if err != nil {
			panic(err)
		}
		messages[fmt.Sprintf("msg-%d", i+1)] = bz
	}

	tempDomain := *domain
	tempDomain.ChainId = math.NewHexOrDecimal256(int64(chainID))
	typedData := apitypes.TypedData{
		Types:       msgTypes,
		PrimaryType: "Tx",
		Domain:      tempDomain,
		Message:     messages,
	}

	fmt.Printf("typedData %v \n", typedData)

	return typedData, nil
}
