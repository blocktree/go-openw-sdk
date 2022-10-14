module github.com/blocktree/go-openw-sdk/v2

go 1.12

require (
	github.com/astaxie/beego v1.12.0
	github.com/blocktree/go-owcdrivers v1.2.22 // indirect
	github.com/blocktree/go-owcrypt v1.1.7
	github.com/blocktree/openwallet/v2 v2.4.3
	github.com/google/uuid v1.2.0
	github.com/tidwall/gjson v1.9.3
)

//replace github.com/blocktree/go-owcdrivers => ../go-owcdrivers
//replace github.com/blocktree/openwallet/v2 => ../openwallet
