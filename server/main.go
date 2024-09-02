package main

import "github.com/gorilla/websocket"

func main() {
	env := EnvConfig()

	db := DBConnection(env)

	finnhubWSConn := connectToFinnhub(env)
}

func connectToFinnhub(env *Env) *websocket.Conn{

}