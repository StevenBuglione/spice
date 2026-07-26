package mail

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	netmail "net/mail"
	"strings"
	"testing"
	"time"
)

var mailTestDate = time.Date(2026, time.July, 26, 18, 30, 0, 0, time.UTC)

func TestMessageSerializesDeterministicMultipartWithoutBcc(t *testing.T) {
	t.Parallel()

	spec := validMessageSpec()
	first, err := NewMessage(spec)
	if err != nil {
		t.Fatalf("NewMessage() error = %v", err)
	}
	second, err := NewMessage(spec)
	if err != nil {
		t.Fatalf("second NewMessage() error = %v", err)
	}
	if !bytes.Equal(first.Bytes(), second.Bytes()) {
		t.Fatal("message serialization is not deterministic")
	}
	if first.ID() != "order-41@example.com" ||
		first.EnvelopeFrom() != "orders@example.com" {
		t.Fatalf("message identity = %q, from = %q", first.ID(), first.EnvelopeFrom())
	}
	if recipients := first.Recipients(); !equalStrings(recipients, []string{
		"alice@example.com",
		"ops@example.com",
		"audit@example.com",
	}) {
		t.Fatalf("recipients = %v", recipients)
	}
	recipients := first.Recipients()
	recipients[0] = "changed@example.com"
	content := first.Bytes()
	content[0] = 'X'
	if first.Recipients()[0] == "changed@example.com" ||
		first.Bytes()[0] == 'X' {
		t.Fatal("message accessors exposed mutable state")
	}

	parsed, err := netmail.ReadMessage(bytes.NewReader(first.Bytes()))
	if err != nil {
		t.Fatalf("ReadMessage() error = %v", err)
	}
	if parsed.Header.Get("Bcc") != "" ||
		!strings.Contains(parsed.Header.Get("To"), "alice@example.com") ||
		!strings.Contains(parsed.Header.Get("Cc"), "ops@example.com") ||
		parsed.Header.Get("Message-ID") != "<order-41@example.com>" {
		t.Fatalf("message headers = %#v", parsed.Header)
	}
	subject, err := new(mime.WordDecoder).DecodeHeader(parsed.Header.Get("Subject"))
	if err != nil || subject != "Order 41 is ready ✓" {
		t.Fatalf("decoded subject = %q, %v", subject, err)
	}
	mediaType, parameters, err := mime.ParseMediaType(parsed.Header.Get("Content-Type"))
	if err != nil || mediaType != "multipart/mixed" {
		t.Fatalf("content type = %q, %v", parsed.Header.Get("Content-Type"), err)
	}
	mixed := multipart.NewReader(parsed.Body, parameters["boundary"])
	bodyPart, err := mixed.NextPart()
	if err != nil {
		t.Fatalf("read mixed body part: %v", err)
	}
	alternativeType, alternativeParameters, err := mime.ParseMediaType(
		bodyPart.Header.Get("Content-Type"),
	)
	if err != nil || alternativeType != "multipart/alternative" {
		t.Fatalf("body content type = %q, %v", bodyPart.Header.Get("Content-Type"), err)
	}
	alternative := multipart.NewReader(bodyPart, alternativeParameters["boundary"])
	text := readMailPart(t, alternative, "text/plain")
	html := readMailPart(t, alternative, "text/html")
	if text != "Order 41\r\nis ready." ||
		html != "<h1>Order 41</h1>\r\n<p>Ready.</p>" {
		t.Fatalf("decoded bodies: text=%q html=%q", text, html)
	}
	if _, nextErr := alternative.NextPart(); !errors.Is(nextErr, io.EOF) {
		t.Fatalf("extra alternative part error = %v", nextErr)
	}

	attachmentPart, err := mixed.NextPart()
	if err != nil {
		t.Fatalf("read attachment: %v", err)
	}
	disposition, dispositionParameters, err := mime.ParseMediaType(
		attachmentPart.Header.Get("Content-Disposition"),
	)
	if err != nil ||
		disposition != "attachment" ||
		dispositionParameters["filename"] != "order-41.txt" {
		t.Fatalf(
			"attachment disposition = %q %#v, %v",
			disposition,
			dispositionParameters,
			err,
		)
	}
	decoded, err := io.ReadAll(base64.NewDecoder(base64.StdEncoding, attachmentPart))
	if err != nil {
		t.Fatalf("decode attachment: %v", err)
	}
	if string(decoded) != "attachment contents" {
		t.Fatalf("attachment = %q", decoded)
	}
	if _, err := mixed.NextPart(); !errors.Is(err, io.EOF) {
		t.Fatalf("extra mixed part error = %v", err)
	}
	assertCRLFAndBase64Lines(t, first.Bytes())
}

