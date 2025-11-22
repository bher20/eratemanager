package main

import (
    "log"
    "net/http"
    "os"

    "github.com/bher20/eratemanager/internal/api"
)

func main() {
    port := os.Getenv("PORT")
    if port == "" {
        port = "8000"
    }

    mux := api.NewMux()

    addr := ":" + port
    log.Printf("eRateManager listening on %s", addr)
    if err := http.ListenAndServe(addr, mux); err != nil {
        log.Fatalf("server failed: %v", err)
    }
}
