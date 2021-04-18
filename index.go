package main

import (
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strings"
)

func indexHandler(sessions *SessionStore) http.HandlerFunc {
	return func (w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// redirect to root path if index.html is requested
		if path == "/index.html" {
			http.Redirect(w, r, "/", 301)
			return
		}
		// serve any valid asset files
		fileRequest := filepath.Base(path)
		fileExt := filepath.Ext(fileRequest)
		if servableFileExt(fileExt) {
			http.ServeFile(w, r, filepath.Join("www", fileRequest))
			return
		}
		// at this point, any other routes besides the root path are invalid
		if path != "/" {
			http.NotFound(w, r)
			return
		}

		// generate page templates
		files := []string{
			"www/home.gohtml",
			"www/layout.gohtml",
		}
		ts, err := template.ParseFiles(files...)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		// get session information from cookie
		var sessionId string
		cookie, err := r.Cookie(cookieName)
		if err != nil {
			log.Printf("serving index: getting session cookie: %s\n", err)
		} else {
			sessionId = cookie.Value
		}
		log.Printf("serving index: getting session for %s\n", sessionId)
		session := sessions.Get(sessionId)
		log.Printf("serving index: this session authenticated? %v\n", session.Authenticated)

		// execute the template with the appropriate session
		err = ts.Execute(w, session)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}

func servableFileExt(ext string) bool {
	lowerExt := strings.ToLower(ext)
	validExtentions := []string{".js", ".css", ".png", ".map.js"}
	for _, validExt := range validExtentions {
		if lowerExt == validExt {
			return true
		}
	}
	return false
}