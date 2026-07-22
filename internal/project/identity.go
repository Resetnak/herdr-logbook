package project

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var scpRemote = regexp.MustCompile(`^(?:[^@/]+@)?([^:/]+):(.+)$`)

func SanitizeRemote(remote string) (string, error) {
	remote = strings.TrimSpace(remote)
	if remote == "" {
		return "", fmt.Errorf("empty Git remote")
	}
	if !strings.Contains(remote, "://") {
		if match := scpRemote.FindStringSubmatch(remote); match != nil {
			return remoteFingerprint(match[1], match[2])
		}
	}

	parsed, err := url.Parse(remote)
	if err != nil || parsed.Hostname() == "" {
		return "", fmt.Errorf("invalid Git remote")
	}
	host := strings.ToLower(parsed.Hostname())
	if port := parsed.Port(); port != "" {
		host += ":" + port
	}
	return remoteFingerprint(host, parsed.Path)
}

func remoteFingerprint(host, path string) (string, error) {
	host = strings.ToLower(strings.TrimSpace(host))
	path = strings.Trim(strings.TrimSpace(path), "/")
	path = strings.TrimSuffix(path, ".git")
	if host == "" || path == "" {
		return "", fmt.Errorf("git remote must contain a host and repository path")
	}
	return host + "/" + path, nil
}

func StableID(identity string) string {
	sum := sha256.Sum256([]byte(identity))
	return "p_" + hex.EncodeToString(sum[:])
}
