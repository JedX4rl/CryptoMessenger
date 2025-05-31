package symmetric

import (
	"errors"
	"fmt"
	"sync"
)

func (c *CipherContext) EncryptECB(data []byte) ([]byte, error) {
	if len(data)&(c.blockSize-1) != 0 {
		return nil, errors.New("block size must be a multiple of the block size")
	}

	numberOfBlocks := len(data) / c.blockSize
	encrypted := make([]byte, len(data))

	wg := &sync.WaitGroup{}
	errChan := make(chan error, numberOfBlocks)

	for i := 0; i < numberOfBlocks; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pos := i * c.blockSize
			block := data[pos : pos+c.blockSize]

			encryptedBlock, err := c.cipher.Encrypt(block)
			if err != nil {
				errChan <- fmt.Errorf("encryption failed at block %d: %w", i, err)
			}

			copy(encrypted[pos:], encryptedBlock)
		}()
	}

	wg.Wait()
	close(errChan)

	if err, ok := <-errChan; ok {
		return nil, err
	}

	return encrypted, nil
}

func (c *CipherContext) DecryptECB(data []byte) ([]byte, error) {
	if len(data)&(c.blockSize-1) != 0 {
		return nil, errors.New("block size must be a multiple of the block size")
	}

	numberOfBlocks := len(data) / c.blockSize
	decrypted := make([]byte, len(data))

	wg := &sync.WaitGroup{}
	errChan := make(chan error, numberOfBlocks)

	for i := 0; i < numberOfBlocks; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pos := i * c.blockSize
			block := data[pos : pos+c.blockSize]

			decryptedBlock, err := c.cipher.Decrypt(block)
			if err != nil {
				errChan <- fmt.Errorf("decryption failed at block %d: %w", i, err)
			}

			copy(decrypted[pos:], decryptedBlock)
		}()
	}

	wg.Wait()
	close(errChan)

	if err, ok := <-errChan; ok {
		return nil, err
	}

	return decrypted, nil
}

func (c *CipherContext) EncryptCBC(data []byte) ([]byte, error) {
	if len(data)&(c.blockSize-1) != 0 {
		return nil, errors.New("block size must be a multiple of the block size")
	}
	if len(c.iv) != c.blockSize {
		return nil, errors.New("iv length must be equal to block size")
	}

	encrypted := make([]byte, len(data))
	previousBlock := make([]byte, c.blockSize)
	numberOfBlocks := len(data) / c.blockSize
	copy(previousBlock, c.iv)

	for i := 0; i < numberOfBlocks; i++ {
		pos := i * c.blockSize
		currBlock := data[pos : pos+c.blockSize]

		xoredBlock := xorBlocks(currBlock, previousBlock)
		encryptedBlock, err := c.cipher.Encrypt(xoredBlock)
		if err != nil {
			return nil, fmt.Errorf("encryption failed at block %d, %w", i, err)
		}

		copy(encrypted[pos:], encryptedBlock)
		copy(previousBlock, encryptedBlock)
	}
	return encrypted, nil
}

func (c *CipherContext) DecryptCBC(data []byte) ([]byte, error) {
	if len(data)&(c.blockSize-1) != 0 {
		return nil, errors.New("block size must be a multiple of the block size")
	}
	if len(c.iv) != c.blockSize {
		return nil, errors.New("iv length must be equal to block size")
	}

	decrypted := make([]byte, len(data))
	previousBlock := make([]byte, c.blockSize)
	numberOfBlocks := len(data) / c.blockSize
	copy(previousBlock, c.iv)

	for i := 0; i < numberOfBlocks; i++ {
		pos := i * c.blockSize
		currBlock := data[pos : pos+c.blockSize]

		decryptedBlock, err := c.cipher.Decrypt(currBlock)
		if err != nil {
			return nil, fmt.Errorf("decryption failed at block %d, %w", i, err)
		}

		xoredBlock := xorBlocks(decryptedBlock, previousBlock)
		copy(decrypted[pos:], xoredBlock)
		copy(previousBlock, currBlock)
	}

	return decrypted, nil
}

