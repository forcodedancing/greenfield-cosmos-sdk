package testutil

import (
	"cosmossdk.io/math"
	"encoding/base64"
	"fmt"
	"github.com/cosmos/cosmos-sdk/testutil"
	govcli "github.com/cosmos/cosmos-sdk/x/gov/client/cli"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/suite"
	tmcli "github.com/tendermint/tendermint/libs/cli"
	"os"

	"github.com/cosmos/cosmos-sdk/client/flags"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	"github.com/cosmos/cosmos-sdk/testutil/network"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/slashing/client/cli"
)

type testParams struct {
	description string
	proposalID  string
	from        string
}

var impeachTestcases = []testParams{
	{
		description: "impeach the last validator",
		proposalID:  "1",
		from:        "0x7b5Fe22B5446f7C62Ea27B8BD71CeF94e03f3dF2",
	},
	{
		description: "impeach the last validator again",
		proposalID:  "2",
		from:        "0x7b5Fe22B5446f7C62Ea27B8BD71CeF94e03f3dF2",
	},
	{
		description: "impeach the last validator, but invalid from",
		proposalID:  "3",
		from:        "0x8b70dC9B691fCeB4e1c69dF8cbF8c077AD4b5853",
	},
}

type ImpeachTestSuite struct {
	suite.Suite

	cfg         network.Config
	network     *network.Network
	proposalIDs []string
}

func NewImpeachTestSuite(cfg network.Config) *ImpeachTestSuite {
	return &ImpeachTestSuite{cfg: cfg}
}

// SetupSuite executes bootstrapping logic before all the tests, i.e. once before
// the entire suite, start executing.
func (s *ImpeachTestSuite) SetupSuite() {
	s.T().Log("setting up e2e test suite")

	var err error
	s.network, err = network.New(s.T(), s.T().TempDir(), s.cfg)
	s.Require().NoError(err)

	s.Require().NoError(s.network.WaitForNextBlock())

	for _, testcase := range impeachTestcases {
		s.submitProposal(testcase)
		s.voteProposal(testcase.proposalID)
		s.proposalIDs = append(s.proposalIDs, testcase.proposalID)
	}
	clientCtx := s.network.Validators[0].ClientCtx
	args := []string{impeachTestcases[1].proposalID, fmt.Sprintf("--%s=json", tmcli.OutputFlag)}
	out, err := clitestutil.ExecTestCLICmd(clientCtx, govcli.GetCmdQueryProposal(), args)
	println(out.String())

	args2 := []string{fmt.Sprintf("--%s=json", tmcli.OutputFlag)}
	out2, _ := clitestutil.ExecTestCLICmd(clientCtx, govcli.GetCmdQueryProposals(), args2)
	println(out2.String())
}

// TearDownSuite performs cleanup logic after all the tests, i.e. once after the
// entire suite, has finished executing.
func (s *ImpeachTestSuite) TearDownSuite() {
	s.T().Log("tearing down e2e test suite")
	s.network.Cleanup()
}

func (s *ImpeachTestSuite) TestImpeachValidatorSuccessful() {
	proposalID := s.proposalIDs[0]
	targetVal := s.network.Validators[len(s.network.Validators)-1]
	clientCtx := s.network.Validators[0].ClientCtx

	// query proposal
	args := []string{proposalID, fmt.Sprintf("--%s=json", tmcli.OutputFlag)}
	out, err := clitestutil.ExecTestCLICmd(clientCtx, govcli.GetCmdQueryProposal(), args)
	s.Require().NoError(err)
	var proposal v1.Proposal
	s.Require().NoError(clientCtx.Codec.UnmarshalJSON(out.Bytes(), &proposal), out.String())
	s.Require().Equal(v1.ProposalStatus_PROPOSAL_STATUS_PASSED, proposal.Status, out.String())

	// query validator
	queryCmd := cli.GetQueryCmd()
	res, err := clitestutil.ExecTestCLICmd(
		clientCtx, queryCmd,
		[]string{targetVal.Address.String(), fmt.Sprintf("--%s=json", tmcli.OutputFlag)},
	)
	s.Require().NoError(err)
	var result types.Validator
	s.Require().NoError(clientCtx.Codec.UnmarshalJSON(res.Bytes(), &result))
	s.Require().NotEqual(result.GetStatus(), types.Bonded, fmt.Sprintf("validator %s not in bonded status", targetVal.Address.String()))
	s.Require().Equal(result.Jailed, true)
}

