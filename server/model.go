package main

type Candle struct{
	Symbol string `json:"symbol"`
	Open string `json:"open"`
	Close string `json:"close"`
	High string `json:"high"`
	Low string `json:"low"`
	TimeStamp string `json:"timestamp"`
}