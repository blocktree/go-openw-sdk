package webapp

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"time"

	"github.com/awnumar/memguard"
	"github.com/blocktree/go-openw-sdk/v2/openwsdk/dto"
	"github.com/blocktree/go-openw-sdk/v2/web/common"
	"github.com/blocktree/openwallet/v2/hdkeystore"
	"github.com/codahale/sss"
	ecc "github.com/godaddy-x/eccrypto"
	"github.com/godaddy-x/freego/cache"
	"github.com/godaddy-x/freego/node"
	"github.com/godaddy-x/freego/utils"
	"github.com/godaddy-x/freego/utils/crypto"
	"github.com/godaddy-x/freego/utils/jwt"
)

var (
	server   *node.WsServer
	keyCache = cache.NewLocalCache(1, 1)
)

type TaskSharding struct {
	TaskID       string
	ShardKey     string // 客户端公钥加密数据
	ShardKeySize int
	ExpiredTime  int64
}

// CreateShardingTask
// 分片数据下发流程：0.任务已创建 10.服务端已下发上传公钥通知 20.节点已上传公钥 30.服务端已下发拉取分片数据通知 40.节点拉取分片数据成功
func CreateShardingTask() (string, error) {
	if server == nil {
		return "", errors.New("ws server not initialized")
	}

	nodes := server.GetConnManager().GetAllSubjectDevices()

	subjects := make([]string, 0, len(nodes))
	for s := range nodes {
		subjects = append(subjects, s)
	}
	sort.Strings(subjects) // 确保顺序稳定

	if !utils.CheckInt(len(subjects), 3, 5) {
		return "", errors.New(utils.AddStr("online nodes: ", len(subjects), " (3 or 5 required)"))
	}

	taskID := utils.GetUUID(true)
	expiredTime := utils.UnixSecond() + 15 // 15秒过期,应该每个操作成功重置15秒有效期
	for _, subject := range subjects {
		res := &dto.CliShardingTaskRes{
			TaskID:      taskID,
			ExpiredTime: expiredTime,
			Status:      10,
		}
		if err := server.GetConnManager().SendToSubject(subject, "shardingPre", res); err != nil {
			return "", err
		}
		if err := keyCache.Put(utils.FNV1a64(utils.AddStr(subject, taskID)), res, 20); err != nil {
			return "", err
		}
	}

	// 轮询检查多节点提交公钥状态
	maxWait := time.After(10 * time.Second)
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()

	waitForAll := func() error {
		for {
			select {
			case <-maxWait:
				return errors.New("timeout waiting for all nodes to submit public keys")
			case <-ticker.C:
				allReady := true
				for _, subject := range subjects {
					cacheKey := utils.FNV1a64(utils.AddStr(subject, taskID))
					v, ok, _ := keyCache.Get(cacheKey, nil)
					if !ok || v == nil || v.(*dto.CliShardingTaskRes).Status != 20 {
						allReady = false
						break
					}
				}
				if allReady {
					return nil
				}
			}
		}
	}

	if err := waitForAll(); err != nil {
		return "", err
	}

	seed := memguard.NewBufferRandom(64)
	defer seed.Destroy()

	n := byte(len(subjects))
	var k byte
	if n == 3 {
		k = byte(2)
	} else if n == 5 {
		k = byte(3)
	}

	shares, err := sss.Split(n, k, seed.Bytes())
	if err != nil {
		return "", err
	}

	if len(shares) != len(subjects) {
		return "", errors.New("share length mismatch: " + strconv.Itoa(len(shares)))
	}

	// 提取所有 x 值并排序
	var xs []byte
	for x := range shares {
		xs = append(xs, x)
	}
	sort.Slice(xs, func(i, j int) bool { return xs[i] < xs[j] })

	if len(xs) != len(subjects) {
		return "", errors.New("share sort length mismatch: " + strconv.Itoa(len(shares)))
	}

	keyID := hdkeystore.ComputeKeyID(seed.Bytes())

	// 检查节点公钥提交完成，然后发送通知让节点拉取加密分片数据
	for i, subject := range subjects {
		cacheKey := utils.FNV1a64(utils.AddStr(subject, taskID))
		value, b, err := keyCache.Get(cacheKey, nil)
		if err != nil {
			return "", errors.New("task cache error: " + err.Error())
		}
		if !b || value == nil {
			return "", errors.New("task cache miss: " + subject + " - " + taskID)
		}

		task := value.(*dto.CliShardingTaskRes)

		x := xs[i] // x = 1, 2, 3...
		key := shares[x]

		pub, err := ecc.LoadECDHPublicKeyFromBase64(task.PublicKey)
		if err != nil {
			return "", errors.New("task public key invalid: " + err.Error())
		}

		shardKey, err := ecc.Encrypt(nil, pub.Bytes(), key, utils.Str2Bytes(subject))
		if err != nil {
			return "", errors.New("task encrypt share key error: " + task.TaskID + " - " + subject + " : " + err.Error())
		}
		task.KeyID = keyID
		task.ShardKeySize = len(key)
		task.ShardKeyIndex = int(x)
		task.ShardKey = utils.Base64Encode(shardKey)
		task.ExpiredTime = utils.UnixSecond() + 15
		task.Status = 30
		if err := server.GetConnManager().SendToSubject(subject, "shardingPost", &dto.CliShardingTaskRes{TaskID: taskID}); err != nil {
			return "", err
		}
		if err := keyCache.Put(cacheKey, task, 20); err != nil {
			return "", err
		}
	}

	return keyID, nil
}

