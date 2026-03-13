// webapp/tview_app.go

package app

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"unicode"

	"github.com/blocktree/go-openw-sdk/v2/openwsdk"

	DIC "github.com/godaddy-x/freego/common"
	"github.com/godaddy-x/freego/utils/crypto"

	"github.com/blocktree/go-openw-sdk/v2/openwsdk/dto"
	"github.com/godaddy-x/freego/ex"

	"github.com/gdamore/tcell/v2"
	"github.com/howeyc/gopass"
	"github.com/rivo/tview"
)

// === 全局 HTTP 服务状态管理 ===
var (
	httpServerMu     sync.Mutex
	isServiceRunning bool

	wsServerMu         sync.Mutex
	isWSServiceRunning bool
)

const (
	menuCreateMPCWallet = iota
	menuCreateECDSA
	menuStartHttp
	menuStartWebsocket
	menuExit
)

// 启动 HTTP 服务（仅当未运行时）
func startHTTPService() error {
	httpServerMu.Lock()
	defer httpServerMu.Unlock()

	if isServiceRunning {
		return fmt.Errorf("service already running")
	}

	web := NewHTTP()

	go func() {
		defer func() {
			httpServerMu.Lock()
			isServiceRunning = false
			httpServerMu.Unlock()

			if r := recover(); r != nil {
				log.Printf("[ERROR] Panic in StartHttpNode: %v", r)
			}
			log.Println("[Service] HTTP service exited")
		}()

		StartHttpNode(web)
	}()

	isServiceRunning = true
	log.Println("[Service] HTTP service started successfully")
	return nil
}

// 启动 WebSocket 服务（仅当未运行时）
func startWSService() error {
	wsServerMu.Lock()
	defer wsServerMu.Unlock()

	if isWSServiceRunning {
		return fmt.Errorf("WebSocket service already running")
	}

	go func() {
		defer func() {
			wsServerMu.Lock()
			isWSServiceRunning = false
			wsServerMu.Unlock()

			if r := recover(); r != nil {
				log.Printf("[ERROR] Panic in StartWebSocketNode: %v", r)
			}
			log.Println("[Service] WebSocket service exited")
		}()

		NewSocket()
	}()

	isWSServiceRunning = true
	log.Println("[Service] WebSocket service started successfully")
	return nil
}

// ==================== 应用入口 ====================
func RunApplication() {
	app := tview.NewApplication()
	showMainMenu(app)
	if err := app.Run(); err != nil {
		panic(err)
	}
}

func menuNumber(index int32) rune {
	return '1' + index
}

// ==================== 主菜单 ====================
func showMainMenu(app *tview.Application) {
	header := tview.NewTextView()
	header.SetText("🔐 OpenWallet CLI – Manage your cryptographic wallets\n( Use ↑↓ to navigate, Enter to select, or press 1–6 )")
	header.SetTextColor(tcell.ColorYellow)
	header.SetDynamicColors(true)
	header.SetBorder(false)

	list := tview.NewList()
	list.SetBorder(false)

	status := ""
	httpServerMu.Lock()
	if isServiceRunning {
		status = " (running)"
	}
	httpServerMu.Unlock()

	wsStatus := ""
	wsServerMu.Lock()
	if isWSServiceRunning {
		wsStatus = " (running)"
	}
	wsServerMu.Unlock()

	list.AddItem("Create MPC Wallet (TSS)", "Multi-node TSS keygen (3 or 5 nodes online)", menuNumber(menuCreateMPCWallet), nil)
	list.AddItem("Generate ECDSA", "Print base64-encoded ECDSA key pair to terminal", menuNumber(menuCreateECDSA), nil) // ← 新增
	list.AddItem("Start HTTP Service"+status, "Launch HTTP signing API", menuNumber(menuStartHttp), nil)
	list.AddItem("Start WebSocket Service"+wsStatus, "Launch WebSocket signing API", menuNumber(menuStartWebsocket), nil)
	list.AddItem("Exit", "Quit the app", menuNumber(menuExit), nil)

	list.SetSelectedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		switch index {
		case menuCreateMPCWallet:
			showCreateMPCKeyWallet(app)
		case menuCreateECDSA:
			showGenerateECDSA(app)
		case menuStartHttp:
			showHttpService(app)
		case menuStartWebsocket:
			showWSService(app)
		case menuExit:
			openwsdk.DestroyMemoryObject()
			app.Stop()
			os.Exit(0)
		}
	})

	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			return event
		}
		return event
	})

	layout := tview.NewFlex().SetDirection(tview.FlexRow)
	layout.AddItem(header, 3, 1, false)
	layout.AddItem(list, 0, 1, true)
	app.SetRoot(layout, true)
}