func TestMessageSupportsTextHTMLAndAlternativeBodies(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		text      string
		html      string
		mediaType string
	}{
		{name: "text", text: "hello", mediaType: "text/plain"},
		{name: "HTML", html: "<p>hello</p>", mediaType: "text/html"},
		{
			name:      "alternative",
			text:      "hello",
			html:      "<p>hello</p>",
			mediaType: "multipart/alternative",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			spec := validMessageSpec()
			spec.TextBody = test.text
			spec.HTMLBody = test.html
			spec.Attachments = nil
			message, err := NewMessage(spec)
			if err != nil {
				t.Fatalf("NewMessage() error = %v", err)
			}
			parsed, err := netmail.ReadMessage(bytes.NewReader(message.Bytes()))
			if err != nil {
				t.Fatalf("ReadMessage() error = %v", err)
			}
			mediaType, _, err := mime.ParseMediaType(parsed.Header.Get("Content-Type"))
			if err != nil || mediaType != test.mediaType {
				t.Fatalf("content type = %q, %v", mediaType, err)
			}
		})
	}
}

func TestMessageDeduplicatesEnvelopeRecipientsInStableOrder(t *testing.T) {
	t.Parallel()

	spec := validMessageSpec()
	spec.To = []string{"first@example.com", "duplicate@example.com"}
	spec.Cc = []string{"duplicate@example.com", "second@example.com"}
	spec.Bcc = []string{"first@example.com", "third@example.com"}
	message, err := NewMessage(spec)
	if err != nil {
		t.Fatalf("NewMessage() error = %v", err)
	}
	if recipients := message.Recipients(); !equalStrings(recipients, []string{
		"first@example.com",
		"duplicate@example.com",
		"second@example.com",
		"third@example.com",
	}) {
		t.Fatalf("recipients = %v", recipients)
	}
}

func TestMessageCopiesInputsAndAcceptsExactBounds(t *testing.T) {
	spec := validMessageSpec()
	spec.Subject = strings.Repeat("s", maxSubjectBytes)
	spec.TextBody = strings.Repeat("t", maxBodyBytes)
	spec.HTMLBody = ""
	spec.To = make([]string, maxRecipients)
	for index := range spec.To {
		spec.To[index] = fmt.Sprintf("recipient-%03d@example.com", index)
	}
	spec.Cc = nil
	spec.Bcc = nil
	spec.Attachments = []AttachmentSpec{
		{Filename: "maximum.bin", Data: make([]byte, maxAttachmentBytes)},
		{Filename: "aggregate.bin", Data: make([]byte, maxAttachmentTotal-maxAttachmentBytes)},
	}
	spec.Attachments[0].Data[0] = 1

	message, err := NewMessage(spec)
	if err != nil {
		t.Fatalf("NewMessage() at exact bounds error = %v", err)
	}
	original := message.Bytes()
	spec.To[0] = "changed@example.com"
	spec.Attachments[0].Data[0] = 2
	if message.Recipients()[0] != "recipient-000@example.com" {
		t.Fatal("message retained the caller-owned recipient slice")
	}
	if !bytes.Equal(message.Bytes(), original) {
		t.Fatal("message retained caller-owned attachment data")
	}
	if len(original) > maxSerializedMailBytes {
		t.Fatalf("serialized message size = %d", len(original))
	}
}

func TestMessageRejectsSerializedOverflow(t *testing.T) {
	spec := validMessageSpec()
	spec.TextBody = strings.Repeat("\x00", maxBodyBytes)
	spec.HTMLBody = strings.Repeat("\x00", maxBodyBytes)
	spec.Attachments = []AttachmentSpec{
		{Filename: "maximum.bin", Data: make([]byte, maxAttachmentBytes)},
		{Filename: "aggregate.bin", Data: make([]byte, maxAttachmentTotal-maxAttachmentBytes)},
	}

	_, err := NewMessage(spec)
	if err == nil || !strings.Contains(err.Error(), "serialized message exceeds") {
		t.Fatalf("NewMessage() error = %v", err)
	}
}

