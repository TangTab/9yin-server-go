package auth

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/local/9yin-go-server/internal/transport"
)

const capturedIdentity = "acct:v1:112cd36f1e1d2bac408e0428aabef6e399080c2bc33407de6677e60dec97c2a6"

func capturedLogin(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "protocol", "connect-first-flight.bin"))
	if err != nil {
		t.Fatal(err)
	}
	reader, err := transport.NewFrameReader(bytes.NewReader(raw), 1024)
	if err != nil {
		t.Fatal(err)
	}
	message, err := reader.ReadFrame(transport.InitialKey)
	if err != nil {
		t.Fatal(err)
	}
	return message
}

func TestCapturedLoginIdentity(t *testing.T) {
	identity, err := ParseLoginAccountIdentity(capturedLogin(t))
	if err != nil {
		t.Fatal(err)
	}
	if got := identity.String(); got != capturedIdentity {
		t.Fatalf("identity = %s", got)
	}
	if got := identity.Prefix(); got != capturedIdentity[:len("acct:v1:")+12] {
		t.Fatalf("prefix = %s", got)
	}
}

func TestPasswordMutationDoesNotChangeIdentity(t *testing.T) {
	message := capturedLogin(t)
	accountLength := int(binary.LittleEndian.Uint32(message[0x15:0x19]))
	passwordLengthOffset := 0x19 + accountLength
	passwordLength := int(binary.LittleEndian.Uint32(message[passwordLengthOffset : passwordLengthOffset+4]))
	passwordStart := passwordLengthOffset + 4
	mutated := append([]byte(nil), message...)
	for index := passwordStart; index < passwordStart+passwordLength; index++ {
		mutated[index] ^= 0x5A
	}
	first, err := ParseLoginAccountIdentity(message)
	if err != nil {
		t.Fatal(err)
	}
	second, err := ParseLoginAccountIdentity(mutated)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatal("password material changed the account identity")
	}
}

func TestAccountMutationChangesIdentity(t *testing.T) {
	message := capturedLogin(t)
	mutated := append([]byte(nil), message...)
	mutated[0x19] ^= 1
	first, _ := ParseLoginAccountIdentity(message)
	second, err := ParseLoginAccountIdentity(mutated)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("account slot mutation did not change identity")
	}
}

func TestMalformedLoginMessagesAreRejected(t *testing.T) {
	valid := capturedLogin(t)
	tests := [][]byte{
		nil,
		{0x03},
		valid[:24],
		append([]byte(nil), valid[:40]...),
		append(append([]byte(nil), valid...), 0),
	}
	badAccountLength := append([]byte(nil), valid...)
	binary.LittleEndian.PutUint32(badAccountLength[0x15:0x19], 15)
	tests = append(tests, badAccountLength)
	for index, message := range tests {
		if _, err := ParseLoginAccountIdentity(message); !errors.Is(err, ErrInvalidLogin) {
			t.Fatalf("case %d error = %v", index, err)
		}
	}
}

func FuzzParseLoginAccountIdentity(f *testing.F) {
	f.Add([]byte{0x02})
	f.Fuzz(func(t *testing.T, message []byte) {
		_, _ = ParseLoginAccountIdentity(message)
	})
}
