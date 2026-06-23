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

func TestBufferGetPutReuse(t *testing.T) {
	b := GetBuffer()
	if b.Len() != 0 {
		t.Fatalf("expected empty buffer, got len %d", b.Len())
	}
	b.WriteString("payload")
	PutBuffer(b)

	b2 := GetBuffer()
	if b2.Len() != 0 {
		t.Fatalf("expected reset buffer on reuse, got len %d", b2.Len())
	}
	PutBuffer(b2)
}

func TestBufferPutNilAndOversizedAreSafe(t *testing.T) {
	PutBuffer(nil) // must not panic

	b := GetBuffer()
	b.Grow(maxSharedCapacity * 2) // oversized
	PutBuffer(b)                  // dropped, not recycled; must not panic

	b2 := GetBuffer()
	if b2.Cap() > maxSharedCapacity {
		t.Fatalf("oversized buffer should not have been recycled, got cap %d", b2.Cap())
	}
	PutBuffer(b2)
}

func TestReaderGetPutReuse(t *testing.T) {
	r := GetReader([]byte("first"))
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	if string(got) != "first" {
		t.Fatalf("unexpected read: %q", got)
	}
	PutReader(r)

	r2 := GetReader([]byte("second"))
	got2, err := io.ReadAll(r2)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	if string(got2) != "second" {
		t.Fatalf("reader not reinitialized on reuse: %q", got2)
	}
	PutReader(r2)
}

func TestReaderPutClearsData(t *testing.T) {
	r := GetReader([]byte("data"))
	PutReader(r)

	// After PutReader the reader must not reference the old data (Len reports 0).
	if r.Len() != 0 {
		t.Fatalf("expected reader to release its data on Put, Len = %d", r.Len())
	}
}

func TestReaderPutNilIsSafe(t *testing.T) {
	PutReader(nil) // must not panic
}
