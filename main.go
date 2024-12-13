package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main(){
	writerFunc := func(w http.ResponseWriter, r *http.Request){
		w.Write([]byte("Wecome Shyam Kuntal"))
	}


	http.HandleFunc("/", writerFunc)
	http.HandleFunc("POST /register", Register)
	http.HandleFunc("GET /login", Login)

	go ConsumeEvents()

	Port := os.Getenv("PORT")
	fmt.Printf("Server is Running on %s", Port)
	log.Fatal(http.ListenAndServe(Port, nil))
}