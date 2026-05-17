# 拉取上游变更

本仓库的 `src/api`、`src/admin`、`src/user` 三个目录的代码源自上游开源项目。如果上游有新功能或安全修复需要 cherry-pick 过来,本页是流程。

> **大多数用户不需要这一节**——下载 release 或 `git pull` main 就够了。只有当你想自己维护 fork 才相关。

## 同步流程

```bash
# 1) 一次性配置 3 个上游远端 (后续都用这套别名)
git remote add upstream-api   https://github.com/dujiao-next/dujiao-next.git
git remote add upstream-admin https://github.com/dujiao-next/admin.git
git remote add upstream-user  https://github.com/dujiao-next/user.git

# 2) fetch + subtree merge 到对应子目录
git fetch upstream-api   main && git subtree pull --prefix=src/api   upstream-api   main -m "merge: upstream api"
git fetch upstream-admin main && git subtree pull --prefix=src/admin upstream-admin main -m "merge: upstream admin"
git fetch upstream-user  main && git subtree pull --prefix=src/user  upstream-user  main -m "merge: upstream user"
```

> monorepo 当前各模块基线: `src/api@faacb3f`,`src/admin@5d0ce38`,`src/user@6b67ef9` (2026-05-14)

## 冲突处理

`git subtree pull` 跟 `git merge` 行为一致——有冲突时会:

1. 标 conflict 文件
2. 你手动选择保留 ours / theirs / 合并
3. `git add` + `git commit` 完成 merge

最常见冲突在:
- `package.json` 依赖版本
- `i18n/index.ts` 国际化字串(我们改过的品牌名 `Dujiao-Shop` 会被改回)
- `views/admin/SiteConnections.vue`(protocol 字段保留 `dujiao-next` 不能改)

## 谨慎拉取

定期同步上游能拿到 bug fix 和新功能,但也会:
- 把你不想要的功能(比如广告位)拉回来
- 跟你已有的二开冲突
- 触发 i18n 反向品牌字串

建议:**先 fetch 不 merge,看 `git log upstream-api/main ^HEAD --oneline` 决定要不要合**。
