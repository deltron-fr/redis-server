package server

import (
	"strings"
	"testing"
	"time"
)

func newTestServer() *Server {
	return NewServer()
}

func run(t *testing.T, s *Server, name string, args ...string) (string, error) {
	t.Helper()
	handler, ok := s.Commands[name]
	if !ok {
		t.Fatalf("unknown command %q", name)
	}
	return handler(&Client{}, Command{Args: args})
}

func mustRun(t *testing.T, s *Server, name string, args ...string) string {
	t.Helper()
	resp, err := run(t, s, name, args...)
	if err != nil {
		t.Fatalf("%s %v: unexpected error: %v", name, args, err)
	}
	return resp
}

// ── SET / GET ──────────────────────────────────────────────────────────

func TestSetAndGet(t *testing.T) {
	s := newTestServer()

	got := mustRun(t, s, "SET", "mykey", "hello")
	if got != "+OK\r\n" {
		t.Fatalf("SET: want +OK, got %q", got)
	}

	got = mustRun(t, s, "GET", "mykey")
	if got != "$5\r\nhello\r\n" {
		t.Fatalf("GET: want hello, got %q", got)
	}
}

func TestGetNonExistentKey(t *testing.T) {
	s := newTestServer()

	got := mustRun(t, s, "GET", "nokey")
	if got != "$-1\r\n" {
		t.Fatalf("GET nonexistent: want nil bulk string, got %q", got)
	}
}

func TestSetOverwrite(t *testing.T) {
	s := newTestServer()

	mustRun(t, s, "SET", "key", "first")
	mustRun(t, s, "SET", "key", "second")

	got := mustRun(t, s, "GET", "key")
	if got != "$6\r\nsecond\r\n" {
		t.Fatalf("GET after overwrite: want second, got %q", got)
	}
}

func TestSetWithPXExpiry(t *testing.T) {
	s := newTestServer()

	got := mustRun(t, s, "SET", "expkey", "val", "PX", "100")
	if got != "+OK\r\n" {
		t.Fatalf("SET PX: want +OK, got %q", got)
	}

	got = mustRun(t, s, "GET", "expkey")
	if got != "$3\r\nval\r\n" {
		t.Fatalf("GET before expiry: want val, got %q", got)
	}

	time.Sleep(150 * time.Millisecond)

	got = mustRun(t, s, "GET", "expkey")
	if got != "$-1\r\n" {
		t.Fatalf("GET after PX expiry: want nil, got %q", got)
	}
}

func TestSetWithEXExpiry(t *testing.T) {
	s := newTestServer()

	got := mustRun(t, s, "SET", "exkey", "val", "EX", "1")
	if got != "+OK\r\n" {
		t.Fatalf("SET EX: want +OK, got %q", got)
	}

	got = mustRun(t, s, "GET", "exkey")
	if got != "$3\r\nval\r\n" {
		t.Fatalf("GET before expiry: want val, got %q", got)
	}

	time.Sleep(1100 * time.Millisecond)

	got = mustRun(t, s, "GET", "exkey")
	if got != "$-1\r\n" {
		t.Fatalf("GET after EX expiry: want nil, got %q", got)
	}
}

func TestSetWrongArgCount(t *testing.T) {
	s := newTestServer()
	_, err := run(t, s, "SET", "onlykey")
	if err == nil {
		t.Fatal("SET with 1 arg: expected error")
	}
}

func TestGetWrongArgCount(t *testing.T) {
	s := newTestServer()
	_, err := run(t, s, "GET")
	if err == nil {
		t.Fatal("GET with 0 args: expected error")
	}
}

// ── RPUSH / LPUSH ─────────────────────────────────────────────────────

func TestRPush(t *testing.T) {
	s := newTestServer()

	got := mustRun(t, s, "RPUSH", "list", "a")
	if got != ":1\r\n" {
		t.Fatalf("RPUSH first: want :1, got %q", got)
	}

	got = mustRun(t, s, "RPUSH", "list", "b", "c")
	if got != ":3\r\n" {
		t.Fatalf("RPUSH multi: want :3, got %q", got)
	}

	got = mustRun(t, s, "LRANGE", "list", "0", "-1")
	want := "*3\r\n$1\r\na\r\n$1\r\nb\r\n$1\r\nc\r\n"
	if got != want {
		t.Fatalf("LRANGE after RPUSH:\nwant: %q\ngot:  %q", want, got)
	}
}

