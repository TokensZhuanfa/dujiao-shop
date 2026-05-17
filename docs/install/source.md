# 从源码编译 (开发者)

## 系统要求

- Go 1.25+
- Node.js 24+
- Redis 7+

## 步骤

```bash
git clone https://github.com/TokensZhuanfa/Dujiao-Shop.git
cd Dujiao-Shop

# api
cd src/api
go run ./cmd/server                  # 开发模式

# 或 build:
go build -tags release -o dujiao-api ./cmd/server
```

```bash
# admin (默认走独立 nginx)
cd src/admin && npm ci && npm run dev      # localhost:5173
# 或 build:
npm run build                              # 产物在 dist/
```

```bash
# user
cd src/user && npm ci && npm run dev       # localhost:5174
```

## 编出**单文件全栈** (api 内嵌前端)

```bash
cd src/admin && npm run build:fullstack && cd -
cd src/user  && npm run build              && cd -
mkdir -p src/api/internal/web/dist
cp -r src/admin/dist src/api/internal/web/dist/admin
cp -r src/user/dist  src/api/internal/web/dist/user
cd src/api && go build -tags 'release fullstack' -o dujiao-api ./cmd/server
```

## 配置文件位置

启动时按以下顺序查找 `config.yml`:

1. `--config <path>` 命令行参数
2. `./config.yml` 当前目录
3. `/etc/dujiao/config.yml`

```bash
cp deploy/config.template.yml config.yml
vi config.yml
./dujiao-api
```

## 开发调试

```bash
# api 全文搜索
grep -rn "你要搜的关键字" src/api/internal

# admin 前端开发服务器(热更新)
cd src/admin && npm run dev

# 同时跑 api + admin + user (3 个终端)
```
