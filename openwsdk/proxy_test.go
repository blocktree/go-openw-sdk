/*
 * Copyright 2019 The openwallet Authors
 * This file is part of the openwallet library.
 *
 * The openwallet library is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The openwallet library is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Lesser General Public License for more details.
 */

package openwsdk

import (
	"github.com/blocktree/openwallet/v2/log"
	"github.com/blocktree/openwallet/v2/owtp"
	"testing"
)

func TestAPINode_ServeProxyNode(t *testing.T) {

	var (
		endRunning = make(chan bool, 1)
	)

	api := testNewAPINode()
	proxyNode, err := api.ServeProxyNode(":7088")
	if err != nil {
		log.Errorf("ServeProxyNode error: %v\n", err)
		return
	}

	proxyNode.SetProxyRequestHandler(func(ctx *owtp.Context) bool {
		log.Infof("Call ProxyRequestHandler")
		log.Infof("proxy server handle method: %s", ctx.Method)
		log.Infof("request params: %v", ctx.Params())
		return true
	})

	proxyNode.SetProxyResponseHandler(func(ctx *owtp.Context) bool {
		log.Infof("Call ProxyResponseHandler")
		log.Infof("response: %+v", ctx.Resp)
		return true
	})

	<-endRunning
}
