package goutil

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
)

// Crypto Crypto
type Crypto struct {
	key string
}

// GetCrypto Get Crypto
func GetCrypto(key string) Crypto {
	return Crypto{key: key}
}

// MD5 get md5
func (o *Crypto) MD5(value string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(value)))
}

// SHA1 get sha1
func (o *Crypto) SHA1(value string) string {
	return fmt.Sprintf("%x", sha1.Sum([]byte(value)))
}

// EncryptAES Decrypt AES algorithm
func (o *Crypto) EncryptAES(message string) (encmess string) {
	plainText := []byte(message)
	//The byte data type represents ASCII characters and the rune data type represents a more broader set of Unicode characters that are encoded in UTF-8 format.

	block, err := aes.NewCipher([]byte(o.key))
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

// DecryptAES Decrypt AES algorithm
func (o *Crypto) DecryptAES(securemess string) (decodedmess string) {
	if securemess == "" {
		return
	}

	cipherText, err := base64.URLEncoding.DecodeString(securemess)
	if err != nil {
		return
	}

	block, err := aes.NewCipher([]byte(o.key))
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
