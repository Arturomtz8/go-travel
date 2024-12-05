package utils

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
)

func CompressData(data []byte) ([]byte, error) {
	var compressedBuffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&compressedBuffer)

	_, err := gzipWriter.Write(data)
	if err != nil {
		return nil, fmt.Errorf("failed to write data to gzip writer: %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %v", err)
	}
	return compressedBuffer.Bytes(), nil
}

func DecompressData(compressedData []byte) ([]byte, error) {
	compressedBuffer := bytes.NewBuffer(compressedData)
	gzipReader, err := gzip.NewReader(compressedBuffer)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %v", err)
	}
	defer gzipReader.Close()
	decompressed, err := io.ReadAll(gzipReader)
	if err != nil {
		return nil, fmt.Errorf("failed to read decompressed data: %v", err)
	}

	return decompressed, nil
}
