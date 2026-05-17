# Telegram Bot 服务介绍

dujiao-shop 内置 **Telegram Bot 通知**(免费,无第三方依赖)。本页是技术介绍,配置实操见 [通知中心配置](/guide/notifications#telegram-bot-通知)。

## 能干什么

| 场景 | 触发 | 效果 |
|---|---|---|
| 新订单提醒 | 用户付款成功 | 你的 Telegram 收到一条带订单号 / 商品 / 金额的消息 |
| 退款提醒 | 后台/支付商触发退款 | 同上 |
| 大额订单 | 金额超过你设的阈值 | 单独高亮提醒 |
| 库存预警 | 商品库存 < N | 提示该补卡密 / 加号池 |
| 验活告警 | Codex 号池连续 banned 数 > N | 提示该挑批新账号 |
| 部署事件 | api 启动 / 异常 panic | 第一时间知道服务挂了 |

## 跟 Telegram Login 区别

dujiao-shop 还支持**用 Telegram 账号登录前台**(`telegram_auth`),那是给买家用的,跟通知 bot 是**两个独立功能**:

| | Telegram Bot 通知 | Telegram 登录 |
|---|---|---|
| 谁用 | 站长(后台运营自己) | 买家(前台注册/登录) |
| 配置 | 后台 → 通知中心 | `config.yml` 的 `telegram_auth.*` |
| 推荐打开 | ✓ | 看你前台对 Telegram 用户多不多 |

两个功能可以**用同一个 Bot Token**,只是登录场景需要更多字段(`bot_username` / `mini_app_url` / `login_expire_seconds`)。

## 安全提示

- Bot token 等于密码,**绝不**贴到 git / 截图 / 公开 issue
- 群通知模式:把 bot 加进去之前,确认群里没有竞争对手 / 客户
- Mini App 模式需要 HTTPS,且 OAuth 校验逻辑已在 `src/api/internal/service/telegram_auth_service.go`,**勿绕过验签**

## 自托管 vs 公开 bot

你的 Bot 是**你自己的私有 bot**(`@your_dujiao_bot`),不是大家共用的。意味着:

- ✅ token 失效只影响你自己
- ✅ Telegram 不会知道你卖了多少订单(消息只发到你的 chat)
- ⚠️ Bot 进群后,所有群成员都能看到通知 → 别拉客户群

## 排查

| 现象 | 排查 |
|---|---|
| 配完后台测试不收到 | bot 没找你说过话(私聊场景必须先 `/start`)/ chat ID 错 / token 失效 |
| 群通知不到 | bot 没在群里 / 群禁止 bot 发消息 |
| 消息延迟 | 你的服务器到 Telegram api 的网络。中国大陆机推荐挂 CF Workers 中转 |
