package main

import (
	"flag"
	"fmt"
	"strings"

	webapp "github.com/blocktree/go-openw-sdk/v2/web"
	"github.com/blocktree/go-openw-sdk/v2/web/common"
)

// 测试参数
// 服务端私钥： x97SIkFTlNUpTsNAdsHURjFQZ3BLL/iQ8bBW+hHV7mk=
// 服务端公钥： BDTL1IlMt+k2glN0Rnwzt7hX8cxWougeorB7hBTTheAqNELXRGTln6oPzqvL0WMhHkruudnFGMAemYsEby8iu80=
// 客户端私钥： uckgLxKoRjSHKjlsqa1gfYlHmza0DTRl/cRdV6DEaNY=
// 客户端公钥： BNlt+QZ0StVPUuVlY5SLEijB0J51PvtMYK/F3Nd4HSs7khBFQhvyUosm/DG1mmspYiOS0Zd/yCFd4uyWQR7YWEI=

// 测试钱包密码：dcdba32174fdf3f9a08a48b9fa838b68

func main() {
	// 命令行参数处理
	configFile := flag.String("config", "cli_config.yaml", "configuration file path")
	initConfig := flag.Bool("init", false, "initialize default configuration example file")
	flag.Parse()

	// 初始化默认配置文件
	if *initConfig {
		common.CreateDefaultCliConfigExample()
		fmt.Printf("Default configuration created at: %s\n", "cli_config_example.yaml")
		return
	}

	// 生成日志文件名：配置文件名（去扩展名）+ "_log"
	logFileName := strings.TrimSuffix(*configFile, ".yaml") + "_log"

	// 初始化配置文件
	common.NewBaseConfig(*configFile, logFileName)

	webapp.NewSocket()

	// 启动交互式控制台
	webapp.RunApplication()
}