func TestLPush(t *testing.T) {
	s := newTestServer()

	got := mustRun(t, s, "LPUSH", "list", "a")
	if got != ":1\r\n" {
		t.Fatalf("LPUSH first: want :1, got %q", got)
	}

	got = mustRun(t, s, "LPUSH", "list", "b", "c")
	if got != ":3\r\n" {
		t.Fatalf("LPUSH multi: want :3, got %q", got)
	}

	got = mustRun(t, s, "LRANGE", "list", "0", "-1")
	want := "*3\r\n$1\r\nc\r\n$1\r\nb\r\n$1\r\na\r\n"
	if got != want {
		t.Fatalf("LRANGE after LPUSH:\nwant: %q\ngot:  %q", want, got)
	}
}

func TestRPushWrongArgCount(t *testing.T) {
	s := newTestServer()
	_, err := run(t, s, "RPUSH", "list")
	if err == nil {
		t.Fatal("RPUSH with 1 arg: expected error")
	}
}

// ── LRANGE ─────────────────────────────────────────────────────────────

func TestLRangeSubset(t *testing.T) {
	s := newTestServer()
	mustRun(t, s, "RPUSH", "list", "a", "b", "c", "d", "e")

	got := mustRun(t, s, "LRANGE", "list", "1", "3")
	want := "*3\r\n$1\r\nb\r\n$1\r\nc\r\n$1\r\nd\r\n"
	if got != want {
		t.Fatalf("LRANGE subset:\nwant: %q\ngot:  %q", want, got)
	}
}

func TestLRangeNegativeIndices(t *testing.T) {
	s := newTestServer()
	mustRun(t, s, "RPUSH", "list", "a", "b", "c", "d", "e")

	got := mustRun(t, s, "LRANGE", "list", "-3", "-1")
	want := "*3\r\n$1\r\nc\r\n$1\r\nd\r\n$1\r\ne\r\n"
	if got != want {
		t.Fatalf("LRANGE neg indices:\nwant: %q\ngot:  %q", want, got)
	}
}

func TestLRangeNonExistentKey(t *testing.T) {
	s := newTestServer()

	got := mustRun(t, s, "LRANGE", "nolist", "0", "-1")
	if got != "*0\r\n" {
		t.Fatalf("LRANGE nonexistent: want empty array, got %q", got)
	}
}

func TestLRangeStartBeyondLength(t *testing.T) {
	s := newTestServer()
	mustRun(t, s, "RPUSH", "list", "a", "b")

	got := mustRun(t, s, "LRANGE", "list", "10", "20")
	if got != "*0\r\n" {
		t.Fatalf("LRANGE out of range: want empty array, got %q", got)
	}
}

// ── LLEN ───────────────────────────────────────────────────────────────

func TestLLenNonExistent(t *testing.T) {
	s := newTestServer()

	got := mustRun(t, s, "LLEN", "nolist")
	if got != ":0\r\n" {
		t.Fatalf("LLEN nonexistent: want :0, got %q", got)
	}
}

func TestLLen(t *testing.T) {
	s := newTestServer()
	mustRun(t, s, "RPUSH", "list", "a", "b", "c")

	got := mustRun(t, s, "LLEN", "list")
	if got != ":3\r\n" {
		t.Fatalf("LLEN: want :3, got %q", got)
	}
}

// ── LPOP ───────────────────────────────────────────────────────────────

func TestLPopSingle(t *testing.T) {
	s := newTestServer()
	mustRun(t, s, "RPUSH", "list", "a", "b", "c")

	got := mustRun(t, s, "LPOP", "list")
	if got != "$1\r\na\r\n" {
		t.Fatalf("LPOP single: want a, got %q", got)
	}

	got = mustRun(t, s, "LLEN", "list")
	if got != ":2\r\n" {
		t.Fatalf("LLEN after LPOP: want :2, got %q", got)
	}
}

