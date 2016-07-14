package crypt

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"os"

	"golang.org/x/crypto/scrypt"
)

const (
	encryptionSaltLen = 32
	encryptionPrefix  = "v0:"
)

// CheckEncryption peeks into the backup to see if it encrypted
// if it is, then we need to have the CRYPTO_PASSWORD env var
// or we cant restore it at all
func CheckEncryption(source string) (bool, error) {
	backupData, err := ioutil.ReadFile(source)
	if err != nil {
		return false, fmt.Errorf("[ERR] Unable to read backupfile: %v", err)
	}
	// try and peek in to see if we have an encrypted backup
	if bytes.HasPrefix(backupData, []byte(encryptionPrefix)) {
		return true, nil
	}
	return false, nil
}

// EncryptFile takes a file input and encrypts it with a passphrase
func EncryptFile(sourceFile string, passphrase string) error {
	source, err := ioutil.ReadFile(sourceFile)
	if err != nil {
		return fmt.Errorf("Unable to read backup file at %s to encrypt: %v", source, err)
	}

	salt := make([]byte, encryptionSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return fmt.Errorf("[ERR] Unable to generate salt for encryption: %v", err)
	}

	key, err := scrypt.Key([]byte(passphrase), salt, 16384, 8, 1, encryptionSaltLen)
	if err != nil {
		return fmt.Errorf("[ERR] Unable to generate scrypt key: %v", err)
	}

	aesCipher, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("[ERR] Unable to generate aes cipher: %v", err)
	}

	gcm, err := cipher.NewGCM(aesCipher)
	if err != nil {
		return fmt.Errorf("[ERR] Unable to create GCM: %v", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return fmt.Errorf("[ERR] Unable to generate nonce for encryption: %v", err)
	}

	sealedData := gcm.Seal(nil, nonce, source, nil)
	var ciphertext bytes.Buffer
	ciphertext.Write([]byte(encryptionPrefix))
	ciphertext.Write(salt)
	ciphertext.Write(nonce)
	ciphertext.Write(sealedData)
	sealedData = nil

	encryptedfile, err := os.OpenFile(sourceFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("[ERR] Unable to open file for encrypted write: %v", err)
	}
	defer encryptedfile.Close()

	_, err = encryptedfile.Write(ciphertext.Bytes())
	if err != nil {
		return fmt.Errorf("[ERR] Unable to write to encrypted file: %v", err)
	}
	return nil
}

// DecryptFile takes a file input and decrypts it with a passphrase
func DecryptFile(sourceFile string, passphrase string) error {
	ciphertext, err := ioutil.ReadFile(sourceFile)
	if err != nil {
		return fmt.Errorf("[ERR] Unable to read backupfile: %v", err)
	}
	ciphertext = ciphertext[len(encryptionPrefix):]
	salt := ciphertext[:encryptionSaltLen]
	ciphertext = ciphertext[encryptionSaltLen:]

	key, err := scrypt.Key([]byte(passphrase), salt, 16384, 8, 1, encryptionSaltLen)
	if err != nil {
		return fmt.Errorf("[ERR] Unable to generate scrypt key: %v", err)
	}

	aesCipher, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("[ERR] Unable to generate aes cipher: %v", err)
	}

	gcm, err := cipher.NewGCM(aesCipher)
	if err != nil {
		return fmt.Errorf("[ERR] Unable to create GCM: %v", err)
	}

	nonce := ciphertext[:gcm.NonceSize()]
	ciphertext = ciphertext[gcm.NonceSize():]

	output, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return fmt.Errorf("[ERR] Unable to decrypt data (possible bad CRYPTO_PASSWORD: %v", err)
	}

	if err := ioutil.WriteFile(sourceFile, output, os.FileMode(0644)); err != nil {
		return fmt.Errorf("Error decrypting file to %s: %v", sourceFile, err)
	}
	return nil
}
