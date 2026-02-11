## 创建钱包文件步骤非常重要，请按照规范操作

### passwords 内容示例 (注意文件内容不应该有换行)
dcdba32174fdf3f9a08a48b9fa838b68

### 1. 创建目录（如不存在）
```bash
sudo mkdir -p /secure/wallets
```

### 2. 写入文件（推荐使用编辑器，避免 echo 留痕）
```bash
sudo nano /secure/wallets/passwords
```

💡 提示：在 nano 中输入密码后，不要按回车，直接保存退出（Ctrl+O → 回车 → Ctrl+X）。

或使用无换行写入命令：
```bash
echo -n "your_password_here" | sudo tee /secure/wallets/passwords > /dev/null
```

### 3. 设置权限：仅属主可读写
```bash
sudo chmod 600 /secure/wallets/passwords
sudo chown $(whoami):$(whoami) /secure/wallets/passwords
```

### 4. 触发操作（无需传密码，服务端自动读取文件）

创建钱包：
```bash
curl -X POST http://127.0.0.1:9422/api/CreateWallet -H "alias:testwallet1"
```

解锁钱包：
```bash
curl -X POST http://127.0.0.1:9422/api/UnlockWallet -H "filename:testwallet1-VzYK21Vem6WBXHXZmSRYGN4iaE6n2naF6z.key"
```

### 5. 安全清理（操作完成后务必执行）

高安全擦除（推荐）：
```bash
shred -u /secure/wallets/passwords
```

或简单删除（确保目录权限安全）：
```bash
rm /secure/wallets/passwords
```

⚠️ **重要说明**

- 密码文件内容必须无换行、无空格
- 所有操作依赖同一密码文件 /secure/wallets/passwords
- 服务端不会删除该文件，需用户手动清理
- 切勿将此文件提交到版本控制或共享存储