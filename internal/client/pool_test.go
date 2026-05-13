package client

import "testing"

func TestChannelPoolReusesClientConnForMatchingTargetAndConfig(t *testing.T) {
	pool := NewChannelPool()
	maxReceive := 1024
	config := DialConfig{
		Authority:               "spanner-emulator",
		MaxReceiveMessageLength: &maxReceive,
	}

	first, err := pool.Acquire("passthrough:///pool-reuse", config)
	if err != nil {
		t.Fatalf("first acquire returned error: %v", err)
	}
	second, err := pool.Acquire("passthrough:///pool-reuse", config)
	if err != nil {
		t.Fatalf("second acquire returned error: %v", err)
	}

	firstConn, err := first.Conn()
	if err != nil {
		t.Fatalf("first Conn returned error: %v", err)
	}
	secondConn, err := second.Conn()
	if err != nil {
		t.Fatalf("second Conn returned error: %v", err)
	}
	if firstConn != secondConn {
		t.Fatal("matching channel leases did not reuse the same ClientConn")
	}

	if err := first.Close(); err != nil {
		t.Fatalf("first close returned error: %v", err)
	}
	if _, err := first.Conn(); err != ErrChannelClosed {
		t.Fatalf("closed first lease Conn error = %v, want ErrChannelClosed", err)
	}
	if _, err := second.Conn(); err != nil {
		t.Fatalf("second lease Conn returned error after first close: %v", err)
	}
	if err := second.Close(); err != nil {
		t.Fatalf("second close returned error: %v", err)
	}
}

func TestChannelPoolSeparatesDifferentEffectiveOptions(t *testing.T) {
	pool := NewChannelPool()
	firstMaxReceive := 1024
	secondMaxReceive := 2048

	first, err := pool.Acquire("passthrough:///pool-options", DialConfig{MaxReceiveMessageLength: &firstMaxReceive})
	if err != nil {
		t.Fatalf("first acquire returned error: %v", err)
	}
	second, err := pool.Acquire("passthrough:///pool-options", DialConfig{MaxReceiveMessageLength: &secondMaxReceive})
	if err != nil {
		t.Fatalf("second acquire returned error: %v", err)
	}
	t.Cleanup(func() {
		first.Close()
		second.Close()
	})

	firstConn, err := first.Conn()
	if err != nil {
		t.Fatalf("first Conn returned error: %v", err)
	}
	secondConn, err := second.Conn()
	if err != nil {
		t.Fatalf("second Conn returned error: %v", err)
	}
	if firstConn == secondConn {
		t.Fatal("different effective channel options reused the same ClientConn")
	}
}

func TestChannelPoolClosesPhysicalConnAfterLastLease(t *testing.T) {
	pool := NewChannelPool()

	first, err := pool.Acquire("passthrough:///pool-release", DialConfig{})
	if err != nil {
		t.Fatalf("first acquire returned error: %v", err)
	}
	second, err := pool.Acquire("passthrough:///pool-release", DialConfig{})
	if err != nil {
		t.Fatalf("second acquire returned error: %v", err)
	}

	originalConn, err := first.Conn()
	if err != nil {
		t.Fatalf("first Conn returned error: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("first close returned error: %v", err)
	}
	if err := second.Close(); err != nil {
		t.Fatalf("second close returned error: %v", err)
	}

	third, err := pool.Acquire("passthrough:///pool-release", DialConfig{})
	if err != nil {
		t.Fatalf("third acquire returned error: %v", err)
	}
	defer third.Close()

	newConn, err := third.Conn()
	if err != nil {
		t.Fatalf("third Conn returned error: %v", err)
	}
	if newConn == originalConn {
		t.Fatal("pool reused ClientConn after all leases were closed")
	}
}
