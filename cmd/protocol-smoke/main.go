// Command protocol-smoke replays one captured first-flight frame against a
// running local server and decodes its first response with the real transport.
package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/local/9yin-go-server/internal/transport"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:19061", "server address")
	fixture := flag.String("fixture", `testdata\protocol\connect-first-flight.bin`, "captured encoded first-flight frame")
	flag.Parse()
	wire, err := os.ReadFile(*fixture)
	if err != nil {
		fatal(err)
	}
	conn, err := net.DialTimeout("tcp", *addr, 3*time.Second)
	if err != nil {
		fatal(err)
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		fatal(err)
	}
	if err := writeAll(conn, wire); err != nil {
		fatal(err)
	}
	reader, err := transport.NewFrameReader(conn, 1<<20)
	if err != nil {
		fatal(err)
	}
	response, err := reader.ReadFrame(transport.InitialKey)
	if err != nil {
		fatal(err)
	}
	if len(response) == 0 {
		fatal(fmt.Errorf("server returned an empty frame"))
	}
	fmt.Printf("opcode=0x%02X decoded_len=%d\n", response[0], len(response))
}

func writeAll(conn net.Conn, data []byte) error {
	for len(data) != 0 {
		n, err := conn.Write(data)
		if err != nil {
			return err
		}
		if n == 0 {
			return fmt.Errorf("zero-byte socket write")
		}
		data = data[n:]
	}
	return nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
