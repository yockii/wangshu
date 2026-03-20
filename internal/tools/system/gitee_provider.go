package system

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/creativeprojects/go-selfupdate"
)

type GiteeRelease struct {
	ID              int64         `json:"id"`
	TagName         string        `json:"tag_name"`
	Name            string        `json:"name"`
	Body            string        `json:"body"`
	CreatedAt       string        `json:"created_at"`
	Assets          []*GiteeAsset `json:"assets"`
	Author          *GiteeUser    `json:"author"`
	Prerelease      bool          `json:"prerelease"`
	Draft           bool          `json:"draft"`
	TargetCommitish string        `json:"target_commitish"`
}

type GiteeAsset struct {
	ID                 int64  `json:"id"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Name               string `json:"name"`
}

type GiteeUser struct {
	ID                int64  `json:"id"`
	Login             string `json:"login"`
	Name              string `json:"name"`
	AvatarURL         string `json:"avatar_url"`
	URL               string `json:"url"`
	HTMLURL           string `json:"html_url"`
	Remark            string `json:"remark"`
	FFollowersURL     string `json:"followers_url"`
	FollowingURL      string `json:"following_url"`
	GistsURL          string `json:"gists_url"`
	StarredURL        string `json:"starred_url"`
	SubscriptionsURL  string `json:"subscriptions_url"`
	OrganizationsURL  string `json:"organizations_url"`
	ReposURL          string `json:"repos_url"`
	EventsURL         string `json:"events_url"`
	ReceivedEventsURL string `json:"received_events_url"`
	Type              string `json:"type"`
}

func (r *GiteeRelease) GetID() int64 {
	return r.ID
}

func (r *GiteeRelease) GetTagName() string {
	return r.TagName
}

func (r *GiteeRelease) GetDraft() bool {
	return r.Draft
}

func (r *GiteeRelease) GetPrerelease() bool {
	return r.Prerelease
}

func (r *GiteeRelease) GetPublishedAt() time.Time {
	t, _ := time.Parse(time.RFC3339, r.CreatedAt)
	return t
}

func (r *GiteeRelease) GetReleaseNotes() string {
	return r.Body
}

func (r *GiteeRelease) GetAssets() []selfupdate.SourceAsset {
	assets := make([]selfupdate.SourceAsset, len(r.Assets))
	for i, asset := range r.Assets {
		assets[i] = asset
	}
	return assets
}

func (r *GiteeRelease) GetName() string {
	return r.Name
}

func (r *GiteeRelease) GetURL() string {
	return fmt.Sprintf("%s/%s/releases/%s", r.Author.Login, r.Name, r.TagName)
}

func (r *GiteeAsset) GetID() int64 {
	return r.ID
}

func (r *GiteeAsset) GetName() string {
	return r.Name
}

func (r *GiteeAsset) GetSize() int {
	return 0
}

func (r *GiteeAsset) GetBrowserDownloadURL() string {
	return r.BrowserDownloadURL
}

type GiteeSource struct {
	baseUrl string
}

func NewGiteeSource() (*GiteeSource, error) {
	return &GiteeSource{
		baseUrl: "https://gitee.com/",
	}, nil
}

func (s *GiteeSource) ListReleases(ctx context.Context, repository selfupdate.Repository) ([]selfupdate.SourceRelease, error) {
	owner, repo, err := repository.GetSlug()
	if err != nil {
		return nil, err
	}

	uri, err := url.JoinPath(s.baseUrl, "/api/v5/repos", owner, repo, "releases")
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: time.Second * 5,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var giteeReleases []*GiteeRelease
	if err := json.NewDecoder(resp.Body).Decode(&giteeReleases); err != nil {
		return nil, err
	}

	// id赋值，规则：release_id + 顺序
	for _, rel := range giteeReleases {
		for j, asset := range rel.Assets {
			asset.ID = rel.GetID()*10000000 + int64(j)
		}
	}

	releases := make([]selfupdate.SourceRelease, len(giteeReleases))
	for i, rel := range giteeReleases {
		releases[i] = rel
	}
	return releases, nil
}

func (s *GiteeSource) DownloadReleaseAsset(ctx context.Context, rel *selfupdate.Release, assetID int64) (io.ReadCloser, error) {
	if rel == nil {
		return nil, selfupdate.ErrInvalidRelease
	}
	var downloadUrl string
	if rel.AssetID == assetID {
		downloadUrl = rel.AssetURL
	} else if rel.ValidationAssetID == assetID {
		downloadUrl = rel.ValidationAssetURL
	}
	if downloadUrl == "" {
		return nil, fmt.Errorf("asset ID %d: %w", assetID, selfupdate.ErrAssetNotFound)
	}

	client := &http.Client{
		Timeout: time.Second * 60,
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadUrl, http.NoBody)
	if err != nil {
		return nil, err
	}
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		response.Body.Close()
		return nil, fmt.Errorf("HTTP request failed with status code %d", response.StatusCode)
	}

	return response.Body, nil
}

// Verify interface
var _ selfupdate.Source = &GiteeSource{}
