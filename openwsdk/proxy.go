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
	"fmt"
	"github.com/blocktree/openwallet/v2/log"
	"github.com/blocktree/openwallet/v2/owtp"
)

// ProxyNode 代理节点，用于承担转发客户端的请求到openw-server，返回结果给客户端
type ProxyNode struct {
	node                 *owtp.OWTPNode
	config               *APINodeConfig
	parent               *APINode
	proxyRequestHandler  func(ctx *owtp.Context) bool //请求前的自定义处理
	proxyResponseHandler func(ctx *owtp.Context) bool //相应后的自定义处理
}

// NewProxyNode 创建一个代理节点实例
func NewProxyNode(config *APINodeConfig) *ProxyNode {

	connectCfg := owtp.ConnectConfig{}
	connectCfg.Address = config.Host
	connectCfg.EnableSSL = config.EnableSSL
	connectCfg.EnableSignature = config.EnableSignature
	connectCfg.ConnectType = config.ConnectType
	node := owtp.NewNode(owtp.NodeConfig{
		Cert:       config.Cert,
		TimeoutSEC: config.TimeoutSEC,
	})

	t := &ProxyNode{
		node:   node,
		config: config,
	}

	return t
}

//StopProxyNode 停止代理节点
func (api *APINode) StopProxyNode() error {

	if api.proxyNode == nil {
		return fmt.Errorf("transmit node is not inited")
	}

	api.proxyNode.Close()
	api.proxyNode = nil

	return nil
}

//ServeProxyNode 开启代理服务
func (api *APINode) ServeProxyNode(address string) (*ProxyNode, error) {

	if api == nil {
		return nil, fmt.Errorf("APINode is not inited")
	}

	if api.proxyNode != nil {
		return nil, fmt.Errorf("proxy node is inited")
	}

	proxyNode := NewProxyNode(&APINodeConfig{
		Host:               address,
		ConnectType:        owtp.HTTP,
		AppID:              api.config.AppID,
		AppKey:             api.config.AppKey,
		Cert:               api.config.Cert,
		EnableSignature:    api.config.EnableSignature,
		EnableKeyAgreement: api.config.EnableKeyAgreement,
	})

	proxyNode.parent = api
	api.proxyNode = proxyNode

	//在准备方法中实现转发
	proxyNode.node.HandlePrepareFunc(proxyNode.proxyServerHandler)

	//开启监听
	proxyNode.Listen()

	return proxyNode, nil
}

//ProxyNode 代理节点
func (api *APINode) ProxyNode() (*ProxyNode, error) {
	if api.proxyNode == nil {
		return nil, fmt.Errorf("proxy node is not inited")
	}
	return api.proxyNode, nil
}

//APINode
func (proxyNode *ProxyNode) APINode() (*APINode, error) {
	if proxyNode.parent == nil {
		return nil, fmt.Errorf("proxy node is not inited")
	}
	return proxyNode.parent, nil
}

//OWTPNode
func (proxyNode *ProxyNode) OWTPNode() (*owtp.OWTPNode, error) {
	if proxyNode.node == nil {
		return nil, fmt.Errorf("proxy node is not inited")
	}
	return proxyNode.node, nil
}

//Listen 启动监听
func (proxyNode *ProxyNode) Listen() {

	//开启监听
	log.Infof("Proxy node IP %s start to listen [%s] connection...", proxyNode.config.Host, proxyNode.config.ConnectType)

	proxyNode.node.Listen(owtp.ConnectConfig{
		Address:     proxyNode.config.Host,
		ConnectType: proxyNode.config.ConnectType,
		//EnableSignature: true,
	})
}

//Close 关闭监听
func (proxyNode *ProxyNode) Close() {
	proxyNode.node.Close()
}

//SetProxyRequestHandler 通过设置请求处理器，你可以在请求被中转前进行一些自定义操作
func (proxyNode *ProxyNode) SetProxyRequestHandler(h func(ctx *owtp.Context) bool) {
	proxyNode.proxyRequestHandler = h
}

//SetProxyResponseHandler 通过设置响应处理器，你可以在响应被中转前进行一些自定义操作
func (proxyNode *ProxyNode) SetProxyResponseHandler(h func(ctx *owtp.Context) bool) {
	proxyNode.proxyResponseHandler = h
}

//proxyServerHandler 代理服务同步请求转发处理器
func (proxyNode *ProxyNode) proxyServerHandler(ctx *owtp.Context) {

	//转发给openw-server节点
	var (
		pass bool
	)

	//代理转发请求前的处理
	if proxyNode.proxyRequestHandler != nil {
		pass = proxyNode.proxyRequestHandler(ctx)
	}

	if pass {
		resp, err := proxyNode.parent.OWTPNode().CallSync(HostNodeID, ctx.Method, ctx.Params().Raw)
		if err != nil {
			ctx.ResponseStopRun(nil, owtp.ErrBadRequest, err.Error())
			return
		}

		ctx.ResponseStopRun(resp.Result, resp.Status, resp.Msg)

		//代理转发相应处理
		if proxyNode.proxyResponseHandler != nil {
			proxyNode.proxyResponseHandler(ctx)
		}
	}
}
