package mcl

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// The endpoints of the Microsoft login.
const (
	// MSAAuthURL is the endpoint the user logs in at.
	MSAAuthURL = "https://login.microsoftonline.com/consumers/oauth2/v2.0/authorize"
	// MSATokenURL is the endpoint the authorization code is exchanged at.
	MSATokenURL = "https://login.microsoftonline.com/consumers/oauth2/v2.0/token"
	// MSARefreshURL is the endpoint a refresh token is exchanged at.
	MSARefreshURL = "https://login.live.com/oauth20_token.srf"
	// MSAScope is the scope the login requests.
	MSAScope = "XboxLive.signin offline_access"

	xboxLiveAuthURL     = "https://user.auth.xboxlive.com/user/authenticate"
	xboxLiveXSTSURL     = "https://xsts.auth.xboxlive.com/xsts/authorize"
	minecraftAuthURL    = "https://api.minecraftservices.com/authentication/login_with_xbox"
	minecraftProfileURL = "https://api.minecraftservices.com/minecraft/profile"
)

// Account is a Minecraft account.
type Account struct {
	// ID is the UUID of the player.
	ID string `json:"id"`
	// Name is the in-game name of the player.
	Name string `json:"name"`
	// AccessToken is the access token of the Minecraft session.
	AccessToken string `json:"access_token"`
	// RefreshToken is the token that refreshes the Microsoft session.
	RefreshToken string `json:"refresh_token"`
	// Skins are the skins of the account.
	Skins []Skin `json:"skins"`
	// Capes are the capes of the account.
	Capes []Cape `json:"capes"`
}

// Skin is a skin of an account.
type Skin struct {
	ID      string `json:"id"`
	State   string `json:"state"`
	URL     string `json:"url"`
	Variant string `json:"variant,omitempty"`
}

// Cape is a cape of an account.
type Cape struct {
	ID    string `json:"id"`
	State string `json:"state"`
	URL   string `json:"url"`
	Alias string `json:"alias,omitempty"`
}

// LoginData holds the data of a login that is in progress.
type LoginData struct {
	// URL is the address the user has to open to log in.
	URL string
	// State identifies the login request. It has to be sent back with the
	// authorization code.
	State string
	// CodeVerifier is the PKCE code verifier. It has to be sent back with
	// the authorization code.
	CodeVerifier string
}

// LoginURL returns the address the user has to open to log in.
//
// The redirect URI has to be registered at the Azure app. For a more secure
// alternative use SecureLoginData.
func LoginURL(clientID, redirectURI string) string {
	parameters := url.Values{
		"client_id":     {clientID},
		"response_type": {"code"},
		"redirect_uri":  {redirectURI},
		"response_mode": {"query"},
		"scope":         {MSAScope},
	}
	return MSAAuthURL + "?" + parameters.Encode()
}

// SecureLoginData returns the data of a login that uses PKCE and a state.
//
// It prevents cross-site request forgery and the injection of an
// authorization code. The state is generated when it is empty.
func SecureLoginData(clientID, redirectURI, state string) LoginData {
	if state == "" {
		state = GenerateState()
	}

	codeVerifier, codeChallenge := generatePKCEData()
	parameters := url.Values{
		"client_id":             {clientID},
		"response_type":         {"code"},
		"redirect_uri":          {redirectURI},
		"response_mode":         {"query"},
		"scope":                 {MSAScope},
		"state":                 {state},
		"code_challenge":        {codeChallenge},
		"code_challenge_method": {"S256"},
	}

	return LoginData{
		URL:          MSAAuthURL + "?" + parameters.Encode(),
		State:        state,
		CodeVerifier: codeVerifier,
	}
}

// GenerateState returns a random state for a login.
func GenerateState() string {
	return randomToken(16)
}

// URLContainsAuthCode reports whether the given URL contains an authorization
// code.
func URLContainsAuthCode(rawURL string) bool {
	_, ok := AuthCodeFromURL(rawURL)
	return ok
}

// AuthCodeFromURL returns the authorization code of the given URL.
func AuthCodeFromURL(rawURL string) (string, bool) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", false
	}
	code := parsed.Query().Get("code")
	if code == "" {
		return "", false
	}
	return code, true
}

