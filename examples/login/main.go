// Command login prints the URL a user has to open to log in with a Microsoft
// account and finishes the login once the redirect URI is called.
//
// Usage:
//
//	go run ./examples/login <client id> <redirect uri>
//
// The redirect URI has to be registered at the Azure app.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/syaniclauncher/minecraft-lib-golang/mcl"
)

func main() {
	if len(os.Args) < 3 {
		log.Fatal("usage: login <client id> <redirect uri>")
	}
	clientID, redirectURI := os.Args[1], os.Args[2]

	data := mcl.SecureLoginData(clientID, redirectURI, "")
	fmt.Println("open this URL in a browser:")
	fmt.Println(data.URL)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		code, err := mcl.ParseAuthCodeURL(r.URL.String(), data.State)
		if err != nil {
			http.Error(w, "invalid login: "+err.Error(), http.StatusBadRequest)
			return
		}
		fmt.Fprintln(w, "the login was successful")

		account, err := mcl.CompleteLogin(context.Background(), clientID, "", redirectURI, code, data.CodeVerifier)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("logged in as %s (%s)\n", account.Name, account.ID)
		fmt.Println("the refresh token is:", account.RefreshToken)
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}
