package mcl_test

import (
	"crypto/sha256"
	"encoding/base64"
	"net/url"
	"strings"
	"testing"

	"github.com/syaniclauncher/minecraft-lib-golang/mcl"
)

func TestLoginURL(t *testing.T) {
	rawURL := mcl.LoginURL("client-id", "http://localhost:8080")

	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse the login url: %v", err)
	}
	if !strings.HasPrefix(rawURL, mcl.MSAAuthURL) {
		t.Errorf("the login url is %q", rawURL)
	}

	query := parsed.Query()
	if query.Get("client_id") != "client-id" {
		t.Errorf("the client id is %q", query.Get("client_id"))
	}
	if query.Get("redirect_uri") != "http://localhost:8080" {
		t.Errorf("the redirect uri is %q", query.Get("redirect_uri"))
	}
	if query.Get("response_type") != "code" {
		t.Errorf("the response type is %q", query.Get("response_type"))
	}
	if query.Get("scope") != mcl.MSAScope {
		t.Errorf("the scope is %q", query.Get("scope"))
	}
}

func TestSecureLoginData(t *testing.T) {
	data := mcl.SecureLoginData("client-id", "http://localhost:8080", "")
	if data.State == "" {
		t.Error("the state should be generated")
	}
	if len(data.CodeVerifier) != 128 {
		t.Errorf("the code verifier has %d characters, want 128", len(data.CodeVerifier))
	}

	parsed, err := url.Parse(data.URL)
	if err != nil {
		t.Fatalf("parse the login url: %v", err)
	}
	query := parsed.Query()
	if query.Get("state") != data.State {
		t.Errorf("the state in the url is %q, want %q", query.Get("state"), data.State)
	}
	if query.Get("code_challenge_method") != "S256" {
		t.Errorf("the code challenge method is %q", query.Get("code_challenge_method"))
	}

	sum := sha256.Sum256([]byte(data.CodeVerifier))
	want := base64.RawURLEncoding.EncodeToString(sum[:])
	if query.Get("code_challenge") != want {
		t.Errorf("the code challenge does not match the verifier")
	}
}

func TestSecureLoginDataKeepsTheState(t *testing.T) {
	data := mcl.SecureLoginData("client-id", "http://localhost:8080", "my-state")
	if data.State != "my-state" {
		t.Errorf("the state is %q, want my-state", data.State)
	}
}

func TestAuthCodeFromURL(t *testing.T) {
	rawURL := "http://localhost:8080/?code=the-code&state=the-state"

	if !mcl.URLContainsAuthCode(rawURL) {
		t.Error("the url should contain an authorization code")
	}
	code, ok := mcl.AuthCodeFromURL(rawURL)
	if !ok || code != "the-code" {
		t.Errorf("the code is %q", code)
	}

	if mcl.URLContainsAuthCode("http://localhost:8080/?state=the-state") {
		t.Error("the url should not contain an authorization code")
	}
	if _, ok := mcl.AuthCodeFromURL("http://localhost:8080/?state=the-state"); ok {
		t.Error("the url should not contain an authorization code")
	}
}

func TestParseAuthCodeURL(t *testing.T) {
	rawURL := "http://localhost:8080/?code=the-code&state=the-state"

	code, err := mcl.ParseAuthCodeURL(rawURL, "the-state")
	if err != nil {
		t.Fatalf("ParseAuthCodeURL: %v", err)
	}
	if code != "the-code" {
		t.Errorf("the code is %q", code)
	}

	if _, err := mcl.ParseAuthCodeURL(rawURL, "another-state"); err == nil {
		t.Error("ParseAuthCodeURL should refuse a wrong state")
	}
	if _, err := mcl.ParseAuthCodeURL("http://localhost:8080/?state=the-state", "the-state"); err == nil {
		t.Error("ParseAuthCodeURL should fail without a code")
	}
}

func TestGenerateState(t *testing.T) {
	first := mcl.GenerateState()
	second := mcl.GenerateState()
	if first == "" || second == "" {
		t.Fatal("the state should not be empty")
	}
	if first == second {
		t.Error("two states should not be equal")
	}
}