// ParseAuthCodeURL returns the authorization code of the given URL and checks
// that the state matches.
//
// It returns an error when the state does not match, so an attacker cannot
// inject an authorization code of their own.
func ParseAuthCodeURL(rawURL, state string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	query := parsed.Query()
	if state != "" && query.Get("state") != state {
		return "", fmt.Errorf("the state of the login does not match")
	}
	code := query.Get("code")
	if code == "" {
		return "", fmt.Errorf("the url does not contain an authorization code")
	}
	return code, nil
}

// CompleteLogin finishes the login and returns the account.
//
// The client secret is deprecated and only needed for old Azure apps. The
// code verifier is only needed when the login was started with
// SecureLoginData.
func CompleteLogin(ctx context.Context, clientID, clientSecret, redirectURI, authCode, codeVerifier string) (*Account, error) {
	token, err := getAuthorizationToken(ctx, clientID, clientSecret, redirectURI, authCode, codeVerifier)
	if err != nil {
		return nil, err
	}

	accessToken, err := getMinecraftToken(ctx, token.AccessToken)
	if err != nil {
		return nil, err
	}

	account, err := getProfile(ctx, accessToken)
	if err != nil {
		return nil, err
	}
	account.AccessToken = accessToken
	account.RefreshToken = token.RefreshToken
	return account, nil
}

// CompleteRefresh refreshes a login and returns the account.
//
// The refresh token is the one that was returned by CompleteLogin.
func CompleteRefresh(ctx context.Context, clientID, clientSecret, refreshToken string) (*Account, error) {
	token, err := refreshAuthorizationToken(ctx, clientID, clientSecret, refreshToken)
	if err != nil {
		return nil, err
	}

	accessToken, err := getMinecraftToken(ctx, token.AccessToken)
	if err != nil {
		return nil, err
	}

	account, err := getProfile(ctx, accessToken)
	if err != nil {
		return nil, err
	}
	account.AccessToken = accessToken
	account.RefreshToken = token.RefreshToken
	return account, nil
}

// authorizationToken is the answer of the token endpoint of Microsoft.
type authorizationToken struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
	RefreshToken string `json:"refresh_token"`
	Error        string `json:"error"`
}

// getAuthorizationToken exchanges an authorization code for a token.
func getAuthorizationToken(ctx context.Context, clientID, clientSecret, redirectURI, authCode, codeVerifier string) (authorizationToken, error) {
	form := url.Values{
		"client_id":    {clientID},
		"scope":        {MSAScope},
		"code":         {authCode},
		"redirect_uri": {redirectURI},
		"grant_type":   {"authorization_code"},
	}
	if clientSecret != "" {
		form.Set("client_secret", clientSecret)
	}
	if codeVerifier != "" {
		form.Set("code_verifier", codeVerifier)
	}

	var token authorizationToken
	if err := postForm(ctx, MSATokenURL, form, nil, &token); err != nil {
		return token, err
	}
	if token.AccessToken == "" {
		return token, fmt.Errorf("the authorization failed: %s", token.Error)
	}
	return token, nil
}

// refreshAuthorizationToken exchanges a refresh token for a new token.
func refreshAuthorizationToken(ctx context.Context, clientID, clientSecret, refreshToken string) (authorizationToken, error) {
	form := url.Values{
		"client_id":     {clientID},
		"scope":         {MSAScope},
		"refresh_token": {refreshToken},
		"grant_type":    {"refresh_token"},
	}
	if clientSecret != "" {
		form.Set("client_secret", clientSecret)
	}

	var token authorizationToken
	if err := postForm(ctx, MSARefreshURL, form, nil, &token); err != nil {
		return token, err
	}
	if token.Error != "" {
		return token, fmt.Errorf("%w: %s", ErrInvalidRefreshToken, token.Error)
	}
	if token.AccessToken == "" {
		return token, fmt.Errorf("%w", ErrInvalidRefreshToken)
	}
	return token, nil
}

