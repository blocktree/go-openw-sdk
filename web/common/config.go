package common

import (
	"errors"
	DIC "github.com/godaddy-x/freego/common"
	"github.com/godaddy-x/freego/utils"
	"github.com/godaddy-x/freego/zlog"
	"time"
)

type Extract struct {
	AppID            string   `yaml:"appID" json:"appID"`
	AppKey           string   `yaml:"appKey" json:"appKey"`
	TradeKey         string   `yaml:"tradeKey" json:"tradeKey"`                 // 交易单签名校验
	WalletDir        string   `yaml:"walletDir" json:"walletDir"`               // 钱包文件夹
	SubmitBlacklist  []string `yaml:"submitBlacklist" json:"submitBlacklist"`   // 签名黑名单
	SummaryWhitelist []string `yaml:"summaryWhitelist" json:"summaryWhitelist"` // 汇总白名单
	RemoteWhitelist  []string `yaml:"remoteWhitelist" json:"remoteWhitelist"`   // 请求白名单
}

type YamlConfigExtract struct {
	DIC.YamlConfig `yaml:",inline"`
	Extract        *Extract `yaml:"extract,omitempty"`
}

var defaultAllYamlConfig *YamlConfigExtract

func InitAllConfig(path string) (err error) {
	defaultAllYamlConfig = &YamlConfigExtract{}
	if err := utils.ReadLocalYamlConfig(path, defaultAllYamlConfig); err != nil {
		return err
	}
	return nil
}

func GetAllConfig() *YamlConfigExtract {
	if defaultAllYamlConfig == nil || !defaultAllYamlConfig.CheckReady() {
		panic(errors.New("yaml config not ready"))
	}
	return defaultAllYamlConfig
}

func InitLogger(config *DIC.ZapConfig, fileName string) {
	if config == nil {
		panic("Log config cannot be nil")
	}
	c := zlog.ZapConfig{}
	c.Layout = config.Layout
	if len(config.Location) > 0 {
		loc, err := time.LoadLocation(config.Location)
		if err != nil {
			panic("zap log location error: " + err.Error())
		}
		c.Location = loc
	}
	c.Level = config.Level
	c.Console = config.Console
	// Only set file config if provided, otherwise logs will output to console only
	if config.FileConfig != nil {
		c.FileConfig = &zlog.FileConfig{
			Compress:   config.FileConfig.Compress,
			Filename:   config.FileConfig.Filename + fileName,
			MaxAge:     config.FileConfig.MaxAge,
			MaxBackups: config.FileConfig.MaxBackups,
			MaxSize:    config.FileConfig.MaxSize,
		}
	}
	zlog.InitDefaultLog(&c)
}

// NewBaseConfig projectName：项目名称对应yaml配置节点 logFileName：日志输出文件名
func NewBaseConfig(configName, logFileName string) {
	// 初始化配置文件
	if err := InitAllConfig(configName); err != nil {
		panic(errors.New("read config error: " + err.Error()))
	}
	config := GetAllConfig()
	// Initialize default logger
	InitLogger(config.GetLoggerConfig(DIC.MASTER), utils.AddStr(logFileName, ".log"))
}