// ==================== 创建钱包 ====================
func isValidAlias(alias string) bool {
	if len(alias) == 0 {
		return false
	}
	for _, r := range alias {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// ==================== mode=1创建钱包列表 ====================
func showCreateWallet(app *tview.Application) {
	app.Suspend(func() {
		fmt.Print("\n")
		fmt.Println("🔐 Create New Wallet")
		fmt.Println("────────────────────")
		fmt.Println("( Password input is hidden. Press Enter to submit. )")
		fmt.Println()

		fmt.Print("Enter alias (letters + digits only, non-empty): ")
		var aliasInput string
		fmt.Scanln(&aliasInput)

		alias := strings.TrimSpace(aliasInput)
		if !isValidAlias(alias) {
			fmt.Println("\n❌ Error: Alias must be non-empty and contain only letters and digits (e.g., myWallet123).")
			fmt.Print("\nPress Enter to return to main menu...")
			fmt.Scanln()
			return
		}

		fmt.Print("Enter password (min 8 chars, input is HIDDEN): ")
		password1, err := gopass.GetPasswd()
		if err != nil {
			fmt.Printf("\nInput error: %v\n", err)
			fmt.Print("Press Enter to return to main menu...")
			fmt.Scanln()
			return
		}

		fmt.Print("Confirm password (input is HIDDEN): ")
		password2, err := gopass.GetPasswd()
		if err != nil {
			DIC.ClearData(password1)
			fmt.Printf("\nInput error: %v\n", err)
			fmt.Print("Press Enter to return to main menu...")
			fmt.Scanln()
			return
		}

		if !bytes.Equal(password1, password2) {
			fmt.Println("\n❌ Error: Passwords do not match.")
			DIC.ClearData(password1)
			DIC.ClearData(password2)
			fmt.Print("Press Enter to return to main menu...")
			fmt.Scanln()
			return
		}
		DIC.ClearData(password2)

		if len(password1) < 8 {
			fmt.Println("\n❌ Error: Password must be at least 8 characters.")
			DIC.ClearData(password1)
			fmt.Print("Press Enter to return to main menu...")
			fmt.Scanln()
			return
		}

		res := &dto.CliCreateWalletRes{}
		err = cliService.CreateWallet(alias, password1, res)
		DIC.ClearData(password1)

		if err != nil {
			fmt.Printf("\n❌ Create failed: %v\n", ex.Catch(err).Msg)
			fmt.Print("Press Enter to return to main menu...")
			fmt.Scanln()
			return
		}

		fmt.Printf("\n✅ Wallet created successfully!\n")
		fmt.Printf("   Wallet ID: %s\n", res.WalletID)
		fmt.Printf("   Alias    : %s\n", alias)
		fmt.Print("\nPress Enter to return to main menu...")
		fmt.Scanln()
	})
	showMainMenu(app)
}

// showCreateMPCKeyWallet 通过轮询多节点完成 TSS keygen，落盘 mpc_keys 后返回 KeyID。
func showCreateMPCKeyWallet(app *tview.Application) {
	app.Suspend(func() {
		fmt.Print("\n")
		fmt.Println("🔐 Create MPC Wallet (TSS Keygen)")
		fmt.Println("────────────────────")
		fmt.Println("Ensure 3 or 5 nodes are online and WebSocket service is running.")
		fmt.Println()

		fmt.Print("Enter alias for this MPC wallet (letters + digits only, non-empty): ")
		var aliasInput string
		fmt.Scanln(&aliasInput)
		alias := strings.TrimSpace(aliasInput)
		if !isValidAlias(alias) {
			fmt.Println("\n❌ Error: Alias must be non-empty and contain only letters and digits (e.g., mpcWallet1).")
			fmt.Print("Press Enter to return to main menu...")
			fmt.Scanln()
			return
		}

		walletID, err := CreateMPCKeygenTask(alias)
		if err != nil {
			fmt.Printf("\n❌ MPC keygen failed: %v\n", err.Error())
			fmt.Print("Press Enter to return to main menu...")
			fmt.Scanln()
			return
		}

		fmt.Printf("\n✅ MPC wallet created successfully!\n")
		fmt.Printf("   walletID: %s\n", walletID)
		fmt.Printf("   alias   : %s\n", alias)
		fmt.Printf("   Saved to: %s/%s.json\n", GetAllConfig().Extract.WalletDir, walletID)
		fmt.Print("\nPress Enter to return to main menu...")
		fmt.Scanln()
	})
	showMainMenu(app)
}

// showTestMPCSign 通过 CreateMPCSignTask 触发一次固定消息哈希的 MPC 签名，用于快速联调。
func showTestMPCSign(app *tview.Application) {
	app.Suspend(func() {
		fmt.Print("\n")
		fmt.Println("🧪 Test MPC Sign (TSS)")
		fmt.Println("──────────────────────")
		fmt.Println("This will use a fixed existing KeyID and a fixed 32-byte hash to run a distributed TSS signature.")
		fmt.Println("Ensure WebSocket service is running and all nodes for this KeyID are online.")
		fmt.Println()

		// 固定使用已存在的 KeyID（测试用）
		const walletID = "WGE8bUCAQBw3WH6JyYRPw2ocWGSugKiBDP"
		fmt.Printf("Using fixed KeyID: %s\n", walletID)

		// 固定的 32 字节消息哈希（64 位 hex），仅用于测试
		const msgHashHex = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		fmt.Printf("Using fixed msgHashHex: %s\n", msgHashHex)

		sigHex, err := CreateMPCSignTask(walletID, msgHashHex)
		if err != nil {
			fmt.Printf("\n❌ MPC sign failed: %v\n", err.Error())
			fmt.Print("Press Enter to return to main menu...")
			fmt.Scanln()
			return
		}

		fmt.Printf("\n✅ MPC sign succeeded!\n")
		fmt.Printf("   walletID         : %s\n", walletID)
		fmt.Printf("   MsgHash (hex) : %s\n", msgHashHex)
		fmt.Printf("   Signature(hex): %s\n", sigHex)
		fmt.Print("\nPress Enter to return to main menu...")
		fmt.Scanln()
	})
	showMainMenu(app)
}

// ==================== mode=1钱包列表与解锁 ====================
func showWalletList(app *tview.Application) {
	req := &dto.CliFindWalletListReq{}
	res := &dto.CliFindWalletListRes{}
	if err := cliService.FindWalletList(req, res); err != nil {
		app.Suspend(func() {
			fmt.Printf("\n❌ Failed to load wallets: %v\n", ex.Catch(err).Msg)
			fmt.Print("Press Enter to return to main menu...")
			fmt.Scanln()
		})
		showMainMenu(app)
		return
	}

	if len(res.Result) == 0 {
		app.Suspend(func() {
			fmt.Println("\nℹ️  No wallets found.")
			fmt.Println("Please create one first.")
			fmt.Print("Press Enter to return to main menu...")
			fmt.Scanln()
		})
		showMainMenu(app)
		return
	}

	header := tview.NewTextView()
	header.SetText("🔐 Select a wallet to unlock\n( Press ESC to return to main menu )\n")
	header.SetTextColor(tcell.ColorYellow)
	header.SetDynamicColors(true)
	header.SetBorder(false)

	walletList := tview.NewList()
	walletList.SetBorder(false)

	for k, w := range res.Result {
		displayName := fmt.Sprintf("%s (%s)", w.Alias, w.WalletID)
		desc := fmt.Sprintf("File: %s-%s.key", w.Alias, w.WalletID)
		shortcut := rune('0' + k + 1)
		walletList.AddItem(displayName, desc, shortcut, nil)
	}

	walletList.SetSelectedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		selected := res.Result[index]
		unlockedSuccessfully := false

		app.Suspend(func() {
			fmt.Printf("\n🔐 Unlocking wallet: %s-%s\n", selected.Alias, selected.WalletID)
			fmt.Print("Enter password (input is HIDDEN): ")

			password, err := gopass.GetPasswd()
			if err != nil {
				fmt.Printf("Input error: %v\n", err)
				fmt.Print("Press Enter to return to wallet list...")
				fmt.Scanln()
				return
			}

			res := &dto.CliUnlockWalletRes{}
			err = cliService.UnlockWallet(fmt.Sprintf("%s-%s.key", selected.Alias, selected.WalletID), password, res)
			DIC.ClearData(password)

			if err != nil {
				fmt.Printf("\n❌ Unlock failed: %v\n", ex.Catch(err).Msg)
				fmt.Print("Press Enter to return to wallet list...")
				fmt.Scanln()
				return
			}

			fmt.Println("✅ Wallet unlocked successfully!")
			fmt.Print("Press Enter to return to main menu...")
			fmt.Scanln()
			unlockedSuccessfully = true
		})

		if unlockedSuccessfully {
			showMainMenu(app)
		}
	})

	walletList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			showMainMenu(app)
			return nil
		}
		return event
	})

	layout := tview.NewFlex().SetDirection(tview.FlexRow)
	layout.AddItem(header, 3, 1, false)
	layout.AddItem(walletList, 0, 1, true)
	app.SetRoot(layout, true)
}

