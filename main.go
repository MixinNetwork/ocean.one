package main

import "context"

func main() {
	ex := NewExchange()
	ex.PollMixinNetwork(context.Background())
}
