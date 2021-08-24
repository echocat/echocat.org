package main

import (
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"io"
)

type sha256reader struct {
	delegate io.Reader
	hash     hash.Hash
}

func newSha256reader(delegate io.Reader) *sha256reader {
	return &sha256reader{
		delegate: delegate,
		hash:     sha256.New(),
	}
}

func (instance *sha256reader) Read(p []byte) (n int, err error) {
	n, err = instance.delegate.Read(p)
	if err != nil || n <= 0 {
		return
	}
	_, err = instance.hash.Write(p[:n])
	return
}

func (instance *sha256reader) Sum() []byte {
	return instance.hash.Sum(nil)
}

func (instance *sha256reader) SumString() string {
	return hex.EncodeToString(instance.Sum())
}
