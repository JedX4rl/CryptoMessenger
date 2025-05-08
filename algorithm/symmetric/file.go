package symmetric

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func (c *CipherContext) EncryptFile(inputPath, outputPath string) error {
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		return fmt.Errorf("inputPath %s does not exist", inputPath)
	}

	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	inputFile, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("cannot open input file: %w", err)
	}
	defer inputFile.Close()

	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("cannot open output file: %w", err)
	}
	defer outputFile.Close()

	buffer := make([]byte, c.blockSize*1024)
	for {
		n, err := inputFile.Read(buffer)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}

		encrypted, err := c.Encrypt(buffer[:n])
		if err != nil {
			return err
		}

		if _, err := outputFile.Write(encrypted); err != nil {
			return err
		}
	}

	return nil
}

func (c *CipherContext) DecryptFile(inputPath, outputPath string) error {
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		return fmt.Errorf("inputPath %s does not exist", inputPath)
	}

	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	inputFile, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("cannot open input file: %w", err)
	}
	defer inputFile.Close()

	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("cannot open output file: %w", err)
	}
	defer outputFile.Close()

	buffer := make([]byte, c.blockSize*1024)
	for {
		n, err := inputFile.Read(buffer)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}

		decrypted, err := c.Decrypt(buffer[:n])
		if err != nil {
			return err
		}

		if _, err := outputFile.Write(decrypted); err != nil {
			return err
		}
	}

	return nil
}

func (c *CipherContext) EncryptFileAsync(inputPath, outputPath string) (<-chan struct{}, <-chan error) {
	successChan := make(chan struct{}, 1)
	errorChan := make(chan error, 1)

	go func() {
		defer close(successChan)
		defer close(errorChan)

		if err := c.EncryptFile(inputPath, outputPath); err != nil {
			errorChan <- err
			return
		}
		successChan <- struct{}{}
	}()

	return successChan, errorChan
}

func (c *CipherContext) DecryptFileAsync(inputPath, outputPath string) (<-chan struct{}, <-chan error) {
	successChan := make(chan struct{}, 1)
	errorChan := make(chan error, 1)

	go func() {
		defer close(successChan)
		defer close(errorChan)

		if err := c.DecryptFile(inputPath, outputPath); err != nil {
			errorChan <- err
			return
		}
		successChan <- struct{}{}
	}()

	return successChan, errorChan
}

func test() {
}
