# go-openw-api-sdk

基于OWTP协议库，封装所有与openw-server钱包服务API交互方法。用于集成到go语言开发下的应用方系统。

## 概述

[TOC]

## Build development environment

The requirements to build OpenWallet are:

- Golang version 1.10 or later
- govendor (a third party package management tool)
- xgo (Go CGO cross compiler)
- Properly configured Go language environment
- Golang supported operating system

## 依赖库管理工具govendor

### 安装govendor

```shell

go get -u -v github.com/kardianos/govendor

```

### 使用govendor

```shell

#进入到项目目录
$ cd $GOPATH/src/github.com/blocktree/OpenWallet

#初始化vendor目录
$ govendor init

#查看vendor目录
[root@CC54425A openwallet]# ls
commands  main.go  vendor

#将GOPATH中本工程使用到的依赖包自动移动到vendor目录中
#说明：如果本地GOPATH没有依赖包，先go get相应的依赖包
$ govendor add +external
或使用缩写： govendor add +e

#Go 1.6以上版本默认开启 GO15VENDOREXPERIMENT 环境变量，可忽略该步骤。
#通过设置环境变量 GO15VENDOREXPERIMENT=1 使用vendor文件夹构建文件。
#可以选择 export GO15VENDOREXPERIMENT=1 或 GO15VENDOREXPERIMENT=1 go build 执行编译
$ export GO15VENDOREXPERIMENT=1

# 如果$GOPATH下已更新本地库，可执行命令以下命令，同步更新vendor包下的库
# 例如本地的$GOPATH/github.com/blocktree/下的组织项目更新后，可执行下面命令同步更新vendor
$ govendor update +v

```

### 依赖blocktree本地库

github.com/blocktree/go-owcrypt
github.com/blocktree/go-owcdrivers
github.com/blocktree/OpenWallet
github.com/blocktree/go-openw-server

## 源码编译跨平台工具

### 安装xgo（支持跨平台编译C代码）

[官方github](https://github.com/karalabe/xgo)

xgo的使用依赖docker。并且把要跨平台编译的项目文件加入到File sharing。

```shell

$ go get github.com/karalabe/xgo
...
$ xgo -h
...

```

---

## go-openw-sdk
   
基于OWTP协议库，封装所有与openw-server钱包服务API交互方法。用于集成到go语言开发下的应用方系统。

---