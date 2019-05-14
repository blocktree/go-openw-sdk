module github.com/blocktree/go-openw-sdk

go 1.12

require (
	github.com/astaxie/beego v1.11.1
	github.com/blocktree/go-owcdrivers v1.0.8
	github.com/blocktree/go-owcrypt v1.0.1
	github.com/blocktree/openwallet v1.4.1
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/google/uuid v1.1.1
	github.com/tidwall/gjson v1.2.1
	golang.org/x/net v0.0.0-20190213061140-3a22650c66bd // indirect
)

//replace github.com/blocktree/go-owcdrivers => ../go-owcdrivers
