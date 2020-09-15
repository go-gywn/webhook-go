package common

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strconv"
)

// PanicIf panic if error
func PanicIf(err error) {
	if err != nil {
		panic(err)
	}
}

// ParseInt string -> int
func ParseInt(value string) int {
	if value == "" {
		return 0
	}
	val, _ := strconv.Atoi(value)
	return val
}

// IntString int -> string
func IntString(value int) string {
	return strconv.Itoa(value)
}

// MD5 get md5
func MD5(value string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(value)))
}

// SHA1 get sha1
func SHA1(value string) string {
	return fmt.Sprintf("%x", sha1.Sum([]byte(value)))
}

// Encrypt encrypt
func Encrypt(message string) (encmess string, err error) {
	return encrypt(Cfg.Key, message)
}

// Decrypt decrypt
func Decrypt(securemess string) (decodedmess string, err error) {
	return decrypt(Cfg.Key, securemess)
}

// Encrypt encrypt
func encrypt(key string, message string) (encmess string, err error) {
	plainText := []byte(message)
	//The byte data type represents ASCII characters and the rune data type represents a more broader set of Unicode characters that are encoded in UTF-8 format.

	block, err := aes.NewCipher([]byte(key))
	//NewCipher creates and returns a new cipher.Block. The key argument should be the AES key
	if err != nil {
		return
	}

	//IV needs to be unique, but doesn't have to be secure.
	//It's common to put it at the beginning of the ciphertext.
	cipherText := make([]byte, aes.BlockSize+len(plainText)) //make([]자료형, 길이)
	iv := cipherText[:aes.BlockSize]
	if _, err = io.ReadFull(rand.Reader, iv); err != nil {
		return
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(cipherText[aes.BlockSize:], plainText)

	//returns to base64 encoded string
	encmess = base64.URLEncoding.EncodeToString(cipherText)
	return
}

// Decrypt dncrypt
func decrypt(key string, securemess string) (decodedmess string, err error) {
	if securemess == "" {
		return
	}

	cipherText, err := base64.URLEncoding.DecodeString(securemess)
	if err != nil {
		return
	}

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return
	}

	if len(cipherText) < aes.BlockSize {
		err = errors.New("Ciphertext block size is too short")
		return
	}

	//IV needs to be unique, but doesn't have to be secure.
	//It's common to put it at the beginning of the ciphertext.
	iv := cipherText[:aes.BlockSize]
	cipherText = cipherText[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	// XORKeyStream can work in-place if the two arguments are the same.
	stream.XORKeyStream(cipherText, cipherText)

	decodedmess = string(cipherText)
	return
}