func (s *ImpeachTestSuite) TestDoubleImpeachValidator() {
	proposalID := s.proposalIDs[1]
	targetVal := s.network.Validators[len(s.network.Validators)-1]
	clientCtx := s.network.Validators[0].ClientCtx

	// query proposal
	args := []string{proposalID, fmt.Sprintf("--%s=json", tmcli.OutputFlag)}
	out, err := clitestutil.ExecTestCLICmd(clientCtx, govcli.GetCmdQueryProposal(), args)
	s.Require().NoError(err)
	var proposal v1.Proposal
	s.Require().NoError(clientCtx.Codec.UnmarshalJSON(out.Bytes(), &proposal), out.String())
	s.Require().Equal(v1.ProposalStatus_PROPOSAL_STATUS_PASSED, proposal.Status, out.String())

	// query validator
	queryCmd := cli.GetQueryCmd()
	res, err := clitestutil.ExecTestCLICmd(
		clientCtx, queryCmd,
		[]string{targetVal.Address.String(), fmt.Sprintf("--%s=json", tmcli.OutputFlag)},
	)
	s.Require().NoError(err)
	var result types.Validator
	s.Require().NoError(clientCtx.Codec.UnmarshalJSON(res.Bytes(), &result))
	s.Require().NotEqual(result.GetStatus(), types.Bonded, fmt.Sprintf("validator %s not in bonded status", targetVal.Address.String()))
	s.Require().Equal(result.Jailed, true)
}

func (s *ImpeachTestSuite) TestQueryInvalidFromAddress() {
	proposalID := s.proposalIDs[2]
	clientCtx := s.network.Validators[0].ClientCtx

	// query proposal, should not be found because of invalid from the proposal will be rejected.
	args := []string{proposalID, fmt.Sprintf("--%s=json", tmcli.OutputFlag)}
	_, err := clitestutil.ExecTestCLICmd(clientCtx, govcli.GetCmdQueryProposal(), args)
	s.Require().Error(err)
}

func (s *ImpeachTestSuite) submitProposal(params testParams) {
	val := s.network.Validators[0]
	clientCtx := val.ClientCtx

	// Always impeach the last validator.
	targetVal := s.network.Validators[len(s.network.Validators)-1]

	args := []string{
		s.impeachValidatorProposal(targetVal.Address, params.from).Name(),
		fmt.Sprintf("--%s=%s", flags.FlagFrom, val.ValAddress.String()),
		fmt.Sprintf("--gas=%s", fmt.Sprintf("%d", flags.DefaultGasLimit+100000)),
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(10))).String()),
	}

	res, err := clitestutil.ExecTestCLICmd(clientCtx, govcli.NewCmdSubmitProposal(), args)
	println(fmt.Sprintf("res=%s", res.String()))
	println(fmt.Errorf("err=%s", err))
	s.Require().NoError(err)
}

func (s *ImpeachTestSuite) impeachValidatorProposal(valAddr sdk.AccAddress, from string) *os.File {
	propMetadata := []byte{}
	//propMetadata := fmt.Sprintf(`
	//{
	//		"title": %s,
	//		"authors": [""],
	//		"summary": %s,
	//		"details": "",
	//		"proposal_forum_url": "",
	//		"vote_option_context": "",
	//}
	//	`,
	//		"Impeach Validator",
	//		"Impeach Validator",
	//	)

	proposal := fmt.Sprintf(`
{
	"messages": [
		{
			"@type": "/cosmos.slashing.v1beta1.MsgImpeach",
			"from":"%s",
			"validator_address":"%s"
		}
	],
	"metadata": "%s",
	"title": "Impeach Validator",
	"authors": [""],
	"summary": "Impeach Validator",
	"details": "",
	"proposal_forum_url": "",
	"vote_option_context": "",
	"deposit": "%s"
}`,
		from,
		valAddr.String(),
		base64.StdEncoding.EncodeToString(propMetadata),
		sdk.NewCoin(s.cfg.BondDenom, math.NewInt(10000000)),
	)

	proposalFile := testutil.WriteToNewTempFile(s.T(), proposal)
	return proposalFile
}

func (s *ImpeachTestSuite) voteProposal(proposalID string) {
	for i := 0; i < len(s.network.Validators); i++ {
		clientCtx := s.network.Validators[i].ClientCtx
		clientCtx.Client = s.network.Validators[0].RPCClient

		// The last validator vote no, others vote yes.
		voteOption := "yes"
		if i == len(s.network.Validators)-1 {
			voteOption = "no"
		}

		args := []string{
			proposalID,
			voteOption,
			fmt.Sprintf("--%s=%s", flags.FlagFrom, s.network.Validators[i].Address.String()),
			fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
			fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
			fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(10))).String()),
		}

		_, err := clitestutil.ExecTestCLICmd(clientCtx, govcli.NewCmdVote(), args)
		s.Require().NoError(err)
	}
}
