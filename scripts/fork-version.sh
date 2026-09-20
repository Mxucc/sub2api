#!/usr/bin/env bash
#
# fork-version.sh —— 本 fork 的版本号计算（唯一实现，本地与 CI 共用）
# =============================================================================
# 协议（详见 docs/FORK_RELEASE.md）：
#
#   上游正式版： Wei-Shaw/sub2api        版本 0.2.7   标签 v0.2.7   镜像 ghcr.io/wei-shaw/sub2api
#   本 fork 构建：<owner>/sub2api        版本 0.2.7-1a2b3c4d  标签 v0.2.7-1a2b3c4d
#                                        镜像 ghcr.io/<owner>/sub2api
#
#   * 版本号 = <上游基线版本>-<短SHA>：基线取 backend/cmd/server/VERSION（跟随上游，只读不改），
#     SHA 取当前提交 => 每个 commit 一个不可变版本号。
#   * 仓库里的 VERSION 文件永远等于上游基线版本；fork 构建只把完整版本号写进构建产物。
#
# 用法：
#   scripts/fork-version.sh                     # 打印版本号/标签/镜像（人类可读）
#   scripts/fork-version.sh --format version    # 0.2.7-1a2b3c4d
#   scripts/fork-version.sh --format tag        # v0.2.7-1a2b3c4d
#   scripts/fork-version.sh --format env        # KEY=VALUE，供 CI 写入 $GITHUB_OUTPUT
#   scripts/fork-version.sh --format json
#   scripts/fork-version.sh --owner Mxucc       # 指定镜像归属（默认取 origin 的 GitHub owner）
#   scripts/fork-version.sh --base 0.2.8        # 覆盖上游基线
#   scripts/fork-version.sh --sha 1a2b3c4d
# =============================================================================

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION_FILE="$REPO_ROOT/backend/cmd/server/VERSION"

FORMAT="all"
BASE=""
SHA=""
OWNER=""
REGISTRY="ghcr.io"
IMAGE_NAME="sub2api"

die() { printf '\033[31merror:\033[0m %s\n' "$1" >&2; exit 1; }

usage() {
  sed -n '2,25p' "${BASH_SOURCE[0]}" | sed 's/^#\{1,2\} \{0,1\}//'
  exit "${1:-0}"
}

while [ $# -gt 0 ]; do
  case "$1" in
    --format) FORMAT="${2:-}"; shift 2 ;;
    --base) BASE="${2:-}"; shift 2 ;;
    --sha) SHA="${2:-}"; shift 2 ;;
    --owner) OWNER="${2:-}"; shift 2 ;;
    --registry) REGISTRY="${2:-}"; shift 2 ;;
    --name) IMAGE_NAME="${2:-}"; shift 2 ;;
    -h|--help) usage 0 ;;
    *) die "未知参数：$1（用 --help 查看用法）" ;;
  esac
done

# ------------------------------------------------------------------ 上游基线
if [ -z "$BASE" ]; then
  [ -f "$VERSION_FILE" ] || die "找不到 $VERSION_FILE"
  BASE="$(tr -d ' \t\r\n' < "$VERSION_FILE")"
fi
# 容错：若文件里被写入过带后缀的版本（历史遗留），只取 X.Y.Z 部分
BASE="${BASE%%-*}"
printf '%s' "$BASE" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+$' \
  || die "上游基线版本格式非法：'$BASE'（应为 X.Y.Z，来自 $VERSION_FILE）"

# ------------------------------------------------------------------ 提交 SHA
if [ -z "$SHA" ]; then
  if command -v git >/dev/null 2>&1 && git -C "$REPO_ROOT" rev-parse --git-dir >/dev/null 2>&1; then
    SHA="$(git -C "$REPO_ROOT" rev-parse --short=8 HEAD)"
  else
    SHA="nogit000"
  fi
fi
printf '%s' "$SHA" | grep -Eq '^[0-9a-f]{4,40}$' \
  || die "提交 SHA 非法：'$SHA'（应为十六进制短 SHA）"

VERSION="$BASE-$SHA"
TAG="v$VERSION"

# ------------------------------------------------------------------ 镜像归属
if [ -z "$OWNER" ] && command -v git >/dev/null 2>&1; then
  REMOTE_URL="$(git -C "$REPO_ROOT" config --get remote.origin.url 2>/dev/null || true)"
  case "$REMOTE_URL" in
    *github.com[:/]*)
      OWNER="$(printf '%s' "$REMOTE_URL" | sed -E 's#.*github\.com[:/]([^/]+)/.*#\1#')"
      ;;
  esac
fi
if [ -n "$OWNER" ]; then
  OWNER_LOWER="$(printf '%s' "$OWNER" | tr '[:upper:]' '[:lower:]')"
  IMAGE="$REGISTRY/$OWNER_LOWER/$IMAGE_NAME"
else
  IMAGE="$REGISTRY/<owner>/$IMAGE_NAME"
fi

MAJOR="$(printf '%s' "$BASE" | cut -d. -f1)"
MINOR="$(printf '%s' "$BASE" | cut -d. -f2)"
LINE_TAG="$MAJOR.$MINOR"

# 与 .goreleaser.yaml 产出的多架构 manifest 标签保持一致：
#   <构建版本>（不可变）、<major>.<minor>（滚动）、latest（滚动）、<major>；
#   另有 <版本>-amd64 / -arm64 单架构标签
PUSH="$(printf '%s\n' "$IMAGE:$VERSION" "$IMAGE:$LINE_TAG" "$IMAGE:latest")"

# ------------------------------------------------------------------ 输出
case "$FORMAT" in
  base) printf '%s\n' "$BASE" ;;
  sha) printf '%s\n' "$SHA" ;;
  version) printf '%s\n' "$VERSION" ;;
  tag) printf '%s\n' "$TAG" ;;
  image) printf '%s\n' "$IMAGE" ;;
  images) printf '%s\n' "$PUSH" ;;
  env)
    printf 'base=%s\n' "$BASE"
    printf 'sha=%s\n' "$SHA"
    printf 'version=%s\n' "$VERSION"
    printf 'tag=%s\n' "$TAG"
    printf 'image=%s\n' "$IMAGE"
    ;;
  json)
    printf '{\n'
    printf '  "base": "%s",\n' "$BASE"
    printf '  "sha": "%s",\n' "$SHA"
    printf '  "version": "%s",\n' "$VERSION"
    printf '  "tag": "%s",\n' "$TAG"
    printf '  "image": "%s",\n' "$IMAGE"
    printf '  "image_tags": [\n'
    printf '    "%s:%s",\n' "$IMAGE" "$VERSION"
    printf '    "%s:%s",\n' "$IMAGE" "$LINE_TAG"
    printf '    "%s:latest"\n' "$IMAGE"
    printf '  ]\n'
    printf '}\n'
    ;;
  all)
    cat <<EOF
上游基线版本 : $BASE        （来自 backend/cmd/server/VERSION，跟随上游）
构建提交     : $SHA
构建版本     : $VERSION
发布标签     : $TAG
镜像         :
  $IMAGE:$VERSION  （不可变，每个 commit 一个）
  $IMAGE:$LINE_TAG        （滚动：$LINE_TAG 线最新构建）
  $IMAGE:latest           （滚动：最新构建）
  （另有 $IMAGE:$VERSION-amd64 / -arm64 单架构标签）
EOF
    ;;
  *) die "不支持的 --format：$FORMAT" ;;
esac
