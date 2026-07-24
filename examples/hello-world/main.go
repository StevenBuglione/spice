package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
)

// @Controller(prefix="/users")
type UserController struct{}

// @Get(path="/{id}")
func (UserController) GetUser(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(writer).Encode(map[string]string{
		"id":      request.PathValue("id"),
		"message": "hello from Spice",
	})
}

func newHandler() http.Handler {
	controller := UserController{}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /users/{id}", controller.GetUser)
	return mux
}

func main() {
	listen := flag.String("listen", ":8080", "HTTP listen address")
	check := flag.Bool("check", false, "build the application, print its route, and exit")
	flag.Parse()

	if *check {
		fmt.Println("Spice example ready: GET /users/{id}")
		return
	}

	log.Printf("Spice example listening on %s", *listen)
	if err := http.ListenAndServe(*listen, newHandler()); err != nil {
		log.Fatal(err)
	}
}
