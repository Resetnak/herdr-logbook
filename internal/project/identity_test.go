package project

import (
	"strings"
	"testing"
)

func TestSanitizeRemoteRemovesCredentialsAndNormalizesForms(t *testing.T) {
	tests := map[string]string{
		"https://user:secret@GitHub.COM/Org/Repo.git?token=secret": "github.com/Org/Repo",
		"git@GitHub.COM:Org/Repo.git":                              "github.com/Org/Repo",
		"ssh://git@GitLab.COM/group/repo.git":                      "gitlab.com/group/repo",
	}
	for input, want := range tests {
		got, err := SanitizeRemote(input)
		if err != nil {
			t.Fatalf("SanitizeRemote(%q) error = %v", input, err)
		}
		if got != want {
			t.Fatalf("SanitizeRemote(%q) = %q, want %q", input, got, want)
		}
		if strings.Contains(got, "secret") || strings.Contains(got, "user") || strings.Contains(got, "git@") {
			t.Fatalf("SanitizeRemote(%q) leaked credentials in %q", input, got)
		}
	}
}

func TestStableIDUsesSHA256(t *testing.T) {
	got := StableID("github.com/Org/Repo")
	if len(got) != 66 || !strings.HasPrefix(got, "p_") {
		t.Fatalf("StableID() = %q", got)
	}
	if got != StableID("github.com/Org/Repo") || got == StableID("github.com/Org/Other") {
		t.Fatal("StableID() is not deterministic and distinct")
	}
}

func TestSanitizeRemoteErrorDoesNotEchoCredentials(t *testing.T) {
	input := "https://user:top-secret@/missing-host"
	_, err := SanitizeRemote(input)
	if err == nil {
		t.Fatal("SanitizeRemote() error = nil")
	}
	if strings.Contains(err.Error(), "top-secret") || strings.Contains(err.Error(), "user") {
		t.Fatalf("SanitizeRemote() leaked credentials in error: %v", err)
	}
}
