package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"codek7/common/pb"

	"github.com/lumbrjx/codek7/gateway/pkg/utils"
)

func (a API) Register(w http.ResponseWriter, r *http.Request) {
	var user struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	hashedPassword, err := utils.HashPassword(user.Password)
	if err != nil {
		http.Error(w, "Failed to register user the request", http.StatusInternalServerError)
	}
	ur, err := a.RepoClient.CreateUser(r.Context(), &pb.CreateUserRequest{
		Username: user.Username,
		Password: hashedPassword,
		Email:    user.Email,
	})
	if err != nil {
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(ur); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (a API) Login(w http.ResponseWriter, r *http.Request) {
	var user struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	hashedPassword, err := utils.HashPassword(user.Password)
	if err != nil {
		http.Error(w, "Failed to login user the request", http.StatusInternalServerError)
	}
	println("Username:", user.Username)
	res, err := a.RepoClient.GetUser(r.Context(), &pb.GetUserRequest{
		Username: user.Username,
	})
	if err != nil {
		fmt.Println("Failed to get user: %v", err)
		http.Error(w, "Failed to get user", http.StatusInternalServerError)
		return
	}

	fmt.Println("password:", res.Password)
	fmt.Println("hashedPassword:", hashedPassword)
	if res == nil || utils.CheckPasswordHash(user.Password, res.Password) == false {
		http.Error(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	jwtT, err := utils.GenToken(res.Id)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    jwtT,
		Path:     "/",
		MaxAge:   3600, // 1 hour
		HttpOnly: true,
	})

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(res); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
	// Redirect to home page after successful login
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a API) Logout(w http.ResponseWriter, r *http.Request) {
	// delete cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
