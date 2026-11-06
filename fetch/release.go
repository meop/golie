package fetch

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
)

// LatestTag asks which of a GitHub project's releases is newest, by where its
// release page's "latest" address redirects, which names the tag. releases is
// the page with a trailing slash: "https://github.com/<owner>/<repo>/releases/".
// It is a plain page, not the API, so no token and no rate limit come into it.
func LatestTag(ctx context.Context, releases string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, releases+"latest", nil)
	if err != nil {
		return "", err
	}
	noFollow := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := noFollow.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	location := resp.Header.Get("Location")
	tag := path.Base(location)
	if resp.StatusCode < 300 || resp.StatusCode >= 400 || !strings.Contains(location, "/tag/") || tag == "" {
		return "", fmt.Errorf("%slatest names no release (%s)", releases, resp.Status)
	}
	return tag, nil
}

// Get returns the body at url, which must be small: a checksum manifest, a
// signature. More than a MiB is an error rather than read.
func Get(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", url, resp.Status)
	}
	const limit = 1 << 20
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if len(body) > limit {
		return nil, fmt.Errorf("%s: more than %d bytes", url, limit)
	}
	return body, nil
}

// SumFor returns the sha256 a checksum manifest (sha256sum's format, as
// SHA256SUMS and gh's checksums.txt are) lists for file.
func SumFor(manifest []byte, file string) (string, error) {
	scanner := bufio.NewScanner(bytes.NewReader(manifest))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 2 && strings.TrimPrefix(fields[1], "*") == file {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("the release's checksums do not list %s", file)
}
