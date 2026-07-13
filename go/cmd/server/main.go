package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

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

	if rawKeys := os.Getenv("PAYFLOW_API_KEYS"); rawKeys != "" {
		keys := strings.Split(rawKeys, ",")
		api = api.WithAuth(payflow.NewAPIKeyAuth(keys))
		log.Printf("api key auth enabled (%d key(s))", len(keys))
	} else {
		log.Println("PAYFLOW_API_KEYS not set; API is open with no auth")
	}

	rps := 5.0
	if raw := os.Getenv("PAYFLOW_RATE_LIMIT_RPS"); raw != "" {
		if parsed, err := strconv.ParseFloat(raw, 64); err == nil {
			rps = parsed
		}
	}
	api = api.WithRateLimiter(payflow.NewRateLimiter(rps, rps*4))
	log.Printf("rate limiting enabled: %.1f req/s per caller", rps)

	if webhookURL := os.Getenv("PAYFLOW_WEBHOOK_URL"); webhookURL != "" {
		secret := os.Getenv("PAYFLOW_WEBHOOK_SECRET")
		api = api.WithWebhooks(payflow.NewWebhookDispatcher(webhookURL, secret))
		log.Printf("webhooks enabled: %s", webhookURL)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("payflow listening on :%s (event log: %s)", port, logPath)
	if err := http.ListenAndServe(":"+port, api.Routes()); err != nil {
		log.Fatal(err)
	}
}
