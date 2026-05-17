# Codex 号池

OpenAI / ChatGPT 账号管理体系。后台 `号池管理 → Codex` 进入。

## 功能

- **自动 token 轮换**: refresh_token 临过期前自动换 access_token,无需人工
- **额度刷新**: 定时拉 `/backend-api/wham/usage` 拿 5h / 7d 窗口的 `used_percent`,实时反映可用度
- **状态识别**: `ok` / `needs_refresh` / `banned` / `invalid` 四态自动迁移
- **预占语义**: 下单事务里**原子**占住一个 `ok` 账号(`reserved_order_id` + `reserved_at`),付款转 `sold`,**超时 / 取消自动归还**(对齐文本卡密的 reservation 语义)
- **库存即时反映**: 商品 `auto_secret_kind = codex_pool` 时,库存等于 `池里 ok && !reserved` 的账号数

## 数据模型

`codex_accounts` 表关键字段:

| 字段 | 类型 | 用途 |
|---|---|---|
| `email` | varchar | 账号邮箱 |
| `account_id` | varchar | chatgpt_account_id |
| `plan` | varchar | plus / pro / free / team |
| `status` | varchar | ok / needs_refresh / banned / invalid |
| `access_token` | text | OAuth access |
| `refresh_token` | text | OAuth refresh |
| `access_exp` | int64 | unix ts (秒) |
| `sold` | bool | 已售出 |
| `sold_order_no` | varchar | 关联订单号 |
| `reserved_order_id` | uint | 预占的订单 ID,0 = 未预占 |
| `primary_used_percent` | float | 5h 窗口已用 % |
| `secondary_used_percent` | float | 7d 窗口已用 % |
| `banned_at` | datetime | 首次识别为 banned 的时间 |

## 商品对接

后台 → 商品管理 → 编辑:

- 把 `auto_secret_kind` 设为 `codex_pool`
- 库存自动 = 池里可用账号数(无须手填)

下单流水:

1. 用户提交订单 → api 事务里 `UPDATE codex_accounts SET reserved_order_id=?, reserved_at=NOW() WHERE status='ok' AND reserved_order_id=0 LIMIT 1`
2. 付款成功 → `UPDATE ... SET sold=true, sold_order_no=?, sold_at=NOW() WHERE id=?`
3. 订单超时 / 取消 → `UPDATE ... SET reserved_order_id=0, reserved_at=NULL WHERE reserved_order_id=?`(归还)

## 交付格式: CpaMC / Sub2api 双下载

买家订单详情页有两个按钮:
- **逐个账号**: 单独下载某一个账号的 JSON(标准 OpenAI auth.json 格式)
- **打包**: 所有账号合并成一个 zip / sub2api 配置

源码:`src/admin/src/views/orders/CodexAccountsModal.vue` + `src/api/internal/handler/order_codex_account_handler.go`

## 验活机制

后台每 N 分钟跑一次:

1. 拉取每条 `ok` 账号的 `/backend-api/wham/usage`
2. 401 / 403 → `banned`
3. token 即将过期 → 用 refresh_token 换新 access_token → 更新 `last_at_updated_at`
4. usage 拉取成功 → 更新 `primary_used_percent` / `secondary_used_percent`
5. `ban_fail_count` 200 时归零,防止偶发误判堆积

可在配置里调验活频率:
```yaml
codex_pool:
  health_check_interval: 5m  # 默认 5 分钟
  ban_fail_threshold: 3       # 连续 N 次失败才标 banned
```
