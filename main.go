package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func main() {
	port := os.Getenv("APP_PORT")
	clientId := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")
	redirectURL := os.Getenv("REDIRECT_URL")

	mux := http.NewServeMux()
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
		//time from when the connection is accepted to when the request body is fully read
		ReadTimeout: 5 * time.Second,
		//time from the end of the request header read to the end of the response write
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// create in-memory, thread-safe session store
	sessions := &SessionStore{}

	// map path prefixes to handlers
	mux.HandleFunc("/", indexHandler(sessions))
	mux.HandleFunc("/oauth/", oauthHandler(clientId, clientSecret, redirectURL, sessions))

	// serve until an interrupt signal is received from the OS
	stop := make(chan os.Signal)
	signal.Notify(stop, os.Interrupt)
	go func() {
		log.Printf("Now serving on http://localhost:%s\n", port)
		log.Println(srv.ListenAndServe())
	}()
	<-stop

	log.Printf("shutting down ...\n")

	// notify via context of shutdown with up to a 10 second wait for cleanup
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal(err)
	}
}