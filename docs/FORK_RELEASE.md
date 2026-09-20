# 定制版（fork）发布协议

本仓库 `Mxucc/sub2api` 是基于上游 [Wei-Shaw/sub2api](https://github.com/Wei-Shaw/sub2api)
的**定制版**。本文定义两边如何共存：**上游发布的是正式版，我们发布的是基于正式版的定制构建。**

```
上游正式版    Wei-Shaw/sub2api        版本 0.2.7            标签 v0.2.7
                                    镜像 ghcr.io/wei-shaw/sub2api:0.2.7

本仓库定制版  Mxucc/sub2api           版本 0.2.7-g1a9d49e1   标签 v0.2.7-g1a9d49e1
                                    镜像 ghcr.io/mxucc/sub2api:0.2.7-g1a9d49e1
```

---

## 1. 版本号与标签规范

### 1.1 规则

| 项 | 规则 | 示例 |
| --- | --- | --- |
| **上游基线版本** | 取自 `backend/cmd/server/VERSION`，**跟随上游，我们只读不改** | `0.2.7` |
| **构建标识** | 本次提交的短 SHA（8 位，前缀 `g`） | `g1a9d49e1` |
| **构建版本** | `<上游基线版本>-g<短SHA>` | `0.2.7-g1a9d49e1` |
| **Git 标签** | `v<构建版本>`（annotated tag） | `v0.2.7-g1a9d49e1` |
| **镜像标签** | `<构建版本>`、`<major>.<minor>`、`latest` | `0.2.7-g1a9d49e1` / `0.2` / `latest` |

要点：

* 版本号**永远**以上游版本号开头 ⇒ 一眼就能看出「这是上游哪个正式版 + 我们的哪个提交」。
* **每个 commit 一个不可变版本号**：同一个 commit 重复构建得到同一个标签，天然幂等、可复现。
* 上游升级到 `0.2.8` 后（合并上游 main），我们的下一个构建自动变成 `0.2.8-gxxxxxxxx`，无需人工干预。
* 上游正式版标签是 `v0.2.7`（无连字符），我们的标签是 `v0.2.7-gxxxxxxxx`（有连字符），两者可机械区分；
  `.github/workflows/release.yml`（上游流水线）已显式排除 `v*-*`，不会把我们的代码打成「官方正式版」。

### 1.2 `backend/cmd/server/VERSION` 的规则（重要）

这个文件是**上游基线版本的唯一来源**：

* 它由上游维护（上游打 tag 后由上游 CI 回写，我们合并上游 main 时一并带过来）。
* **我们任何自动化流程都不修改它**（`fork-release.yml` 只在工作区临时写入构建产物版本，不提交）。
* 本地构建（`make build`）在打过 tag 的 checkout 上会优先用 tag 作为版本号（`backend/scripts/resolve-version.sh`）。

### 1.3 镜像标签语义

以 `ghcr.io/mxucc/sub2api` 为例：

| 镜像标签 | 含义 | 出现于 | 建议用途 |
| --- | --- | --- | --- |
| `0.2.7-g2681bd17` | 某个 commit 的构建，**不可变** | 两种配置 | 生产环境固定版本 ✅ |
| `0.2.7-g2681bd17-amd64` / `-arm64` | 单架构镜像 | 两种配置（简化配置仅 amd64） | 特殊平台 |
| `latest` | 最新构建（滚动，每个 commit 都会移动） | 两种配置 | 开发/测试 |
| `0.2`（`<major>.<minor>`） | 该 minor 线的最新构建（滚动） | 仅**完整配置** | 只升小版本的场景 |
| `0`（`<major>`） | 该 major 线的最新构建（滚动） | 仅**完整配置** | 一般不用 |

> 生产环境请**固定用带短 SHA 的标签**；`latest` / `0.2` 每次提交都会变。

两种配置由仓库变量 `SIMPLE_RELEASE` 决定（见 3.2）：

* `SIMPLE_RELEASE=true`（当前仓库的设置）：只构建 **linux/amd64** 的 GHCR 镜像（`<版本>`、`<版本>-amd64`、`latest`），
  跳过 arm64、二进制归档与校验和，速度最快，Release 名字带 `(Simple)` 后缀。
* 未设置 / `false`：构建 **amd64 + arm64 多架构**镜像（`<版本>`、`<版本>-amd64`、`<版本>-arm64`、`latest`、`0.2`、`0`），
  同时产出各平台二进制归档与 `checksums.txt`（脚本安装 / 应用内自更新可用）。
* 也可以随时用 workflow_dispatch 手动跑一次并把 `simple_release` 取消勾选（临时覆盖该变量）。

---

## 2. 发布流程（自动）

**推送即发布**：任何推送到 `main` 的提交都会自动完成 `构建 → 打标签 → 推镜像 → 建 Release`。

```
git add -A && git commit -m "feat: xxx"
git push origin main          # ← 到此为止，剩下交给 GitHub Actions
```

`.github/workflows/fork-release.yml` 依次执行：

1. **prepare**：用 `scripts/fork-version.sh` 计算 `<基线>-g<短SHA>` → 创建并推送 annotated tag → 检查该 commit 是否已经发过版。
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
| `UPDATE_GITHUB_TOKEN` | 可选 | 私有仓库或触发 GitHub API 限流时填写 |

> **这两个变量都必须在「运行时」设置**（容器环境变量 / systemd 环境文件 / config.yaml 的 env），
> 不是构建期变量。`SUB2API_UPDATE_REPO` 不设置时，应用会去查上游 `Wei-Shaw/sub2api`，
> 于是永远看不到我们自己的发布。
>
> 用镜像部署时改 `docker-compose.yml` 的环境变量并 `docker compose up -d` 重建容器：
>
> ```yaml
> services:
>   sub2api:
>     environment:
>       SUB2API_UPDATE_REPO: Mxucc/sub2api
> ```

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
git push origin main                   # 推送后自动按新的上游基线发版（0.2.8-gxxxxxxxx）
```

* 只读报告：`scripts/fork-sync-upstream.sh --no-merge`
* 合并指定上游版本：`scripts/fork-sync-upstream.sh --tag v0.2.8`
* 冲突处理时 `backend/cmd/server/VERSION` **必须采用上游版本**（脚本会提示命令）。

### 本地查看将发布的版本

```bash
scripts/fork-version.sh                 # 人类可读
scripts/fork-version.sh --format version   # 0.2.7-g1a9d49e1
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
    image: ghcr.io/mxucc/sub2api:0.2.7-g1a9d49e1
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
    image: ghcr.io/mxucc/sub2api:0.2.7-g1a9d49e1
    environment:
      SUB2API_UPDATE_REPO: Mxucc/sub2api
```

---

## 6. 禁止事项（会导致两个版本互相污染）

1. **不要给本仓库推上游的裸标签**（`v0.2.7`）。我们的标签永远带 `-g<短SHA>`；
   `release.yml` 已排除这类标签，即使误推也不会构建，但保持干净更好。
2. **不要手工修改 `backend/cmd/server/VERSION`**（除合并上游时的自动结果）。它是上游基线，被改写会导致版本号错乱。
3. **不要用 `latest` 作为生产版本**（每个 commit 都会移动）。
4. **不要在没有设置 `SUB2API_UPDATE_REPO` 的定制部署上点「一键更新」**（会被覆盖成官方版）。
5. **不要删除/复用已发布的 tag**（`v0.2.7-g1a9d49e1` 必须永远指向同一个 commit）。

---

## 7. 已知取舍与常见问题

### 7.1 为什么「检查更新」看不到我们自己的发布？

三个原因叠加，缺一不可修：

1. **没设置 `SUB2API_UPDATE_REPO`** —— 应用默认只查上游 `Wei-Shaw/sub2api`（见 §3.4）。
   这样即使点了「检查更新」，看到的也是官方版本，与我们的 `0.2.7-g<sha>` 一比就是「已是最新」。
2. **我们的 Release 曾被标成 pre-release** —— 版本号 `0.2.7-g<sha>` 在 semver 里是预发布，
   GoReleaser 的 `prerelease: auto` 就会把 Release 标成 Pre-release；
   而 GitHub 的 `/releases/latest` 接口**会跳过预发布版本**，于是更新检查直接 404（前端只显示「已是最新」）。
   现在 `fork-release.yml` 在发布后会自动执行
   `gh release edit <tag> --prerelease=false --latest`，把定制版构建提升为正式 Release（Latest）。
3. **同一上游基线的版本比较** —— `0.2.7-gAAAA` 与 `0.2.7-gBBBB` 只比 `X.Y.Z` 的话是「相同版本」。
   后端在检测到 `SUB2API_UPDATE_REPO` 已配置时，会改用 **releases 列表顺序**判断
   （列表按发布倒序；当前构建之前存在条目即为有更新），因此同一基线的新提交也能正确提示更新。

对应修复后的行为：

| 场景 | 结果 |
| --- | --- |
| 上游基线提升（0.2.7 → 0.2.8） | 「有新版本可用」，显示 `0.2.8-gxxxxxxx` |
| 同一基线的新提交（gAAAA → gBBBB） | 「有新版本可用」，显示 `0.2.7-gBBBBBBB` |
| 当前就是最新构建 | 「已是最新版本」 |

**自检命令**（在部署机器上）：

```bash
# 1) 应用内检查更新走的是哪个仓库（应为你自己的仓库）
docker compose exec sub2api printenv SUB2API_UPDATE_REPO

# 2) 该仓库的 /releases/latest 是否可用（404 = 没有正式 Release，全是预发布）
curl -s -o /dev/null -w '%{http_code}\n' https://api.github.com/repos/Mxucc/sub2api/releases/latest

# 3) 直接看接口返回
docker compose exec sub2api \
  curl -s "localhost:8080/api/v1/admin/system/check-updates?force=true" | head -c 400
```

### 7.2 为什么「一键更新」不可用 / 报找不到归档？

「一键更新」需要该 Release 里有**当前平台的二进制归档 + checksums**。
仓库变量 `SIMPLE_RELEASE=true` 时只构建镜像（`archives: []`、`checksum.enable=false`），
因此按钮会被替换为提示文案（`version.noBinaryAsset`），请按界面上给出的镜像标签命令升级：

```bash
# 1) 改 compose 里的 image: ghcr.io/mxucc/sub2api:<新版本>
# 2) docker compose up -d
```

想要「一键更新」可用：把仓库变量 `SIMPLE_RELEASE` 设为 `false`（或删除），
下次发布会同时产出各平台归档与 `checksums.txt`（构建时间更长）。

> 容器部署推荐始终用镜像标签升级（可回滚、可审计），不要依赖应用内自更新覆盖容器内的二进制。


### 7.3 其它常见问题

**Q：构建太慢怎么办？**
A：把仓库变量 `SIMPLE_RELEASE` 设为 `true`，只构建 x86_64 的 GHCR 镜像（跳过 arm64、归档、校验和）。

**Q：上游发布正式版后我们怎么办？**
A：先 `scripts/fork-sync-upstream.sh` 合并（基线版本随之上移），推送后会自动发 `0.2.8-gxxxxxxxx`；
官方正式版仍在上游仓库，两边互不影响。

**Q：已经发布的旧版本要不要补标记成正式 Release？**
A：只有「最新那个」需要（`/releases/latest` 只认它）。补标记：
`gh release edit v0.2.7-g605b448e --prerelease=false --latest`。
历史版本保持 Pre-release 也无影响 —— 应用判定「是否有更新」用的是发布列表顺序，与 pre-release 标记无关。

**Q：每个 commit 都发版，镜像会不会太多？**
A：GHCR 对公开包免费，`latest` 是滚动标签；需要清理旧标签时可用 GitHub API 批量删除
（`gh api --method DELETE /user/packages/container/sub2api/versions/<id>`）。

---

## 8. 相关文件

| 文件 | 作用 |
| --- | --- |
| `.github/workflows/fork-release.yml` | 定制版流水线：每次提交构建、发版、推镜像，并把 Release 提升为 Latest |
| `.github/workflows/release.yml` | 上游流水线，保留但排除 `v*-*` 标签（不处理我们的构建） |
| `backend/internal/service/update_service.go` | 应用内「检查更新/回滚」：支持 `SUB2API_UPDATE_REPO`，并按发布列表判断定制版新构建 |
| `scripts/fork-version.sh` | 版本号计算的唯一实现（本地 + CI 共用） |
| `scripts/fork-sync-upstream.sh` | 合并上游正式版 |
| `.goreleaser.yaml` / `.goreleaser.simple.yaml` | 上游构建配置，镜像命名按仓库所有者自动落到我们的命名空间 |
