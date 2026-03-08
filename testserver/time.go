package main

import (
	"fmt"
	"net/http"
	"time"
)

func init() {
	http.HandleFunc("/time", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Server time: %s\n", time.Now().Format("15:04:05"))
	})
}
