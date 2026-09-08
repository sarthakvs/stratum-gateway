package main

import (
	"fmt"
	"os"
)

func main() {
	cfg, err := LoadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Config:", err)
		os.Exit(1)
	}
	fmt.Println("Stratum-gateway is running on ...", cfg.Addr)
}
