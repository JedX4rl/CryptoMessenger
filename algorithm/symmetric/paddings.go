package symmetric

import (
	"bytes"
	"crypto/rand"
	"errors"
)

func (c *CipherContext) addPadding(data []byte) ([]byte, error) {
	if data == nil || len(data) == 0 {
		return nil, errors.New("data cannot be empty")
	}

	paddingSize := c.blockSize - (len(data) & (int(c.blockSize) - 1))
	if paddingSize == 0 {
		paddingSize = c.blockSize
	}

	switch c.padding {
	case Zeros:
		return ZerosPadding(data, paddingSize), nil
	case AnsiX923:
		return ANSIX923Padding(data, paddingSize), nil
	case PKCS7:
		return PKCS7Padding(data, paddingSize), nil
	case Iso10126:
		return ISO10126Padding(data, paddingSize)
	default:
		return nil, errors.New("invalid padding padding")
	}
}

func (c *CipherContext) removePadding(data []byte) ([]byte, error) {
	if data == nil || len(data) == 0 {
		return nil, errors.New("data cannot be empty")
	}
	switch c.padding {
	case Zeros:
		return removeZerosPadding(data), nil
	case AnsiX923:
		return removeANSIX923Padding(data)
	case PKCS7:
		return removePKCS7Padding(data)
	case Iso10126:
		return removeISO10126Padding(data)
	default:
		return nil, errors.New("invalid padding padding")
	}
}

func ZerosPadding(data []byte, paddingSize int) []byte {
	padding := bytes.Repeat([]byte{0}, paddingSize)
	return append(data, padding...)
}

func removeZerosPadding(data []byte) []byte {
	return bytes.TrimRight(data, "\x00")
}

func ANSIX923Padding(data []byte, paddingSize int) []byte {
	padding := append(bytes.Repeat([]byte{0}, paddingSize-1), byte(paddingSize))
	return append(data, padding...)
}

func removeANSIX923Padding(data []byte) ([]byte, error) {
	paddingSize := int(data[len(data)-1])
	if paddingSize > len(data) {
		return nil, errors.New("invalid padding size")
	}
	return data[:len(data)-paddingSize], nil
}

func PKCS7Padding(data []byte, paddingSize int) []byte {
	padding := bytes.Repeat([]byte{byte(paddingSize)}, paddingSize)
	return append(data, padding...)
}

func removePKCS7Padding(data []byte) ([]byte, error) {
	paddingSize := int(data[len(data)-1])
	if paddingSize > len(data) {
		return nil, errors.New("invalid padding size")
	}
	for i := 1; i <= paddingSize; i++ {
		if data[len(data)-i] != byte(paddingSize) {
			return nil, errors.New("invalid PKCS7 padding")
		}
	}
	return data[:len(data)-paddingSize], nil
}

func ISO10126Padding(data []byte, paddingSize int) ([]byte, error) {
	padding := make([]byte, paddingSize)
	if _, err := rand.Read(padding[:paddingSize-1]); err != nil {
		return nil, err
	}
	padding[paddingSize-1] = byte(paddingSize)
	return append(data, padding...), nil
}

func removeISO10126Padding(data []byte) ([]byte, error) {
	paddingSize := int(data[len(data)-1])
	if paddingSize > len(data) {
		return nil, errors.New("invalid padding size")
	}
	return data[:len(data)-paddingSize], nil
}
