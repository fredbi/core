package shared

import (
	"io"
	"testing"
)

func TestBytesBorrowRedeemReuse(t *testing.T) {
	s, redeem := Bytes.BorrowWithRedeem()
	if s.Len() != 0 {
		t.Fatalf("expected empty buffer, got len %d", s.Len())
	}
	s.Append([]byte("hello")...)
	if got := string(s.Slice()); got != "hello" {
		t.Fatalf("unexpected contents: %q", got)
	}
	redeem()

	s2, redeem2 := Bytes.BorrowWithRedeem()
	if s2.Len() != 0 {
		t.Fatalf("expected reset buffer on reuse, got len %d", s2.Len())
	}
	redeem2()
}

func TestBufferBorrowRedeemReuse(t *testing.T) {
	b := BorrowBuffer()
	if b.Len() != 0 {
		t.Fatalf("expected empty buffer, got len %d", b.Len())
	}
	b.WriteString("payload")
	RedeemBuffer(b)

	b2 := BorrowBuffer()
	if b2.Len() != 0 {
		t.Fatalf("expected reset buffer on reuse, got len %d", b2.Len())
	}
	RedeemBuffer(b2)
}

func TestBufferRedeemNilAndOversizedAreSafe(t *testing.T) {
	RedeemBuffer(nil) // must not panic

	b := BorrowBuffer()
	b.Grow(maxSharedCapacity * 2) // oversized
	RedeemBuffer(b)               // dropped, not recycled; must not panic

	b2 := BorrowBuffer()
	if b2.Cap() > maxSharedCapacity {
		t.Fatalf("oversized buffer should not have been recycled, got cap %d", b2.Cap())
	}
	RedeemBuffer(b2)
}

func TestReaderBorrowRedeemReuse(t *testing.T) {
	r := BorrowReader([]byte("first"))
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	if string(got) != "first" {
		t.Fatalf("unexpected read: %q", got)
	}
	RedeemReader(r)

	r2 := BorrowReader([]byte("second"))
	got2, err := io.ReadAll(r2)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	if string(got2) != "second" {
		t.Fatalf("reader not reinitialized on reuse: %q", got2)
	}
	RedeemReader(r2)
}

func TestReaderRedeemClearsData(t *testing.T) {
	r := BorrowReader([]byte("data"))
	RedeemReader(r)

	// After RedeemReader the reader must not reference the old data (Len reports 0).
	if r.Len() != 0 {
		t.Fatalf("expected reader to release its data on Redeem, Len = %d", r.Len())
	}
}

func TestReaderRedeemNilIsSafe(t *testing.T) {
	RedeemReader(nil) // must not panic
}
