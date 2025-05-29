package symmetric

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func (c *CipherContext) EncryptFile(ctx context.Context, inputPath, outputPath string, progress func(done, total int)) error {
	inputInfo, err := os.Stat(inputPath)
	if err != nil {
		return fmt.Errorf("failed to stat input file: %w", err)
	}
	if inputInfo.IsDir() {
		return fmt.Errorf("inputPath %s is a directory", inputPath)
	}

	fileSize := inputInfo.Size()
	chunkSize := int64(c.blockSize * 1024)
	totalChunks := int((fileSize + chunkSize - 1) / chunkSize)

	inputFile, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("cannot open input file: %w", err)
	}
	defer inputFile.Close()

	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("cannot open output file: %w", err)
	}
	defer outputFile.Close()

	buffer := make([]byte, chunkSize)
	chunkIndex := 0

	for {
		select {
		case <-ctx.Done():
			_ = os.Remove(outputPath)
			return ctx.Err()
		default:
		}

		n, err := inputFile.Read(buffer)
		if err != nil && err != io.EOF {
			return fmt.Errorf("read error: %w", err)
		}
		if n == 0 {
			break
		}

		encrypted, err := c.Encrypt(buffer[:n], chunkIndex, totalChunks)
		if err != nil {
			return fmt.Errorf("encrypt chunk %d failed: %w", chunkIndex, err)
		}

		if _, err := outputFile.Write(encrypted); err != nil {
			return fmt.Errorf("write error: %w", err)
		}

		chunkIndex++
		progress(chunkIndex, totalChunks)
	}

	return nil
}

func (c *CipherContext) DecryptFile(inputPath, outputPath string, progress func(done, total int)) error {
	inputInfo, err := os.Stat(inputPath)
	if err != nil {
		return fmt.Errorf("failed to stat input file: %w", err)
	}
	if inputInfo.IsDir() {
		return fmt.Errorf("inputPath %s is a directory", inputPath)
	}

	// Вычисляем общее количество чанков
	fileSize := inputInfo.Size()
	chunkSize := int64(c.blockSize * 1024)
	totalChunks := int((fileSize + chunkSize - 1) / chunkSize)

	inputFile, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("cannot open input file: %w", err)
	}
	defer inputFile.Close()

	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("cannot open output file: %w", err)
	}
	defer outputFile.Close()

	buffer := make([]byte, chunkSize)
	chunkIndex := 0

	for {
		n, err := inputFile.Read(buffer)
		if err != nil && err != io.EOF {
			return fmt.Errorf("read error: %w", err)
		}
		if n == 0 {
			break
		}

		decrypted, err := c.Decrypt(buffer[:n], chunkIndex, totalChunks)
		if err != nil {
			return fmt.Errorf("decrypt chunk %d failed: %w", chunkIndex, err)
		}

		if _, err = outputFile.Write(decrypted); err != nil {
			return fmt.Errorf("write error: %w", err)
		}

		chunkIndex++
		progress(chunkIndex, totalChunks)
	}

	return nil
}

func (c *CipherContext) EncryptFileAsync(ctx context.Context, inputPath, outputPath string, progress func(done, total int)) (<-chan struct{}, <-chan error) {
	successChan := make(chan struct{}, 1)
	errorChan := make(chan error, 1)

	go func() {
		defer close(successChan)
		defer close(errorChan)

		if err := c.EncryptFile(ctx, inputPath, outputPath, progress); err != nil {
			errorChan <- err
			return
		}
		successChan <- struct{}{}
	}()

	return successChan, errorChan
}

func (c *CipherContext) DecryptFileAsync(inputPath, outputPath string, progress func(done, total int)) (<-chan struct{}, <-chan error) {
	successChan := make(chan struct{}, 1)
	errorChan := make(chan error, 1)

	go func() {
		defer close(successChan)
		defer close(errorChan)

		if err := c.DecryptFile(inputPath, outputPath, progress); err != nil {
			errorChan <- err
			return
		}
		successChan <- struct{}{}
	}()

	return successChan, errorChan
}

func test() {
}
