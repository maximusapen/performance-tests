/*******************************************************************************
 *
 * OCO Source Materials
 * , 5737-D43
 * (C) Copyright IBM Corp. 2020, 2023 All Rights Reserved.
 * The source code for this program is not  published or otherwise divested of
 * its trade secrets,  * irrespective of what has been deposited with
 * the U.S. Copyright Office.
 ******************************************************************************/

package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

// KeyEnvVar is the name of encryption key environment variable.
const KeyEnvVar = "STAGE_GLOBAL_ARMPERF_CRYPTOKEY"

// GenerateKey will generate a random 32 byte key
func GenerateKey() (string, error) {
	keyBytes := make([]byte, 32)
	_, err := rand.Read(keyBytes)
	if err != nil {
		return "", fmt.Errorf("Error generating ciper key : %s", err.Error())
	}

	key := fmt.Sprintf("%x", keyBytes)

	os.Setenv(KeyEnvVar, key)

	return key, nil
}

// Encrypt will encrypt the given input data with the armada_performance_crytpo_key
// The crypto key is stored in vault and must be set in the STAGE_GLOBAL_ARMPERF_CRYPTOKEY env var before calling this method
// INPUT: Text to encrypt
// OUTPUT: Hex encoded encrypted input data
func Encrypt(plaintext string) (string, error) {
	ck := os.Getenv(KeyEnvVar)
	if len(ck) == 0 {
		return "", fmt.Errorf("No cipher key specified. Please set \"%s\" environment variable", KeyEnvVar)
	}

	key, err := hex.DecodeString(ck)
	if err != nil {
		return "", fmt.Errorf("Malformed cipher key. Hexadecimal string expected : %s", err.Error())
	}

	cb, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("Failed to create cipher block : %s", err.Error())
	}

	aesgcm, err := cipher.NewGCM(cb)
	if err != nil {
		return "", fmt.Errorf("Failed to generate GCM block cipher : %s", err.Error())
	}

	iv := make([]byte, aesgcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", fmt.Errorf("Failed to generate initialization vector : %s", err.Error())
	}

	ciphertext := aesgcm.Seal(iv, iv, []byte(plaintext), nil)

	return hex.EncodeToString(ciphertext), nil
}

// Decrypt will decrypt the given input data with the STAGE_GLOBAL_ARMPERF_CRYPTOKEY
// The crypto key is stored in vault and must be set in the STAGE_GLOBAL_ARMPERF_CRYPTOKEY env var before calling this method
// INPUT: Hex encoded encrypted data
// OUTPUT: Decrypted text
func Decrypt(data string) (string, error) {
	ciphertext, err := hex.DecodeString(data)
	if err != nil {
		return "", fmt.Errorf("Invalid cipher text supplied : %s", err.Error())
	}

	ck := os.Getenv(KeyEnvVar)
	if len(ck) == 0 {
		return "", fmt.Errorf("No cipher key specified. Please set \"%s\" environment variable", KeyEnvVar)
	}

	key, err := hex.DecodeString(ck)
	if err != nil {
		return "", fmt.Errorf("Failed to process cipher key : %s", err.Error())
	}

	cb, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("Failed to create cipher block : %s", err.Error())
	}

	aesgcm, err := cipher.NewGCM(cb)
	if err != nil {
		return "", fmt.Errorf("Failed to generate GCM block cipher : %s", err.Error())
	}

	ivSize := aesgcm.NonceSize()
	if len(ciphertext) < ivSize {
		return "", fmt.Errorf("Missing initialisation vector in cipher text")
	}

	iv, ciphertext := ciphertext[:ivSize], ciphertext[ivSize:]
	plaintext, err := aesgcm.Open(nil, iv, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("Failed to decrypt cipher text : %s", err.Error())
	}

	return string(plaintext), nil
}
