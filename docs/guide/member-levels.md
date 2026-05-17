# 会员等级

dujiao-shop 自带等级体系,可按"累计消费"或手工设定升级。

## 配置等级

后台 **会员管理 → 会员等级** → 新建:

| 字段 | 说明 |
|---|---|
| 等级名 | `LV0 普通会员` / `LV1 银卡` / ... |
| 升级条件 | `total_spent >= N`(累计消费金额) |
| 折扣 | 全场默认折扣(0.95 = 95 折);单 SKU 价可在商品里覆盖 |
| 头像装饰 | 等级图标(可选)|
| 描述 | 用户中心展示用 |

## 升级机制

- **自动**:用户付款后异步任务触发 `total_spent` 累加 → 重算等级
- **手动**:后台 → 用户详情 → "设为 LV3"

降级:**目前不自动降**(防止用户退款后心理预期落空)。手动操作。

## 等级权益

- **全场折扣**:`member_level_prices` 表写每个 SKU 的等级价(详见 [优惠券与活动价](/guide/coupons))
- **优惠券限定**:某些优惠券限"LV2 及以上才能用"
- **礼品卡兑换**:某些礼品卡限"LV3 及以上才能兑"
- **专属商品**:商品也能限会员等级才能下单(`min_level_id` 字段)

## 用户视角

用户中心首屏展示:
- 当前等级 + 头像装饰
- 距下一级还差多少累计消费
- 当前等级的折扣 / 权益列表

## SQL 速查

```sql
-- 看每个等级有多少用户
SELECT ml.name, count(u.id) AS user_count
FROM member_levels ml LEFT JOIN users u ON u.member_level_id = ml.id
GROUP BY ml.id ORDER BY ml.id;

-- 强升一个用户
UPDATE users SET member_level_id = 3 WHERE id = 42;
```

## 排查

| 现象 | 原因 |
|---|---|
| 用户付款后等级没升 | 异步任务可能堆积,看 `journalctl -u dujiao-api -f \| grep level_upgrade` |
| 用户看到的折扣价 vs 后台不一样 | `member_level_prices` 表里有具体 SKU 的覆盖值;清空后回到等级默认折扣 |
