package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type Response struct {
	Status string      `json:"status"`
	Msg    string      `json:"msg"`
	Data   interface{} `json:"data"`
}

var jwtSecretKey = []byte(os.Getenv("JWT_SECRET_KEY"))
var sendGridAPIKey = os.Getenv("SENDGRID_API_KEY")

func getCollection() *mongo.Collection {
	client, err := ConnectDB()
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB in getCollection: %v", err)
		return nil
	}
	return client.Database("testdb").Collection("userevent")
}

func Register(w http.ResponseWriter, r *http.Request) {
	var user User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	user.Email = strings.ToLower(user.Email)
	event := map[string]interface{}{
		"event_type": "user_registration",
		"payload": map[string]interface{}{
			"user_id": user.Email,
			"email":   user.Email,
			"name":    "John Doe",
			"password": user.Password,
		},
		"status":      "pending",
		"retry_count": 0,
		"created_at":  time.Now().UTC(),
		"updated_at":  time.Now().UTC(),
	}

	var collection = getCollection()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if user.Email == "" {
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}
	if user.Password == "" {
		http.Error(w, "Password is required atleast 1 character", http.StatusBadRequest)
		return
	}

	result, err := collection.InsertOne(ctx, event)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := Response{
		Status: "ok",
		Msg:    "User Ceated Successfully",
		Data:   map[string]interface{}{"email": user.Email, "event": event, "mongo": result},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func Login(w http.ResponseWriter, r *http.Request) {
	var loginUser User

	err := json.NewDecoder(r.Body).Decode(&loginUser)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	loginUser.Email = strings.ToLower(loginUser.Email)
	if loginUser.Email == "" {
		http.Error(w, "Email is required", http.StatusBadRequest)
		return
	}
	if loginUser.Password == "" {
		http.Error(w, "Password is required", http.StatusBadRequest)
		return
	}

	collection := getCollection()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var foundUser map[string]interface{}
	err = collection.FindOne(ctx, bson.M{"payload.email": loginUser.Email}).Decode(&foundUser)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		} else {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}
	storedPassword := foundUser["payload"].(map[string]interface{})["password"].(string)
	if storedPassword != loginUser.Password {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return

	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": loginUser.Email,
		"exp":   time.Now().Add(24 * time.Hour).Unix(),
	})
	
	tokenStr, err := token.SignedString(jwtSecretKey)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	response := Response{
		Status: "ok",
		Msg:    "Login successful",
		Data:   map[string]string{"token": tokenStr},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func sendWelcomeEmail(email string) error {
	from := mail.NewEmail("shyam kuntal", "shyam.kuntal@digivatelabs.com")
	subject := "Testing the Service"
	to := mail.NewEmail("New User", email)
	plainTextContent := "Welcome to our service. We're glad to have you!"
	htmlContent := "<strong>Welcome to our service. We're glad to have you!</strong>"
	message := mail.NewSingleEmail(from, subject, to, plainTextContent, htmlContent)

	client := sendgrid.NewSendClient(sendGridAPIKey)
	response, err := client.Send(message)
	if err != nil {
		log.Println(err)
	} else {
		fmt.Println(response.StatusCode)
		fmt.Println(response.Body)
		fmt.Println(response.Headers)
	}
	return nil
}

func ConsumeEvents() {
	collection := getCollection()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	for {
		var event map[string]interface{}
		log.Println("Fetching pending events...")
		err := collection.FindOne(ctx, bson.M{"status": "pending"}).Decode(&event)
		
		if err != nil {
			if err == mongo.ErrNoDocuments {
				log.Println("No pending events found. Retrying...")
				time.Sleep(5 * time.Second)
				continue
			}
			log.Printf("Failed to fetch event: %v", err)

			time.Sleep(5 * time.Second) 
			continue
		}

		retryCount, ok := event["retry_count"].(int)
		if !ok {
			log.Println("Invalid retry_count type, skipping event.")
			// continue
		}
		if retryCount > 5 {
			log.Printf("Moving event %v to Dead Letter Queue due to exceeding retries.", event["_id"])
			moveToDeadLetterQueue(event)
			continue
		}
		err = sendWelcomeEmail(event["payload"].(map[string]interface{})["email"].(string))

		if err != nil {
			log.Printf("Failed to process event %v: %v", event["_id"], err)

			event["retry_count"] = retryCount + 1
			event["updated_at"] = time.Now().UTC()
			_, updateErr := collection.UpdateOne(ctx, bson.M{"_id": event["_id"]}, bson.M{"$set": event})
			if updateErr != nil {
				log.Printf("Failed to update retry_count for event %v: %v", event["_id"], updateErr)
			}
			continue
		}

		log.Printf("Successfully processed event %v", event["_id"])
		event["status"] = "processed"
		event["updated_at"] = time.Now().UTC()
		_, updateErr := collection.UpdateOne(ctx, bson.M{"_id": event["_id"]}, bson.M{"$set": event})
		if updateErr != nil {
			log.Printf("Failed to mark event %v as processed: %v", event["_id"], updateErr)
		}
	}
}


func moveToDeadLetterQueue(event map[string]interface{}) {
	deadLetterQueue := getCollection().Database().Collection("dead_letter_queue")
	event["failure_reason"] = "Email service failure"
	event["failed_at"] = time.Now().UTC()

	_, err := deadLetterQueue.InsertOne(context.Background(), event)
	if err != nil {
		log.Fatalf("Failed to insert event into dead-letter queue: %v", err)
	}                
}
