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

	if jarPath := os.Getenv("PAYFLOW_RISK_JAR"); jarPath != "" {
		store.SetRiskEvaluator(payflow.NewProcessRiskEvaluator(jarPath))
		log.Printf("risk engine enabled: %s", jarPath)
	} else {
		log.Println("PAYFLOW_RISK_JAR not set; large charges will skip the risk check")
	}

	api := payflow.NewAPI(store).WithLogPath(logPath)

	log.Printf("payflow listening on :8080 (event log: %s)", logPath)
	if err := http.ListenAndServe(":8080", api.Routes()); err != nil {
		log.Fatal(err)
	}
}
