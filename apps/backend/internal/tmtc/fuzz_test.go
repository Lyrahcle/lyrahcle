package tmtc

import "testing"

func FuzzDecodeAX25Hex(f *testing.F) {
	seeds := []string{
		"",
		"00",
		"gg",
		hexSeed(),
		"82a0a4a6",
		"ffffffffffffffffffffffffffffffff",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, frameHex string) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("DecodeAX25Hex panicked on input %q: %v", frameHex, r)
			}
		}()
		_, _ = DecodeAX25Hex(frameHex)
	})
}

func FuzzDecodeAX25(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0x82, 0xa0, 0xa4, 0xa6})
	f.Add(make([]byte, 16))
	f.Add(make([]byte, 1000))

	f.Fuzz(func(t *testing.T, data []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("DecodeAX25 panicked on %d-byte input: %v", len(data), r)
			}
		}()
		_, _ = DecodeAX25(data)
	})
}

func hexSeed() string {
	frame := buildFrame("APRS", "N0CALL", 0, 7, "test")
	out := make([]byte, len(frame)*2)
	const hexdigits = "0123456789abcdef"
	for i, b := range frame {
		out[i*2] = hexdigits[b>>4]
		out[i*2+1] = hexdigits[b&0x0F]
	}
	return string(out)
}

func FuzzDecodeCCSDSHex(f *testing.F) {
	seeds := []string{
		"",
		"00",
		"gg",
		"0864c02a000b54454c454d455452592d4f4b",
		"ffffffffffffffffffffffffffffffff",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, packetHex string) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("DecodeCCSDSHex panicked on input %q: %v", packetHex, r)
			}
		}()
		_, _ = DecodeCCSDSHex(packetHex)
	})
}

func FuzzDecodeCCSDS(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0x08, 0x64, 0xc0, 0x2a, 0x00, 0x0b})
	f.Add(make([]byte, 6))
	f.Add(make([]byte, 1000))

	f.Fuzz(func(t *testing.T, data []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("DecodeCCSDS panicked on %d-byte input: %v", len(data), r)
			}
		}()
		_, _ = DecodeCCSDS(data)
	})
}
