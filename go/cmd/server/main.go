package main

import (
	"log"
	"net/http"

	"payflow"
)

func main() {
	store := payflow.NewStore()
	api := payflow.NewAPI(store)

	log.Println("payflow listening on :8080")
	if err := http.ListenAndServe(":8080", api.Routes()); err != nil {
		log.Fatal(err)
	}
}
