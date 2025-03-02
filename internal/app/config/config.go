package config

import (
	"flag"
)

var FlagRunAddr string
var FlagResultHostAddr string

func ParseFlags() {
	flag.StringVar(&FlagRunAddr, "a", "http://localhost:8080", "address and port to run server")
	flag.StringVar(&FlagResultHostAddr, "b", "", "address and port in result link")
	flag.Parse()
}
