## 解锁钱包文件步骤非常重要，请按照规范操作

### passwords 内容示例
`{
"W7czZcnnWCCGDCAyfXA1eb4KQTsvfN3NZH": "mySecretPass1",
"WE2AYr1jWiKuzoaCvZa1fWuMeSBxx7M6Xh": "mySecretPass2",
"W9xyzAbcDefGhiJklMnoPqrStuVwxYz123": "mySecretPass3"
}`

### 1.创建目录（如不存在）
sudo mkdir -p /secure/wallets

### 2.写入文件（推荐使用编辑器，避免 echo 留痕）
sudo nano /secure/wallets/passwords

### 3.设置权限：仅属主可读写
sudo chmod 600 /secure/wallets/passwords
sudo chown $(whoami):$(whoami) /secure/wallets/passwords

### 4.通过 @ 读取文件内容作为 body
curl -X POST http://127.0.0.1:9422/api/UnlockWallet -H "filename:testwallet1-WLnRikQZPp9WUqQEEh4i7bVpDnWEBMmNQg.key" -d @e:\\passwords

### 5.安全覆写并删除（需安装 secure-delete）
shred -u /secure/wallets/passwords

### 5.或简单删除（如果文件在安全目录且权限正确）
rm /secure/wallets/passwords