# 二进制部署 (轻量)

适合不想装 docker 的小内存 VPS。**单文件全栈**,内嵌 admin/user 前端,**无需 nginx**。

## 快速开始

```bash
# 1. 下载最新 release 的 fullstack 版
#    把 URL 里的 v1.0.1 替换成最新 tag (见 https://github.com/TokensZhuanfa/Dujiao-Shop/releases)
curl -L -O https://github.com/TokensZhuanfa/Dujiao-Shop/releases/download/v1.0.1/dujiao-shop-fullstack_v1.0.1_linux_amd64.tar.gz

# 2. 解压 + 安装 (装 systemd unit + 系统用户 dujiao)
tar xzf dujiao-shop-fullstack_*.tar.gz
cd dujiao-shop-fullstack_*/
sudo ./install.sh

# 3. 改配置
sudo vi /opt/dujiao/config.yml

# 4. 启服务
sudo systemctl enable --now dujiao
sudo journalctl -u dujiao -f

# 5. 浏览器访问 http://你的IP:8080
#    admin 界面在 /admin/  user 界面在 /
```

ARM 服务器把 URL 里 `linux_amd64` 换成 `linux_arm64`,其他步骤一致。

## 两种 release 包对比

| 包名 | 包含 | 适合 |
|---|---|---|
| `dujiao-shop-fullstack_*` | dujiao-api(内嵌 admin+user 前端) + admin-tool | **单文件全栈**,不需 nginx,5 分钟跑起 |
| `dujiao-shop-headless_*`  | dujiao-api-headless + admin-tool + nginx 模板 | 已有 nginx,想自己托管 admin/user |

> 想要 nginx 托管前端的,还需要单独下载 `dujiao-shop-admin-*.zip` 和 `dujiao-shop-user-*.zip` 解到 nginx root,详见 [宝塔面板部署](/install/baota)。

## 数据存放位置

```
/opt/dujiao/
├── dujiao-api            # 主二进制
├── admin-tool            # 运维 CLI
├── config.yml            # 你的配置 (不会被升级覆盖)
├── db/                   # SQLite 数据库
├── uploads/              # 商品图 / 用户上传
├── credentials/          # 文件型卡密
└── logs/
```

## 反向代理 (HTTPS)

二进制本身只跑 HTTP。生产环境前面要套 nginx + cert (Let's Encrypt 或 Cloudflare)。

### Fullstack 版的最小 nginx 配置

```nginx
server {
    listen 443 ssl http2;
    server_name your.domain.com;

    ssl_certificate     /etc/letsencrypt/live/your.domain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/your.domain.com/privkey.pem;

    client_max_body_size 100M;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## 升级

```bash
cd /tmp && curl -L -O https://github.com/TokensZhuanfa/Dujiao-Shop/releases/download/v1.0.1/dujiao-shop-fullstack_v1.0.1_linux_amd64.tar.gz
tar xzf dujiao-shop-fullstack_*.tar.gz
cd dujiao-shop-fullstack_*/
sudo ./install.sh   # 自动覆盖二进制,保留 config.yml + 数据库
sudo systemctl restart dujiao
```

## 运维 CLI

```bash
# 列出所有管理员
sudo -u dujiao /opt/dujiao/admin-tool list-admins

# 重置某管理员的 2FA (TOTP 丢了用)
sudo -u dujiao /opt/dujiao/admin-tool reset-2fa --username admin
```

## 卸载

```bash
sudo systemctl disable --now dujiao
sudo rm /etc/systemd/system/dujiao.service
sudo rm -rf /opt/dujiao    # ⚠️ 含数据库,删前先备份
sudo userdel dujiao
```