func TestMessageRejectsInvalidContracts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*MessageSpec)
	}{
		{name: "empty ID", mutate: func(spec *MessageSpec) { spec.ID = "" }},
		{name: "spaced ID", mutate: func(spec *MessageSpec) { spec.ID = " id@example.com" }},
		{name: "missing ID domain", mutate: func(spec *MessageSpec) { spec.ID = "id@" }},
		{name: "multiple ID separators", mutate: func(spec *MessageSpec) { spec.ID = "id@@example.com" }},
		{name: "ID brackets", mutate: func(spec *MessageSpec) { spec.ID = "<id@example.com>" }},
		{name: "Unicode ID", mutate: func(spec *MessageSpec) { spec.ID = "idé@example.com" }},
		{name: "long ID", mutate: func(spec *MessageSpec) {
			spec.ID = strings.Repeat("x", maxMessageIDBytes) + "@example.com"
		}},
		{name: "zero date", mutate: func(spec *MessageSpec) { spec.Date = time.Time{} }},
		{name: "empty from", mutate: func(spec *MessageSpec) { spec.From = "" }},
		{name: "invalid from", mutate: func(spec *MessageSpec) { spec.From = "not-an-address" }},
		{name: "Unicode envelope", mutate: func(spec *MessageSpec) {
			spec.From = "sender@éxample.com"
		}},
		{name: "from injection", mutate: func(spec *MessageSpec) {
			spec.From = "sender@example.com\r\nBcc: stolen@example.com"
		}},
		{name: "no recipients", mutate: func(spec *MessageSpec) {
			spec.To, spec.Cc, spec.Bcc = nil, nil, nil
		}},
		{name: "too many recipients", mutate: func(spec *MessageSpec) {
			spec.To = make([]string, maxRecipients+1)
			for index := range spec.To {
				spec.To[index] = fmt.Sprintf("recipient-%d@example.com", index)
			}
		}},
		{name: "invalid to", mutate: func(spec *MessageSpec) { spec.To = []string{"invalid"} }},
		{name: "invalid cc", mutate: func(spec *MessageSpec) { spec.Cc = []string{"invalid"} }},
		{name: "invalid bcc", mutate: func(spec *MessageSpec) { spec.Bcc = []string{"invalid"} }},
		{name: "invalid reply-to", mutate: func(spec *MessageSpec) { spec.ReplyTo = "invalid" }},
		{name: "empty subject", mutate: func(spec *MessageSpec) { spec.Subject = "" }},
		{name: "spaced subject", mutate: func(spec *MessageSpec) { spec.Subject = " subject" }},
		{name: "subject injection", mutate: func(spec *MessageSpec) {
			spec.Subject = "subject\r\nBcc: stolen@example.com"
		}},
		{name: "long subject", mutate: func(spec *MessageSpec) {
			spec.Subject = strings.Repeat("x", maxSubjectBytes+1)
		}},
		{name: "no body", mutate: func(spec *MessageSpec) {
			spec.TextBody, spec.HTMLBody = "", ""
		}},
		{name: "long text body", mutate: func(spec *MessageSpec) {
			spec.TextBody = strings.Repeat("x", maxBodyBytes+1)
		}},
		{name: "long HTML body", mutate: func(spec *MessageSpec) {
			spec.HTMLBody = strings.Repeat("x", maxBodyBytes+1)
		}},
		{name: "too many attachments", mutate: func(spec *MessageSpec) {
			spec.Attachments = make([]AttachmentSpec, maxAttachments+1)
		}},
		{name: "empty attachment filename", mutate: func(spec *MessageSpec) {
			spec.Attachments[0].Filename = ""
		}},
		{name: "attachment path", mutate: func(spec *MessageSpec) {
			spec.Attachments[0].Filename = "../secret.txt"
		}},
		{name: "attachment backslash", mutate: func(spec *MessageSpec) {
			spec.Attachments[0].Filename = `directory\secret.txt`
		}},
		{name: "attachment filename injection", mutate: func(spec *MessageSpec) {
			spec.Attachments[0].Filename = "file.txt\r\nBcc"
		}},
		{name: "long attachment filename", mutate: func(spec *MessageSpec) {
			spec.Attachments[0].Filename = strings.Repeat("x", maxFilenameBytes+1)
		}},
		{name: "empty attachment", mutate: func(spec *MessageSpec) {
			spec.Attachments[0].Data = nil
		}},
		{name: "large attachment", mutate: func(spec *MessageSpec) {
			spec.Attachments[0].Data = make([]byte, maxAttachmentBytes+1)
		}},
		{name: "attachment total", mutate: func(spec *MessageSpec) {
			spec.Attachments = []AttachmentSpec{
				{Filename: "one.bin", Data: make([]byte, 9<<20)},
				{Filename: "two.bin", Data: make([]byte, 9<<20)},
			}
		}},
		{name: "invalid attachment media type", mutate: func(spec *MessageSpec) {
			spec.Attachments[0].ContentType = "invalid"
		}},
		{name: "multipart attachment", mutate: func(spec *MessageSpec) {
			spec.Attachments[0].ContentType = "multipart/mixed"
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			spec := validMessageSpec()
			test.mutate(&spec)
			if _, err := NewMessage(spec); err == nil {
				t.Fatal("NewMessage() unexpectedly succeeded")
			}
		})
	}
}

