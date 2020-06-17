package main

import (
	"bytes"
	"cloud.google.com/go/storage"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
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

	token := os.Getenv("token")
	channel := os.Getenv("channel")
	client := http.Client{}
	slackMessage := SlackMessage{Channel: channel}
	gcs, err := storage.NewClient(r.Context())
	if err != nil {
		log.Printf("gcs client: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	csccNotification := &CsccNotification{}
	err = json.Unmarshal(m.Message.Data, csccNotification)
	if err := json.Unmarshal(body, &m); err != nil {
		log.Printf("data.Unmarshal: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	switch csccNotification.Finding.Category {
	case "OPEN_FIREWALL":
		slackMessage.Text = fmt.Sprintf("%s: Command Center event on category `%s`\nYou can inspect the resource here: %s",
			csccNotification.Finding.EventTime, csccNotification.Finding.Category, csccNotification.Finding.ExternalURI)
	case "PUBLIC_BUCKET_ACL":
		bucketName := strings.Replace(csccNotification.Finding.ResourceName, "//storage.googleapis.com/", "", 1)
		policy, err := gcs.Bucket(bucketName).IAM().Policy(r.Context())
		if err != nil {
			log.Printf("policy: %v", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		for _, role := range policy.Roles() {
			if policy.HasRole("allUsers", role) {
				policy.Remove("allUsers", role)
			}
			if policy.HasRole("allAuthenticatedUsers", role) {
				policy.Remove("allAuthenticatedUsers", role)
			}
		}
		err = gcs.Bucket(bucketName).IAM().SetPolicy(r.Context(), policy)
		if err != nil {
			log.Printf("apply policy: %v", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		slackMessage.Text = fmt.Sprintf("%s: Command Center event on category `%s`\nThe public authorization has been automatically removed on the resource: %s",
			csccNotification.Finding.EventTime, csccNotification.Finding.Category, csccNotification.Finding.ExternalURI)
	}

	jsonSlacMessage, err := json.Marshal(slackMessage)
	if err := json.Unmarshal(body, &m); err != nil {
		log.Printf("slack.marshal: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	request, err := http.NewRequest("POST", "https://slack.com/api/chat.postMessage", bytes.NewReader(jsonSlacMessage))
	if err := json.Unmarshal(body, &m); err != nil {
		log.Printf("new request: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	request.Header.Add("Content-type", "application/json")
	request.Header.Add("Authorization", "Bearer "+token)

	response, err := client.Do(request)
	if err := json.Unmarshal(body, &m); err != nil {
		log.Printf("request.do: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	resp, err := ioutil.ReadAll(response.Body)
	if err := json.Unmarshal(body, &m); err != nil {
		log.Printf("response: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	fmt.Println(string(resp))
	w.WriteHeader(http.StatusOK)
}

type SlackMessage struct {
	Channel string `json:"channel"`
	Text    string `json:"text"`
}

type CsccNotification struct {
	NotificationConfigName string `json:"notificationConfigName"`
	Finding                struct {
		Name             string `json:"name"`
		Parent           string `json:"parent"`
		ResourceName     string `json:"resourceName"`
		State            string `json:"state"`
		Category         string `json:"category"`
		ExternalURI      string `json:"externalUri"`
		SourceProperties struct {
			Recommendation        string   `json:"Recommendation"`
			ReactivationCount     float64  `json:"ReactivationCount"`
			ExceptionInstructions string   `json:"ExceptionInstructions"`
			Explanation           string   `json:"Explanation"`
			ProjectID             string   `json:"ProjectId"`
			ScannerName           string   `json:"ScannerName"`
			SeverityLevel         string   `json:"SeverityLevel"`
			ResourcePath          []string `json:"ResourcePath"`
			Allowed               string   `json:"Allowed"`
			SourceRanges          string   `json:"SourceRanges"`
			AllowedIPRange        string   `json:"AllowedIpRange"`
			ActivationTrigger     string   `json:"ActivationTrigger"`
			ExternalSourceRanges  []string `json:"ExternalSourceRanges"`
		} `json:"sourceProperties"`
		SecurityMarks struct {
			Name string `json:"name"`
		} `json:"securityMarks"`
		EventTime  time.Time `json:"eventTime"`
		CreateTime time.Time `json:"createTime"`
	} `json:"finding"`
}
