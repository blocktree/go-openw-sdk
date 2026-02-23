// webapp/tview_app.go

package webapp

import (
	"bytes"
	"fmt"
	"strings"
	"unicode"

	"github.com/godaddy-x/freego/ex"

	"log"
	"os"

	"github.com/blocktree/go-openw-sdk/v2/openwsdk/dto"

	"github.com/gdamore/tcell/v2"
	"github.com/howeyc/gopass"
	"github.com/rivo/tview"
)

func RunApplication() {
	app := tview.NewApplication()
	showMainMenu(app)
	if err := app.Run(); err != nil {
		panic(err)
	}
}

func showMainMenu(app *tview.Application) {
	// === 创建主菜单标题和提示 ===
	header := tview.NewTextView()
	header.SetText("🔐 OpenWallet CLI – Manage your cryptographic wallets\n( Use ↑↓ to navigate, Enter to select, or press 1–4 )")
	header.SetTextColor(tcell.ColorYellow)
	header.SetDynamicColors(true)
	header.SetBorder(false)

	// === 创建菜单列表 ===
	list := tview.NewList()
	list.SetBorder(false)

	list.AddItem("Create Wallet", "Generate new cryptographic keys", '1', nil)
	list.AddItem("Unlock Wallet", "Load and decrypt an existing wallet", '2', nil)
	list.AddItem("Start Service", "Launch HTTP signing API", '3', nil)
	list.AddItem("Exit", "Quit the application", '4', nil)

	list.SetSelectedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		switch index {
		case 0:
			showCreateWallet(app)
		case 1:
			showWalletList(app)
		case 2:
			log.Println("[Action] Start Service")
			// TODO: startHTTPService()
		case 3:
			app.Stop()
			os.Exit(0)
		}
	})

	// ESC 也可返回（虽然已经是主菜单，但保持一致性）
	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			// 主菜单按 ESC 直接退出（可选），或忽略
			// 这里我们选择忽略，或你也可以退出
			// app.Stop(); os.Exit(0)
			return event // 允许默认行为（无操作）
		}
		return event
	})

	// === 布局：顶部提示 + 菜单列表 ===
	layout := tview.NewFlex().SetDirection(tview.FlexRow)
	layout.AddItem(header, 3, 1, false) // 固定 3 行高
	layout.AddItem(list, 0, 1, true)    // 剩余空间给列表

	app.SetRoot(layout, true)
}

// 校验 alias：非空且仅包含字母和数字
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

// 安全创建钱包流程（使用 gopass，无回显）
func showCreateWallet(app *tview.Application) {
	app.Suspend(func() {
		// === 清晰的标题和说明 ===
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
			clearPassword(password1)
			fmt.Printf("\nInput error: %v\n", err)
			fmt.Print("Press Enter to return to main menu...")
			fmt.Scanln()
			return
		}

		if !bytes.Equal(password1, password2) {
			fmt.Println("\n❌ Error: Passwords do not match.")
			clearPassword(password1)
			clearPassword(password2)
			fmt.Print("Press Enter to return to main menu...")
			fmt.Scanln()
			return
		}

		clearPassword(password2)

		if len(password1) < 8 {
			fmt.Println("\n❌ Error: Password must be at least 8 characters.")
			clearPassword(password1)
			// password2 已经在前面清零了
			fmt.Print("Press Enter to return to main menu...")
			fmt.Scanln()
			return
		}

		// 调用服务创建钱包
		res := &dto.CliCreateWalletRes{}
		err = CliService.CreateWallet(alias, password1, res)
		clearPassword(password1)

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

	// 返回主菜单
	showMainMenu(app)
}

// 安全清零密码内存
func clearPassword(p []byte) {
	if p == nil {
		return
	}
	for i := range p {
		p[i] = 0
	}
}

// ===== 钱包列表 =====
func showWalletList(app *tview.Application) {
	req := &dto.CliFindWalletListReq{}
	res := &dto.CliFindWalletListRes{}
	if err := CliService.FindWalletList(req, res); err != nil {
		showMessage(app, fmt.Sprintf("Failed to load wallets: %v", err))
		return
	}

	if len(res.Result) == 0 {
		showMessage(app, "No wallets found. Please create one first.")
		return
	}

	// === 创建提示文本 ===
	header := tview.NewTextView()
	header.SetText("🔐 Select a wallet to unlock\n( Press ESC to return to main menu )\n")
	header.SetTextColor(tcell.ColorYellow)
	header.SetDynamicColors(true)
	header.SetBorder(false)

	// === 创建钱包列表 ===
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

		// 标记是否解锁成功
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
			err = CliService.UnlockWallet(fmt.Sprintf("%s-%s.key", selected.Alias, selected.WalletID), password, res)
			clearPassword(password)

			if err != nil {
				fmt.Printf("\n❌ Unlock failed: %v\n", ex.Catch(err).Msg)
				fmt.Print("Press Enter to return to wallet list...")
				fmt.Scanln()
				return
			}

			fmt.Println("✅ Wallet unlocked successfully!")
			fmt.Print("Press Enter to return to main menu...")
			fmt.Scanln()

			// 标记成功（注意：不能在这里 SetRoot！）
			unlockedSuccessfully = true
		})

		// Suspend 已结束，现在可以安全切换界面
		if unlockedSuccessfully {
			showMainMenu(app)
		}
		// 如果失败，什么也不做 → 自动留在钱包列表
	})

	// ESC 返回主菜单（保留原有逻辑）
	walletList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			showMainMenu(app)
			return nil
		}
		return event
	})

	// === 布局：顶部提示 + 列表 ===
	layout := tview.NewFlex().SetDirection(tview.FlexRow)
	layout.AddItem(header, 3, 1, false)    // 高度 3 行
	layout.AddItem(walletList, 0, 1, true) // 剩余空间

	app.SetRoot(layout, true)
}

// 显示消息弹窗
func showMessage(app *tview.Application, message string) {
	modal := tview.NewModal()
	modal.SetText(message)
	modal.AddButtons([]string{"OK"})
	modal.SetDoneFunc(func(_ int, _ string) {
		showMainMenu(app)
	})
	app.SetRoot(modal, false)
}