// handleShardingPost 节点拉取临时公钥加密的分片数据
func handleShardingPost(ctx context.Context, connCtx *node.ConnectionContext, body []byte) (interface{}, error) {
	request := &dto.CliShardingTaskReq{}
	if err := utils.JsonUnmarshal(body, &request); err != nil {
		return nil, err
	}
	subject := connCtx.GetUserIDString()
	cacheKey := utils.FNV1a64(utils.AddStr(subject, request.TaskID))
	value, b, err := keyCache.Get(cacheKey, nil)
	if err != nil {
		return nil, err
	}
	if !b || value == nil {
		return nil, errors.New("task not found: " + request.TaskID)
	}
	task := value.(*dto.CliShardingTaskRes)
	if task.ExpiredTime < utils.UnixSecond() {
		_ = keyCache.Del(cacheKey)
		return nil, errors.New("task expired: " + request.TaskID)
	}
	task.ExpiredTime = utils.UnixSecond() + 15
	task.Status = 40

	if err := keyCache.Put(cacheKey, task, 20); err != nil {
		return nil, err
	}
	return task, nil
}

func NewSocket() {
	// 创建WebSocket服务器实例
	server = node.NewWsServer(node.SubjectDeviceUnique)

	// 添加JWT配置参数
	jwtConfig := common.GetAllConfig().GetJwtConfig(project)
	_ = server.AddJwtConfig(jwt.JwtConfig{TokenKey: jwtConfig.TokenKey, TokenAlg: jwtConfig.TokenAlg, TokenExp: jwtConfig.TokenExp, TokenTyp: jwtConfig.TokenTyp})

	// 添加系统基本参数
	serverConfig := common.GetAllConfig().GetServerConfig(project)

	// 添加ECDSA配置参数,服务端私钥和客户端公钥
	for _, v := range serverConfig.Keys {
		cipher, err := crypto.CreateS256ECDSAWithBase64(v.PrivateKey, v.PublicKey)
		if err != nil {
			panic("create ecdsa object error: " + err.Error())
		}
		_ = server.AddCipher(v.Name, cipher)
	}

	// 添加Local缓存参数,默认空即可
	server.AddLocalCache(nil)

	// 配置连接池
	if err := server.NewPool(100, 10, 5, 30); err != nil {
		panic(err)
	}

	if err := server.AddRouter("/ws/shardingPost", handleShardingPost, &node.RouterConfig{}); err != nil {
		panic(err)
	}

	if err := server.AddRouter("/ws/mpcTempPublicKey", handleTempPublicKey, &node.RouterConfig{}); err != nil {
		panic(err)
	}

	if err := server.AddRouter("/ws/mpcKeygenResult", handleMpcKeygenResult, &node.RouterConfig{}); err != nil {
		panic(err)
	}
	if err := server.AddRouter("/ws/mpcKeygenMsg", handleMpcKeygenMsg, &node.RouterConfig{}); err != nil {
		panic(err)
	}

	// 分布式签名相关路由
	if err := server.AddRouter("/ws/mpcSignResult", handleMpcSignResult, &node.RouterConfig{}); err != nil {
		panic(err)
	}
	if err := server.AddRouter("/ws/mpcSignMsg", handleMpcSignMsg, &node.RouterConfig{}); err != nil {
		panic(err)
	}

	if err := server.StartWebsocket(utils.AddStr(serverConfig.Addr, ":", serverConfig.Port+100)); err != nil {
		panic(err)
	}

}
