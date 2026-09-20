//go:build unit

package service

import (
	"context"
	"errors"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type updateServiceCacheStub struct {
	data string
}

func (s *updateServiceCacheStub) GetUpdateInfo(context.Context) (string, error) {
	if s.data == "" {
		return "", errors.New("cache miss")
	}
	return s.data, nil
}

func (s *updateServiceCacheStub) SetUpdateInfo(_ context.Context, data string, _ time.Duration) error {
	s.data = data
	return nil
}

type updateServiceGitHubClientStub struct {
	release        *GitHubRelease
	recentReleases []*GitHubRelease
	recentErr      error
}

func (s *updateServiceGitHubClientStub) FetchLatestRelease(context.Context, string) (*GitHubRelease, error) {
	return s.release, nil
}

func (s *updateServiceGitHubClientStub) FetchRecentReleases(context.Context, string, int) ([]*GitHubRelease, error) {
	return s.recentReleases, s.recentErr
}

func (s *updateServiceGitHubClientStub) DownloadFile(context.Context, string, string, int64) error {
	panic("DownloadFile should not be called when no update is available")
}

func (s *updateServiceGitHubClientStub) FetchChecksumFile(context.Context, string) ([]byte, error) {
	panic("FetchChecksumFile should not be called when no update is available")
}

func TestUpdateServicePerformUpdateNoUpdateReturnsSentinel(t *testing.T) {
	svc := NewUpdateService(
		&updateServiceCacheStub{},
		&updateServiceGitHubClientStub{
			release: &GitHubRelease{
				TagName: "v0.1.132",
				Name:    "v0.1.132",
			},
		},
		"0.1.132",
		"release",
	)

	err := svc.PerformUpdate(context.Background())

	require.Error(t, err)
	require.True(t, errors.Is(err, ErrNoUpdateAvailable))
	require.ErrorIs(t, err, ErrNoUpdateAvailable)
}

func newRollbackTestService(current string, releases []*GitHubRelease) *UpdateService {
	return NewUpdateService(
		&updateServiceCacheStub{},
		&updateServiceGitHubClientStub{recentReleases: releases},
		current,
		"release",
	)
}

func TestUpdateServiceListRollbackVersionsFiltersAndCaps(t *testing.T) {
	releases := []*GitHubRelease{
		{TagName: "v0.1.148", PublishedAt: "2026-07-09T00:00:00Z"},                       // newer than current: excluded
		{TagName: "v0.1.147", PublishedAt: "2026-07-08T00:00:00Z"},                       // current: excluded
		{TagName: "v0.1.146-rc1", PublishedAt: "2026-07-07T12:00:00Z", Prerelease: true}, // prerelease: excluded
		{TagName: "v0.1.146", PublishedAt: "2026-07-07T00:00:00Z"},
		{TagName: "v0.1.145", PublishedAt: "2026-07-06T00:00:00Z", Draft: true}, // draft: excluded
		{TagName: "v0.1.144", PublishedAt: "2026-07-05T00:00:00Z"},
		{TagName: "v0.1.144", PublishedAt: "2026-07-05T00:00:00Z"}, // duplicate: excluded
		{TagName: "v0.1.143", PublishedAt: "2026-07-04T00:00:00Z"},
		{TagName: "v0.1.142", PublishedAt: "2026-07-03T00:00:00Z"}, // beyond cap of 3: excluded
	}
	svc := newRollbackTestService("0.1.147", releases)

	versions, err := svc.ListRollbackVersions(context.Background())

	require.NoError(t, err)
	require.Len(t, versions, 3)
	require.Equal(t, "0.1.146", versions[0].Version)
	require.Equal(t, "0.1.144", versions[1].Version)
	require.Equal(t, "0.1.143", versions[2].Version)
}

func TestUpdateServiceListRollbackVersionsSortsUnorderedInput(t *testing.T) {
	releases := []*GitHubRelease{
		{TagName: "v0.1.144"},
		{TagName: "v0.1.146"},
		{TagName: "v0.1.145"},
	}
	svc := newRollbackTestService("0.1.147", releases)

	versions, err := svc.ListRollbackVersions(context.Background())

	require.NoError(t, err)
	require.Len(t, versions, 3)
	require.Equal(t, "0.1.146", versions[0].Version)
	require.Equal(t, "0.1.145", versions[1].Version)
	require.Equal(t, "0.1.144", versions[2].Version)
}

func TestUpdateServiceListRollbackVersionsEmptyWhenNoneOlder(t *testing.T) {
	releases := []*GitHubRelease{
		{TagName: "v0.1.147"},
		{TagName: "v0.1.148"},
	}
	svc := newRollbackTestService("0.1.147", releases)

	versions, err := svc.ListRollbackVersions(context.Background())

	require.NoError(t, err)
	require.Empty(t, versions)
}

func TestUpdateServiceListRollbackVersionsPropagatesFetchError(t *testing.T) {
	svc := NewUpdateService(
		&updateServiceCacheStub{},
		&updateServiceGitHubClientStub{recentErr: errors.New("github unavailable")},
		"0.1.147",
		"release",
	)

	_, err := svc.ListRollbackVersions(context.Background())

	require.Error(t, err)
	require.Contains(t, err.Error(), "github unavailable")
}

func TestUpdateServiceRollbackToVersionRejectsDisallowedTargets(t *testing.T) {
	releases := []*GitHubRelease{
		{TagName: "v0.1.148"},
		{TagName: "v0.1.147"},
		{TagName: "v0.1.146"},
		{TagName: "v0.1.145"},
		{TagName: "v0.1.144"},
		{TagName: "v0.1.143"},
		{TagName: "v0.1.142"},
	}
	svc := newRollbackTestService("0.1.147", releases)

	for _, target := range []string{
		"",         // empty
		"0.1.147",  // current version
		"v0.1.147", // current version with prefix
		"0.1.148",  // newer than current
		"0.1.142",  // older than the 3 most recent
		"9.9.9",    // nonexistent
	} {
		err := svc.RollbackToVersion(context.Background(), target)
		require.ErrorIs(t, err, ErrRollbackVersionNotAllowed, "target %q should be rejected", target)
	}
}

func TestUpdateServiceRollbackToVersionAcceptsVPrefix(t *testing.T) {
	// No platform asset in the release: the target passes the allowlist check
	// and fails later at asset lookup, proving the version itself was accepted.
	releases := []*GitHubRelease{
		{TagName: "v0.1.147"},
		{TagName: "v0.1.146"},
	}
	svc := newRollbackTestService("0.1.147", releases)

	err := svc.RollbackToVersion(context.Background(), "v0.1.146")

	require.Error(t, err)
	require.NotErrorIs(t, err, ErrRollbackVersionNotAllowed)
	require.Contains(t, err.Error(), "no compatible release found")
}

// ---------------------------------------------------------------------------
// 定制版（fork）发布仓库：SUB2API_UPDATE_REPO 指向自己的仓库时，
// 发布是 semver 预发布（0.2.7-g<sha>），必须走 releases 列表而不是 /releases/latest。
// ---------------------------------------------------------------------------

func newCustomRepoTestService(current string, releases []*GitHubRelease, official *GitHubRelease) *UpdateService {
	return NewUpdateService(
		&updateServiceCacheStub{},
		&updateServiceGitHubClientStub{release: official, recentReleases: releases},
		current,
		"release",
	)
}

func TestUpdateServiceCustomRepoDetectsNewerBuildOfSameBase(t *testing.T) {
	t.Setenv(updateRepoEnv, "Mxucc/sub2api")

	svc := newCustomRepoTestService("0.2.7-gaaaa1111", []*GitHubRelease{
		{TagName: "v0.2.7-gbbbb2222", Name: "newer"},
		{TagName: "v0.2.7-gaaaa1111", Name: "current"},
	}, nil)

	info, err := svc.CheckUpdate(context.Background(), true)

	require.NoError(t, err)
	require.True(t, info.HasUpdate, "同一上游基线的新构建应判定为有更新")
	require.Equal(t, "0.2.7-gbbbb2222", info.LatestVersion)
}

func TestUpdateServiceCustomRepoCurrentBuildIsLatest(t *testing.T) {
	t.Setenv(updateRepoEnv, "Mxucc/sub2api")

	svc := newCustomRepoTestService("0.2.7-gbbbb2222", []*GitHubRelease{
		{TagName: "v0.2.7-gbbbb2222"},
		{TagName: "v0.2.7-gaaaa1111"},
	}, nil)

	info, err := svc.CheckUpdate(context.Background(), true)

	require.NoError(t, err)
	require.False(t, info.HasUpdate)
}

func TestUpdateServiceCustomRepoDetectsUpstreamBaseBump(t *testing.T) {
	t.Setenv(updateRepoEnv, "Mxucc/sub2api")

	svc := newCustomRepoTestService("0.2.7-gbbbb2222", []*GitHubRelease{
		{TagName: "v0.2.8-gcccc3333"},
	}, nil)

	info, err := svc.CheckUpdate(context.Background(), true)

	require.NoError(t, err)
	require.True(t, info.HasUpdate)
	require.Equal(t, "0.2.8-gcccc3333", info.LatestVersion)
}

func TestUpdateServiceCustomRepoSkippedDrafts(t *testing.T) {
	t.Setenv(updateRepoEnv, "Mxucc/sub2api")

	svc := newCustomRepoTestService("0.2.7-gaaaa1111", []*GitHubRelease{
		{TagName: "v0.2.7-gzzzz9999", Draft: true},
		{TagName: "v0.2.7-gbbbb2222"},
		{TagName: "v0.2.7-gaaaa1111"},
	}, nil)

	info, err := svc.CheckUpdate(context.Background(), true)

	require.NoError(t, err)
	require.Equal(t, "0.2.7-gbbbb2222", info.LatestVersion)
	require.True(t, info.HasUpdate)
}

func TestUpdateServiceCustomRepoWarnsWhenNoBinaryArchive(t *testing.T) {
	t.Setenv(updateRepoEnv, "Mxucc/sub2api")

	svc := newCustomRepoTestService("0.2.7-gaaaa1111", []*GitHubRelease{
		{TagName: "v0.2.7-gbbbb2222"},
	}, nil)

	info, err := svc.CheckUpdate(context.Background(), true)

	require.NoError(t, err)
	require.NotEmpty(t, info.Warning, "无二进制归档时应提示改用镜像升级")
}

func TestUpdateServiceCustomRepoNoWarningWhenArchiveExists(t *testing.T) {
	t.Setenv(updateRepoEnv, "Mxucc/sub2api")

	svc := newCustomRepoTestService("0.2.7-gaaaa1111", []*GitHubRelease{
		{
			TagName: "v0.2.7-gbbbb2222",
			Assets:  []GitHubAsset{{Name: "sub2api_0.2.7-gbbbb2222_" + svcPlatformForTest() + ".tar.gz"}},
		},
	}, nil)

	info, err := svc.CheckUpdate(context.Background(), true)

	require.NoError(t, err)
	require.Empty(t, info.Warning)
}

func TestUpdateServiceOfficialRepoStillUsesLatestEndpoint(t *testing.T) {
	// 未配置 SUB2API_UPDATE_REPO 时保持官方语义：
	// 只用 /releases/latest（不会把预发布版本当成更新）。
	t.Setenv(updateRepoEnv, "")

	svc := newCustomRepoTestService("0.2.7", []*GitHubRelease{
		{TagName: "v0.2.8-rc.1", Prerelease: true},
	}, &GitHubRelease{TagName: "v0.2.7", Name: "official"})

	info, err := svc.CheckUpdate(context.Background(), true)

	require.NoError(t, err)
	require.False(t, info.HasUpdate)
	require.Equal(t, "0.2.7", info.LatestVersion)
	require.Equal(t, "official", info.ReleaseInfo.Name)
}

func svcPlatformForTest() string {
	return runtime.GOOS + "_" + runtime.GOARCH
}