func TestLPopWithCount(t *testing.T) {
	s := newTestServer()
	mustRun(t, s, "RPUSH", "list", "a", "b", "c", "d")

	got := mustRun(t, s, "LPOP", "list", "2")
	want := "*2\r\n$1\r\na\r\n$1\r\nb\r\n"
	if got != want {
		t.Fatalf("LPOP count:\nwant: %q\ngot:  %q", want, got)
	}

	got = mustRun(t, s, "LLEN", "list")
	if got != ":2\r\n" {
		t.Fatalf("LLEN after LPOP count: want :2, got %q", got)
	}
}

func TestLPopCountExceedsLength(t *testing.T) {
	s := newTestServer()
	mustRun(t, s, "RPUSH", "list", "a", "b")

	got := mustRun(t, s, "LPOP", "list", "5")
	want := "*2\r\n$1\r\na\r\n$1\r\nb\r\n"
	if got != want {
		t.Fatalf("LPOP count>len:\nwant: %q\ngot:  %q", want, got)
	}
}

func TestLPopEmptyList(t *testing.T) {
	s := newTestServer()

	got := mustRun(t, s, "LPOP", "nolist")
	if got != "$-1\r\n" {
		t.Fatalf("LPOP empty: want nil, got %q", got)
	}
}

// ── BLPOP ──────────────────────────────────────────────────────────────

func TestBLPopWithExistingData(t *testing.T) {
	s := newTestServer()
	mustRun(t, s, "RPUSH", "list", "x", "y")

	got := mustRun(t, s, "BLPOP", "list", "1")
	want := "*2\r\n$4\r\nlist\r\n$1\r\nx\r\n"
	if got != want {
		t.Fatalf("BLPOP:\nwant: %q\ngot:  %q", want, got)
	}
}

func TestBLPopTimeout(t *testing.T) {
	s := newTestServer()

	start := time.Now()
	got := mustRun(t, s, "BLPOP", "emptylist", "1")
	elapsed := time.Since(start)

	if got != "*-1\r\n" {
		t.Fatalf("BLPOP timeout: want nil array, got %q", got)
	}
	if elapsed < 900*time.Millisecond {
		t.Fatalf("BLPOP returned too early: %v", elapsed)
	}
}

// ── TYPE ───────────────────────────────────────────────────────────────

func TestTypeString(t *testing.T) {
	s := newTestServer()
	mustRun(t, s, "SET", "strkey", "val")

	got := mustRun(t, s, "TYPE", "strkey")
	if got != "+string\r\n" {
		t.Fatalf("TYPE string: want +string, got %q", got)
	}
}

func TestTypeStream(t *testing.T) {
	s := newTestServer()
	mustRun(t, s, "XADD", "stream", "1-1", "field", "val")

	got := mustRun(t, s, "TYPE", "stream")
	if got != "+stream\r\n" {
		t.Fatalf("TYPE stream: want +stream, got %q", got)
	}
}

func TestTypeNone(t *testing.T) {
	s := newTestServer()

	got := mustRun(t, s, "TYPE", "nonexistent")
	if got != "+none\r\n" {
		t.Fatalf("TYPE none: want +none, got %q", got)
	}
}

func TestTypeExpiredKey(t *testing.T) {
	s := newTestServer()
	mustRun(t, s, "SET", "ttlkey", "val", "PX", "100")

	time.Sleep(150 * time.Millisecond)

	got := mustRun(t, s, "TYPE", "ttlkey")
	if got != "+none\r\n" {
		t.Fatalf("TYPE expired: want +none, got %q", got)
	}
}

// ── XADD ───────────────────────────────────────────────────────────────

func TestXAddManualID(t *testing.T) {
	s := newTestServer()

	got := mustRun(t, s, "XADD", "stream", "1-1", "name", "alice")
	if got != "$3\r\n1-1\r\n" {
		t.Fatalf("XADD manual ID: want 1-1, got %q", got)
	}
}

func TestXAddAutoSeq(t *testing.T) {
	s := newTestServer()
	mustRun(t, s, "XADD", "stream", "5-1", "k", "v")

	got := mustRun(t, s, "XADD", "stream", "5-*", "k", "v2")
	if got != "$3\r\n5-2\r\n" {
		t.Fatalf("XADD auto-seq same ms: want 5-2, got %q", got)
	}

	got = mustRun(t, s, "XADD", "stream", "7-*", "k", "v3")
	if got != "$3\r\n7-0\r\n" {
		t.Fatalf("XADD auto-seq new ms: want 7-0, got %q", got)
	}
}

