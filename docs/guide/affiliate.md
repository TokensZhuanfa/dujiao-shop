# 分销推广 (联盟营销)

让用户通过专属链接推广,带新人下单后拿佣金。后台 **会员管理 → 分销** 配置。

## 启用

后台 → 系统设置 → 分销:

| 字段 | 说明 |
|---|---|
| 启用 | 总开关 |
| 佣金比例 | 推广人拿订单金额的 N%(默认 10%) |
| 确认天数 | 订单付款后过 N 天才生效(防退款薅羊毛,默认 7 天) |
| 最小提现金额 | 累计佣金 ≥ N 才能提现申请 |
| 支持的提现通道 | 支付宝 / 银行卡 / USDT 等(让用户自填账号) |

## 推广人流程

1. 用户进个人中心 → 分销 → 拿到自己的推广链接 `https://your.com/?aff=USER_ID`
2. 分享链接(微信群 / 朋友圈 / 博客)
3. 访客点击 → 浏览器 cookie 写入 `aff=USER_ID`,30 天有效
4. 访客注册 / 下单时,系统读 cookie,写入 `affiliate_profiles.referrer_id`
5. 该用户后续每笔订单 → 推广人拿佣金

## 数据模型

| 表 | 用途 |
|---|---|
| `affiliate_profiles` | 谁推广了谁,绑定关系 |
| `affiliate_clicks` | 推广链接点击日志(防刷分析) |
| `affiliate_commissions` | 每笔订单产生的佣金记录 |
| `affiliate_withdraw_requests` | 提现申请,后台审核 |

## 佣金状态

- **`pending`**:订单已付款,但未过 `confirm_days`
- **`confirmed`**:可提现
- **`refunded`**:订单退款后扣回(`confirmed` 不可逆,`pending` 直接撤)
- **`withdrawn`**:已发放给推广人

## 提现处理

后台 **分销 → 提现申请** → 看待审核列表:

1. 核对账号信息(支付宝 / USDT 地址)
2. 线下打款 / 链上转账
3. 后台标"已发放" → 状态变 `withdrawn`

> 自动打款**不内置**(各支付商 API 不同,你自己接你的),用线下手动 + 后台标记最简单。

## 防作弊

- 同 IP 自推自不计佣金
- 同 IP / 同设备指纹连续注册多个号 → 后台人工标记 `disabled`
- 退款触发佣金回扣

## SQL 速查

```sql
-- 待审核提现总额
SELECT count(*), sum(amount) FROM affiliate_withdraw_requests WHERE status='pending';

-- 某推广人累计佣金
SELECT
  sum(CASE WHEN status='confirmed' THEN amount END) AS withdrawable,
  sum(CASE WHEN status='pending' THEN amount END) AS pending
FROM affiliate_commissions WHERE referrer_id = 42;
```
