package server

import (
	"compress/gzip"
	"io"
	"sync"
)

const (
	DEFAULT_COMPRESSION_LEVEL = gzip.DefaultCompression
	DEFAULT_BUFFER_SIZE = 1024
)

type Compressor struct {
	level int // Compression level (e.g., gzip.BestSpeed, gzip.DefaultCompression)
	mu sync.Mutex
	buf []byte
}

func NewCompressor(level int) *Compressor {
	return &Compressor{
		level: level,
		buf: make([]byte, DEFAULT_BUFFER_SIZE),
	}
}


func (c *Compressor) Compress(data []byte, w io.Writer) error {
	c.mu.Lock()
	defer c.mu.Unlock()

    gzipWriter, err := gzip.NewWriterLevel(w, c.level)
	if err != nil {
		return  err
	}

	_, err = gzipWriter.Write(data)
	if err != nil {
		return err
	}

	return  nil
}


func (c *Compressor) Decompress(r io.Reader) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

    gzipReader, err := gzip.NewReader(r)
    if err != nil {
        return nil, err
    }
    defer gzipReader.Close()

    decompressedData, err := io.ReadAll(gzipReader)
    if err != nil {
        return nil, err
    }

    return decompressedData, nil
}
