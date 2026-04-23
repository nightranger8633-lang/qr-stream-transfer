package main

import (
	"image"

	"github.com/skip2/go-qrcode"
)

func QRImage(payload string, size int) (image.Image, error) {
	if size < 256 {
		size = 256
	}
	qr, err := qrcode.New(payload, qrcode.Medium)
	if err != nil {
		return nil, err
	}
	return qr.Image(size), nil
}
