package testutil

import (
	"fmt"
	"math/big"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"

	"github.com/mage-coven/fury/app"
	"github.com/mage-coven/fury/tests/e2e/runner"
)

const (
	FundedAccountName = "whale"
	// use coin type 60 so we are compatible with accounts from `fury add keys --eth <name>`
	// these accounts use the ethsecp256k1 signing algorithm that allows the signing client
	// to manage both sdk & evm txs.
	Bip44CoinType = 60

	IbcPort    = "transfer"
	IbcChannel = "channel-0"
)

type E2eTestSuite struct {
	suite.Suite

	config SuiteConfig
	runner runner.NodeRunner

	Fury *Chain
	Ibc  *Chain

	UpgradeHeight        int64
	DeployedErc20Address common.Address
}

func (suite *E2eTestSuite) SetupSuite() {
	var err error
	fmt.Println("setting up test suite.")
	app.SetSDKConfig()

	suiteConfig := ParseSuiteConfig()
	suite.config = suiteConfig
	suite.UpgradeHeight = suiteConfig.FuryUpgradeHeight
	suite.DeployedErc20Address = common.HexToAddress(suiteConfig.FuryErc20Address)

	runnerConfig := runner.Config{
		FuryConfigTemplate: suiteConfig.FuryConfigTemplate,

		IncludeIBC: suiteConfig.IncludeIbcTests,
		ImageTag:   "local",

		EnableAutomatedUpgrade:  suiteConfig.IncludeAutomatedUpgrade,
		FuryUpgradeName:         suiteConfig.FuryUpgradeName,
		FuryUpgradeHeight:       suiteConfig.FuryUpgradeHeight,
		FuryUpgradeBaseImageTag: suiteConfig.FuryUpgradeBaseImageTag,

		SkipShutdown: suiteConfig.SkipShutdown,
	}
	suite.runner = runner.NewFuryNode(runnerConfig)

	chains := suite.runner.StartChains()
	furychain := chains.MustGetChain("fury")
	suite.Fury, err = NewChain(suite.T(), furychain, suiteConfig.FundedAccountMnemonic)
	if err != nil {
		suite.runner.Shutdown()
		suite.T().Fatalf("failed to create fury chain querier: %s", err)
	}

	if suiteConfig.IncludeIbcTests {
		ibcchain := chains.MustGetChain("ibc")
		suite.Ibc, err = NewChain(suite.T(), ibcchain, suiteConfig.FundedAccountMnemonic)
		if err != nil {
			suite.runner.Shutdown()
			suite.T().Fatalf("failed to create ibc chain querier: %s", err)
		}
	}

	suite.InitFuryEvmData()
}

func (suite *E2eTestSuite) TearDownSuite() {
	fmt.Println("tearing down test suite.")
	// close all account request channels
	suite.Fury.Shutdown()
	if suite.Ibc != nil {
		suite.Ibc.Shutdown()
	}
	// gracefully shutdown docker container(s)
	suite.runner.Shutdown()
}

func (suite *E2eTestSuite) SkipIfIbcDisabled() {
	if !suite.config.IncludeIbcTests {
		suite.T().SkipNow()
	}
}

func (suite *E2eTestSuite) SkipIfUpgradeDisabled() {
	if !suite.config.IncludeAutomatedUpgrade {
		suite.T().SkipNow()
	}
}

// FuryHomePath returns the OS-specific filepath for the fury home directory
// Assumes network is running with kvtool installed from the sub-repository in tests/e2e/kvtool
func (suite *E2eTestSuite) FuryHomePath() string {
	return filepath.Join("kvtool", "full_configs", "generated", "fury", "initstate", ".fury")
}

// BigIntsEqual is a helper method for comparing the equality of two big ints
func (suite *E2eTestSuite) BigIntsEqual(expected *big.Int, actual *big.Int, msg string) {
	suite.Truef(expected.Cmp(actual) == 0, "%s (expected: %s, actual: %s)", msg, expected.String(), actual.String())
}
