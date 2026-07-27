package protocol

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
)

func TestMessageRoundTrip(t *testing.T) {
	input := Request{
		JSONRPC: "2.0",
		ID:      7,
		Method:  "initialize",
	}
	var framed bytes.Buffer
	if err := WriteMessage(&framed, input); err != nil {
		t.Fatalf("WriteMessage() error = %v", err)
	}
	var output Request
	if err := ReadMessage(bufio.NewReader(&framed), &output); err != nil {
		t.Fatalf("ReadMessage() error = %v", err)
	}
	if output.JSONRPC != input.JSONRPC ||
		output.ID != input.ID ||
		output.Method != input.Method ||
		len(output.Params) != 0 {
		t.Fatalf("output = %#v, want %#v", output, input)
	}
}

func TestReadMessageRejectsInvalidFraming(t *testing.T) {
	for _, input := range []string{
		"\r\n{}",
		"Other: 2\r\n\r\n{}",
		"Content-Length: nope\r\n\r\n",
		"Content-Length: 2\r\nContent-Length: 2\r\n\r\n{}",
		"Content-Length: 4\r\n\r\n{}",
	} {
		var output Request
		err := ReadMessage(bufio.NewReader(strings.NewReader(input)), &output)
		if err == nil {
			t.Fatalf("ReadMessage(%q) error = nil", input)
		}
	}
}

func TestWriteMessageRejectsNilWriter(t *testing.T) {
	if err := WriteMessage(nil, Request{}); err == nil {
		t.Fatal("WriteMessage(nil) error = nil")
	}
}

func FuzzReadMessage(f *testing.F) {
	f.Add([]byte("Content-Length: 2\r\n\r\n{}"))
	f.Add([]byte("Content-Length: 0\r\n\r\n"))
	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) > 1<<20 {
			input = input[:1<<20]
		}
		var output Request
		if err := ReadMessage(
			bufio.NewReader(bytes.NewReader(input)),
			&output,
		); err != nil {
			return
		}
	})
}
