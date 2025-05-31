package pkg

import (
	"CryptoMessenger/algorithm/symmetric"
	_ "CryptoMessenger/algorithm/symmetric"
	"fmt"
	"strings"
)

func ParseCipherMode(mode string) (symmetric.CipherMode, error) {
	switch mode {
	case "ECB":
		return symmetric.ECB, nil
	case "CBC":
		return symmetric.CBC, nil
	case "PCBC":
		return symmetric.PCBC, nil
	case "CFB":
		return symmetric.CFB, nil
	case "OFB":
		return symmetric.OFB, nil
	case "CTR":
		return symmetric.CTR, nil
	case "RandomDelta":
		return symmetric.RandomDelta, nil
	default:
		return 0, fmt.Errorf("unknown cipher mode: %s", mode)
	}
}

func ParsePaddingMode(padding string) (symmetric.PaddingMode, error) {
	switch strings.ToUpper(padding) {
	case "ZEROS":
		return symmetric.Zeros, nil
	case "ANSIX923":
		return symmetric.AnsiX923, nil
	case "PKCS7":
		return symmetric.PKCS7, nil
	case "ISO10126":
		return symmetric.Iso10126, nil
	default:
		return 0, fmt.Errorf("unknown padding mode: %s", padding)
	}
}
