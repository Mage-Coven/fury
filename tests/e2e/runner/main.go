package runner

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/cenkalti/backoff/v4"
)

type Config struct {
	FuryConfigTemplate string

	ImageTag   string
	IncludeIBC bool

	EnableAutomatedUpgrade  bool
	FuryUpgradeName         string
	FuryUpgradeHeight       int64
	FuryUpgradeBaseImageTag string

	SkipShutdown bool
}

// NodeRunner is responsible for starting and managing docker containers to run a node.
type NodeRunner interface {
	StartChains() Chains
	Shutdown()
}

// FuryNodeRunner manages and runs a single Fury node.
type FuryNodeRunner struct {
	config    Config
	furyChain *ChainDetails
}

var _ NodeRunner = &FuryNodeRunner{}

func NewFuryNode(config Config) *FuryNodeRunner {
	return &FuryNodeRunner{
		config: config,
	}
}

func (k *FuryNodeRunner) StartChains() Chains {
	installKvtoolCmd := exec.Command("./scripts/install-kvtool.sh")
	installKvtoolCmd.Stdout = os.Stdout
	installKvtoolCmd.Stderr = os.Stderr
	if err := installKvtoolCmd.Run(); err != nil {
		panic(fmt.Sprintf("failed to install kvtool: %s", err.Error()))
	}

	log.Println("starting fury node")
	kvtoolArgs := []string{"testnet", "bootstrap", "--fury.configTemplate", k.config.FuryConfigTemplate}
	if k.config.IncludeIBC {
		kvtoolArgs = append(kvtoolArgs, "--ibc")
	}
	if k.config.EnableAutomatedUpgrade {
		kvtoolArgs = append(kvtoolArgs,
			"--upgrade-name", k.config.FuryUpgradeName,
			"--upgrade-height", fmt.Sprint(k.config.FuryUpgradeHeight),
			"--upgrade-base-image-tag", k.config.FuryUpgradeBaseImageTag,
		)
	}
	startFuryCmd := exec.Command("kvtool", kvtoolArgs...)
	startFuryCmd.Env = os.Environ()
	startFuryCmd.Env = append(startFuryCmd.Env, fmt.Sprintf("FURY_TAG=%s", k.config.ImageTag))
	startFuryCmd.Stdout = os.Stdout
	startFuryCmd.Stderr = os.Stderr
	log.Println(startFuryCmd.String())
	if err := startFuryCmd.Run(); err != nil {
		panic(fmt.Sprintf("failed to start fury: %s", err.Error()))
	}

	k.furyChain = &furyChain

	err := k.waitForChainStart()
	if err != nil {
		k.Shutdown()
		panic(err)
	}
	log.Println("fury is started!")

	chains := NewChains()
	chains.Register("fury", k.furyChain)
	if k.config.IncludeIBC {
		chains.Register("ibc", &ibcChain)
	}
	return chains
}

func (k *FuryNodeRunner) Shutdown() {
	if k.config.SkipShutdown {
		log.Printf("would shut down but SkipShutdown is true")
		return
	}
	log.Println("shutting down fury node")
	shutdownFuryCmd := exec.Command("kvtool", "testnet", "down")
	shutdownFuryCmd.Stdout = os.Stdout
	shutdownFuryCmd.Stderr = os.Stderr
	if err := shutdownFuryCmd.Run(); err != nil {
		panic(fmt.Sprintf("failed to shutdown kvtool: %s", err.Error()))
	}
}

func (k *FuryNodeRunner) waitForChainStart() error {
	// exponential backoff on trying to ping the node, timeout after 30 seconds
	b := backoff.NewExponentialBackOff()
	b.MaxInterval = 5 * time.Second
	b.MaxElapsedTime = 30 * time.Second
	if err := backoff.Retry(k.pingFury, b); err != nil {
		return fmt.Errorf("failed to start & connect to chain: %s", err)
	}
	b.Reset()
	// the evm takes a bit longer to start up. wait for it to start as well.
	if err := backoff.Retry(k.pingEvm, b); err != nil {
		return fmt.Errorf("failed to start & connect to chain: %s", err)
	}
	return nil
}

func (k *FuryNodeRunner) pingFury() error {
	log.Println("pinging fury chain...")
	url := fmt.Sprintf("http://localhost:%s/status", k.furyChain.RpcPort)
	res, err := http.Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		return fmt.Errorf("ping to status failed: %d", res.StatusCode)
	}
	log.Println("successfully started Fury!")
	return nil
}

func (k *FuryNodeRunner) pingEvm() error {
	log.Println("pinging evm...")
	url := fmt.Sprintf("http://localhost:%s", k.furyChain.EvmPort)
	res, err := http.Get(url)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	// when running, it should respond 405 to a GET request
	if res.StatusCode != 405 {
		return fmt.Errorf("ping to evm failed: %d", res.StatusCode)
	}
	log.Println("successfully pinged EVM!")
	return nil
}
