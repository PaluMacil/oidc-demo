package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const cookieName = "session"

type GoogleClaims struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Link          string `json:"link"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
}

func oauthHandler(clientId, clientSecret, redirectURL string, sessions *SessionStore) http.HandlerFunc {
	oauth2Config := &oauth2.Config{
		RedirectURL:  redirectURL,
		ClientID:     clientId,
		ClientSecret: clientSecret,
		Scopes: []string{"https://www.googleapis.com/auth/userinfo.profile",
			"https://www.googleapis.com/auth/userinfo.email"},
		Endpoint: google.Endpoint,
	}

	return func (w http.ResponseWriter, r *http.Request) {
		pathParts := strings.Split(r.URL.Path, "/")
		if r.Method != "GET" || len(pathParts) != 4 {
			log.Printf("oauth: path invalid length or not a GET: %s %s\n", r.Method, r.URL.Path)
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		step := pathParts[3]
		switch step {
		case "login":
			// generate the login url for google auth; it will look something like this:
			// https://accounts.google.com/o/oauth2/auth
			//  ?client_id=your-client-id.apps.googleusercontent.com
			//  &redirect_uri=http%3A%2F%2Flocalhost%3A4949%2Foauth%2Fgoogle%2Fcallback
			//  &response_type=code
			//  &scope=https%3A%2F%2Fwww.googleapis.com%2Fauth%2Fuserinfo.profile+https%3A%2F%2Fwww.googleapis.com%2Fauth%2Fuserinfo.email
			//  &state=state
			// ideally, save a randomly generated state and then check that a valid saved state was used in the callback
			url := oauth2Config.AuthCodeURL("state")
			http.Redirect(w, r, url, http.StatusTemporaryRedirect)
			return
		case "callback":
			// you should have a code value upon return from the auth provider which you can use to get an access token
			// with authorization to use the scopes allowed by the user and auth provider
			code := r.FormValue("code")
			token, err := oauth2Config.Exchange(r.Context(), code)
			if err != nil {
				log.Printf("callback: code exchange failed with '%s'\n", err)
				http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
				return
			}

			response, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				log.Printf("callback: getting access_token: %s\n", err)
				return
			}
			defer response.Body.Close()

			// parse the returned claims into a struct for access in the application
			claims := &GoogleClaims{}
			err = json.NewDecoder(response.Body).Decode(claims)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				log.Printf("callback: decoding oauth claims: %s\n", err)
				return
			}
			// if the email is not verified, treat this auth as a failure
			if !claims.VerifiedEmail {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				log.Printf("callback: email was not verified in auth provider\n")
				return
			}
			log.Printf("callback: claims loaded from auth provider\n")

			// set a session based upon claims; in a real application, you might look up or create a user in a
			// database and build the session from that. You aren't guaranteed a stable set of claims from the
			// third party auth provider, so trading this set of claims for your own auth token and/or structures
			// is usually ideal
			session := sessions.Set(*claims)
			cookie := &http.Cookie{
				Name:       cookieName,
				Value:      session.ID,
				Path:       "/",
				Expires:    time.Now().AddDate(0,0,7),
				HttpOnly:   true,
				SameSite:   http.SameSiteLaxMode,
			}
			log.Printf("callback: created session %s for %s (%s)\n", session.ID, session.Name, session.Email)
			http.SetCookie(w, cookie)
			// if you use the state field, you can store a redirect value there, but beware of
			// how you generate the state that you don't introduce a way for attackers to cause
			// blind redirects
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		case "logout":
			cookie, err := r.Cookie(cookieName)
			if err != nil {
				log.Printf("logout: getting session cookie: %s\n", err)
				http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
				return
			}
			sessionId := cookie.Value
			// set session cookie to expire in past (deletes cookie)
			expireInPast := time.Now().Add(-7 * 24 * time.Hour)
			cookie.Expires = expireInPast
			deletionCookie := &http.Cookie{
				Name:       cookieName,
				Value:      "",
				Path:       "/",
				Expires:    expireInPast,
				MaxAge:     -1,
				HttpOnly:   true,
				SameSite:   http.SameSiteLaxMode,
			}
			http.SetCookie(w, deletionCookie)
			// delete session from session store
			log.Printf("logout: set cookie to expire, now deleting session %s\n", sessionId)
			sessions.Delete(sessionId)
			http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
			return
		default:
			log.Printf("invalid oauth step: %s\n", step)
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		}
	}
}
