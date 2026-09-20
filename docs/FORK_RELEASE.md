# 定制版（fork）发布协议

本仓库 `Mxucc/sub2api` 是基于上游 [Wei-Shaw/sub2api](https://github.com/Wei-Shaw/sub2api)
的**定制版**。本文定义两边如何共存：**上游发布的是正式版，我们发布的是基于正式版的定制构建。**

```
上游正式版    Wei-Shaw/sub2api        版本 0.2.7            标签 v0.2.7
                                    镜像 ghcr.io/wei-shaw/sub2api:0.2.7

本仓库定制版  Mxucc/sub2api           版本 0.2.7-1a9d49e1   标签 v0.2.7-1a9d49e1
                                    镜像 ghcr.io/mxucc/sub2api:0.2.7-1a9d49e1
```

---

## 1. 版本号与标签规范

### 1.1 规则

| 项 | 规则 | 示例 |
| --- | --- | --- |
| **上游基线版本** | 取自 `backend/cmd/server/VERSION`，**跟随上游，我们只读不改** | `0.2.7` |
| **构建标识** | 本次提交的短 SHA（8 位） | `1a9d49e1` |
| **构建版本** | `<上游基线版本>-<短SHA>` | `0.2.7-1a9d49e1` |
| **Git 标签** | `v<构建版本>`（annotated tag） | `v0.2.7-1a9d49e1` |
| **镜像标签** | `<构建版本>`、`<major>.<minor>`、`latest` | `0.2.7-1a9d49e1` / `0.2` / `latest` |

要点：

* 版本号**永远**以上游版本号开头 ⇒ 一眼就能看出「这是上游哪个正式版 + 我们的哪个提交」。
* **每个 commit 一个不可变版本号**：同一个 commit 重复构建得到同一个标签，天然幂等、可复现。
* 上游升级到 `0.2.8` 后（合并上游 main），我们的下一个构建自动变成 `0.2.8-xxxxxxxx`，无需人工干预。
* 上游正式版标签是 `v0.2.7`（无连字符），我们的标签是 `v0.2.7-xxxxxxxx`（有连字符），两者可机械区分；
  `.github/workflows/release.yml`（上游流水线）已显式排除 `v*-*`，不会把我们的代码打成「官方正式版」。

### 1.2 `backend/cmd/server/VERSION` 的规则（重要）

这个文件是**上游基线版本的唯一来源**：

* 它由上游维护（上游打 tag 后由上游 CI 回写，我们合并上游 main 时一并带过来）。
* **我们任何自动化流程都不修改它**（`fork-release.yml` 只在工作区临时写入构建产物版本，不提交）。
* 本地构建（`make build`）在打过 tag 的 checkout 上会优先用 tag 作为版本号（`backend/scripts/resolve-version.sh`）。

### 1.3 镜像标签语义

以 `ghcr.io/mxucc/sub2api` 为例：

| 镜像标签 | 含义 | 建议用途 |
| --- | --- | --- |
| `0.2.7-1a9d49e1` | 某个 commit 的构建，**不可变** | 生产环境固定版本 ✅ |
| `0.2.7-1a9d49e1-amd64` / `-arm64` | 单架构构建（GoReleaser 直接产出） | 特殊平台 |
| `0.2` | 该 minor 线的最新构建（滚动） | 只升小版本的场景 |
| `latest` | 最新构建（滚动，每个 commit 都会移动） | 开发/测试 |
| `0` | 该 major 线的最新构建（滚动） | 一般不用 |

> 生产环境请**固定用带 SHA 的标签**；`latest` / `0.2` 每次提交都会变。

---

## 2. 发布流程（自动）

**推送即发布**：任何推送到 `main` 的提交都会自动完成 `构建 → 打标签 → 推镜像 → 建 Release`。

```
git add -A && git commit -m "feat: xxx"
git push origin main          # ← 到此为止，剩下交给 GitHub Actions
```

`.github/workflows/fork-release.yml` 依次执行：

1. **prepare**：用 `scripts/fork-version.sh` 计算 `<基线>-<短SHA>` → 创建并推送 annotated tag → 检查该 commit 是否已经发过版。
2. **build-frontend**：构建前端产物。
3. **release**：GoReleaser 编译二进制、构建多架构镜像并推送到本仓库命名空间，同时创建 GitHub Release
   （Release 正文 = tag 消息 = 自动生成的提交列表）。

幂等性：**同一个 commit 只会发一次版**。重复推送/重跑会直接跳过（不会报错、不会覆盖）。

手动补发/重建（例如当时 CI 失败、或想给历史 commit 补镜像）：
GitHub → Actions → *Fork Release* → Run workflow，可指定 `ref`（commit/标签）与 `force`（重建）。

---

## 3. 首次配置清单（在哪配、配什么）

### 3.1 仓库：Settings → Actions → General

| 配置 | 值 | 为什么 |
| --- | --- | --- |
| Workflow permissions | **Read and write permissions** | 需要创建 tag/Release、推送 GHCR 镜像 |
| Allow GitHub Actions to create and approve pull requests | 不需要 | — |

### 3.2 仓库：Settings → Secrets and variables → Actions

| 类型 | 名称 | 必需 | 说明 |
| --- | --- | --- | --- |
| Secret | `DOCKERHUB_USERNAME` | 可选 | 想同时发 Docker Hub 时填 |
| Secret | `DOCKERHUB_TOKEN` | 可选 | Docker Hub Access Token |
| Variable | `UPDATE_REPO` | 建议 | 例如 `Mxucc/sub2api`，让应用内「检查更新/一键更新/回滚」指向我们自己的 Release |
| Variable | `DOCKER_IMAGE` | 建议 | 例如 `ghcr.io/mxucc/sub2api`，让界面里显示的回滚命令指向我们的镜像 |
| Variable | `SIMPLE_RELEASE` | 可选 | `true` = 只构建 x86_64 GHCR 镜像（更快，跳过其它产物） |

* **GHCR 不需要任何 Secret**：使用内置 `GITHUB_TOKEN`（`packages: write`），镜像天然落在 `ghcr.io/mxucc/sub2api`（仓库所有者命名空间），与上游 `ghcr.io/wei-shaw/sub2api` 完全隔离。
* `UPDATE_REPO` / `DOCKER_IMAGE` 是**构建期注入前端的变量**（`VITE_`），改了要等下一次构建才生效。

### 3.3 GitHub Packages 可见性（首次发布后手动点一次）

Actions → 第一次构建完成后，打开
`https://github.com/users/Mxucc/packages/container/sub2api/settings`
→ **Change visibility → Public**（公开仓库要让别人免登录 `docker pull` 就必须设为 Public）。

私有仓库保持 Private 也可以，使用者需要 `docker login ghcr.io`。

### 3.4 服务端环境变量（运行定制版镜像/二进制时）

| 变量 | 值 | 作用 |
| --- | --- | --- |
| `SUB2API_UPDATE_REPO` | `Mxucc/sub2api` | 让「检查更新 / 一键更新 / 回滚」用自己的 Release，**避免被更新回官方正式版**（默认值是上游仓库） |

> 不设置这个变量时，应用会把上游正式版当作「有更新」，一键更新会把定制版二进制覆盖成官方版 —— 定制部署请务必设置。

### 3.5 本地仓库

```bash
git remote add upstream https://github.com/Wei-Shaw/sub2api.git   # 只在需要同步上游时用
git fetch upstream --tags
```

---

## 4. 跟随上游（日常维护）

```bash
scripts/fork-sync-upstream.sh          # 拉取并合并 upstream/main（会报告基线版本变化）
# 解决冲突 → 跑测试
make test-frontend && make -C backend test
git push origin main                   # 推送后自动按新的上游基线发版（0.2.8-xxxxxxxx）
```

* 只读报告：`scripts/fork-sync-upstream.sh --no-merge`
* 合并指定上游版本：`scripts/fork-sync-upstream.sh --tag v0.2.8`
* 冲突处理时 `backend/cmd/server/VERSION` **必须采用上游版本**（脚本会提示命令）。

### 本地查看将发布的版本

```bash
scripts/fork-version.sh                 # 人类可读
scripts/fork-version.sh --format version   # 0.2.7-1a9d49e1
make fork-version                       # 同上
```

### 本地构建镜像（不上传）

```bash
make fork-image                         # 用仓库根目录 Dockerfile 构建 sub2api:local
```

---

## 5. 部署（使用我们自己的镜像）

```yaml
# docker-compose.yml（片段）
services:
  sub2api:
    # ✅ 固定 commit 版本（推荐）
    image: ghcr.io/mxucc/sub2api:0.2.7-1a9d49e1
    # ⚠️ 上游官方镜像是 ghcr.io/wei-shaw/sub2api:*，不要混用
```

`deploy/` 下的 compose 文件与 `install.sh` 属于上游文件（默认引用 `weishaw/sub2api` 与上游 Release），
定制部署建议用**自己的 compose 覆盖文件**，避免与上游 rebase 冲突：

```bash
docker compose -f docker-compose.yml -f docker-compose.fork.yml up -d
```

```yaml
# docker-compose.fork.yml
services:
  sub2api:
    image: ghcr.io/mxucc/sub2api:0.2.7-1a9d49e1
    environment:
      SUB2API_UPDATE_REPO: Mxucc/sub2api
```

---

## 6. 禁止事项（会导致两个版本互相污染）

1. **不要给本仓库推上游的裸标签**（`v0.2.7`）。我们的标签永远带 `-<短SHA>`；
   `release.yml` 已排除这类标签，即使误推也不会构建，但保持干净更好。
2. **不要手工修改 `backend/cmd/server/VERSION`**（除合并上游时的自动结果）。它是上游基线，被改写会导致版本号错乱。
3. **不要用 `latest` 作为生产版本**（每个 commit 都会移动）。
4. **不要在没有设置 `SUB2API_UPDATE_REPO` 的定制部署上点「一键更新」**（会被覆盖成官方版）。
5. **不要删除/复用已发布的 tag**（`v0.2.7-1a9d49e1` 必须永远指向同一个 commit）。

---

## 7. 已知取舍与常见问题

**Q：同一上游基线的不同构建，为什么应用内不提示「有新版本」？**
A：应用的版本比较只看 `X.Y.Z`（`parseVersion` 在 `-` 处截断），所以 `0.2.7-aaaa` 与 `0.2.7-bbbb` 视为同一版本。
这是刻意的：定制版靠镜像标签升级，不靠应用内自更新。

**Q：每个 commit 都发版，镜像会不会太多？**
A：GHCR 对公开包免费，`latest` / `0.2` 是滚动标签；如需要清理旧标签，可用 GitHub API 批量删除
（`gh api --method DELETE /user/packages/container/sub2api/versions/<id>`）。

**Q：为什么 Release 被标记为 pre-release？**
A：GoReleaser 配置为 `prerelease: auto`，`0.2.7-xxxxxxxx` 在 semver 里属于预发布版本，因此 GitHub 会标为 pre-release
（副作用：GitHub 的 `/releases/latest` 接口不会返回它，应用内自更新也就不会误判）。
如需改为正式 Release，在 `.goreleaser.yaml` 里把 `release.prerelease: auto` 改成 `false`（注意这是上游文件，改动会在 rebase 时冲突）。

**Q：构建太慢怎么办？**
A：把仓库变量 `SIMPLE_RELEASE` 设为 `true`，只构建 x86_64 的 GHCR 镜像（跳过 arm64、归档、校验和）。

**Q：上游发布正式版后我们怎么办？**
A：先 `scripts/fork-sync-upstream.sh` 合并（基线版本随之上移），推送后会自动发 `0.2.8-xxxxxxxx`；
官方正式版仍在上游仓库，两边互不影响。

---

## 8. 相关文件

| 文件 | 作用 |
| --- | --- |
| `.github/workflows/fork-release.yml` | 定制版流水线：每次提交构建并发版（本协议的唯一自动化入口） |
| `.github/workflows/release.yml` | 上游流水线，保留但排除 `v*-*` 标签（不处理我们的构建） |
| `scripts/fork-version.sh` | 版本号计算的唯一实现（本地 + CI 共用） |
| `scripts/fork-sync-upstream.sh` | 合并上游正式版 |
| `.goreleaser.yaml` / `.goreleaser.simple.yaml` | 上游构建配置，镜像命名按仓库所有者自动落到我们的命名空间 |
