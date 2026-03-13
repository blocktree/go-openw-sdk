package alg_ecdsa_test

import (
	"crypto/sha256"
	"encoding/hex"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/blocktree/go-openw-sdk/v2/mpc/alg_ecdsa"
	"github.com/bnb-chain/tss-lib/crypto"
	"github.com/bnb-chain/tss-lib/tss"
)

func TestExampleInitKey(t *testing.T) {
	alg_ecdsa.ExampleInitKey()
}

func TestExampleLoadAndSign(t *testing.T) {
	alg_ecdsa.ExampleLoadAndSign()
}

func TestPartyIDs(t *testing.T) {
	ids := alg_ecdsa.PartyIDs([]string{"node1", "node2", "node3"})
	if len(ids) != 3 {
		t.Fatalf("expected 3 party ids, got %d", len(ids))
	}
	if ids[0].Index == ids[1].Index {
		t.Error("party indices should differ after sort")
	}
}

func TestKeygenAndSignInProcess(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TSS keygen+sign in short mode (slow)")
	}
	nodeIDs := []string{"n1", "n2", "n3"}
	threshold := 2
	errCh := make(chan *tss.Error, 8)

	router := &alg_ecdsa.InProcessRouter{ErrCh: errCh}
	result, err := alg_ecdsa.RunKeygen(nodeIDs, threshold, router, alg_ecdsa.KeygenConfig{
		PreParamsTimeout: 1 * time.Minute,
	}, nil)
	if err != nil {
		t.Fatalf("keygen failed: %v", err)
	}
	if result == nil || len(result.SaveData) != 3 {
		t.Fatalf("keygen result invalid")
	}
	if result.KeyID == "" {
		t.Error("keyID should be set")
	}

	msgHash := new(big.Int).SetBytes([]byte("hello world 32 bytes hash!!!!!!"))
	if msgHash.BitLen() > 256 {
		msgHash = new(big.Int).SetBytes(msgHash.Bytes()[:32])
	}

	sigResult, err := alg_ecdsa.RunSign(nodeIDs, result.SaveData, msgHash, router)
	if err != nil {
		t.Fatalf("sign failed: %v", err)
	}
	if sigResult == nil || len(sigResult.Signature) != 64 {
		t.Fatalf("signature invalid: %+v", sigResult)
	}
}

// TestDeriveAccountIDFromRootPub 演示如何从根公钥派生出 AccountID（无 seed）。
// 根公钥来自 keygen 结果的 SaveData[].ECDSAPub；chainCode 用 ChainCodeFromKeyID(KeyID)；path 用 PathFromAccountIndex。
func TestDeriveAccountIDFromRootPub(t *testing.T) {
	// 1) 构造根公钥（测试用：用曲线上的一个点；实际来自 keygen 后 save.ECDSAPub）
	ec := tss.S256()
	rootScalar := big.NewInt(12345)
	rootPoint, err := crypto.NewECPoint(ec, ec.Params().Gx, ec.Params().Gy)
	if err != nil {
		t.Fatal(err)
	}
	rootPoint = rootPoint.ScalarMult(rootScalar)

	// 2) chainCode：无 seed 时用 KeyID 派生；这里用固定 KeyID 模拟
	keyID := "test-mpc-key-id"
	chainCode := alg_ecdsa.ChainCodeFromKeyID(keyID)
	if len(chainCode) != 32 {
		t.Fatalf("chainCode length want 32, got %d", len(chainCode))
	}

	// 3) 路径：账户索引 0、1 对应不同 AccountID
	path0 := alg_ecdsa.PathFromAccountIndex(0)
	path1 := alg_ecdsa.PathFromAccountIndex(1)

	// 4) 派生子公钥
	delta0, childPub0, err := alg_ecdsa.DeriveChildPubFromPath(rootPoint, chainCode, path0)
	if err != nil {
		t.Fatalf("derive path0: %v", err)
	}
	delta1, childPub1, err := alg_ecdsa.DeriveChildPubFromPath(rootPoint, chainCode, path1)
	if err != nil {
		t.Fatalf("derive path1: %v", err)
	}

	// 5) 子公钥 → 公钥 hex（生产环境可交给 openwallet.GenAccountID(pubKeyHex)）
	pubHex0 := alg_ecdsa.PubKeyToHex(childPub0)
	pubHex1 := alg_ecdsa.PubKeyToHex(childPub1)
	if len(pubHex0) != 130 || len(pubHex1) != 130 {
		t.Fatalf("pub hex length want 130, got %d / %d", len(pubHex0), len(pubHex1))
	}

	// 6) AccountID：这里用 SHA256(公钥) 的 hex 作为示例；业务里可用 openwallet.GenAccountID(pubHex)
	h0 := sha256.Sum256([]byte(pubHex0))
	h1 := sha256.Sum256([]byte(pubHex1))
	accountID0 := hex.EncodeToString(h0[:])
	accountID1 := hex.EncodeToString(h1[:])
	if accountID0 == accountID1 {
		t.Fatal("different paths must yield different AccountID")
	}

	t.Logf("path0 delta=%s accountID=%s", delta0.Text(10), accountID0)
	t.Logf("path1 delta=%s accountID=%s", delta1.Text(10), accountID1)

	// 同一 path 再次派生，结果应一致
	delta0Again, childPub0Again, err := alg_ecdsa.DeriveChildPubFromPath(rootPoint, chainCode, path0)
	if err != nil {
		t.Fatal(err)
	}
	if delta0Again.Cmp(delta0) != 0 {
		t.Error("delta should be deterministic for same path")
	}
	if alg_ecdsa.PubKeyToHex(childPub0Again) != pubHex0 {
		t.Error("child pub should be deterministic for same path")
	}
	_ = accountID0
	_ = accountID1
}

