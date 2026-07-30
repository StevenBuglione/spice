package conversion

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"
)

func TestBuiltInCodecs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		run  func() error
	}{
		{
			name: "string",
			run: func() error {
				value, err := String().Parse("spice")
				if err != nil || value != "spice" {
					return errors.New("string did not round trip")
				}
				return nil
			},
		},
		{
			name: "boolean",
			run: func() error {
				value, err := Boolean().Parse("true")
				if err != nil || !value {
					return errors.New("boolean did not parse")
				}
				formatted, err := Boolean().Format(value)
				if err != nil || formatted != "true" {
					return errors.New("boolean did not format")
				}
				return nil
			},
		},
		{
			name: "signed integer",
			run: func() error {
				codec, err := SignedInteger(16)
				if err != nil {
					return err
				}
				value, err := codec.Parse("-32768")
				if err != nil || value != math.MinInt16 {
					return errors.New("integer did not parse")
				}
				_, err = codec.Format(math.MaxInt16 + 1)
				if !errors.Is(err, ErrInvalidValue) {
					return errors.New("integer accepted an out-of-range value")
				}
				return nil
			},
		},
		{
			name: "unsigned integer",
			run: func() error {
				codec, err := UnsignedInteger(8)
				if err != nil {
					return err
				}
				value, err := codec.Parse("255")
				if err != nil || value != math.MaxUint8 {
					return errors.New("unsigned integer did not parse")
				}
				return nil
			},
		},
		{
			name: "float",
			run: func() error {
				codec, err := Float(64)
				if err != nil {
					return err
				}
				value, err := codec.Parse("1.25")
				if err != nil || value != 1.25 {
					return errors.New("float did not parse")
				}
				return nil
			},
		},
		{
			name: "duration",
			run: func() error {
				value, err := Duration().Parse("2m30s")
				if err != nil || value != 150*time.Second {
					return errors.New("duration did not parse")
				}
				formatted, err := Duration().Format(value)
				if err != nil || formatted != "2m30s" {
					return errors.New("duration did not format")
				}
				return nil
			},
		},
		{
			name: "time",
			run: func() error {
				codec, err := Time(time.DateOnly, time.UTC)
				if err != nil {
					return err
				}
				value, err := codec.Parse("2026-07-30")
				if err != nil {
					return err
				}
				formatted, err := codec.Format(value)
				if err != nil || formatted != "2026-07-30" {
					return errors.New("time did not round trip")
				}
				return nil
			},
		},
		{
			name: "URL",
			run: func() error {
				value, err := URL().Parse("https://example.com/orders")
				if err != nil || value.Host != "example.com" {
					return errors.New("URL did not parse")
				}
				return nil
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if err := test.run(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestInvalidBuiltInValuesDoNotLeakRawInput(t *testing.T) {
	t.Parallel()

	const secret = "not-a-number-secret"
	checks := []func() error{
		func() error {
			_, err := ParseBoolean(secret)
			return err
		},
		func() error {
			_, err := ParseSignedInteger(secret, 64)
			return err
		},
		func() error {
			_, err := ParseDuration(secret)
			return err
		},
		func() error {
			_, err := URL().Parse(secret)
			return err
		},
	}
	for _, check := range checks {
		err := check()
		if err == nil {
			t.Fatal("invalid value was accepted")
		}
		if strings.Contains(err.Error(), secret) {
			t.Fatalf("error leaks raw input: %v", err)
		}
	}
}

func TestConverterComposition(t *testing.T) {
	t.Parallel()

	integer, err := SignedInteger(64)
	if err != nil {
		t.Fatal(err)
	}
	positive := ConverterFunc[int64, string](func(value int64) (string, error) {
		if value <= 0 {
			return "", errors.New("must be positive")
		}
		return "order", nil
	})
	converter := Then[string, int64, string](integer, positive)
	value, err := converter.Convert("42")
	if err != nil || value != "order" {
		t.Fatalf("Convert() = %q, %v", value, err)
	}
	if _, err := converter.Convert("-1"); err == nil {
		t.Fatal("Convert() accepted a negative value")
	}
}

func TestCustomCodecAndInvalidConstruction(t *testing.T) {
	t.Parallel()

	codec, err := NewCodec(
		"order.ID",
		func(value string) (string, error) { return "id:" + value, nil },
		func(value string) (string, error) {
			return strings.TrimPrefix(value, "id:"), nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if codec.Name() != "order.ID" {
		t.Fatalf("Name() = %q", codec.Name())
	}
	value, err := codec.Parse("42")
	if err != nil || value != "id:42" {
		t.Fatalf("Parse() = %q, %v", value, err)
	}

	if _, err := NewCodec[string]("", nil, nil); err == nil {
		t.Fatal("NewCodec() accepted an invalid definition")
	}
	if _, err := SignedInteger(7); err == nil {
		t.Fatal("SignedInteger() accepted bit size 7")
	}
	if _, err := UnsignedInteger(7); err == nil {
		t.Fatal("UnsignedInteger() accepted bit size 7")
	}
	if _, err := Float(16); err == nil {
		t.Fatal("Float() accepted bit size 16")
	}
	if _, err := Time("", time.UTC); err == nil {
		t.Fatal("Time() accepted an empty layout")
	}
	if _, err := Time(time.DateOnly, nil); err == nil {
		t.Fatal("Time() accepted a nil location")
	}

	var zero Codec[string]
	if _, err := zero.Parse("value"); err == nil {
		t.Fatal("zero Codec.Parse() succeeded")
	}
	if _, err := zero.Format("value"); err == nil {
		t.Fatal("zero Codec.Format() succeeded")
	}
	var nilConverter ConverterFunc[string, string]
	if _, err := nilConverter.Convert("value"); err == nil {
		t.Fatal("nil ConverterFunc succeeded")
	}
}

func TestThenRejectsMissingConverters(t *testing.T) {
	t.Parallel()

	var first Converter[string, string]
	if _, err := Then[string, string, string](
		first,
		String(),
	).Convert("value"); err == nil {
		t.Fatal("Then() accepted a nil first converter")
	}
	var second Converter[string, string]
	if _, err := Then[string, string, string](
		String(),
		second,
	).Convert("value"); err == nil {
		t.Fatal("Then() accepted a nil second converter")
	}
}

func ExampleThen() {
	integer, err := SignedInteger(64)
	if err != nil {
		fmt.Println("conversion unavailable")
		return
	}
	label := ConverterFunc[int64, string](func(value int64) (string, error) {
		return fmt.Sprintf("order-%d", value), nil
	})
	value, err := Then[string, int64, string](
		integer,
		label,
	).Convert("42")
	if err != nil {
		fmt.Println("conversion failed")
		return
	}
	fmt.Println(value)
	// Output: order-42
}
