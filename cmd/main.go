package main

import (
    "fmt"
    "log"
    "net/http"
)

func main() {
    
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "Сервер работает")
    })

    log.Println("Сервер запущен на http://localhost:8080")
    err := http.ListenAndServe(":8080", nil)
    if err != nil {
        log.Fatalf("Ошибка сервера: %v", err)
    }
}