package main

import "context"

func main() {
	ex := NewExchange()
	ex.Run(context.Background())
}