// TestDeriveAccountIDFromKeygenResult 用 keygen 得到的根公钥派生 AccountID（完整流程，需跑 keygen，short 下跳过）。
func TestDeriveAccountIDFromKeygenResult(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping keygen in short mode")
	}
	nodeIDs := []string{"n1", "n2", "n3"}
	errCh := make(chan *tss.Error, 8)
	router := &alg_ecdsa.InProcessRouter{ErrCh: errCh}
	result, err := alg_ecdsa.RunKeygen(nodeIDs, 2, router, alg_ecdsa.KeygenConfig{PreParamsTimeout: 1 * time.Minute}, nil)
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}

	// 根公钥来自任意一份 SaveData
	rootPub := result.SaveData[0].ECDSAPub
	keyID := result.KeyID
	chainCode := alg_ecdsa.ChainCodeFromKeyID(keyID)
	path := alg_ecdsa.PathFromAccountIndex(0)

	delta, childPub, err := alg_ecdsa.DeriveChildPubFromPath(rootPub, chainCode, path)
	if err != nil {
		t.Fatalf("derive: %v", err)
	}

	pubHex := alg_ecdsa.PubKeyToHex(childPub)
	h := sha256.Sum256([]byte(pubHex))
	accountID := hex.EncodeToString(h[:])

	t.Logf("KeyID=%s AccountID(hex)=%s delta=%s", keyID, accountID, delta.Text(10))
	if accountID == "" || pubHex == "" {
		t.Fatal("AccountID and pubHex must be non-empty")
	}
}

// TestRealDeriveAccountIDFromMpcKeys 从本地 mpc_keys 目录加载已存在的 keygen 结果，派生出真实 AccountID。
// 需先有 mpc_keys 目录（由 ExampleInitKey 或 RunKeygen + FileKeyStore.Save 生成）。
// 若 mpc_keys 在 mpc 子目录下，使用 mpc/mpc_keys；若在项目根目录则使用 mpc_keys。
func TestRealDeriveAccountIDFromMpcKeys(t *testing.T) {
	baseDir := "mpc_keys"
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		baseDir = "mpc/mpc_keys"
	}
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		t.Fatalf("未找到 mpc_keys 目录（已尝试 ./mpc_keys 与 ./mpc/mpc_keys），请先跑 keygen 并 Save 到 mpc_keys")
	}
	keyStore := alg_ecdsa.NewFileKeyStore(baseDir)

	// 1) 枚举 keyID：mpc_keys 下的子目录名
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		t.Fatalf("read mpc_keys dir: %v（请先在项目根目录执行测试，并确保已运行过 keygen 且 Save 到 mpc_keys）", err)
	}
	var keyIDs []string
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			keyIDs = append(keyIDs, e.Name())
		}
	}
	if len(keyIDs) == 0 {
		t.Fatalf("mpc_keys 下没有找到 keyID 子目录，请先跑 keygen 并 Save 到 mpc_keys")
	}
	sort.Strings(keyIDs)
	keyID := keyIDs[0]
	if len(keyIDs) > 1 {
		t.Logf("存在多个 keyID，使用第一个: %s", keyID)
	}

	// 2) 枚举该 key 下的 nodeID：mpc_keys/keyID/*.json 的文件名去掉 .json
	keyDir := filepath.Join(baseDir, keyID)
	nodeEntries, err := os.ReadDir(keyDir)
	if err != nil {
		t.Fatalf("read key dir %s: %v", keyDir, err)
	}
	var nodeIDs []string
	for _, e := range nodeEntries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			nodeID := strings.TrimSuffix(e.Name(), ".json")
			nodeIDs = append(nodeIDs, nodeID)
		}
	}
	if len(nodeIDs) < 2 {
		t.Fatalf("key %s 下至少需要 2 个节点的 SaveData，当前: %v", keyID, nodeIDs)
	}
	sort.Strings(nodeIDs)

	// 3) 加载该 key 下所有节点的 SaveData
	_, keysInOrder, err := keyStore.LoadAllForKey(keyID, nodeIDs)
	if err != nil {
		t.Fatalf("LoadAllForKey: %v", err)
	}

	// 4) 根公钥取任意一份 SaveData 的 ECDSAPub
	rootPub := keysInOrder[0].ECDSAPub
	if rootPub == nil {
		t.Fatal("SaveData 中 ECDSAPub 为空")
	}

	// 5) chainCode + path 派生子公钥
	chainCode := alg_ecdsa.ChainCodeFromKeyID(keyID)
	path := alg_ecdsa.PathFromAccountIndex(0)
	delta, childPub, err := alg_ecdsa.DeriveChildPubFromPath(rootPub, chainCode, path)
	if err != nil {
		t.Fatalf("DeriveChildPubFromPath: %v", err)
	}

	// 6) 子公钥 → 公钥 hex → AccountID（示例用 SHA256 hex；业务可用 openwallet.GenAccountID(pubHex)）
	pubHex := alg_ecdsa.PubKeyToHex(childPub)
	h := sha256.Sum256([]byte(pubHex))
	accountID := hex.EncodeToString(h[:])

	t.Logf("KeyID:    %s", keyID)
	t.Logf("NodeIDs:  %v", nodeIDs)
	t.Logf("PubHex:   %s", pubHex)
	t.Logf("AccountID: %s", accountID)
	t.Logf("Delta(path=0): %s", delta.Text(10))

	if accountID == "" || pubHex == "" {
		t.Fatal("AccountID 或 PubHex 为空")
	}
}
