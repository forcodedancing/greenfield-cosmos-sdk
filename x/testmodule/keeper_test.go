package testmodule

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestTestKeeper(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockBankKeeper := NewMockBankKeeper(ctrl)

	testKeeper := NewTestKeeper(mockBankKeeper)

	mockBankKeeper.EXPECT().TestFunc(gomock.Any()).Return(1)

	require.Equal(t, 1, testKeeper.TestBankKeeper(1))

	mockBankKeeper.EXPECT().TestFunc(gomock.Eq(10)).Return(10).AnyTimes()
	mockBankKeeper.EXPECT().TestFunc(gomock.Eq(20)).Return(20).AnyTimes()

	require.Equal(t, 10, testKeeper.TestBankKeeper(10))
	require.Equal(t, 20, testKeeper.TestBankKeeper(20))
}
