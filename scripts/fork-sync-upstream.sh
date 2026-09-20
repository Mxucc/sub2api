#!/usr/bin/env bash
#
# fork-sync-upstream.sh —— 把上游正式版合并进本 fork
# =============================================================================
# 上游（正式版）：https://github.com/Wei-Shaw/sub2api
# 本 fork 的所有修改都应该是「叠加在上游之上的提交」，因此日常维护就是不断重复：
#
#   1) scripts/fork-sync-upstream.sh          # 拉取上游并合并（默认合并 main）
#   2) 解决冲突 / 跑测试
#   3) scripts/fork-release.sh --push         # 按新的上游基线发布 vX.Y.Z-1
#
# 用法：
#   scripts/fork-sync-upstream.sh                 # 拉取并合并 upstream/main
#   scripts/fork-sync-upstream.sh --no-merge      # 只看上游有什么新东西
#   scripts/fork-sync-upstream.sh --tag v0.2.8    # 合并指定的上游标签
#   scripts/fork-sync-upstream.sh --upstream Wei-Shaw/sub2api
# =============================================================================

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
UPSTREAM_REPO="Wei-Shaw/sub2api"
UPSTREAM_REMOTE="upstream"
TARGET=""
MERGE=1

usage() {
  sed -n '2,18p' "${BASH_SOURCE[0]}" | sed 's/^#\{1,2\} \{0,1\}//'
  exit "${1:-0}"
}

die() { printf '\033[31merror:\033[0m %s\n' "$1" >&2; exit 1; }
warn() { printf '\033[33mwarning:\033[0m %s\n' "$1" >&2; }
info() { printf '%s\n' "$1"; }

while [ $# -gt 0 ]; do
  case "$1" in
    --upstream) UPSTREAM_REPO="${2:-}"; shift 2 ;;
    --remote) UPSTREAM_REMOTE="${2:-}"; shift 2 ;;
    --tag) TARGET="${2:-}"; shift 2 ;;
    --no-merge) MERGE=0; shift ;;
    -h|--help) usage 0 ;;
    *) die "未知参数：$1（用 --help 查看用法）" ;;
  esac
done

cd "$REPO_ROOT"

[ -z "$(git status --porcelain)" ] || die "工作区不干净，请先提交或 stash（git status）"

# ------------------------------------------------------------ 确保上游 remote
if ! git remote get-url "$UPSTREAM_REMOTE" >/dev/null 2>&1; then
  info "添加 upstream remote：$UPSTREAM_REPO"
  git remote add "$UPSTREAM_REMOTE" "https://github.com/$UPSTREAM_REPO.git"
fi

info "拉取上游（$UPSTREAM_REMOTE → $UPSTREAM_REPO）…"
git fetch --tags --prune "$UPSTREAM_REMOTE"

if [ -n "$TARGET" ]; then
  MERGE_REF="refs/tags/$TARGET"
  git rev-parse -q --verify "$MERGE_REF" >/dev/null || die "找不到上游标签 $TARGET"
else
  MERGE_REF="$UPSTREAM_REMOTE/HEAD"
  git rev-parse -q --verify refs/remotes/"$UPSTREAM_REMOTE"/HEAD >/dev/null 2>&1 \
    || MERGE_REF="$UPSTREAM_REMOTE/main"
fi

LOCAL_BASE="$(tr -d ' \t\r\n' < backend/cmd/server/VERSION)"
UPSTREAM_BASE="$(git show "$MERGE_REF:backend/cmd/server/VERSION" 2>/dev/null | tr -d ' \t\r\n' || echo '?')"
UPSTREAM_TAGS="$(git tag -l 'v[0-9]*' --sort=-v:refname | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' || true)"
LATEST_UPSTREAM_TAG="$(printf '%s\n' "$UPSTREAM_TAGS" | head -1)"
[ -n "$LATEST_UPSTREAM_TAG" ] || LATEST_UPSTREAM_TAG="(本地无上游正式版标签)"

AHEAD="$(git rev-list --count "$MERGE_REF"..HEAD 2>/dev/null || echo '?')"
BEHIND="$(git rev-list --count HEAD.."$MERGE_REF" 2>/dev/null || echo '?')"

info ""
info "============================================================"
info " 上游合并检查"
info "============================================================"
info " 本 fork 的 VERSION : $LOCAL_BASE"
info " 上游的 VERSION     : $UPSTREAM_BASE"
info " 上游最新正式版标签  : $LATEST_UPSTREAM_TAG"
info " 待合并引用         : $MERGE_REF"
info " 领先上游           : $AHEAD 个提交（我们的改动）"
info " 落后上游           : $BEHIND 个提交（待合并）"
info "============================================================"
info ""

if [ "$BEHIND" = "0" ]; then
  info "✅ 已经和上游同步，无需合并。"
  [ "$MERGE" -eq 1 ] && info "可以继续开发，然后 scripts/fork-release.sh --push 发布。"
  exit 0
fi

if [ "$MERGE" -ne 1 ]; then
  info "（--no-merge：仅报告）待合并的提交预览："
  git log --oneline --no-merges HEAD.."$MERGE_REF" | head -25
  exit 0
fi

if [ "$LOCAL_BASE" != "$UPSTREAM_BASE" ] && [ "$UPSTREAM_BASE" != "?" ]; then
  warn "上游基线版本将从 $LOCAL_BASE 变为 $UPSTREAM_BASE：合并后构建号会从 -1 重新开始。"
fi

info "开始合并 $MERGE_REF …"
if ! git merge --no-edit "$MERGE_REF"; then
  info ""
  warn "合并出现冲突。处理建议："
  info "  1) 解决冲突文件（git status 查看）："
  info "       git checkout --theirs -- <文件>   # 采用上游版本"
  info "       git checkout --ours   -- <文件>   # 保留我们的版本"
  info "     然后 git add <文件> && git commit"
  info "  2) backend/cmd/server/VERSION 必须采用上游版本（它是上游基线，不是我们的构建号）："
  info "       git checkout --theirs -- backend/cmd/server/VERSION 2>/dev/null || true"
  info "  3) 放弃本次合并：git merge --abort"
  exit 1
fi

NEW_BASE="$(tr -d ' \t\r\n' < backend/cmd/server/VERSION)"
info ""
info "✅ 合并完成。当前上游基线：$NEW_BASE"
info ""
info "下一步："
info "  1) 跑测试：make test-frontend && make -C backend test"
info "  2) 发布：  scripts/fork-release.sh --push"