func TestMessageBoundaryAvoidsCallerContent(t *testing.T) {
	t.Parallel()

	spec := validMessageSpec()
	normalized, err := normalizeMessage(spec)
	if err != nil {
		t.Fatalf("normalize message: %v", err)
	}
	first := messageBoundary(normalized, "mixed")
	spec.TextBody += first
	normalized, err = normalizeMessage(spec)
	if err != nil {
		t.Fatalf("normalize message with boundary: %v", err)
	}
	if second := messageBoundary(normalized, "mixed"); second == first {
		t.Fatal("boundary collided with caller content")
	}
}

func FuzzNewMessage(f *testing.F) {
	f.Add("subject", "text", "<p>HTML</p>", "attachment.txt")
	f.Add("✓", "\x00\r\n", "<script>", "../file")
	f.Fuzz(func(
		t *testing.T,
		subject string,
		textBody string,
		htmlBody string,
		filename string,
	) {
		spec := validMessageSpec()
		spec.Subject = subject
		spec.TextBody = textBody
		spec.HTMLBody = htmlBody
		spec.Attachments[0].Filename = filename
		if _, constructionErr := NewMessage(spec); constructionErr != nil {
			return
		}
	})
}

func ExampleMessage() {
	message, err := NewMessage(MessageSpec{
		ID:       "order-41@example.com",
		Date:     mailTestDate,
		From:     "Orders <orders@example.com>",
		To:       []string{"customer@example.com"},
		Subject:  "Order 41 is ready",
		TextBody: "Your order is ready.",
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(message.ID(), message.Recipients())
	// Output: order-41@example.com [customer@example.com]
}

func BenchmarkNewMessage(b *testing.B) {
	spec := validMessageSpec()
	b.ReportAllocs()
	for b.Loop() {
		if _, err := NewMessage(spec); err != nil {
			b.Fatal(err)
		}
	}
}

func validMessageSpec() MessageSpec {
	return MessageSpec{
		ID:       "order-41@example.com",
		Date:     mailTestDate.Add(999 * time.Millisecond),
		From:     "Spice Orders <orders@example.com>",
		To:       []string{"Alice <alice@example.com>"},
		Cc:       []string{"ops@example.com"},
		Bcc:      []string{"audit@example.com", "alice@example.com"},
		ReplyTo:  "support@example.com",
		Subject:  "Order 41 is ready ✓",
		TextBody: "Order 41\nis ready.",
		HTMLBody: "<h1>Order 41</h1>\n<p>Ready.</p>",
		Attachments: []AttachmentSpec{{
			Filename:    "order-41.txt",
			ContentType: "text/plain; charset=utf-8",
			Data:        []byte("attachment contents"),
		}},
	}
}

func readMailPart(
	t *testing.T,
	reader *multipart.Reader,
	expectedMediaType string,
) string {
	t.Helper()
	part, err := reader.NextPart()
	if err != nil {
		t.Fatalf("read %s part: %v", expectedMediaType, err)
	}
	mediaType, _, err := mime.ParseMediaType(part.Header.Get("Content-Type"))
	if err != nil || mediaType != expectedMediaType {
		t.Fatalf("part content type = %q, %v", mediaType, err)
	}
	content, err := io.ReadAll(part)
	if err != nil {
		t.Fatalf("read %s body: %v", expectedMediaType, err)
	}
	return string(content)
}

func assertCRLFAndBase64Lines(t *testing.T, content []byte) {
	t.Helper()
	if bytes.Contains(bytes.ReplaceAll(content, []byte("\r\n"), nil), []byte("\n")) {
		t.Fatal("serialized message contains a bare line feed")
	}
	lines := strings.Split(string(content), "\r\n")
	inBase64 := false
	for _, line := range lines {
		if strings.EqualFold(line, "Content-Transfer-Encoding: base64") {
			inBase64 = true
			continue
		}
		if inBase64 && strings.HasPrefix(line, "--") {
			inBase64 = false
		}
		if inBase64 && len(line) > base64LineBytes {
			t.Fatalf("base64 line length = %d", len(line))
		}
	}
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

var _ Sender = senderFunc(nil)

type senderFunc func(context.Context, Message) error

func (sender senderFunc) Send(ctx context.Context, message Message) error {
	return sender(ctx, message)
}
