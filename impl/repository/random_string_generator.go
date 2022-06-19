package repository

import (
	"bytes"
	"encoding/hex"
	"io"
	"math/rand"
	"sync"
)

type RandStringGenerator struct {
	randMu         sync.Mutex
	rand           *rand.Rand
	byteLen        uint
	bufPool        sync.Pool
	encoderFactory func(r io.Writer) io.Writer
}

func NewRandStringGenerator(seed int64, byteLen uint, encoderFactory func(r io.Writer) io.Writer) *RandStringGenerator {
	if encoderFactory == nil {
		encoderFactory = hex.NewEncoder
	}
	return &RandStringGenerator{
		rand:    rand.New(rand.NewSource(seed)),
		byteLen: byteLen,
		bufPool: sync.Pool{
			New: func() any {
				buf := bytes.NewBuffer(make([]byte, 32))
				buf.Reset()
				return buf
			},
		},
		encoderFactory: encoderFactory,
	}
}

func (f *RandStringGenerator) Generate() (randomStr string, err error) {
	buf := f.bufPool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		f.bufPool.Put(buf)
	}()

	encoder := f.encoderFactory(buf)

	f.randMu.Lock()
	_, err = io.CopyN(encoder, f.rand, int64(f.byteLen))
	f.randMu.Unlock()

	if cl, ok := encoder.(io.Closer); ok {
		cl.Close()
	}

	if err != nil {
		return
	}
	return buf.String(), nil
}
