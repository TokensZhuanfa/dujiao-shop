# 更新日志

完整发布历史在 [GitHub Releases](https://github.com/TokensZhuanfa/Dujiao-Shop/releases) 页面。

> Release 由 GitHub Actions 自动构建,**每个版本都附带 SHA-256 校验文件**,下载后请校验。

## v1.0.0 (2026-05-16)

首个独立 release。在上游 [dujiao-next](https://github.com/dujiao-next) 基础上的二次开发汇总。

### 新增

- **Codex 号池**:OpenAI / ChatGPT 账号 token 自动轮换、额度刷新、状态识别(`ok` / `needs_refresh` / `banned` / `invalid`),库存即时反映可用账号数
- **号池型商品**:`auto_secret_kind = codex_pool`,下单事务里原子预占账号,付款后转 sold,超时 / 取消归还(对齐文本卡密 reservation 语义)
- **文件型卡密**:单文件 ≤ 1 GB,从订单页直接下载
- **CpaMC / Sub2api 双格式**:号池账号单个下载 / 全部打包
- **订单号 base32 高熵**:防扫单遍历
- **暴破锁定**:5 分钟内同 IP 失败 5 次锁 15 分钟
- **二进制发布**:`fullstack`(单文件全栈)+ `headless`(api only + nginx 模板)+ `admin-tool`(运维 CLI)
- **GitHub Pages 文档站**:本站

### 默认配置

- bcrypt cost = 10
- 密码最小 8 位(默认要求大小写 + 数字)
- JWT 24 小时过期(用户端 "记住我" 7 天)
- session token_version 机制:改密自动失效所有现有 token

## 发布流程

打 tag → GitHub Actions 自动出 release。本节流程详见 [开源仓库与贡献](/intro/repos)。

## 版本号约定

[语义化版本](https://semver.org):

- **major** `v2.0.0`:破坏性改动(数据库迁移不可逆 / 配置格式变化 / API 大改)
- **minor** `v1.2.0`:新功能,向后兼容
- **patch** `v1.0.1`:bug 修复
