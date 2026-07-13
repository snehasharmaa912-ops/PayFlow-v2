package main

import (
	"log"
	"net/http"
	"os"

	"payflow"
)

func main() {
	logPath := os.Getenv("PAYFLOW_LOG_PATH")
	if logPath == "" {
		logPath = "payflow.log"
	}

	store, err := payflow.ReplayFromLog(logPath)
	if err != nil {
		log.Fatalf("replaying event log: %v", err)
	}

	eventLog, err := payflow.OpenEventLog(logPath)
	if err != nil {
		log.Fatalf("opening event log: %v", err)
	}
	defer eventLog.Close()
	store.SetEventLog(eventLog)

	api := payflow.NewAPI(store).WithLogPath(logPath)

	log.Printf("payflow listening on :8080 (event log: %s)", logPath)
	if err := http.ListenAndServe(":8080", api.Routes()); err != nil {
		log.Fatal(err)
	}
}
