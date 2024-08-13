package cryptoutil_test

import (
	"fmt"
	"testing"

	"github.com/elisasre/go-common/v2/cryptoutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBase64decode(t *testing.T) {
	encoded := "U1VDQ0VTUw=="
	decoded, err := cryptoutil.Base64decode(encoded)
	assert.Nil(t, err)
	assert.Equal(t, "SUCCESS", decoded)

	failing := "^"
	_, err = cryptoutil.Base64decode(failing)
	assert.NotNil(t, err)
}

func TestDecrypt(t *testing.T) {
	// This test makes sure that we don't brake backwards compatibility
	// by modifying both; encrypt and decrypt at the same time.
	const passwd = "mypass"
	var (
		encryptedInput = []byte{
			0xf2, 0x3, 0x92, 0x1f, 0x9b, 0xb4, 0x56, 0xc,
			0x37, 0xb4, 0x33, 0x5f, 0x1, 0xad, 0xe3, 0x66,
			0x99, 0x14, 0x3e, 0x59, 0xc9, 0x19, 0xfe, 0x3b,
			0x6f, 0x34, 0xd2, 0xd9, 0x80, 0xe7, 0x1f, 0x2f,
			0xf2, 0x15, 0xb1, 0x4, 0x3e,
		}
		expectedOutput = []byte("some data")
	)

	data, err := cryptoutil.Decrypt(encryptedInput, passwd)
	require.NoError(t, err)
	require.Equal(t, expectedOutput, data)
}

func TestEncryptDecrypt(t *testing.T) {
	const (
		input  = "supersecret"
		passwd = "testpassword"
	)

	encrypted, err := cryptoutil.Encrypt([]byte(input), passwd)
	require.NoError(t, err)
	data, err := cryptoutil.Decrypt(encrypted, passwd)
	require.NoError(t, err)
	require.Equal(t, input, string(data))
}

func FuzzEncryptDecrypt(f *testing.F) {
	f.Add([]byte("some data"), "passwd")

	f.Fuzz(func(t *testing.T, input []byte, passwd string) {
		encrypted, err := cryptoutil.Encrypt(input, passwd)
		require.NoError(t, err)
		data, err := cryptoutil.Decrypt(encrypted, passwd)
		require.NoError(t, err)
		require.Equal(t, input, data)
	})
}

func ExampleEncrypt() {
	encrypted, _ := cryptoutil.Encrypt([]byte("supersecret"), "testpassword")
	data, _ := cryptoutil.Decrypt(encrypted, "testpassword")
	fmt.Println(string(data))
	// Output: supersecret
}

func ExampleDecrypt() {
	encrypted, _ := cryptoutil.Encrypt([]byte("supersecret"), "testpassword")
	data, _ := cryptoutil.Decrypt(encrypted, "testpassword")
	fmt.Println(string(data))
	// Output: supersecret
}
