package app

import (
	"encoding/json"
	"net/http"
)

// UserController serves the example user API.
//
// @Controller(prefix="/users")
type UserController struct{}

// UserResponse is the public representation returned by the example endpoint.
type UserResponse struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

// GetUser returns a user-shaped example response.
//
// @Get(path="/{id}")
func (*UserController) GetUser(writer http.ResponseWriter, request *http.Request) {
	payload, err := json.Marshal(UserResponse{
		ID:      request.PathValue("id"),
		Message: "hello from Spice",
	})
	if err != nil {
		http.Error(writer, "encode response", http.StatusInternalServerError)
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	if _, err := writer.Write(payload); err != nil {
		return
	}
}

// ControllerProvider constructs the example controller.
//
// @Bean
func ControllerProvider() *UserController {
	return &UserController{}
}

// MuxProvider supplies the exact route table populated by generated adapters.
//
// @Bean
func MuxProvider() *http.ServeMux {
	return http.NewServeMux()
}
