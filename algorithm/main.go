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

	//data, err := os.ReadFile("algorithm/input")
	//if err != nil {
	//	fmt.Println("Ошибка при чтении файла:", err)
	//	return
	//}

	cipher, err := symmetric.NewCipherContext(key, algo, symmetric.ECB, symmetric.AnsiX923, iv, 16)
	if err != nil {
		panic(fmt.Errorf("failed to create symmetric cipher: %w", err))
	}
	_ = cipher

	//if err := cipher.EncryptFile("/Users/nikitatretakov/Downloads/Курсовая бд.pdf", "algorithm/tmp.txt"); err != nil {
	//	panic(fmt.Errorf("failed to encrypt file: %w", err))
	//}
	//
	//if err := cipher.DecryptFile("algorithm/tmp.txt", "algorithm/out.pdf"); err != nil {
	//	panic(fmt.Errorf("failed to decrypt file: %w", err))
	//}

	//encrypted, err := cipher.Encrypt(data)
	//if err != nil {
	//	panic(fmt.Errorf("failed to encrypt data: %w", err))
	//}
	//decrypted, err := cipher.Decrypt(encrypted)
	//if err != nil {
	//	panic(fmt.Errorf("failed to decrypt data: %w", err))
	//}
	//
	//err = os.WriteFile("algorithm/out.pdf", decrypted, 0644)
	//if err != nil {
	//	fmt.Println("Ошибка при записи файла:", err)
	//	return
	//}

}
