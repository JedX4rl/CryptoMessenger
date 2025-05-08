package symmetric

import (
	"errors"
	"fmt"
)

type CipherContext struct {
	key         []byte
	cipher      CipherScheme
	mode        CipherMode
	padding     PaddingMode
	iv          []byte
	extraParams map[string][]byte
	blockSize   int
}

func NewCipherContext(
	key []byte,
	cipher CipherScheme,
	mode CipherMode,
	padding PaddingMode,
	iv []byte,
	blockSize int,
	extraParams ...interface{}) (*CipherContext, error) {

	cryptoContext := &CipherContext{
		key:         key,
		cipher:      cipher,
		mode:        mode,
		padding:     padding,
		iv:          iv,
		blockSize:   blockSize,
		extraParams: make(map[string][]byte),
	}

	if err := cryptoContext.SetKey(key); err != nil {
		return nil, fmt.Errorf("failed to set key: %w", err)
	}

	var paramKey string
	if extraParams != nil {
		tmpKey, ok := extraParams[0].(string)
		if !ok {
			return nil, fmt.Errorf("failed to get param key")
		}
		paramKey = tmpKey
	}

	for i := 1; i < len(extraParams); i++ {
		param, ok := extraParams[i].(int)
		if !ok {
			return nil, fmt.Errorf("invalid param's value")
		}
		cryptoContext.extraParams[paramKey] = append(cryptoContext.extraParams[paramKey], byte(param))
	}
	return cryptoContext, nil
}

func (c *CipherContext) SetKey(key []byte) error {
	if c.cipher == nil {
		return fmt.Errorf("cipher is not initialized")
	}
	return c.cipher.SetKey(key)
}

func (c *CipherContext) Encrypt(data []byte) ([]byte, error) {
	if data == nil || len(data) == 0 {
		return nil, fmt.Errorf("data cannot be empty")
	}

	dataWithPadding, err := c.addPadding(data)
	if err != nil {
		return nil, fmt.Errorf("failed to add padding data: %w", err)
	}

	var encryptedData []byte

	switch c.mode {
	case ECB:
		encryptedData, err = c.EncryptECB(dataWithPadding)
	case CBC:
		encryptedData, err = c.EncryptCBC(dataWithPadding)
	case PCBC:
		encryptedData, err = c.EncryptPCBC(dataWithPadding)
	case CFB:
		encryptedData, err = c.EncryptCFB(dataWithPadding)
	case OFB:
		encryptedData, err = c.EncryptOFB(dataWithPadding)
	case CTR:
		encryptedData, err = c.EncryptCTR(dataWithPadding)
	case RandomDelta:
		encryptedData, err = c.EncryptRandomDelta(dataWithPadding)
	default:
		err = fmt.Errorf("unsupported cipher mode: %d", c.cipher)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to encrypt data: %w", err)
	}

	return encryptedData, nil
}

func (c *CipherContext) Decrypt(data []byte) ([]byte, error) {
	if data == nil || len(data) == 0 {
		return nil, errors.New("data cannot be empty")
	}

	var (
		decryptedData []byte
		err           error
	)

	switch c.mode {
	case ECB:
		decryptedData, err = c.DecryptECB(data)
	case CBC:
		decryptedData, err = c.DecryptCBC(data)
	case PCBC:
		decryptedData, err = c.DecryptPCBC(data)
	case CFB:
		decryptedData, err = c.DecryptCFB(data)
	case OFB:
		decryptedData, err = c.DecryptOFB(data)
	case CTR:
		decryptedData, err = c.DecryptCTR(data)
	default:
		err = fmt.Errorf("unsupported cipher mode: %d", c.cipher)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}

	decryptedData, err = c.removePadding(decryptedData)
	if err != nil {
		return nil, fmt.Errorf("failed to remove padding data: %w", err)
	}
	return decryptedData, nil
}

func (c *CipherContext) EncryptAsync(data []byte) (<-chan []byte, <-chan error) {
	resultChan := make(chan []byte, 1)
	errorChan := make(chan error, 1)

	go func() {
		defer close(resultChan)
		defer close(errorChan)

		encrypted, err := c.Encrypt(data)
		if err != nil {
			errorChan <- err
			return
		}
		resultChan <- encrypted
	}()

	return resultChan, errorChan
}

func (c *CipherContext) DecryptAsync(data []byte) (<-chan []byte, <-chan error) {
	resultChan := make(chan []byte, 1)
	errorChan := make(chan error, 1)

	go func() {
		defer close(resultChan)
		defer close(errorChan)

		decrypted, err := c.Decrypt(data)
		if err != nil {
			errorChan <- err
			return
		}
		resultChan <- decrypted
	}()

	return resultChan, errorChan
}
