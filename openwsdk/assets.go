package openwsdk

//注册钱包管理工具
//func initAssetAdapter() {
//	//注册钱包管理工具
//	log.Debug("Wallet Manager Load Successfully.")
//	openw.RegAssets(ethereum.Symbol, ethereum.NewWalletManager())
//}
//
//// GetAssetsAdapter 获取资产控制器
//func GetAssetsAdapter(symbol string) (openwallet.AssetsAdapter, error) {
//
//	adapter := assets.GetAssets(symbol)
//	if adapter == nil {
//		return nil, fmt.Errorf("assets: %s is not support", symbol)
//	}
//
//	manager, ok := adapter.(openwallet.AssetsAdapter)
//	if !ok {
//		return nil, fmt.Errorf("assets: %s is not support", symbol)
//	}
//
//	return manager, nil
//}
