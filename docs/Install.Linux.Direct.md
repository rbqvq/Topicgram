# Linux 直接安装教程

## 安装

1. 执行安装命令

```shell
bash <(curl -sSL "https://gitlab.com/CoiaPrant/Topicgram/-/raw/master/scripts/install.sh")
```

---

2. 进入 `/opt/Topicgram` 文件夹

---

3. 创建一个名为 `config.json` 的配置文件

```json
{
  "Web": {
    "Type": "tcp",
    "Listen": ":443",
    "Cert": "cert.pem",
    "Key": "private.key"
  },
  "Database": {
    "Type": "sqlite3",
    "SQLite3": {
      "File": "sqlite.db",
      "BusyTimeout": 5000,
      "JournalMode": "WAL"
    }
  },
  "Bot": {
    "Token": "你的 Bot Token",
    "GroupId": 0,
    "LanguageCode": "zh-hans",
    "WebHook": {
      "Host": "你的 WebHook 域名 (非 443 要带端口)"
    }
  },
  "Security": {
    "InsecureSkipVerify": false
  },
  "Proxy": ""
}
```

> 文件编码必须为 `UTF-8`

`:443` 为 WebHook 监听地址, 如果有多个 Bot 请将 `443` 更换成不一样的端口, 然后修改 WebHook Host 设置

> 替换 GroupId 为你的转发群组, 将 Bot 设置为管理员, 授予 **删除消息, 置顶消息, 管理话题** 权限

---

4. 在文件夹下创建 `cert.pem` 和 `private.key`

对应网站证书

> 如果搭配 Cloudflare CDN 使用可使用 回源证书

---

5. 启动

```shell
systemctl enable --now Topicgram # Topicgram 为默认服务名, 如果您安装了多个 Bot 请自行修改服务名称
```
