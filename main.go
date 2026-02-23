package main

import (
	webapp "github.com/blocktree/go-openw-sdk/v2/web"
	"github.com/blocktree/go-openw-sdk/v2/web/common"
)

// 测试参数
// 服务端私钥： x97SIkFTlNUpTsNAdsHURjFQZ3BLL/iQ8bBW+hHV7mk=
// 服务端公钥： BDTL1IlMt+k2glN0Rnwzt7hX8cxWougeorB7hBTTheAqNELXRGTln6oPzqvL0WMhHkruudnFGMAemYsEby8iu80=
// 客户端私钥： uckgLxKoRjSHKjlsqa1gfYlHmza0DTRl/cRdV6DEaNY=
// 客户端公钥： BNlt+QZ0StVPUuVlY5SLEijB0J51PvtMYK/F3Nd4HSs7khBFQhvyUosm/DG1mmspYiOS0Zd/yCFd4uyWQR7YWEI=

func main() {
	// 初始化配置文件
	common.NewBaseConfig("config.yaml", "cli-http")
	// 启动交互式控制台
	webapp.RunApplication()
}