func (c *CipherContext) EncryptPCBC(data []byte) ([]byte, error) {
	if len(data)&(c.blockSize-1) != 0 {
		return nil, errors.New("block size must be a multiple of the block size")
	}
	if len(c.iv) != c.blockSize {
		return nil, errors.New("iv length must be equal to block size")
	}

	encrypted := make([]byte, len(data))
	numberOfBlocks := len(data) / c.blockSize
	previousPlaintext := make([]byte, c.blockSize)
	previousCiphertext := make([]byte, c.blockSize)
	copy(previousCiphertext, c.iv)

	for i := 0; i < numberOfBlocks; i++ {
		pos := i * c.blockSize
		plaintextBlock := data[pos : pos+c.blockSize]
		inputBlock := make([]byte, c.blockSize)
		for j := 0; j < c.blockSize; j++ {
			inputBlock[j] = plaintextBlock[j] ^ previousPlaintext[j] ^ previousCiphertext[j]
		}

		encryptedBlock, err := c.cipher.Encrypt(inputBlock)
		if err != nil {
			return nil, fmt.Errorf("encryption failed at block %d, %w", i, err)
		}

		copy(encrypted[pos:], encryptedBlock)
		copy(previousPlaintext, plaintextBlock)
		copy(previousCiphertext, encryptedBlock)
	}

	return encrypted, nil
}

func (c *CipherContext) DecryptPCBC(data []byte) ([]byte, error) {
	if len(data)&(c.blockSize-1) != 0 {
		return nil, errors.New("block size must be a multiple of the block size")
	}
	if len(c.iv) != c.blockSize {
		return nil, errors.New("iv length must be equal to block size")
	}

	decrypted := make([]byte, len(data))
	numberOfBlocks := len(data) / c.blockSize
	previousPlaintext := make([]byte, c.blockSize)
	previousCiphertext := make([]byte, c.blockSize)
	copy(previousCiphertext, c.iv)

	for i := 0; i < numberOfBlocks; i++ {
		pos := i * c.blockSize
		currBlock := data[pos : pos+c.blockSize]

		decryptedBlock, err := c.cipher.Decrypt(currBlock)
		if err != nil {
			return nil, fmt.Errorf("decryption failed at block %d, %w", i, err)
		}

		for j := 0; j < c.blockSize; j++ {
			decrypted[pos+j] = decryptedBlock[j] ^ previousPlaintext[j] ^ previousCiphertext[j]
		}

		copy(previousPlaintext, decrypted[pos:pos+c.blockSize])
		copy(previousCiphertext, currBlock)
	}

	return decrypted, nil
}

func (c *CipherContext) EncryptCFB(data []byte) ([]byte, error) {
	if len(data)&(c.blockSize-1) != 0 {
		return nil, errors.New("block size must be a multiple of the block size")
	}
	if len(c.iv) != c.blockSize {
		return nil, errors.New("iv length must be equal to block size")
	}

	encrypted := make([]byte, len(data))
	previousBlock := make([]byte, c.blockSize)
	numberOfBlocks := len(data) / c.blockSize
	copy(previousBlock, c.iv)

	for i := 0; i < numberOfBlocks; i++ {
		pos := i * c.blockSize
		encryptedIV, err := c.cipher.Encrypt(previousBlock)
		if err != nil {
			return nil, fmt.Errorf("cannot encrypt IV at block %d, %w", i, err)
		}

		end := pos + c.blockSize
		if end > len(data) {
			end = len(data)
		}

		currBlock := data[pos:end]
		xoredBlock := xorBlocks(currBlock, encryptedIV)

		copy(encrypted[pos:], xoredBlock)
		copy(previousBlock, xoredBlock)
	}

	return encrypted, nil
}

func (c *CipherContext) DecryptCFB(data []byte) ([]byte, error) {
	if len(c.iv) != c.blockSize {
		return nil, errors.New("iv length must be equal to block size")
	}

	decrypted := make([]byte, len(data))
	previousBlock := make([]byte, c.blockSize)
	numberOfBlocks := len(data) / c.blockSize
	copy(previousBlock, c.iv)

	for i := 0; i < numberOfBlocks; i++ {
		pos := i * c.blockSize

		outputBlock, err := c.cipher.Encrypt(previousBlock)
		if err != nil {
			return nil, fmt.Errorf("decryption failed at block %d, %w", i, err)
		}

		end := pos + c.blockSize
		if end > len(data) {
			end = len(data)
		}

		currBlock := data[pos:end]
		xoredBlock := xorBlocks(currBlock, outputBlock)

		copy(decrypted[pos:], xoredBlock)
		copy(previousBlock, currBlock)
	}

	return decrypted, nil
}

