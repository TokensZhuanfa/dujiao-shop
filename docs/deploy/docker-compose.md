# Docker Compose 部署

最简单、推荐第一次部署用。5 分钟跑起来。

## 系统要求

- Debian 12 / Ubuntu 22.04 +
- Docker + Compose plugin
- 至少 1 GB RAM(小内存机会比较紧,建议 2 GB+)

## 部署步骤

```bash
# 1. 克隆仓库到 /opt/dujiao-shop
git clone https://github.com/TokensZhuanfa/Dujiao-Shop.git /opt/dujiao-shop
cd /opt/dujiao-shop/deploy

# 2. 改配置 (jwt_secret / 支付通道 / 邮件等)
cp config.template.yml config.yml
vi config.yml

# 3. 一键部署 (装 docker + 起 4 个容器: redis + api + admin + user)
sudo bash deploy.sh

# 4. 浏览器访问
#   admin: http://你的IP:8081
#   user : http://你的IP:80
```

## 升级

```bash
cd /opt/dujiao-shop && git pull
cd deploy && docker compose up -d --build
```

## 配置必改字段

`config.yml` 至少改下面 4 处:
- `jwt_secret`:`openssl rand -base64 32`
- `redis.address`:默认 `redis:6379`(docker compose 内网解析,**不要改**)
- `payment.*`:你接入的支付通道
- `smtp.*`:邮件发送

## 排查

```bash
# 看所有容器
docker compose ps

# 看 api 日志
docker compose logs -f api

# 重启
docker compose restart api
```
