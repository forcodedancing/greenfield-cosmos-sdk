package tx

import (
	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/signing"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestEIP712MultiHandler(t *testing.T) {
	privKey, pubkey, addr := testdata.KeyTestPubAddrEthSecp256k1(require.New(t))
	_, feePayerPubKey, feePayerAddr := testdata.KeyTestPubAddrEthSecp256k1(require.New(t))
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	interfaceRegistry.RegisterImplementations((*sdk.Msg)(nil), &banktypes.MsgSend{})
	marshaler := codec.NewProtoCodec(interfaceRegistry)

	txConfig := NewTxConfig(marshaler, []signingtypes.SignMode{signingtypes.SignMode_SIGN_MODE_EIP_712})
	txBuilder := txConfig.NewTxBuilder()

	chainID := "greenfield_9000"
	testMemo := "some test memo"
	testMsg1 := banktypes.NewMsgSend(addr, addr, sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(1))))
	testMsg2 := banktypes.NewMsgSend(addr, addr, sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(22222))))
	accNum, accSeq := uint64(1), uint64(2)

	sigData := &signingtypes.SingleSignatureData{
		SignMode: signingtypes.SignMode_SIGN_MODE_EIP_712,
	}

	sig := signingtypes.SignatureV2{
		PubKey:   pubkey,
		Data:     sigData,
		Sequence: accSeq,
	}
	feePayerSig := signingtypes.SignatureV2{
		PubKey:   feePayerPubKey,
		Data:     sigData,
		Sequence: accSeq,
	}

	fee := txtypes.Fee{Amount: sdk.NewCoins(sdk.NewInt64Coin("atom", 150)), GasLimit: 20000}
	tip := &txtypes.Tip{Amount: sdk.NewCoins(sdk.NewInt64Coin("tip-token", 10))}

	err := txBuilder.SetMsgs(testMsg1, testMsg2)
	require.NoError(t, err)
	txBuilder.SetMemo(testMemo)
	txBuilder.SetFeeAmount(fee.Amount)
	txBuilder.SetFeePayer(feePayerAddr)
	txBuilder.SetGasLimit(fee.GasLimit)
	txBuilder.SetTip(tip)

	err = txBuilder.SetSignatures(sig, feePayerSig)
	require.NoError(t, err)

	signingData := signing.SignerData{
		Address:       addr.String(),
		ChainID:       chainID,
		AccountNumber: accNum,
		Sequence:      accSeq,
		PubKey:        pubkey,
	}

	modeHandler := signModeEip712MultiHandler{}

	//t.Log("verify invalid chain ID")
	//_, err = modeHandler.GetSignBytes(signingtypes.SignMode_SIGN_MODE_EIP_712, signingData, txBuilder.GetTx())
	//require.EqualError(t, err, fmt.Sprintf("failed to parse chainID: %s", signingData.ChainID))

	t.Log("verify GetSignBytes correct")
	signingData.ChainID = "greenfield_9000-1"
	signBytes, err := modeHandler.GetSignBytes(signingtypes.SignMode_SIGN_MODE_EIP_712, signingData, txBuilder.GetTx())
	require.NoError(t, err)
	require.NotNil(t, signBytes)

	t.Log("verify that setting signature doesn't change sign bytes")
	expectedSignBytes := signBytes
	sigData.Signature, err = privKey.Sign(signBytes)
	require.NoError(t, err)
	err = txBuilder.SetSignatures(sig)
	require.NoError(t, err)
	signBytes, err = modeHandler.GetSignBytes(signingtypes.SignMode_SIGN_MODE_EIP_712, signingData, txBuilder.GetTx())
	require.NoError(t, err)
	require.Equal(t, expectedSignBytes, signBytes)
}
