package main

import (
	"bytes"
	"encoding/binary"
	"github.com/corona10/goimagehash"
	"github.com/nfnt/resize"
	"image"
	_ "image/jpeg"
	"os"
)

type PHash struct {
	hash [8]uint8
}

func PHashFromFile(filename string) (PHash, error) {
	f, err := os.Open(filename)
	if err != nil {
		return PHash{}, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if img == nil {
		return PHash{}, err
	}
	img = resize.Resize(64, 64, img, resize.Lanczos3)
	pHash, _ := goimagehash.PerceptionHash(img) // 由于保证img非空，所以pHash比不可能返回错误
	var h [8]uint8
	binary.BigEndian.PutUint64(h[:], pHash.GetHash())
	return PHash{hash: h}, nil
}

func byteToHexChar(b byte) byte {
	b &= 0xf
	if b < 10 {
		return b + '0'
	}
	return b + 'A' - 10
}

func (d PHash) String() string {
	buf := bytes.NewBuffer(make([]byte, 0, 16))
	for _, b := range d.hash {
		buf.WriteByte(byteToHexChar((b >> 4) & 0xf))
		buf.WriteByte(byteToHexChar(b & 0xf))
	}
	return buf.String()
}

func (d PHash) WriteToBuf(buf []byte) {
	_ = buf[7]
	for i := range d.hash {
		buf[i] = d.hash[i]
	}
}
