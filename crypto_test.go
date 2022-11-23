package common

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
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

func ExampleEncrypt() {
	encrypted := Encrypt([]byte("supersecret"), "testpassword")
	fmt.Println(string(Decrypt(encrypted, "testpassword")))
	// Output: supersecret
}

func ExampleDecrypt() {
	encrypted := Encrypt([]byte("supersecret"), "testpassword")
	fmt.Println(string(Decrypt(encrypted, "testpassword")))
	// Output: supersecret
}
