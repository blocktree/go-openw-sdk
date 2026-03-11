package main

import (
	"testing"
	"time"

	"github.com/blocktree/go-openw-sdk/v2/openwsdk"
)

func TestRunAllNode(t *testing.T) {
	go func() {
		cliConfig := openwsdk.ReadJson("cli_node0.json")
		RunMPCNode(cliConfig)
	}()
	go func() {
		cliConfig := openwsdk.ReadJson("cli_node1.json")
		RunMPCNode(cliConfig)
	}()
	go func() {
		cliConfig := openwsdk.ReadJson("cli_node2.json")
		RunMPCNode(cliConfig)
	}()
	time.Sleep(2000 * time.Second)
}