func TestXAddZeroMsAutoSeq(t *testing.T) {
	s := newTestServer()

	got := mustRun(t, s, "XADD", "stream", "0-*", "k", "v")
	if got != "$3\r\n0-1\r\n" {
		t.Fatalf("XADD 0-*: want 0-1, got %q", got)
	}
}

func TestXAddRejectsZeroZero(t *testing.T) {
	s := newTestServer()
	_, err := run(t, s, "XADD", "stream", "0-0", "k", "v")
	if err == nil {
		t.Fatal("XADD 0-0: expected error")
	}
}

func TestXAddRejectsSmallerID(t *testing.T) {
	s := newTestServer()
	mustRun(t, s, "XADD", "stream", "5-2", "k", "v")

	_, err := run(t, s, "XADD", "stream", "3-1", "k", "v")
	if err == nil {
		t.Fatal("XADD smaller ID: expected error")
	}
}

func TestXAddRejectsEqualID(t *testing.T) {
	s := newTestServer()
	mustRun(t, s, "XADD", "stream", "5-2", "k", "v")

	_, err := run(t, s, "XADD", "stream", "5-2", "k", "v")
	if err == nil {
		t.Fatal("XADD equal ID: expected error")
	}
}

func TestXAddMultipleFields(t *testing.T) {
	s := newTestServer()

	got := mustRun(t, s, "XADD", "stream", "1-1", "name", "alice", "age", "30")
	if got != "$3\r\n1-1\r\n" {
		t.Fatalf("XADD multi-field: want 1-1, got %q", got)
	}
}

// ── XRANGE ─────────────────────────────────────────────────────────────

func TestXRangeFullRange(t *testing.T) {
	s := newTestServer()
	mustRun(t, s, "XADD", "stream", "1-1", "name", "alice")
	mustRun(t, s, "XADD", "stream", "2-1", "name", "bob")
	mustRun(t, s, "XADD", "stream", "3-1", "name", "charlie")

	got := mustRun(t, s, "XRANGE", "stream", "1-1", "3-1")
	if !strings.HasPrefix(got, "*3\r\n") {
		t.Fatalf("XRANGE full: want 3 entries, got %q", got)
	}
}

func TestXRangeSubset(t *testing.T) {
	s := newTestServer()
	mustRun(t, s, "XADD", "stream", "1-1", "k", "a")
	mustRun(t, s, "XADD", "stream", "2-1", "k", "b")
	mustRun(t, s, "XADD", "stream", "3-1", "k", "c")

	got := mustRun(t, s, "XRANGE", "stream", "2-1", "3-1")
	if !strings.HasPrefix(got, "*2\r\n") {
		t.Fatalf("XRANGE subset: want 2 entries, got %q", got)
	}
}

func TestXRangeStartWithHyphen(t *testing.T) {
	s := newTestServer()
	mustRun(t, s, "XADD", "stream", "1-1", "name", "alice")
	mustRun(t, s, "XADD", "stream", "2-1", "name", "bob")
	mustRun(t, s, "XADD", "stream", "3-1", "name", "charlie")

	got := mustRun(t, s, "XRANGE", "stream", "-", "3-1")
	if !strings.HasPrefix(got, "*3\r\n") {
		t.Fatalf("XRANGE start hyphen: want 3 entries, got %q", got)
	}
}

func TestXRangeStopWithAsterisk(t *testing.T) {
	s := newTestServer()
	mustRun(t, s, "XADD", "stream", "1-1", "name", "alice")
	mustRun(t, s, "XADD", "stream", "2-1", "name", "bob")
	mustRun(t, s, "XADD", "stream", "3-1", "name", "charlie")

	got := mustRun(t, s, "XRANGE", "stream", "1-1", "+")
	if !strings.HasPrefix(got, "*3\r\n") {
		t.Fatalf("XRANGE stop asterisk: want 3 entries, got %q", got)
	}
}