// ==================== mode=3,5钱包解锁禁止描述 ====================
func showDisableUnlockWallet(app *tview.Application) {
	config := GetAllConfig().Extract
	app.Suspend(func() {
		if config.WalletMode == 3 || config.WalletMode == 5 {
			fmt.Printf("\n🔒 Wallet unlocking is disabled in threshold/MPC mode (walletMode=%d).\n", config.WalletMode)
			fmt.Println("   In this mode, private keys are split into shares across multiple nodes.")
		} else {
			fmt.Printf("\n🔒 Wallet unlocking is only available in standalone mode (walletMode=1).\n")
			fmt.Printf("   Your current walletMode (%d) is not valid for local wallet operations.\n", config.WalletMode)
		}
		fmt.Println("   Valid walletMode values:")
		fmt.Println("     • 1 = Standalone (local key storage)")
		fmt.Println("     • 3 = Threshold/MPC (2-of-3 signing)")
		fmt.Println("     • 5 = Threshold/MPC (3-of-5 signing)")
		fmt.Print("\n   Press Enter to return to the main menu...")
		fmt.Scanln()
	})
	showMainMenu(app) // 返回菜单
}

// ==================== 启动HTTP服务 ====================
func showHttpService(app *tview.Application) {
	// 检查是否已有服务在运行
	httpServerMu.Lock()
	alreadyRunning := isServiceRunning
	httpServerMu.Unlock()
	if alreadyRunning {
		app.Suspend(func() {
			fmt.Println("\n⚠️  Service is already running!")
			fmt.Print("Press Enter to return to main menu...")
			fmt.Scanln()
		})
		showMainMenu(app)
		return
	}

	// 启动服务
	if err := startHTTPService(); err != nil {
		app.Suspend(func() {
			fmt.Printf("\n❌ Failed to start service: %v\n", err)
			fmt.Print("Press Enter to return to main menu...")
			fmt.Scanln()
		})
		showMainMenu(app)
	} else {
		app.Suspend(func() {
			fmt.Println("\n✅ HTTP service started successfully!")
			fmt.Println(fmt.Sprintf("   Listening on http://localhost:%d", GetAllConfig().GetServerConfig(project).Port))
			fmt.Print("Press Enter to return to main menu...")
			fmt.Scanln()
		})
		showMainMenu(app)
	}
}

