# 升级与迁移

dujiao-shop 升级**对 config.yml 和数据库通常向后兼容**,跨大版本(v1 → v2)时另说。本文按部署方式分三段。

## 通用准备

任何升级前先:

1. **备份**:见 [备份与恢复](/ops/backup-restore)。`db/` + `uploads/` + `credentials/` + `config.yml` 都要
2. **看 changelog**:[发布页](https://github.com/TokensZhuanfa/Dujiao-Shop/releases) 看新版有没有破坏性改动
3. **错峰升级**:挑订单量低的时段,大版本至少 30 分钟停机预算

## Docker Compose 升级

```bash
cd /opt/dujiao-shop
git pull                    # 拉新代码
cd deploy
docker compose pull         # 拉新 base 镜像 (redis)
docker compose up -d --build   # rebuild + 滚动重启
docker compose logs -f api  # 看启动是否健康
```

回滚:

```bash
git checkout <旧 commit>
docker compose up -d --build
```

## 二进制升级

下载新版 tarball → 解压 → `install.sh` 自动覆盖二进制保留数据:

```bash
cd /tmp
# 把 URL 里的 v1.0.1 替换成你要升到的新 tag
curl -L -O https://github.com/TokensZhuanfa/Dujiao-Shop/releases/download/v1.0.1/dujiao-shop-fullstack_v1.0.1_linux_amd64.tar.gz
tar xzf dujiao-shop-fullstack_*.tar.gz
cd dujiao-shop-fullstack_*/
sudo ./install.sh         # 覆盖二进制,保留 config.yml + db/ + uploads/ + credentials/
sudo systemctl restart dujiao
sudo journalctl -u dujiao -f
```

回滚:把上一版的 tarball 解压再跑一次 `install.sh`。

## 宝塔 / 手动部署升级

后端二进制 + 前端 dist 分开升。把下面所有 URL 里的 `v1.0.1` 整体改成新 tag:

```bash
cd /www/server/dujiao

# 后端
curl -L -o headless.tar.gz https://github.com/TokensZhuanfa/Dujiao-Shop/releases/download/v1.0.1/dujiao-shop-headless_v1.0.1_linux_amd64.tar.gz
tar xzf headless.tar.gz   # 覆盖 dujiao-api-headless + admin-tool + install.sh + INSTALL.md
chmod +x dujiao-api-headless admin-tool install.sh
systemctl restart dujiao-api

# 前端
curl -L -o /tmp/admin.zip https://github.com/TokensZhuanfa/Dujiao-Shop/releases/download/v1.0.1/dujiao-shop-admin-v1.0.1.zip
curl -L -o /tmp/user.zip  https://github.com/TokensZhuanfa/Dujiao-Shop/releases/download/v1.0.1/dujiao-shop-user-v1.0.1.zip
unzip -qo /tmp/admin.zip -d /www/wwwroot/dujiao-admin/
unzip -qo /tmp/user.zip  -d /www/wwwroot/dujiao-user/
chown -R www:www /www/wwwroot/dujiao-{admin,user}
```

## 数据库迁移自动跑

dujiao-api 启动时**自动跑** GORM AutoMigrate:加新列 / 加新索引 / 加新表。

不会做的事:删列 / 改列类型 / 重命名(数据库迁移历史不可逆)。新版本如果需要这种,changelog 会显式说明 + 提供 SQL。

## 跨部署方式迁移

把 dujiao-shop 从 Docker → 二进制(或反向),核心是搬数据:

```bash
# 1. 老机停服务并备份
systemctl stop dujiao-api       # 或 docker compose stop
tar czf /tmp/dujiao-data.tar.gz \
    -C /opt/dujiao db uploads credentials config.yml .secrets

# 2. 新机按目标方式装好框架但 *不* 起服务
sudo ./install.sh

# 3. 把数据搬过去
scp /tmp/dujiao-data.tar.gz new-host:/tmp/
ssh new-host 'tar xzf /tmp/dujiao-data.tar.gz -C /opt/dujiao'

# 4. 新机启
ssh new-host 'systemctl start dujiao && journalctl -u dujiao -f'

# 5. 老机停服务后 DNS 切到新机
```

## 数据库引擎切换(SQLite ↔ MySQL)

见 [备份与恢复 → 数据库迁移](/ops/backup-restore#数据库迁移sqlite--mysql)。

## 大版本(v1 → v2)升级

破坏性改动有 changelog 标 `BREAKING:`。常见处理:

- 配置字段重命名 → 升级前先在老版改成新名,再换 binary
- 数据库列删除 → 升级前 SQL 备份要删的列内容,**升级后无法恢复**
- API 路径变化 → 接你的 dujiao-shop 的外部系统要跟着改