func (c *CipherContext) EncryptOFB(data []byte) ([]byte, error) {
	if len(c.iv) != c.blockSize {
		return nil, errors.New("iv length must be equal to block size")
	}

	encrypted := make([]byte, len(data))
	previousBlock := make([]byte, c.blockSize)
	numberOfBlocks := len(data) / c.blockSize
	copy(previousBlock, c.iv)

	for i := 0; i < numberOfBlocks; i++ {
		pos := i * c.blockSize

		encryptedBlock, err := c.cipher.Encrypt(previousBlock)
		if err != nil {
			return nil, fmt.Errorf("encryption failed at block %d, %w", i, err)
		}

		end := pos + c.blockSize
		if end > len(data) {
			end = len(data)
		}

		currBlock := data[pos:end]
		xoredBlock := xorBlocks(currBlock, encryptedBlock)

		copy(encrypted[pos:], xoredBlock)
		copy(previousBlock, encryptedBlock)
	}

	return encrypted, nil
}

func (c *CipherContext) DecryptOFB(data []byte) ([]byte, error) {
	return c.EncryptOFB(data)
}

func (c *CipherContext) EncryptCTR(data []byte) ([]byte, error) {
	if len(c.iv) != c.blockSize {
		return nil, errors.New("iv length must be equal to block size")
	}

	encrypted := make([]byte, len(data))
	numberOfBlocks := len(data) / c.blockSize

	wg := &sync.WaitGroup{}
	errCh := make(chan error, numberOfBlocks)

	counter := make([]byte, c.blockSize)

	copy(counter, c.iv)

	for i := 0; i < numberOfBlocks; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			pos := i * c.blockSize

			localCounter := make([]byte, c.blockSize)
			copy(localCounter, c.iv)
			incrementCounterBy(localCounter, i)

			encryptedCounter, err := c.cipher.Encrypt(localCounter)
			if err != nil {
				errCh <- fmt.Errorf("encryption failed at block %d, %w", i, err)
				return
			}

			end := pos + c.blockSize
			if end > len(data) {
				end = len(data)
			}
			currBlock := data[pos:end]

			xoredBlock := xorBlocks(currBlock, encryptedCounter)
			copy(encrypted[pos:], xoredBlock)

		}()
	}

	wg.Wait()
	close(errCh)

	if err, ok := <-errCh; ok {
		return nil, err
	}

	return encrypted, nil
}

func (c *CipherContext) DecryptCTR(data []byte) ([]byte, error) {
	return c.EncryptCTR(data)
}

func (c *CipherContext) EncryptRandomDelta(data []byte) ([]byte, error) {
	if len(c.iv) != c.blockSize {
		return nil, errors.New("iv length must be equal to block size")
	}

	randomDelta, exists := c.extraParams["randomDelta"]
	if !exists {
		return nil, errors.New("randomDelta parameter is missing")
	}
	if len(randomDelta) != c.blockSize {
		return nil, errors.New("randomDelta length must match block size")
	}

	blockSize := c.blockSize
	numBlocks := (len(data) + blockSize - 1) / blockSize
	encrypted := make([]byte, len(data))

	wg := sync.WaitGroup{}
	errCh := make(chan error, numBlocks)

	for i := 0; i < numBlocks; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			pos := i * blockSize
			end := pos + blockSize
			if end > len(data) {
				end = len(data)
			}
			block := data[pos:end]

			counterI := computeCounter(c.iv, randomDelta, i)

			encryptedCounter, err := c.cipher.Encrypt(counterI)
			if err != nil {
				errCh <- fmt.Errorf("encryption failed at block %d: %w", i, err)
				return
			}

			for j := 0; j < end-pos; j++ {
				encrypted[pos+j] = block[j] ^ encryptedCounter[j]
			}
		}()
	}

	wg.Wait()
	close(errCh)

	if err, ok := <-errCh; ok {
		return nil, err
	}

	return encrypted, nil
}

func (c *CipherContext) DecryptRandomDelta(data []byte) ([]byte, error) {
	return c.EncryptRandomDelta(data)
}

func xorBlocks(a, b []byte) []byte {
	res := make([]byte, len(a))
	for i := 0; i < len(a); i++ {
		res[i] = a[i] ^ b[i]
	}
	return res
}

func computeCounter(iv, delta []byte, multiplier int) []byte {
	result := make([]byte, len(iv))
	carry := 0

	for i := len(iv) - 1; i >= 0; i-- {
		d := int(delta[i]) * multiplier
		sum := int(iv[i]) + d + carry
		result[i] = byte(sum & 0xFF)
		carry = sum >> 8
	}
	return result
}

func incrementCounterBy(counter []byte, value int) {
	n := len(counter)
	for i := n - 1; i >= 0 && value > 0; i-- {
		sum := int(counter[i]) + (value & 0xFF)
		counter[i] = byte(sum & 0xFF)
		value >>= 8
		if sum > 0xFF {
			value++
		}
	}
}
