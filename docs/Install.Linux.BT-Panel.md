# Linux 宝塔面板安装教程

## 准备工作

预装软件

- Nginx 1.24+
- MySQL (可选)

## 安装

1. 执行安装命令

```shell
bash <(curl -sSL "https://gitlab.com/CoiaPrant/Topicgram/-/raw/master/scripts/install.sh")
```

---

2. 进入 `/opt/Topicgram` 文件夹

---

3. 创建一个名为 `config.json` 的配置文件

- SQLite3

```json
{
  "Web": {
    "Type": "unix",
    "Listen": "/run/Topicgram.sock"
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

- MySQL

```json
{
  "Web": {
    "Type": "unix",
    "Listen": "/run/Topicgram.sock"
  },
  "Database": {
    "Type": "mysql",
    "MySQL": {
      "Host": "localhost",
      "Port": 3306,
      "User": "数据库用户名",
      "Password": "数据库密码",
      "Name": "数据库名称"
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

`/run/Topicgram.sock` 为 `Unix Socket` 监听地址, 如果有多个 Bot 请将 `Topicgram` 更换成不一样的名字

> 替换 GroupId 为你的转发群组, 将 Bot 设置为管理员, 授予 **删除消息, 置顶消息, 管理话题** 权限

---

4. 启动

```shell
systemctl enable --now Topicgram # Topicgram 为默认服务名, 如果您安装了多个 Bot 请自行修改服务名称
```

5.  在 宝塔面板 (aaPanel) 添加网站并配置 SSL (开启强制 HTTPS)

---

6. 申请 SSL (Telegram Bot API 要求可信 HTTPS) / 添加 Cloudflare 反代

7. 配置反向代理

目标 URL 填写 `http://unix:/run/Topicgram.sock` (上文的 `Web` 节 `Listen` 字段)

发送域名 `$host`

> 部分版本的面板无法添加, 可以将 目标 URL 设置为 http://127.0.0.1 然后修改反向代理配置文件

编辑反向代理配置文件为以下内容

```nginx
#PROXY-START/
underscores_in_headers on;

location ^~ / {
    proxy_pass http://unix:/run/Topicgram.sock;
    proxy_set_header Host $host;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection $http_connection;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header REMOTE-HOST $remote_addr;

    add_header X-Cache $upstream_cache_status;
}

#PROXY-END/
```
