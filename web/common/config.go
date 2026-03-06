package common

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	DIC "github.com/godaddy-x/freego/common"
	"github.com/godaddy-x/freego/utils"
	"github.com/godaddy-x/freego/utils/crypto"
	"github.com/godaddy-x/freego/zlog"
)

type Extract struct {
	AppID            string   `yaml:"appID" json:"appID"`
	AppKey           string   `yaml:"appKey" json:"appKey"`
	WalletDir        string   `yaml:"walletDir" json:"walletDir"`               // 钱包文件夹
	WalletMode       int64    `yaml:"walletMode" json:"walletMode"`             // # 钱包模式,1,3,5模式, 1=本地创建密码钱包 3,5进行分片管理（需要部署多个分片节点）
	SignerBlacklist  []string `yaml:"signerBlacklist" json:"signerBlacklist"`   // 转出地址黑名单
	SignerWhitelist  []string `yaml:"signerWhitelist" json:"signerWhitelist"`   // 交易单JSON签名请求IP白名单
	SummaryWhitelist []string `yaml:"summaryWhitelist" json:"summaryWhitelist"` // 汇总地址白名单
	RemoteWhitelist  []string `yaml:"remoteWhitelist" json:"remoteWhitelist"`   // 业务系统请求IP白名单
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
	if !utils.CheckInt64(defaultAllYamlConfig.Extract.WalletMode, 1, 3, 5) {
		return errors.New("wallet mode invalid [1,3,5]")
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

// WriteKeyFile 写入结构内容到文件
func WriteKeyFile(file string, content []byte) error {
	// Create the keystore directory with appropriate permissions
	// in case it is not present yet.
	const dirPerm = 0700
	if err := os.MkdirAll(filepath.Dir(file), dirPerm); err != nil {
		return err
	}
	// Atomic write: create a temporary hidden file first
	// then move it into place. TempFile assigns mode 0600.
	f, err := ioutil.TempFile(filepath.Dir(file), "."+filepath.Base(file)+".tmp")
	if err != nil {
		return err
	}
	if _, err := f.Write(content); err != nil {
		f.Close()
		os.Remove(f.Name())
		return err
	}
	f.Close()
	return os.Rename(f.Name(), file)
}

const defaultConfigExample = `server:
  cli_main:
    gc_limit_mb: 512
    gc_percent: 30
    port: 9422
    keys:
      - name: %d  # client no
        public_key: "%s" # client public key, private key: %s
        private_key: "%s" # server private key, public key: %s

jwt:
  cli_main:
    token_key: "%s"
    token_alg: "HS256"
    token_typ: "JWT"
    token_exp: 3600 # 1h

logger:
  master:
    layout: 0
    location: "Asia/Shanghai"
    level: "error"
    console: false
    file_config:
      max_size: 50
      max_backups: 10
      max_age: 7
      compress: false

extract:
  appID: "%s"
  appKey: "%s"
  tradeKey: "%s"
  walletDir: "%s"
  remoteWhitelist:
    - "127.0.0.1"
  signerBlacklist:
    - "0x2346f1ca41d0161d26f46ec2885721c28fbf1375"
  summaryWhitelist:
    - "0x2346f1ca41d0161d26f46ec2885721c28fbf1375"
`

func CreateDefaultCliConfigExample() {
	eccObj := crypto.EcdsaObject{}
	if err := eccObj.CreateS256ECDSA(); err != nil {
		panic(err)
	}
	if err := eccObj.CreateS256ECDSA(); err != nil {
		panic(err)
	}
	eccObj2 := crypto.EcdsaObject{}
	if err := eccObj2.CreateS256ECDSA(); err != nil {
		panic(err)
	}
	clientNo := utils.NextIID()
	clientPublicKey := eccObj2.PublicKeyBase64
	clientPrivateKey := eccObj2.PrivateKeyBase64
	serverPrivateKey := eccObj.PrivateKeyBase64
	serverPublicKey := eccObj.PublicKeyBase64
	tokenKey := hex.EncodeToString(utils.GetRandomSecure(64))
	appID := hex.EncodeToString(utils.GetRandomSecure(16))
	appKey := hex.EncodeToString(utils.GetRandomSecure(32))
	tradeKey := hex.EncodeToString(utils.GetRandomSecure(32))
	walletDir := "keys"
	path := filepath.Join("cli_config_example.yaml")
	content := fmt.Sprintf(defaultConfigExample, clientNo, clientPublicKey, clientPrivateKey, serverPrivateKey, serverPublicKey, tokenKey, appID, appKey, tradeKey, walletDir)
	if err := WriteKeyFile(path, []byte(content)); err != nil {
		panic(err)
	}
}
