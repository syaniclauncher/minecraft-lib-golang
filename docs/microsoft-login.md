# Microsoft login

The library can log in with a Microsoft account. The process follows the
scheme that is documented at [wiki.vg](https://wiki.vg/Microsoft_Authentication_Scheme).

## Before you start

You need an Azure app:

1. Create an app registration in the
   [Azure portal](https://portal.azure.com/).
2. Add a **Mobile and desktop applications** platform and enter the redirect
   URI your launcher listens on, for example `http://localhost:8080`.
3. Copy the **Application (client) ID**. It is the `clientID` of the
   functions below.

The app needs the permission to use the Minecraft API. When it does not have
it, the login fails with `mcl.ErrAzureAppNotPermitted`.

## The login

The login has three steps: the user opens a URL, the user logs in and your
launcher receives the answer on the redirect URI.

### 1. Build the login URL

```go
data := mcl.SecureLoginData(clientID, "http://localhost:8080", "")
fmt.Println("open this URL:", data.URL)
```

`SecureLoginData` uses PKCE and a state. Both protect the login against
attacks: PKCE makes sure that only the launcher that started the login can
finish it, and the state makes sure that the answer belongs to this login.

Keep `data.State` and `data.CodeVerifier`; both are needed in the next step.

`mcl.LoginURL(clientID, redirectURI)` builds a URL without PKCE. It is
simpler, but it is not recommended.

### 2. Receive the answer

Start a server on the redirect URI and wait for the request:

```go
http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	code, err := mcl.ParseAuthCodeURL(r.URL.String(), data.State)
	if err != nil {
		http.Error(w, "invalid login", http.StatusBadRequest)
		return
	}
	fmt.Fprintln(w, "the login was successful")
	complete(code)
})
http.ListenAndServe(":8080", nil)
```

`ParseAuthCodeURL` checks the state and returns the authorization code. It
fails when the state does not match, so an attacker cannot inject a code of
their own.

### 3. Finish the login

```go
func complete(code string) {
	account, err := mcl.CompleteLogin(ctx, clientID, "", "http://localhost:8080", code, data.CodeVerifier)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(account.Name, account.ID)
	save(account.RefreshToken)
}
```

The client secret is deprecated and only needed for old Azure apps, so it can
be empty.

The returned `mcl.Account` holds the name, the UUID, the access token and the
refresh token of the player. The access token is what the game needs:

```go
options := mcl.Options{
	Username: account.Name,
	UUID:     account.ID,
	Token:    account.AccessToken,
}
```

## Refreshing the session

The access token expires after a few hours. The refresh token is good for
much longer and renews the session:

```go
account, err := mcl.CompleteRefresh(ctx, clientID, "", savedRefreshToken)
if err != nil {
	log.Fatal(err)
}
```

A refresh token that is rejected comes back as `mcl.ErrInvalidRefreshToken`:

```go
if errors.Is(err, mcl.ErrInvalidRefreshToken) {
	// The user has to log in again.
}
```

## The errors

| Error | Meaning |
| --- | --- |
| `mcl.ErrAzureAppNotPermitted` | The Azure app is not allowed to use the Minecraft API. |
| `mcl.ErrAccountNotOwnMinecraft` | The account does not own Minecraft: Java Edition. |
| `mcl.ErrInvalidRefreshToken` | The refresh token was rejected. |
