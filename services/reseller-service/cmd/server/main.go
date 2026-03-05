package main

import (
"fmt"
"net/http"
"os"
)

func main() {
port := os.Getenv("PORT")
if port == "" {
port = "8080"
}
mux := http.NewServeMux()
mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
w.Header().Set("Content-Type", "application/json")
fmt.Fprintf(w, `{"status":"ok","service":"%s"}`, os.Args[0])
})
http.ListenAndServe(":"+port, mux)
}
