package versioncheck

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultRepo      = "kai-zer-ru/buhgalter"
	defaultCacheTTL  = 24 * time.Hour
	githubUserAgent  = "buhgalter-version-check"
	githubAPIBaseURL = "https://api.github.com/repos/"
)

type Checker struct {
	currentVersion string
	repo           string
	cacheTTL       time.Duration
	httpClient     *http.Client

	mu    sync.Mutex
	cache cachedRelease
}

type cachedRelease struct {
	fetchedAt time.Time
	tagName   string
	htmlURL   string
}

type githubRelease struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

type Result struct {
	CurrentVersion  string `json:"current_version"`
	LatestVersion   string `json:"latest_version,omitempty"`
	UpdateAvailable bool   `json:"update_available"`
	ReleaseURL      string `json:"release_url,omitempty"`
}

func NewChecker(currentVersion string) *Checker {
	return &Checker{
		currentVersion: strings.TrimSpace(currentVersion),
		repo:           defaultRepo,
		cacheTTL:       defaultCacheTTL,
		httpClient:     &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Checker) Check(ctx context.Context) Result {
	result := Result{CurrentVersion: c.currentVersion}
	if c.currentVersion == "" {
		return result
	}

	release, ok := c.cachedRelease()
	if !ok {
		var err error
		release, err = c.fetchLatestRelease(ctx)
		if err != nil {
			return result
		}
		c.storeRelease(release)
	}

	latest := normalizeVersion(release.tagName)
	if latest == "" {
		return result
	}

	result.LatestVersion = latest
	result.ReleaseURL = release.htmlURL
	result.UpdateAvailable = compareVersions(c.currentVersion, latest) < 0
	return result
}

func (c *Checker) cachedRelease() (cachedRelease, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cache.tagName == "" || time.Since(c.cache.fetchedAt) >= c.cacheTTL {
		return cachedRelease{}, false
	}
	return c.cache, true
}

func (c *Checker) storeRelease(release cachedRelease) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = release
}

func (c *Checker) fetchLatestRelease(ctx context.Context) (cachedRelease, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		githubAPIBaseURL+c.repo+"/releases/latest",
		nil,
	)
	if err != nil {
		return cachedRelease{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", githubUserAgent)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return cachedRelease{}, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 512))
		return cachedRelease{}, fmt.Errorf("github releases: status %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload githubRelease
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return cachedRelease{}, err
	}
	tagName := strings.TrimSpace(payload.TagName)
	if tagName == "" {
		return cachedRelease{}, fmt.Errorf("github releases: empty tag_name")
	}

	return cachedRelease{
		fetchedAt: time.Now(),
		tagName:   tagName,
		htmlURL:   strings.TrimSpace(payload.HTMLURL),
	}, nil
}

func normalizeVersion(v string) string {
	return strings.TrimPrefix(strings.TrimSpace(v), "v")
}

func compareVersions(current, latest string) int {
	currentParts := versionParts(normalizeVersion(current))
	latestParts := versionParts(normalizeVersion(latest))
	maxLen := len(currentParts)
	if len(latestParts) > maxLen {
		maxLen = len(latestParts)
	}
	for i := 0; i < maxLen; i++ {
		var currentPart, latestPart int
		if i < len(currentParts) {
			currentPart = currentParts[i]
		}
		if i < len(latestParts) {
			latestPart = latestParts[i]
		}
		switch {
		case currentPart < latestPart:
			return -1
		case currentPart > latestPart:
			return 1
		}
	}
	return 0
}

func versionParts(v string) []int {
	rawParts := strings.Split(v, ".")
	parts := make([]int, 0, len(rawParts))
	for _, part := range rawParts {
		part = strings.SplitN(part, "-", 2)[0]
		part = strings.SplitN(part, "+", 2)[0]
		n, err := strconv.Atoi(part)
		if err != nil {
			n = 0
		}
		parts = append(parts, n)
	}
	return parts
}
