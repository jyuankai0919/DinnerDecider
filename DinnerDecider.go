package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/people/v1"
)

var googleOauthConfig = &oauth2.Config{
	RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
	ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
	ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
	Scopes: []string{
		drive.DriveMetadataReadonlyScope,
	},
	Endpoint: google.Endpoint,
}

func main() {
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	http.HandleFunc("/login", handleGoogleLogin)
	http.HandleFunc("/callback", handleGoogleCallback)
	http.HandleFunc("/maps", handleMaps)

	log.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	url := googleOauthConfig.AuthCodeURL("randomstate")
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func handleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	state := r.FormValue("state")
	if state != "randomstate" {
		http.Error(w, "State mismatch", http.StatusBadRequest)
		return
	}

	code := r.FormValue("code")
	token, err := googleOauthConfig.Exchange(context.Background(), code)
	if err != nil {
		http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	client := googleOauthConfig.Client(context.Background(), token)
	service, err := people.New(client)
	if err != nil {
		http.Error(w, "Failed to create people service: "+err.Error(), http.StatusInternalServerError)
		return
	}

	person, err := service.People.Get("people/me").PersonFields("emailAddresses,names").Do()
	if err != nil {
		http.Error(w, "Failed to get user info: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if len(person.EmailAddresses) > 0 {
		email := person.EmailAddresses[0].Value
		fmt.Fprintf(w, "User email: %s\n", email)
	} else {
		fmt.Fprintf(w, "No email addresses found\n")
	}

	if len(person.Names) > 0 {
		name := person.Names[0].DisplayName
		fmt.Fprintf(w, "User name: %s\n", name)
	} else {
		fmt.Fprintf(w, "No names found\n")
	}

	// 假設這裡有一些餐廳數據，你可以根據需要替換成實際的數據或 API 調用
	recommendedRestaurants := []string{"Restaurant A", "Restaurant B", "Restaurant C"}

	fmt.Fprintf(w, "Recommended restaurants for dinner:\n")
	for _, restaurant := range recommendedRestaurants {
		fmt.Fprintf(w, "- %s\n", restaurant)
	}
}
func handleMaps(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("access_token")
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	client := googleOauthConfig.Client(oauth2.NoContext, &oauth2.Token{
		AccessToken: cookie.Value,
	})

	srv, err := drive.New(client)
	if err != nil {
		http.Error(w, "Failed to create Drive client: "+err.Error(), http.StatusInternalServerError)
		return
	}

	files, err := srv.Files.List().Q("mimeType='application/vnd.google-apps.map'").Fields("files(id, name)").Do()
	if err != nil {
		http.Error(w, "Failed to retrieve files: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(files)
}
