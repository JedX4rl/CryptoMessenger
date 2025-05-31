package main

import (
	"CryptoMessenger/algorithm/rc5"
	"CryptoMessenger/algorithm/symmetric"
	"crypto/rand"
	"fmt"
)

func main() {

	key := make([]byte, 64)
	_, err := rand.Read(key[:64])
	if err != nil {
		panic(fmt.Errorf("failed to generate random bytes: %w", err))
	}
	algo, err := rc5.NewRC5(64, 12, uint(len(key)), key)
	if err != nil {
		panic(fmt.Errorf("failed to generate rc5 algorithm: %w", err))
	}

	iv := make([]byte, 64)
	_, err = rand.Read(iv)
	if err != nil {
		panic(fmt.Errorf("failed to generate random bytes: %w", err))
	}

	cipher, err := symmetric.NewCipherContext(key, algo, symmetric.ECB, symmetric.AnsiX923, iv, 16)
	if err != nil {
		panic(fmt.Errorf("failed to create symmetric cipher: %w", err))
	}
	_ = cipher
}
