# 发布流程

dujiao-shop 用 [GoReleaser](https://goreleaser.com) + GitHub Actions 自动发布。流程:

```bash
# 1. 改完代码,本地 commit
git add -A && git commit -m "..."

# 2. 打 tag (语义化版本)
git tag v1.2.3
git push origin main v1.2.3

# 3. GitHub Actions 自动:
#    - build admin/user 前端
#    - embed 进 api/internal/web/
#    - goreleaser 出多平台 tarball
#    - 创建 Release + 上传 artifacts
```

## 发布构件清单

每次 release 自动生成 7 个 artifact:

| 文件 | 大小 | 用途 |
|---|---|---|
| `dujiao-shop-fullstack_*_linux_amd64.tar.gz` | ~27 MB | 单文件全栈,x86_64 |
| `dujiao-shop-fullstack_*_linux_arm64.tar.gz` | ~25 MB | 同上,ARM64 |
| `dujiao-shop-headless_*_linux_amd64.tar.gz` | ~26 MB | api only,x86_64 |
| `dujiao-shop-headless_*_linux_arm64.tar.gz` | ~24 MB | 同上,ARM64 |
| `dujiao-shop-admin-v*.zip` | ~816 KB | admin 前端 dist |
| `dujiao-shop-user-v*.zip` | ~464 KB | user 前端 dist |
| `checksums.txt` | 454 B | SHA-256 校验 |

## 配置文件

| 文件 | 作用 |
|---|---|
| `.github/workflows/release.yml` | CI 流程 |
| `.goreleaser.yml` | GoReleaser 配置(build / archive / release) |
| `deploy/release/install.sh` | tarball 自带的一键安装脚本 |
| `deploy/release/dujiao.service` | systemd unit |
| `deploy/release/README-binary.md` | tarball 内附的 INSTALL.md |

## 文档站发布

`docs/` 目录的改动会触发另一个 workflow [`.github/workflows/docs.yml`](https://github.com/TokensZhuanfa/Dujiao-Shop/blob/main/.github/workflows/docs.yml),自动 build 并发到 GitHub Pages。

**触发条件**: push 到 main + `docs/**` 路径有变化。

**部署目标**: `https://TokensZhuanfa.github.io/Dujiao-Shop/`

## 版本号约定

跟随[语义化版本](https://semver.org):

- **major** (`v2.0.0`):破坏性改动(数据库迁移不可逆 / 配置格式变化 / API 大改)
- **minor** (`v1.2.0`):新功能,向后兼容
- **patch** (`v1.0.1`):bug 修复

不要复用已发布的 tag。如果一定要重发(比如 release artifact build 错),先删旧 tag:

```bash
git tag -d v1.0.0
git push origin :refs/tags/v1.0.0
git tag v1.0.0
git push origin v1.0.0
```
