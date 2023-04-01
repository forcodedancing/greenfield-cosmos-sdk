package testmodule

type TestKeeper struct {
	bankKeeper BankKeeper
}

func NewTestKeeper(bankKeeper BankKeeper) TestKeeper {
	return TestKeeper{
		bankKeeper: bankKeeper,
	}
}

func (k TestKeeper) TestBankKeeper(a int) int {
	return k.bankKeeper.TestFunc(a)
}
