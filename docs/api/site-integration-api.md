# 站点对接 API 文档

跟另一个 dujiao-shop 实例互联用的 endpoint。先看 [站点对接说明](/api/site-integration) 拿到配对的 API Key + Secret。

## 鉴权

每个请求都要 3 个 header:

| Header | 值 |
|---|---|
| `X-DJ-Key` | 主站给的 API Key |
| `X-DJ-Timestamp` | 当前 unix 时间戳(秒) |
| `X-DJ-Nonce` | UUID v4,5 分钟内不重复 |
| `X-DJ-Signature` | `hex(hmac-sha256(secret, key + timestamp + nonce + sha256(body)))` |

签名示例(Go):

```go
import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
)

func sign(secret, key, timestamp, nonce string, body []byte) string {
    bodyHash := sha256.Sum256(body)
    payload := key + timestamp + nonce + hex.EncodeToString(bodyHash[:])
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write([]byte(payload))
    return hex.EncodeToString(mac.Sum(nil))
}
```

签名失败 → 403 拒绝。

## 端点

### 拉商品列表

```
GET /api/v1/upstream/products
```

参数:
- `page` (默认 1)
- `per_page` (默认 50,最大 200)
- `category_id` 可选
- `updated_after` ISO 8601,只拉这时间之后改过的

响应:
```json
{
  "status_code": 0,
  "data": {
    "products": [
      {
        "id": 42,
        "slug": "openai-plus-1m",
        "title": "ChatGPT Plus 1 个月",
        "price": "99.00",
        "stock": 15,
        "auto_secret_kind": "codex_pool",
        "category": "AI 账号"
      }
    ],
    "total": 87,
    "page": 1,
    "per_page": 50
  }
}
```

### 创建分销订单

```
POST /api/v1/upstream/orders
```

Body:
```json
{
  "product_id": 42,
  "sku_id": 100,
  "quantity": 1,
  "downstream_order_no": "DS-XYZ123",
  "customer_email": "buyer@example.com",
  "metadata": { "channel": "agent-foo" }
}
```

响应:
```json
{
  "status_code": 0,
  "data": {
    "upstream_order_no": "O-3KZJ7HW9MX",
    "amount": "99.00",
    "expire_at": "2026-05-18T05:30:00Z"
  }
}
```

`upstream_order_no` 之后查交付状态 / 通知付款用。

### 通知主站付款完成

```
POST /api/v1/upstream/orders/:upstream_order_no/paid
```

Body:
```json
{
  "downstream_payment_id": "pay_xxx",
  "paid_amount": "99.00",
  "paid_at": "2026-05-18T05:15:00Z"
}
```

响应:
```json
{
  "status_code": 0,
  "data": {
    "fulfillment": {
      "status": "ready",
      "payload": "卡密内容..."
    }
  }
}
```

> 注:大部分场景主站还要做"等代理站资金到账后再交付"判定,这一步可在主站后台跑账户对账后人工触发。

### 查询订单状态

```
GET /api/v1/upstream/orders/:upstream_order_no
```

响应包含订单状态 + (若已付款且 ready)卡密 payload。

### 主站回调代理站(主站 → 代理)

主站在订单状态变化时(交付完成 / 退款 / 卡密变更)向**对接时配的回调 URL** POST:

```json
{
  "event": "order.fulfilled",
  "upstream_order_no": "O-3KZJ7HW9MX",
  "downstream_order_no": "DS-XYZ123",
  "fulfillment": {
    "status": "ready",
    "payload": "..."
  },
  "timestamp": "2026-05-18T05:16:00Z"
}
```

带跟普通请求一样的签名 header,代理站验签 → 处理。

## 错误码

| status_code | HTTP | 含义 |
|---|---|---|
| 0 | 200 | success |
| 4001 | 400 | 参数缺失 / 格式错 |
| 4003 | 403 | 鉴权失败(key 不存在 / 签名错 / nonce 重放 / timestamp 偏差 > 5 min) |
| 4004 | 404 | 资源不存在(订单号 / 商品 ID 错) |
| 4090 | 409 | 状态冲突(订单已交付却又来一次付款通知) |
| 4099 | 409 | 库存不足 |
| 5000 | 500 | 主站内部错误,看 `request_id` 找 log |

## 调试

主站后台 → 站点对接 → 选某条 → "请求日志"
- 看代理站最近 30 次请求的 status / latency / 签名是否对

代理站后台同理 → "回调日志" 看主站推过来的 webhook。
