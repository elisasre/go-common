package common

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBase64decode(t *testing.T) {
	encoded := "U1VDQ0VTUw=="
	decoded, err := Base64decode(encoded)
	assert.Nil(t, err)
	assert.Equal(t, "SUCCESS", decoded)

	failing := "^"
	_, err = Base64decode(failing)
	assert.NotNil(t, err)
}

func ExampleBase64decode() {
	out, _ := Base64decode("U1VDQ0VTUw==")
	fmt.Println(out)
	// Output: SUCCESS
}

func ExampleBase64encode() {
	fmt.Println(Base64encode([]byte("SUCCESS")))
	// Output: U1VDQ0VTUw==
}

func TestEncryptDecrypt(t *testing.T) {
	const (
		input  = "supersecret"
		passwd = "testpassword"
	)

	encrypted, err := Encrypt([]byte(input), passwd)
	require.NoError(t, err)
	data, err := Decrypt(encrypted, passwd)
	require.NoError(t, err)
	require.Equal(t, input, string(data))
}

func FuzzEncryptDecrypt(f *testing.F) {
	f.Add([]byte("some data"), "passwd")

	f.Fuzz(func(t *testing.T, input []byte, passwd string) {
		encrypted, err := Encrypt(input, passwd)
		require.NoError(t, err)
		data, err := Decrypt(encrypted, passwd)
		require.NoError(t, err)
		require.Equal(t, input, data)
	})
}

func ExampleEncrypt() {
	encrypted, _ := Encrypt([]byte("supersecret"), "testpassword")
	data, _ := Decrypt(encrypted, "testpassword")
	fmt.Println(string(data))
	// Output: supersecret
}

func ExampleDecrypt() {
	encrypted, _ := Encrypt([]byte("supersecret"), "testpassword")
	data, _ := Decrypt(encrypted, "testpassword")
	fmt.Println(string(data))
	// Output: supersecret
}
