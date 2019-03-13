package main

import (
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

// loginPageHandler takes you to the login page or the root page if you are already logged in.
func loginPageHandler(w http.ResponseWriter, r *http.Request) {
	_, loggedIn := DB.CheckForLoggedInUser(r)
	if loggedIn {
		http.Redirect(w, r, "/", 302)
		return
	}
	err := renderTemplate(w, "login.html", nil)
	if err != nil {
		panic(err)
	}
}

// loginHandler is the API endpoint that handles checking if a login
// is correct and giving the user a session token.
func loginHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	username := r.FormValue("username")
	password := r.FormValue("password")

	if userID, ok := DB.UsernameToID[username]; ok {
		if checkPassword(DB.Passwords[userID], password) {
			log.Info("login is correct", username)
			sessionToken := genSessionToken()
			http.SetCookie(w, &http.Cookie{
				Name:    "sessionToken",
				Value:   sessionToken,
				Expires: time.Now().Add(3 * time.Hour),
			})
			DB.Sessions[sessionToken] = userID
			http.Redirect(w, r, "/", 301)
			return
		}

	} else {
		log.Error("Username not found:", username)
	}
	_ = username
	_ = password
	loginPageHandler(w, r)
}