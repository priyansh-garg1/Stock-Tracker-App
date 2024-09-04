package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

var (
	symbols = []string{"AAPL", "AMZN"}

	tempCandles = make(map[string]*TempCandle)

	clientConns = make(map[*websocket.Conn]string)

	broadcast = make(chan *BroadcastMessage)

	mu sync.Mutex
)

func main() {
	env := EnvConfig()

	db := DBConnection(env)

	finnhubWSConn := connectToFinnhub(env)
	defer finnhubWSConn.Close()

	go handleFinnhubMessages(finnhubWSConn, db)

	go broadcastUpdates()

	http.HandleFunc("/ws", WSHandler)

	http.HandleFunc("/stocks-history", func(w http.ResponseWriter, r *http.Request) {
		StocksHistoryHandler(w, r, db)
	})

	http.HandleFunc("/stock-candles", func(w http.ResponseWriter, r *http.Request) {
		CandlesHandler(w, r, db)
	})

	http.ListenAndServe(fmt.Sprintf(":%s", env.ServerPort), nil)
}

func WSHandler(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading the connection: %e", err)
		return
	}

	defer conn.Close()
	defer func() {
		delete(clientConns, conn)
		log.Println("Client Disconnected")
	}()

	for {
		_, symbol, err := conn.ReadMessage()
		clientConns[conn] = string(symbol)
		log.Printf("New Client Connected ")

		if err != nil {
			log.Printf("Error reading message from client: %e", err)
			break
		}
	}
}
func StocksHistoryHandler(w http.ResponseWriter, r *http.Request, db *gorm.DB) {
	var candles []Candle
	db.Order("timestamp asc").Find(&candles)

	groupedData := make(map[string][]Candle)

	for _, candle := range candles {
		groupedData[candle.Symbol] = append(groupedData[candle.Symbol], candle)
	}

	jsonResponse, _ := json.Marshal(groupedData)
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResponse)
}

func CandlesHandler(w http.ResponseWriter, r *http.Request, db *gorm.DB) {
	symbol := r.URL.Query().Get("symbol")

	var candles []Candle
	db.Where("symbol = ?", symbol).Order("timestamp asc").Find(&candles)

	jsonCandles, _ := json.Marshal(candles)
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonCandles)
}

func connectToFinnhub(env *Env) *websocket.Conn {
	ws, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("wss://ws.finnhub.io?token=%s", env.API_KEY), nil)
	if err != nil {
		panic(err)
	}

	for _, s := range symbols {
		msg, _ := json.Marshal(map[string]interface{}{"type": "subscribe", "symbol": s})
		ws.WriteMessage(websocket.TextMessage, msg)
	}

	return ws
}

func handleFinnhubMessages(ws *websocket.Conn, db *gorm.DB) {

	finnhubMessage := &FinnhubMessage{}

	for {
		if err := ws.ReadJSON(finnhubMessage); err != nil {
			fmt.Printf("Error reading the message: %e", err)
			continue
		}

		if finnhubMessage.Type == "trade" {
			for _, trade := range finnhubMessage.Data {
				processTradeData(&trade, db)
			}
		}
	}

}

func processTradeData(trade *TradeData, db *gorm.DB) {
	mu.Lock()
	defer mu.Unlock()

	symbol := trade.Symbol
	price := trade.Price
	volume := trade.Volume
	timestamp := time.UnixMilli(trade.Timestamp)

	tempCandle, exists := tempCandles[symbol]

	if !exists || timestamp.After(tempCandle.CloseTime) {
		if exists {
			candle := tempCandle.toCandle()

			if err := db.Create(candle).Error; err != nil {
				fmt.Println("Error creating candle to DB: ", err)
			}

			broadcast <- &BroadcastMessage{
				UpdateType: Closed,
				Candle:     candle,
			}
		}

		tempCandle = &TempCandle{
			Symbol:     symbol,
			OpenTime:   timestamp,
			CloseTime:  timestamp.Add(time.Minute),
			OpenPrice:  price,
			ClosePrice: price,
			HighPrice:  price,
			LowPrice:   price,
			Volume:     float64(volume),
		}
	}

	tempCandle.ClosePrice = price
	tempCandle.Volume += float64(volume)
	if price < tempCandle.HighPrice {
		tempCandle.HighPrice = price
	}
	if price < tempCandle.LowPrice {
		tempCandle.LowPrice = price
	}

	tempCandles[symbol] = tempCandle

	broadcast <- &BroadcastMessage{
		UpdateType: Live,
		Candle:     tempCandle.toCandle(),
	}

}

func broadcastUpdates() {
	ticket := time.NewTicker(1 * time.Second)
	defer ticket.Stop()

	var latestUpdate *BroadcastMessage

	for {
		select {
		case update := <-broadcast:
			if update.UpdateType == Closed {
				broadcastToClient(update)

			} else {
				latestUpdate = update
			}

		case <-ticket.C:
			if latestUpdate != nil {
				broadcastToClient(latestUpdate)
			}
			latestUpdate = nil
		}
	}
}

func broadcastToClient(update *BroadcastMessage) {
	jsonUpdate, _ := json.Marshal(update)

	for clientConn, symbol := range clientConns {
		if update.Candle.Symbol == symbol {
			err := clientConn.WriteMessage(websocket.TextMessage, jsonUpdate)
			if err != nil {
				log.Printf("Error sending message to client: %e", err)
				clientConn.Close()
				delete(clientConns, clientConn)
			}
		}
	}
}
