//go:build darwin
// +build darwin

package main

/*
#cgo CFLAGS: -I/usr/local/tdlib-1.8.19/include -I/opt/homebrew/opt/openssl@3/include
#cgo LDFLAGS: -L/usr/local/tdlib-1.8.19/lib -L/opt/homebrew/opt/openssl@3/lib -ltdjson -lstdc++ -lssl -lcrypto -lm -lz
*/
import "C"