// getMinecraftToken exchanges a Microsoft token for a Minecraft token.
func getMinecraftToken(ctx context.Context, microsoftToken string) (string, error) {
	var xbl struct {
		Token         string `json:"Token"`
		DisplayClaims struct {
			XUI []struct {
				UHS string `json:"uhs"`
			} `json:"xui"`
		} `json:"DisplayClaims"`
	}
	err := postJSON(ctx, xboxLiveAuthURL, map[string]any{
		"Properties": map[string]any{
			"AuthMethod": "RPS",
			"SiteName":   "user.auth.xboxlive.com",
			"RpsTicket":  "d=" + microsoftToken,
		},
		"RelyingParty": "http://auth.xboxlive.com",
		"TokenType":    "JWT",
	}, nil, &xbl)
	if err != nil {
		return "", err
	}
	if xbl.Token == "" || len(xbl.DisplayClaims.XUI) == 0 {
		return "", fmt.Errorf("the authentication with Xbox Live failed")
	}
	userHash := xbl.DisplayClaims.XUI[0].UHS

	var xsts struct {
		Token string `json:"Token"`
	}
	err = postJSON(ctx, xboxLiveXSTSURL, map[string]any{
		"Properties": map[string]any{
			"SandboxId":  "RETAIL",
			"UserTokens": []string{xbl.Token},
		},
		"RelyingParty": "rp://api.minecraftservices.com/",
		"TokenType":    "JWT",
	}, nil, &xsts)
	if err != nil {
		return "", err
	}
	if xsts.Token == "" {
		return "", fmt.Errorf("the authentication with Xbox failed")
	}

	var minecraft struct {
		AccessToken string `json:"access_token"`
	}
	err = postJSON(ctx, minecraftAuthURL, map[string]any{
		"identityToken": fmt.Sprintf("XBL3.0 x=%s;%s", userHash, xsts.Token),
	}, nil, &minecraft)
	if err != nil {
		return "", err
	}
	if minecraft.AccessToken == "" {
		return "", fmt.Errorf("%w", ErrAzureAppNotPermitted)
	}
	return minecraft.AccessToken, nil
}

// getProfile returns the profile of an account.
func getProfile(ctx context.Context, accessToken string) (*Account, error) {
	header := http.Header{}
	header.Set("Authorization", "Bearer "+accessToken)

	var profile struct {
		Account
		Error        string `json:"error"`
		ErrorMessage string `json:"errorMessage"`
	}
	if err := getJSON(ctx, minecraftProfileURL, header, &profile); err != nil {
		return nil, err
	}
	if profile.Error == "NOT_FOUND" {
		return nil, fmt.Errorf("%w", ErrAccountNotOwnMinecraft)
	}
	if profile.ID == "" {
		return nil, fmt.Errorf("the profile of the account could not be read: %s", profile.ErrorMessage)
	}
	return &profile.Account, nil
}

// generatePKCEData returns a PKCE code verifier and its challenge.
func generatePKCEData() (codeVerifier, codeChallenge string) {
	verifier := randomToken(96)
	sum := sha256.Sum256([]byte(verifier))
	return verifier, base64.RawURLEncoding.EncodeToString(sum[:])
}

// randomToken returns a random URL safe token of about n bytes.
func randomToken(n int) string {
	buffer := make([]byte, n)
	if _, err := rand.Read(buffer); err != nil {
		return strings.Repeat("0", n*2)
	}
	return base64.RawURLEncoding.EncodeToString(buffer)
}

// getJSON downloads a JSON document and decodes it into v.
func getJSON(ctx context.Context, url string, header http.Header, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("user-agent", userAgent)
	for key, values := range header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("GET %s: %s: %s", url, resp.Status, strings.TrimSpace(string(data)))
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("parse the answer of %s: %w", url, err)
	}
	return nil
}

// postForm sends a form encoded document and decodes the response into v.
func postForm(ctx context.Context, url string, form url.Values, header http.Header, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("user-agent", userAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for key, values := range header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("POST %s: %s: %s", url, resp.Status, strings.TrimSpace(string(data)))
	}
	if v == nil {
		return nil
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("parse the answer of %s: %w", url, err)
	}
	return nil
}
