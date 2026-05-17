# 管理后台使用速览

后台默认在 `https://your.com/admin/`。

## 登录

- **用户名**: 配置文件 `bootstrap.default_admin_username`(默认 `admin`)
- **密码**: 配置文件 `bootstrap.default_admin_password`,**首次登录后请立刻改**
- **2FA**: 设置 → 安全 → 启用 TOTP(强烈建议)

## 左侧导航(常用)

| 菜单 | 作用 |
|---|---|
| **仪表盘** | KPI / 订单趋势 / 收入 / 系统状态 |
| **商品管理** | 商品 CRUD,SKU,分类,标签,定价,库存 |
| **订单管理** | 订单查询 / 退款 / 手动标记 / 导出 |
| **卡密管理** | 文本卡密池 / 文件卡密 / 卡密批次 |
| **号池管理 → Codex** | OpenAI/ChatGPT 账号管理(详见 [Codex 号池](/customization/codex-pool)) |
| **会员管理** | 用户列表 / 等级 / 钱包余额 |
| **优惠券** | 创建优惠码 / 礼品卡 |
| **支付通道** | 接入支付商(详见 [支付通道接入](/admin/payment)) |
| **联盟营销** | 推广员 / 佣金 / 提现 |
| **内容** | Banner / 文章 / 站点配置 |
| **系统设置** | site_name / SMTP / Telegram bot |
| **审计日志** | 谁在什么时候改了什么 |

## 商品 CRUD 关键字段

进**商品管理 → 添加商品**:

| 字段 | 含义 |
|---|---|
| 标题 / Slug | 浏览器 URL 是 `/p/{slug}` |
| 分类 | 商品在前台属于哪一类 |
| 价格 + 划线价 + 会员价 | 划线价不为零会显示打折效果 |
| 描述 | 富文本,可贴图 |
| **`auto_secret_kind`** | **核心**:`text`(文本卡密)/ `file`(文件卡密)/ `codex_pool`(号池账号) |
| 库存 | text/file 类自己填;codex_pool 自动 |
| 单笔限购 | 防黄牛 |
| 是否上架 | 不上架不在前台显示 |

## 订单管理常用操作

- **改状态**: `paid` / `cancelled` / `refunded`
- **手动标记已付**: 用户线下转账场景
- **退款**: 调支付商 refund API + 卡密标 voided
- **重发卡密邮件**: 卖给用户后用户没收到邮件
- **查交付记录**: 看用户收到的卡密原文

## 卡密管理

### 文本卡密

1. **添加批次**: 给一组卡密命名,如 "618-MISC"
2. **导入**:
   - 粘贴(每行一条)
   - 上传 CSV(第 1 列是卡密)
3. **绑定到商品**: 在批次详情设置允许哪些商品消费

### 文件卡密

1. 选商品的 `auto_secret_kind=file`
2. 商品详情 → 上传文件(单文件 ≤ 1 GB)
3. 一个文件 = 1 份库存。买家付款后获 1 份文件下载链接

### 号池卡密(Codex)

详见 [Codex 号池](/customization/codex-pool)。

## 系统设置

### 修改 site_name / icon / Hero 文案

后台 → 系统设置 → 站点 → 改 site_name / hero_title / hero_subtitle / logo URL。

### 邮件

后台 → 系统设置 → 邮件 → 填 SMTP host/port/账号/密码 → "测试邮件"按钮。

### Telegram Bot 通知

后台 → 系统设置 → Telegram Bot:
- 输入 Bot Token(`@BotFather`)
- 加 Chat ID(你自己的 Telegram numeric ID,可问 `@userinfobot`)
- "测试通知"
- 触发:新订单 / 退款 / 大额支付 / 异常

### 多语言

后台显示语言在右上角切。默认 zh-CN,可切 zh-TW / en-US。
**前台**语言由用户浏览器 Accept-Language 自动判断,可在 URL 加 `?locale=en-US` 强制。

## 审计日志

后台 → 审计日志 → 看每条 admin 操作:

| 字段 | 例 |
|---|---|
| 时间 | 2026-05-18 04:30 |
| 操作人 | admin |
| 操作 | UPDATE_ORDER_STATUS |
| 目标 | order:O-AB1234 |
| 详情 | from=pending to=paid |
| IP | 1.2.3.4 |

**多人共管 / 出问题溯源时关键**。建议给每个 admin 单独的账号,不要共用 admin 一个。

## 添加 admin 用户

```bash
# 进 SQLite 直接插入(暂无 admin-tool 命令)
sqlite3 /opt/dujiao/db/dujiao.db <<EOF
INSERT INTO admins(username, password_hash, is_super, created_at)
VALUES ('alice', '$2a$10$...', 0, datetime('now'));
EOF
```

bcrypt hash 生成:`echo -n '密码' | htpasswd -bniBC 10 alice | cut -d: -f2`。
`is_super=0` 表示非超管,可配 RBAC 角色限制权限。

## CLI 速查

```bash
# 列管理员
/opt/dujiao/admin-tool list-admins

# 重置 2FA
/opt/dujiao/admin-tool reset-2fa --username admin
```
