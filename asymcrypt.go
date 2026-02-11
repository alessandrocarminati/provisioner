package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io/ioutil"
	"os"
)

const rsa_block_size = 117

func GenerateKeyPair(PrivateFN string, PublicFN string) error {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return err
	}

	publicKey := &privateKey.PublicKey

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	})
	err = os.WriteFile(PrivateFN, privateKeyPEM, 0644)
	if err != nil {
		return err
	}

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return err
	}
	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicKeyBytes,
	})
	err = os.WriteFile(PublicFN, publicKeyPEM, 0644)
	if err != nil {
		return err
	}
	return nil
}

func EncryptConfig(ConfigFN string, PubkeyFN string) ([]byte, error) {
	publicKeyPEM, err := ioutil.ReadFile(PubkeyFN)
	if err != nil {
		return nil, err
	}
	publicKeyBlock, _ := pem.Decode(publicKeyPEM)
	if publicKeyBlock == nil {
		return nil, errors.New("failed to decode public key PEM: no block found")
	}
	publicKey, err := x509.ParsePKIXPublicKey(publicKeyBlock.Bytes)
	if err != nil {
		return nil, err
	}

	content, err := ioutil.ReadFile(ConfigFN)
	if err != nil {
		return nil, err
	}

	ciphertext := make([]byte, 0, len(content))
	for i := 0; i < len(content); i += rsa_block_size {
		if i+rsa_block_size < len(content) {
			partial, err1 := rsa.EncryptPKCS1v15(rand.Reader, publicKey.(*rsa.PublicKey), content[i:i+rsa_block_size])
			if err1 != nil {
				return nil, err1
			}
			ciphertext = append(ciphertext, partial...)
		} else {
			partial, err1 := rsa.EncryptPKCS1v15(rand.Reader, publicKey.(*rsa.PublicKey), content[i:])
			if err1 != nil {
				return nil, err1
			}
			ciphertext = append(ciphertext, partial...)
		}
	}

	return ciphertext, nil
}
func DecryptConfig(ConfigFN string, PivkeyFN string) ([]byte, error) {

	privateKeyPEM, err := ioutil.ReadFile(PivkeyFN)
	if err != nil {
		return nil, err
	}

	privateKeyBlock, _ := pem.Decode(privateKeyPEM)
	if privateKeyBlock == nil {
		return nil, errors.New("failed to decode private key PEM: no block found")
	}
	privateKey, err := x509.ParsePKCS1PrivateKey(privateKeyBlock.Bytes)
	if err != nil {
		return nil, err
	}

	content, err := ioutil.ReadFile(ConfigFN)
	if err != nil {
		return nil, err
	}

	plaintext := make([]byte, 0, len(content))
	for i := 0; i < len(content); i += privateKey.Size() {
		if i+privateKey.Size() < len(content) {
			partial, err1 := rsa.DecryptPKCS1v15(rand.Reader, privateKey, content[i:i+privateKey.Size()])
			if err1 != nil {
				return nil, err1
			}
			plaintext = append(plaintext, partial...)
		} else {
			partial, err1 := rsa.DecryptPKCS1v15(rand.Reader, privateKey, content[i:])
			if err1 != nil {
				return nil, err1
			}
			plaintext = append(plaintext, partial...)
		}
	}

	return plaintext, nil
}

func WriteFile(FN string, content []byte) error {
	err := ioutil.WriteFile(FN, content, 0644)
	if err != nil {
		return err
	}
	return nil
}
