package main

import (
	"CryptoMessenger/algorithm/rc6"
	"CryptoMessenger/algorithm/symmetric"
	"fmt"
)

func main() {

	key := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	//rc, err := rc5.NewRC5(64, 100, 64, key)
	//if err != nil {
	//	panic(err)
	//}
	rc, err := rc6.NewRC6(key)
	if err != nil {
		fmt.Println(err)
	}
	iv := make([]byte, 16)

	testContext, err := symmetric.NewCipherContext(key, rc, symmetric.ECB, symmetric.Zeros, iv, 16)
	if err != nil {
		panic(err)
	}
	msg := []byte("hello worldDKADKL:ASKdl;kl;k ;lfkdsl;fkkdj kjk12k312ok3 klf;kdsl;fkdlsfl")
	encrypted, err := testContext.Encrypt(msg)
	if err != nil {
		panic(err)
	}
	decrypted, err := testContext.Decrypt(encrypted)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(decrypted))

}
