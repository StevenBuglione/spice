package mailtest_test

import (
	"context"
	"fmt"
	"time"

	spicemail "github.com/spice-framework/spice/mail"
	"github.com/spice-framework/spice/mail/mailtest"
)

func ExampleSender() {
	message, err := spicemail.NewMessage(spicemail.MessageSpec{
		ID:       "welcome-7@example.com",
		Date:     time.Date(2026, time.July, 26, 12, 0, 0, 0, time.UTC),
		From:     "team@example.com",
		To:       []string{"developer@example.com"},
		Subject:  "Welcome",
		TextBody: "Spice is ready.",
	})
	if err != nil {
		panic(err)
	}
	sender, err := mailtest.New(mailtest.Config{Capacity: 10})
	if err != nil {
		panic(err)
	}
	if err := sender.Send(context.Background(), message); err != nil {
		panic(err)
	}
	delivered := sender.Messages()
	fmt.Println(delivered[0].Subject())
	fmt.Println(delivered[0].TextBody())
	// Output:
	// Welcome
	// Spice is ready.
}
