package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

func main() {

	http.HandleFunc("/", HelloPubSub)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}

// HelloPubSub receives and processes a Pub/Sub push message.
func HelloPubSub(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("ioutil.ReadAll: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	fmt.Fprint(w, "headers\n")
	for k, v := range r.Header {
		fmt.Fprintf(w, "%s: %s\n", k, v)
	}
	fmt.Fprint(w, "body\n")
	fmt.Fprint(w, string(body))
	w.WriteHeader(http.StatusOK)
	if r.Header.Get("content-type") != "" {
		w.Header().Add("content-type", r.Header.Get("content-type"))
	}
}
