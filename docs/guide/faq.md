# 常见问题 FAQ

## 安装

### Q. apt 装 redis 启动报 `libjemalloc.so.2: failed to map segment`

**A.** 装了宝塔的机器特有。宝塔自带的 `/usr/local/lib/libjemalloc.so.2` 旧版被 ld.so 优先加载,跟 apt redis 7.x 期望的 jemalloc 5.3 不兼容。修法见 [宝塔部署 2.1](/install/baota#_2-1-装-redis-二选一) — 用 systemd drop-in 加 `LD_PRELOAD` 强制走 apt 路径,或者直接走"宝塔商店装 redis"避开这个坑。

### Q. admin 后台打开白屏,F12 看到 URL 含 `__DJ_ADMIN_BASE__`

**A.** v1.0.0 的 `dujiao-shop-admin-v1.0.0.zip` 含 fullstack 模式占位符,nginx 静态托管会导致全部资源 404 → 白屏。

```bash
find /www/wwwroot/dujiao-admin -type f \( -name '*.html' -o -name '*.js' -o -name '*.css' \) \
  -exec sed -i 's|__DJ_ADMIN_BASE__||g' {} +
```

跑完浏览器 `Ctrl+Shift+R` 强刷。v1.0.1 起 CI 已修(build 两次),新 release 无需此 workaround。

### Q. 上传图片成功但 admin / 前台都不显示,F12 看 `/uploads/xxx.png` 404

**A.** 宝塔/手动部署时常见。`uploads/` 目录在 **api 工作目录** `/www/server/dujiao/uploads/` 里(api 进程负责写),但 admin 站 nginx 根目录是 `/www/wwwroot/dujiao-admin/`,找不到 → 404。

伪静态里加 `^~ /uploads/` 让 nginx 直接 serve api 的 uploads 目录:

```nginx
location ^~ /uploads/ {
    alias /www/server/dujiao/uploads/;
    try_files $uri =404;
    expires 30d;
    add_header Cache-Control "public, immutable";
}
```

⚠️ **`^~` 不能省**。宝塔默认 vhost 自带 `location ~ .*\.(jpg|png|gif|...)$` 这条 regex(图片缓存优化),它的优先级**高于普通前缀 location**。如果你写成 `location /uploads/`,所有 `*.png` / `*.jpg` 请求都被 regex 抢走 → 在站点根目录找 → 404。改成 `^~ /uploads/`(优先前缀)就压过 regex 了。

admin 站和 user 站**都要加**。如果保存后访问还是 403,放权限:`chmod -R o+rX /www/server/dujiao/uploads`。

### Q. 文件实际写在 `/root/dujiaoapi` 而不是 `/www/server/dujiao`,alias 404

**A.** api 的 systemd unit 实际 `WorkingDirectory` 跟文档默认不一致。看一下:

```bash
systemctl cat dujiao-api | grep WorkingDirectory
ps -ef | grep dujiao-api | grep -v grep
```

两条路修:

**A. 改 nginx alias 指向真实路径**(快但 `/root/` 默认 perm 700):
```bash
chmod o+x /root /root/dujiaoapi    # 让 nginx www 用户能进去
# 然后 alias 写 /root/dujiaoapi/uploads/
```

**B. 把 api 数据搬到文档默认 `/www/server/dujiao/`**(推荐,跟文档对齐):
```bash
systemctl stop dujiao-api
mkdir -p /www/server/dujiao
mv /root/dujiaoapi/* /www/server/dujiao/
mv /root/dujiaoapi/.secrets /www/server/dujiao/ 2>/dev/null
sed -i 's|/root/dujiaoapi|/www/server/dujiao|g' /etc/systemd/system/dujiao-api.service
systemctl daemon-reload
systemctl start dujiao-api
```

### Q. api 跑起来 CPU 100%,日志全是 `redis i/o timeout`

**A.** `config.yml` 的 `redis.host` 是 `redis`(docker DNS 名),非 Docker 部署时这个主机名解析到野生 IP。改成 `127.0.0.1` 即可。

### Q. 部署到 GitHub Actions tag 触发 release,但 CI 失败

**A.** 看具体 step error。最常见:
- **archives.files 路径**: `.goreleaser.yml` 改了路径但本机没测,push 前用 `goreleaser release --snapshot --skip publish` 跑一遍
- **缺 `GITHUB_TOKEN`**: workflow 里 `permissions.contents: write` 必须给
- **同名 tag 已发**: GoReleaser 默认拒绝覆盖,要么删旧 tag 再 push,要么 `.goreleaser.yml` 加 `release.mode: replace`

### Q. 外网访问不通,服务器内 curl 能通

**A.** 99% 是防火墙:
```bash
# ufw
ufw allow 8082/tcp
ufw reload

# 或 firewalld
firewall-cmd --permanent --add-port=8082/tcp && firewall-cmd --reload

# 或 iptables 直接
iptables -I INPUT -p tcp --dport 8082 -j ACCEPT
```

还得检查云厂商**安全组**(阿里云/腾讯云/AWS)有没放 8082。

## 后台 / 登录

### Q. admin 密码忘了

**A.** 用 `admin-tool` CLI 重置:
```bash
# 列管理员
/opt/dujiao/admin-tool list-admins

# 暂无内置 reset-password,需要直接改数据库:
systemctl stop dujiao-api
sqlite3 /opt/dujiao/db/dujiao.db
sqlite> UPDATE admins SET password_hash = '$2a$10$...' WHERE username = 'admin';
# bcrypt 生成新 hash:
#   echo -n '你的新密码' | htpasswd -bniBC 10 admin | cut -d: -f2
sqlite> UPDATE admins SET token_version = token_version + 1 WHERE username = 'admin';  # 强制旧 session 失效
sqlite> .exit
systemctl start dujiao-api
```

详见 [admin 后台使用](/admin/usage)。

### Q. 2FA 验证码丢了进不去

**A.**
```bash
/opt/dujiao/admin-tool reset-2fa --username admin
```

如果连 admin-tool 都没装,直接 SQL:
```sql
UPDATE admins SET totp_secret = '', totp_enabled_at = NULL, recovery_codes = '' WHERE username = 'admin';
```

### Q. 登录页能开,提交后转圈不返回

**A.** F12 看 Network。两种典型:
- `Failed to fetch` / CORS error → 后台站点伪静态没贴对,`/api/` 没反代到 8080
- `502 Bad Gateway` → api 进程挂了。`systemctl status dujiao-api`

## 订单 / 支付

### Q. 用户付了钱,订单还是 `pending`

**A.** 看支付商**回调**有没到。
```bash
grep -i 'payment callback' /var/log/dujiao/api.log | tail -30
```
- 没看到任何 callback 日志 → 支付商根本没回调你。检查回调 URL 是不是公网可达,有没拼对 `?provider=xxx&id=xxx`
- 看到 callback 但 `signature mismatch` → 密钥配错
- 看到 callback 但 `order not found` → `out_trade_no` 跟订单号对不上

### Q. 订单超过 15 分钟没付,会自动取消吗

**A.** 会,由 asynq 异步队列周期任务处理。如果 redis 挂了任务跑不起来 → 订单永远 pending → 锁住的卡密/号池永远占用。
```bash
# 看 redis
systemctl status redis-server
redis-cli ping
# 看 queue worker 状态
journalctl -u dujiao-api -f | grep asynq
```

### Q. 退款怎么操作

**A.** 后台 → 订单 → 找到这条 → "退款"按钮。
- **完全退款**: 调支付商 refund API + 卡密标记为 voided
- **部分退款**: 只支持 Stripe / PayPal,易支付一般不支持
- **超过 `max_refund_days`(默认 30 天)**: 后台拒绝退款,只能手工到支付商后台操作 + SQL 标记订单状态

## 卡密 / 号池

### Q. 号池(Codex)账号商品的库存为啥不是手填

**A.** `auto_secret_kind=codex_pool` 的商品库存**自动等于**:`codex_accounts WHERE status='ok' AND sold=false AND reserved_order_id=0` 的行数。后台改库存是无效的,只能加号池账号。

### Q. 库存有但下单失败,提示"库存不足"

**A.** 并发抢购时可能出现。一般是:
- 看 `codex_accounts` 是不是真的有满足条件的(`status='ok'` 且 `reserved_order_id=0`)
- 看是否 `reserved` 但没"释放"——可能 reservation 没归还。手动归还:
  ```sql
  UPDATE codex_accounts SET reserved_order_id=0, reserved_at=NULL
  WHERE reserved_order_id IN (SELECT id FROM orders WHERE status='cancelled');
  ```

### Q. 号池账号怎么批量导入

**A.** 后台 → 号池管理 → Codex → "批量导入" → 选 codex CLI 的 `auth.json` 文件(可一次传多个)。每个 auth.json 解析出 1 行账号。

## 性能 / 资源

### Q. 1GB 内存机能跑吗

**A.** 能,但要:
1. swap 加到 4GB(`fallocate -l 4G /swapfile && mkswap /swapfile && swapon /swapfile`)
2. 用 **二进制 fullstack**(单进程 ~70MB RAM)或**宝塔 headless**,不要 docker(docker 自身吃 200MB+)
3. `redis maxmemory 64mb`(够用)
4. SQLite 不要切 MySQL(MySQL 至少 200MB)

### Q. 后台加载好慢

**A.**
- 看 `/api/v1/admin/orders?per_page=50` 之类响应时间——超 500ms 多半是 SQLite 扫表慢
- 几万订单时,查 `orders` 表是否所有过滤字段都有 index:
  ```sql
  CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id);
  CREATE INDEX IF NOT EXISTS idx_orders_status_created ON orders(status, created_at);
  ```

## 更多

没找到答案?到 [GitHub Issues](https://github.com/TokensZhuanfa/Dujiao-Shop/issues) 提问,**贴上 `journalctl -u dujiao-api --since '10 min ago'` 输出**最快有回应。
