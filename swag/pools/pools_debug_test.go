//go:build poolsdebug

package pools

import (
	"strings"
	"testing"
)

// These tests exercise the instrumented pool and only compile/run under -tags poolsdebug.

func TestDebugDetectsLeak(t *testing.T) {
	ResetTracking()

	p := NewRedeemable[resettable]()
	_, _ = p.BorrowWithRedeem() // borrowed, never redeemed → a leak

	var fake fakeTB
	if AssertNoLeaks(&fake) {
		t.Fatal("expected AssertNoLeaks to report the leak")
	}
	if len(fake.errors) == 0 {
		t.Fatal("expected an Errorf about leaked objects")
	}
	if len(fake.logs) == 0 || !strings.Contains(fake.logs[0], "borrowed at") {
		t.Fatalf("expected a log naming the borrow site, got %v", fake.logs)
	}
	// the recorded borrow site should point at THIS test file, validating the caller() skip depth.
	if !strings.Contains(fake.logs[0], "pools_debug_test.go") {
		t.Fatalf("expected borrow site in pools_debug_test.go, got %q", fake.logs[0])
	}
}

func TestDebugCleanRunHasNoLeak(t *testing.T) {
	ResetTracking()

	p := NewRedeemable[resettable]()
	_, redeem := p.BorrowWithRedeem()
	redeem()

	var fake fakeTB
	if !AssertNoLeaks(&fake) {
		t.Fatalf("clean run should have no leaks, got errors=%v logs=%v", fake.errors, fake.logs)
	}
}

func TestDebugDoubleRedeemRichPanic(t *testing.T) {
	ResetTracking()

	p := NewRedeemable[resettable]()
	_, redeem := p.BorrowWithRedeem()
	redeem()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected a panic on double redeem")
		}
		msg, _ := r.(string)
		if !strings.Contains(msg, "double redeem") {
			t.Fatalf("expected a double-redeem panic, got %q", msg)
		}
	}()
	redeem()
}

func TestDebugForeignRedeemPanics(t *testing.T) {
	ResetTracking()

	p := New[resettable]()
	foreign := &resettable{} // never borrowed from p

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected a panic on redeem of a foreign object")
		}
		msg, _ := r.(string)
		if !strings.Contains(msg, "never handed out") {
			t.Fatalf("expected a never-handed-out panic, got %q", msg)
		}
	}()
	p.Redeem(foreign)
}

func TestDebugABADetected(t *testing.T) {
	ResetTracking()

	p := NewRedeemable[resettable]()

	innerA, redeemA := p.BorrowWithRedeem()
	redeemA() // A is valid, returned to the pool

	innerB, redeemB := p.BorrowWithRedeem()
	if innerB != innerA {
		t.Skip("pool returned a different slot; ABA scenario not reproduced this run")
	}
	defer redeemB() // keep B checked out so the stale redeemA hits the re-borrowed slot

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected a panic on a stale (ABA) redeem")
		}
		msg, _ := r.(string)
		if !strings.Contains(msg, "stale borrow") && !strings.Contains(msg, "ABA") {
			t.Fatalf("expected a stale-borrow/ABA panic, got %q", msg)
		}
	}()
	redeemA() // stale: the slot was re-borrowed by B since this borrow
}
