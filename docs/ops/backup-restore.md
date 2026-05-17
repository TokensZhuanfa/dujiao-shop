# 备份与恢复

dujiao-shop 数据分**4 类**,每类备份方式不同。**生产环境强烈建议每天 cron 备份**。

## 数据分布

| 数据 | 位置(Docker) | 位置(二进制/宝塔) | 必备份 |
|---|---|---|---|
| SQLite 数据库 | volume `api_db` | `/opt/dujiao/db/dujiao.db` 或 `/www/server/dujiao/db/dujiao.db` | ✅ 最关键 |
| Redis (队列 + 缓存) | volume `redis_data` | `/var/lib/redis/dump.rdb` | ⚠️ 一般不备(可重建) |
| 上传文件 | volume `api_uploads` | `/opt/dujiao/uploads/` | ✅ 用户头像 / 商品图 |
| 文件型卡密 | volume `api_credentials` | `/opt/dujiao/credentials/` | ✅ **绝对**别丢,买家会找你要 |
| 日志 | volume `api_logs` | `/opt/dujiao/logs/` | 🟢 不需要 |
| config.yml | `deploy/config.yml` | `/opt/dujiao/config.yml` | ⚠️ secret 在里面,备份后**加密保存** |

## 推荐备份策略

### 简版:整目录 tar 一份(每天)

```bash
#!/usr/bin/env bash
# /opt/dujiao/backup.sh — 加到 crontab "0 3 * * *" 每天 3 点跑
set -euo pipefail

ROOT=/opt/dujiao    # 或 /www/server/dujiao
BACKUP=/data/backup/dujiao
DATE=$(date +%Y%m%d-%H%M%S)
mkdir -p "$BACKUP"

# 1. SQLite 用 .backup 命令保证一致性(直接 cp 在写入时可能损坏)
sqlite3 "$ROOT/db/dujiao.db" ".backup $BACKUP/db-$DATE.sqlite"
gzip "$BACKUP/db-$DATE.sqlite"

# 2. 上传 + 卡密 + config
tar czf "$BACKUP/files-$DATE.tar.gz" \
    -C "$ROOT" uploads credentials config.yml .secrets 2>/dev/null

# 3. 清 30 天前的
find "$BACKUP" -mtime +30 -delete

# 4. (可选) rsync 到异地
# rsync -avz "$BACKUP/" backup-server:/dujiao-bak/
```

`crontab -e`:
```cron
0 3 * * * /opt/dujiao/backup.sh >> /var/log/dujiao-backup.log 2>&1
```

### Docker 部署的备份变种

```bash
# SQLite (停 api 几秒做一致性快照,或 sqlite3 .backup 模式)
docker compose -f /opt/dujiao-shop/deploy/docker-compose.yml \
    exec -T api sqlite3 /app/db/dujiao.db ".backup /tmp/dump.sqlite"
docker cp dujiao-api-1:/tmp/dump.sqlite "$BACKUP/db-$DATE.sqlite"

# 上传 / 卡密 (docker volume 直接 tar)
docker run --rm -v dujiao_api_uploads:/data -v "$BACKUP":/backup alpine \
    tar czf "/backup/uploads-$DATE.tar.gz" -C /data .
docker run --rm -v dujiao_api_credentials:/data -v "$BACKUP":/backup alpine \
    tar czf "/backup/credentials-$DATE.tar.gz" -C /data .
```

## 恢复

### SQLite 恢复

```bash
# 1. 停 api
systemctl stop dujiao-api    # 或 docker compose stop api

# 2. 备份当前(防呆)
mv /opt/dujiao/db/dujiao.db{,.before-restore-$(date +%s)}

# 3. 解 gzip + 落位
gunzip -c /data/backup/dujiao/db-20260518-030000.sqlite.gz > /opt/dujiao/db/dujiao.db
chown www:www /opt/dujiao/db/dujiao.db   # docker 部署改成 dujiao:dujiao

# 4. 启 api 看日志
systemctl start dujiao-api
journalctl -u dujiao-api -f
```

### 文件恢复

```bash
tar xzf /data/backup/dujiao/files-20260518-030000.tar.gz -C /opt/dujiao
chown -R www:www /opt/dujiao/{uploads,credentials}  # 看你部署用户
systemctl restart dujiao-api
```

## 数据库迁移(SQLite → MySQL)

适合数据量上来(几万订单 / 几十万 SKU)时:

```bash
# 1. 装 sqlite3 / mysql-client
apt install -y sqlite3 mysql-client

# 2. dump SQLite
sqlite3 /opt/dujiao/db/dujiao.db .dump > dump.sql

# 3. 改成 MySQL 兼容(替换几个语法)
sed -i \
    -e 's/AUTOINCREMENT/AUTO_INCREMENT/gi' \
    -e 's/BEGIN TRANSACTION/START TRANSACTION/gi' \
    -e 's/PRAGMA[^;]*;//gi' \
    dump.sql

# 4. 导入 MySQL
mysql -uroot -p dujiao < dump.sql

# 5. 改 config.yml
#   driver: mysql
#   dsn: dujiao:password@tcp(127.0.0.1:3306)/dujiao?charset=utf8mb4&parseTime=True

# 6. 重启 api
systemctl restart dujiao-api
```

> SQLite → MySQL 迁移**强烈建议先在测试环境跑一遍**。数据类型、自增主键、JSON 字段都有细微差异。

## 异地备份

把 `/data/backup/dujiao/` 同步到对象存储(七牛/阿里 OSS/B2/S3):

```bash
# 七牛(qshell)
qshell qupload2 --src-dir=/data/backup/dujiao --bucket=your-bucket

# rclone(支持几乎所有云存储)
rclone copy /data/backup/dujiao remote:bak/dujiao --max-age 35d
```

## 验证备份能恢复

**没验证过的备份等于没备份**。每月跑一次:

```bash
# 在测试机上恢复一份昨天的备份,看启不来 → 启起来了 admin 能登录 → 商品列表能查
```
