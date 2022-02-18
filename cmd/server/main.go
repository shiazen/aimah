package main

import (
	"fmt"
	"net/http"
)

func main() {
	handler := func(_ http.ResponseWriter, r *http.Request) {
		// fmt.Printf("Req: %s %s\n", r.Host, r.URL.Path) 
		fmt.Println(r.URL.Path) 
	}
	http.HandleFunc("/", handler)

	http.ListenAndServe(":8080", nil)
}