// ==================== 启动WebSocket服务 ====================
func showWSService(app *tview.Application) {
	// 检查是否已有服务在运行
	wsServerMu.Lock()
	alreadyRunning := isWSServiceRunning
	wsServerMu.Unlock()
	if alreadyRunning {
		app.Suspend(func() {
			fmt.Println("\n⚠️  WebSocket service is already running!")
			fmt.Print("Press Enter to return to main menu...")
			fmt.Scanln()
		})
		showMainMenu(app)
		return
	}

	// 启动服务
	if err := startWSService(); err != nil {
		app.Suspend(func() {
			fmt.Printf("\n❌ Failed to start WebSocket service: %v\n", err)
			fmt.Print("Press Enter to return to main menu...")
			fmt.Scanln()
		})
		showMainMenu(app)
	} else {
		app.Suspend(func() {
			fmt.Println("\n✅ WebSocket service started successfully!")
			// 注意：这里需要你的配置能获取 WebSocket 端口
			// 假设配置中有 WSPort 字段，否则请调整
			wsPort := GetAllConfig().GetServerConfig(project).Port + 100
			fmt.Printf("   Listening on ws://localhost:%d\n", wsPort)
			fmt.Print("Press Enter to return to main menu...")
			fmt.Scanln()
		})
		showMainMenu(app)
	}
}

// ==================== 生成临时 ECDSA 密钥对（Base64 输出） ====================
func showGenerateECDSA(app *tview.Application) {
	// 生成 ECDSA 密钥对 (P-256)
	o := &crypto.EcdsaObject{}
	if err := o.CreateS256ECDSA(); err != nil {
		app.Suspend(func() {
			fmt.Printf("\n❌ Failed to generate ECDSA key: %v\n", err)
			fmt.Print("Press Enter to return to main menu...")
			fmt.Scanln()
		})
		showMainMenu(app)
		return
	}

	// 安全输出到终端
	app.Suspend(func() {
		fmt.Println("\n🔐 Temporary ECDSA Key Pair (Base64-encoded)")
		fmt.Println("──────────────────────────────────────────────")
		fmt.Printf("Public Key (Base64):\n%s\n\n", o.PublicKeyBase64)
		fmt.Printf("Private Key (Base64):\n%s\n\n", o.PrivateKeyBase64)
		fmt.Println("💡 You can copy these for testing. Keys are NOT saved.")
		fmt.Print("Press Enter to return to main menu...")
		fmt.Scanln()
	})

	showMainMenu(app)
}
