package main

import (
	"fmt"
	"time"
)

type Candle struct {
	Symbol    string `json:"symbol"`
	Open      string `json:"open"`
	Close     string `json:"close"`
	High      string `json:"high"`
	Low       string `json:"low"`
	TimeStamp string `json:"timestamp"`
}

type TempCandle struct {
	Symbol     string
	OpenTime   time.Time
	CloseTime  time.Time
	OpenPrice  float64
	ClosePrice float64
	HighPrice  float64
	LowPrice   float64
	Volume     float64
}

type FinnhubMessage struct {
	Data []TradeData `json:"data"`
	Type string      `json:"type"`
}

type TradeData struct {
	Close     []string `json:"c"`
	Price     float64  `json:"p"`
	Symbol    string   `json:"s"`
	Timestamp int64    `json:"t"`
	Volume    int      `json:"v"`
}

type BroadcastMessage struct {
	UpdateType UpdateType `json:"updateType"`
	Candle    *Candle  `json:"candle"`
}

type UpdateType string

const (
	Live UpdateType = "live"
	Closed UpdateType = "closed"
)

func (tc *TempCandle) toCandle() *Candle {
	return &Candle{
		Symbol:    tc.Symbol,
		Open:      fmt.Sprintf("%f",tc.OpenPrice),
		Close:     fmt.Sprintf("%f",tc.ClosePrice),
		High:      fmt.Sprintf("%f",tc.HighPrice),
		Low:       fmt.Sprintf("%f",tc.LowPrice),
		TimeStamp: fmt.Sprintf("%t",tc.CloseTime),
	}
}
