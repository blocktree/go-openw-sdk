module github.com/blocktree/go-openw-sdk

go 1.12

require (
	github.com/astaxie/beego v1.11.1
	github.com/blocktree/go-owcdrivers v1.0.38
	github.com/blocktree/go-owcrypt v1.0.2
	github.com/blocktree/openwallet v1.4.5
	github.com/google/uuid v1.1.1
	github.com/tidwall/gjson v1.2.1
)

//replace github.com/blocktree/go-owcdrivers => ../go-owcdrivers
//replace github.com/blocktree/openwallet => ../openwallet
