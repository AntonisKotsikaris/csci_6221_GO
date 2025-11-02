package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtKey = []byte("supersecretkey")

func main() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter username: ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)

	tokenStr, _ := generateJWT(username)

	conn, err := net.Dial("tcp", "0.0.0.0:9000")
	if err != nil {
		fmt.Println("Connection error:", err)
		return
	}
	defer conn.Close()

	conn.Write([]byte(tokenStr + "\n"))
	go listenServer(conn)
	go sendHeartbeats(conn)

	for {
		text, _ := reader.ReadString('\n')
		conn.Write([]byte(text))
	}
}

func listenServer(conn net.Conn) {
	reader := bufio.NewReader(conn)
	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println(":x: Server disconnected.")
			os.Exit(0)
		}
		fmt.Print(msg)
	}
}

func sendHeartbeats(conn net.Conn) {
	for {
		time.Sleep(5 * time.Second)
		conn.Write([]byte("PING\n"))
	}
}

func generateJWT(username string) (string, error) {
	claims := jwt.MapClaims{
		"username": username,
		"exp":      time.Now().Add(10 * time.Minute).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}
