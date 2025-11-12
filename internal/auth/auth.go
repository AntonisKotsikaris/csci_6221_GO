package auth

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gorilla/sessions"
)

var authHTML string

var (
	templates *template.Template
	store     = sessions.NewCookieStore([]byte("your-secret-key-change-this-in-production"))
)

// User represents a user from auth.json
type User struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

var (
	users    []User
	usersMux sync.RWMutex
	authFile = filepath.Join("DB", "auth.json")
)

func init() {
	templatesPath := filepath.Join("internal", "auth", "views", "auth.html")
	templates = template.Must(template.ParseFiles(templatesPath))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   3600 * 8, // 8 hours
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
	}

	// Load users from auth.json
	if err := loadUsers(); err != nil {
		log.Printf("Warning: Could not load users from auth.json: %v", err)
	} else {
		log.Printf("Successfully loaded %d users from auth.json", len(users))
	}
}

// loadUsers reads users from DB/auth.json
func loadUsers() error {
	usersMux.Lock()
	defer usersMux.Unlock()

	data, err := os.ReadFile(authFile)
	if err != nil {
		return err
	}

	users = []User{}
	if err := json.Unmarshal(data, &users); err != nil {
		return err
	}

	return nil
}

// authenticateUser checks if email and password match any user in auth.json
func authenticateUser(email, password string) (*User, error) {
	usersMux.RLock()
	defer usersMux.RUnlock()

	for _, user := range users {
		if user.Email == email && user.Password == password {
			return &user, nil
		}
	}

	return nil, nil // Return nil if not found
}

// Setup registers all auth-related routes
func Setup() {
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/signup", signupHandler)
	http.HandleFunc("/auth/login", loginPostHandler)
	http.HandleFunc("/auth/signup", signupPostHandler)
	http.HandleFunc("/logout", logoutHandler)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	// If already authenticated, redirect to connect
	session, _ := store.Get(r, "auth-session")
	if auth, ok := session.Values["authenticated"].(bool); ok && auth {
		http.Redirect(w, r, "/connect", http.StatusSeeOther)
		return
	}

	data := map[string]interface{}{
		"Title":  "Login",
		"Action": "/auth/login",
	}
	templates.Execute(w, data)
}

func signupHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title":  "Sign Up",
		"Action": "/auth/signup",
	}
	templates.Execute(w, data)
}

func loginPostHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")

	// Authenticate user from auth.json
	user, err := authenticateUser(email, password)
	if err != nil || user == nil {
		log.Printf("Login failed for %s", email)
		data := map[string]interface{}{
			"Title":  "Login",
			"Action": "/auth/login",
			"Error":  "Invalid email or password",
		}
		templates.Execute(w, data)
		return
	}

	// Create session
	session, _ := store.Get(r, "auth-session")
	session.Values["authenticated"] = true
	session.Values["email"] = user.Email
	session.Values["login_time"] = time.Now().Unix()
	session.Save(r, w)

	log.Printf("User %s logged in successfully", user.Email)

	// Redirect to connect endpoint
	http.Redirect(w, r, "/connect", http.StatusSeeOther)
}

func signupPostHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Redirect(w, r, "/signup", http.StatusSeeOther)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")
	confirmPassword := r.FormValue("confirm_password")

	// Validation
	if password != confirmPassword {
		data := map[string]interface{}{
			"Title":  "Sign Up",
			"Action": "/auth/signup",
			"Error":  "Passwords do not match",
		}
		templates.Execute(w, data)
		return
	}

	if len(password) < 6 {
		data := map[string]interface{}{
			"Title":  "Sign Up",
			"Action": "/auth/signup",
			"Error":  "Password must be at least 6 characters",
		}
		templates.Execute(w, data)
		return
	}

	// Check if user already exists
	usersMux.RLock()
	for _, user := range users {
		if user.Email == email {
			usersMux.RUnlock()
			data := map[string]interface{}{
				"Title":  "Sign Up",
				"Action": "/auth/signup",
				"Error":  "User already exists",
			}
			templates.Execute(w, data)
			return
		}
	}
	usersMux.RUnlock()

	// Add new user to the list
	usersMux.Lock()
	newUser := User{
		Email:    email,
		Password: password,
	}
	users = append(users, newUser)
	usersMux.Unlock()

	// Save to auth.json file
	if err := saveUsers(); err != nil {
		log.Printf("Error saving new user: %v", err)
		data := map[string]interface{}{
			"Title":  "Sign Up",
			"Action": "/auth/signup",
			"Error":  "Error creating account",
		}
		templates.Execute(w, data)
		return
	}

	log.Printf("New user created: %s", email)

	// Auto-login after signup
	session, _ := store.Get(r, "auth-session")
	session.Values["authenticated"] = true
	session.Values["email"] = email
	session.Values["login_time"] = time.Now().Unix()
	session.Save(r, w)

	http.Redirect(w, r, "/connect", http.StatusSeeOther)
}

// saveUsers writes the current users list back to auth.json
func saveUsers() error {
	usersMux.RLock()
	defer usersMux.RUnlock()

	data, err := json.MarshalIndent(users, "", "    ")
	if err != nil {
		return err
	}

	return os.WriteFile(authFile, data, 0644)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "auth-session")

	if email, ok := session.Values["email"].(string); ok {
		log.Printf("User %s logged out", email)
	}

	session.Values["authenticated"] = false
	session.Options.MaxAge = -1 // Delete session
	session.Save(r, w)

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// RequireAuth is a middleware that protects routes
func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := store.Get(r, "auth-session")

		if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
			log.Printf("Unauthorized access attempt to %s", r.URL.Path)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		next(w, r)
	}
}
