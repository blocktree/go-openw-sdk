package openwsdk

////注册钱包管理工具
//func initAssetAdapter() {
//	//注册钱包管理工具
//	log.Debug("Wallet Manager Load Successfully.")
//	assets.RegAssets(ethereum.Symbol, ethereum.NewWalletManager())
//	assets.RegAssets(bitcoin.Symbol, bitcoin.NewWalletManager())
//	assets.RegAssets(litecoin.Symbol, litecoin.NewWalletManager())
//	assets.RegAssets(qtum.Symbol, qtum.NewWalletManager())
//	assets.RegAssets(nebulasio.Symbol, nebulasio.NewWalletManager())
//	assets.RegAssets(bitcoincash.Symbol, bitcoincash.NewWalletManager())
//	assets.RegAssets(ontology.Symbol, ontology.NewWalletManager())
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
