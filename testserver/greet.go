package main

import (
	"fmt"
	"net/http"

	"github.com/kunalsinghdadhwal/hotreload/testserver/utils"
)

func init() {
	http.HandleFunc("/greet", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		if name == "" {
			name = "World"
		}
		fmt.Fprintln(w, utils.Greet(name))
	})
}
