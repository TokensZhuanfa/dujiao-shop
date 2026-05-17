# 上游同步

monorepo 之前是三个独立子仓 (`dujiao-next/{dujiao-next,admin,user}`),现在合并到本仓。

## 同步上游变更

```bash
# 一次性加 3 个上游远端
git remote add upstream-api   https://github.com/dujiao-next/dujiao-next.git
git remote add upstream-admin https://github.com/dujiao-next/admin.git
git remote add upstream-user  https://github.com/dujiao-next/user.git

# fetch + subtree merge
git fetch upstream-api   main && git subtree pull --prefix=src/api   upstream-api   main -m "merge: upstream api"
git fetch upstream-admin main && git subtree pull --prefix=src/admin upstream-admin main -m "merge: upstream admin"
git fetch upstream-user  main && git subtree pull --prefix=src/user  upstream-user  main -m "merge: upstream user"
```

> 当前 monorepo 基于 dujiao-next/dujiao-next@faacb3f、admin@5d0ce38、user@6b67ef9 (2026-05-14)。

## 冲突处理

`git subtree pull` 跟 `git merge` 行为一致——有冲突时会:

1. 标 conflict 文件
2. 你手动选择保留 ours / theirs / 合并
3. `git add` + `git commit` 完成 merge

最常见冲突在:
- `package.json` 依赖版本
- `i18n/index.ts` 国际化字串(我们改了 brand)
- `views/admin/SiteConnections.vue`(protocol 字段保留 `dujiao-next`)

## 谨慎选择

定期同步上游能拿到 bug fix 和新功能,但也会:
- 把上游"广告位"等可能不想要的功能拉回来
- 跟你已有的二开冲突
- 触发前端 i18n 反向品牌字串(`Dujiao-Shop` 被改回 `Dujiao-Next`)

建议:**先 fetch 不 merge,看 diff 决定**。
