package main

import (
	"encoding/json"
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

// PubSubMessage is the payload of a Pub/Sub event.
type PubSubMessage struct {
	Message struct {
		Attribute map[string]string `json:"attributes,omitempty"`
		Data      []byte            `json:"data,omitempty"`
		ID        string            `json:"messageId"`
	} `json:"message"`
	Subscription string `json:"subscription"`
}

// HelloPubSub receives and processes a Pub/Sub push message.
func HelloPubSub(w http.ResponseWriter, r *http.Request) {
	var m PubSubMessage
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("ioutil.ReadAll: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	fmt.Println(string(body))
	if err := json.Unmarshal(body, &m); err != nil {
		log.Printf("json.Unmarshal: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	log.Printf("MessageId = %s\n", m.Message.ID)
	log.Printf("Subscription = %s\n", m.Subscription)
	log.Printf("Attributes\n")
	for k, v := range m.Message.Attribute {
		log.Printf("%s=%s\n", k, v)
	}
	log.Printf("Content\n---\n%s\n---\n", m.Message.Data)

	w.WriteHeader(http.StatusOK)
}
