# 通知中心配置

dujiao-shop 内置 3 个通知通道:邮件 / Telegram Bot / Webhook。后台 **系统设置 → 通知** 启用。

## 邮件通知

走 SMTP,先在 [config.yml](/config/yaml#email) 配好 SMTP host/账号/密码。

后台 → 通知 → 邮件 → 勾选触发场景:

- 新订单(发给运营自己)
- 订单付款(发给买家)
- 卡密交付(发给买家)
- 退款完成(发给买家)
- 注册验证码 / 找回密码

**测试**:配置完点"发送测试邮件"按钮,3 秒内到达即 SMTP 通。

## Telegram Bot 通知

发到你自己的 Telegram(适合实时收到大订单提醒)。

### 1. 创建 Bot

跟 [@BotFather](https://t.me/BotFather) 对话:

```
/newbot
> Bot 名字: My Dujiao Shop Bot
> Bot username: my_dujiao_shop_bot (必须以 _bot 结尾)
< 拿到 token: 1234567890:ABCDEF...
```

### 2. 拿你的 Chat ID

跟 [@userinfobot](https://t.me/userinfobot) 对话 → 它会回你的 numeric ID(如 `123456789`)。

> 群通知:把 bot 加进群,然后用 `https://api.telegram.org/bot<TOKEN>/getUpdates` 查 chat ID(负数)。

### 3. 后台填

后台 → 通知 → Telegram:

| 字段 | 值 |
|---|---|
| Bot Token | `1234567890:ABCDEF...` |
| Chat ID | 你的 ID 或群 ID |
| 触发场景 | 勾"新订单" / "退款" / "支付失败" 等 |

点"测试"按钮 → 应该收到 "测试消息 from dujiao-shop"。

## Webhook 通知

把事件 POST 到任意 URL(给企业微信 / 飞书 / 钉钉 / 自建系统)。

后台 → 通知 → Webhook → 新建:

| 字段 | 说明 |
|---|---|
| URL | 你的接收端,如 `https://oapi.dingtalk.com/robot/send?access_token=...` |
| Secret | 可选,验签密钥(`X-Dujiao-Signature` HMAC-SHA256) |
| 触发场景 | 同上 |
| 消息模板 | Markdown,可用变量 `{order_no}` `{amount}` `{user.email}` 等 |

### 模板变量

| 变量 | 含义 |
|---|---|
| `{event}` | order.paid / order.refunded / user.register / ... |
| `{order_no}` | 订单号 |
| `{amount}` | 订单金额(decimal) |
| `{user.id}` / `{user.email}` | 下单用户 |
| `{products}` | 商品列表(用 ` ` 拼接)|
| `{paid_at}` / `{refunded_at}` | ISO 时间 |

### 签名验证(可选)

如果填了 `Secret`,请求会带 header:
```
X-Dujiao-Signature: <hex(hmac-sha256(secret, body))>
```
你的接收端比对一遍才信。

## 排查

| 现象 | 排查 |
|---|---|
| 邮件不到 | 看 api 日志 `grep -i 'send_email\|smtp'`;垃圾邮件箱;SMTP 端口被运营商封 |
| Telegram bot 没消息 | 确认 chat ID 正确;bot 没被你 block;`curl https://api.telegram.org/bot<TOKEN>/getMe` 看 token 有效 |
| Webhook 不响应 | 看 api 日志的 `webhook_dispatch_failed`;接收端 4xx/5xx 内容 |
