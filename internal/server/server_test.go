package server

import (
	"strings"
	"testing"
	"time"

	"github.com/deltron-fr/redis-server/internal/parser"
)

func newTestServer() *Server {
	return NewServer("")
}

func run(t *testing.T, s *Server, name string, args ...string) (string, error) {
	t.Helper()
	command, ok := s.Commands[name]
	if !ok {
		t.Fatalf("unknown command %q", name)
	}
	return command.Handler(&Client{}, Command{Args: args})
}

func mustRun(t *testing.T, s *Server, name string, args ...string) string {
	t.Helper()
	resp, err := run(t, s, name, args...)
	if err != nil {
		t.Fatalf("%s %v: unexpected error: %v", name, args, err)
	}
	return resp
}

func testIsTxCommand(cmdName string) bool {
	return cmdName == "EXEC" || cmdName == "DISCARD" || cmdName == "WATCH" ||
		cmdName == "MULTI" || cmdName == "UNWATCH"
}

func runWithClient(t *testing.T, s *Server, client *Client, name string, args ...string) string {
	t.Helper()
	resp, err := runWithClientErr(t, s, client, name, args...)
	if err != nil {
		t.Fatalf("%s %v: unexpected error: %v", name, args, err)
	}
	return resp
}

func runWithClientErr(t *testing.T, s *Server, client *Client, name string, args ...string) (string, error) {
	t.Helper()
	command, ok := s.Commands[name]
	if !ok {
		t.Fatalf("unknown command %q", name)
	}
	if client.ClientState == StateTransaction && !testIsTxCommand(name) {
		client.TxQueue = append(client.TxQueue, Command{
			Handler: command.Handler,
			Args:    args,
		})
		return parser.BulkStringOutputParser("QUEUED"), nil
	}
	return command.Handler(client, Command{Args: args})
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

// ── XREAD ──────────────────────────────────────────────────────────────

func TestXReadSingleStream(t *testing.T) {
	s := newTestServer()
	mustRun(t, s, "XADD", "mystream", "1-1", "name", "alice")

	// RESP2: array(1)[ array(2)[ bulk("mystream"),
	//   array(1)[ array(2)[ bulk("1-1"), array(2)[bulk("name"), bulk("alice")] ] ] ] ]
	want := "*1\r\n" +
		"*2\r\n$8\r\nmystream\r\n" +
		"*1\r\n*2\r\n$3\r\n1-1\r\n" +
		"*2\r\n$4\r\nname\r\n$5\r\nalice\r\n"

	got := mustRun(t, s, "XREAD", "STREAMS", "mystream", "0-0")
	if got != want {
		t.Fatalf("XREAD single stream:\nwant: %q\ngot:  %q", want, got)
	}
}

func TestXReadMultipleStreams(t *testing.T) {
	s := newTestServer()
	mustRun(t, s, "XADD", "stream1", "1-1", "k", "a")
	mustRun(t, s, "XADD", "stream2", "2-1", "k", "b")

	got := mustRun(t, s, "XREAD", "STREAMS", "stream1", "stream2", "0-0", "0-0")
	if !strings.HasPrefix(got, "*2\r\n") {
		t.Fatalf("XREAD multi stream: want 2 streams, got %q", got)
	}
	if !strings.Contains(got, "$7\r\nstream1\r\n") || !strings.Contains(got, "$7\r\nstream2\r\n") {
		t.Fatalf("XREAD multi stream: want both stream names, got %q", got)
	}
	if !strings.Contains(got, "$1\r\na\r\n") || !strings.Contains(got, "$1\r\nb\r\n") {
		t.Fatalf("XREAD multi stream: want entries from both streams, got %q", got)
	}
}

func TestXReadNonExistentKey(t *testing.T) {
	s := newTestServer()

	got := mustRun(t, s, "XREAD", "STREAMS", "nosuch", "0-0")
	if got != "*-1\r\n" {
		t.Fatalf("XREAD nonexistent: want nil array, got %q", got)
	}
}

func TestXReadFiltersById(t *testing.T) {
	s := newTestServer()
	mustRun(t, s, "XADD", "stream", "1-1", "k", "v1")
	mustRun(t, s, "XADD", "stream", "2-1", "k", "v2")
	mustRun(t, s, "XADD", "stream", "3-1", "k", "v3")

	// Only the entry strictly greater than 2-1 should be returned.
	want := "*1\r\n" +
		"*2\r\n$6\r\nstream\r\n" +
		"*1\r\n*2\r\n$3\r\n3-1\r\n" +
		"*2\r\n$1\r\nk\r\n$2\r\nv3\r\n"

	got := mustRun(t, s, "XREAD", "STREAMS", "stream", "2-1")
	if got != want {
		t.Fatalf("XREAD filter by ID:\nwant: %q\ngot:  %q", want, got)
	}
}

// ── INCR ───────────────────────────────────────────────────────────────

func TestIncrNewKey(t *testing.T) {
	s := newTestServer()

	got := mustRun(t, s, "INCR", "counter")
	if got != ":1\r\n" {
		t.Fatalf("INCR new key: want :1, got %q", got)
	}

	got = mustRun(t, s, "GET", "counter")
	if got != "$1\r\n1\r\n" {
		t.Fatalf("INCR new key stored as string: want 1, got %q", got)
	}
}

func TestIncrExistingKey(t *testing.T) {
	s := newTestServer()
	mustRun(t, s, "SET", "counter", "5")

	got := mustRun(t, s, "INCR", "counter")
	if got != ":6\r\n" {
		t.Fatalf("INCR existing: want :6, got %q", got)
	}

	got = mustRun(t, s, "INCR", "counter")
	if got != ":7\r\n" {
		t.Fatalf("INCR again: want :7, got %q", got)
	}
}

func TestIncrNonInteger(t *testing.T) {
	s := newTestServer()
	mustRun(t, s, "SET", "key", "abc")

	_, err := run(t, s, "INCR", "key")
	if err == nil {
		t.Fatal("INCR non-integer: expected error")
	}
}

func TestIncrExpiredKey(t *testing.T) {
	s := newTestServer()
	mustRun(t, s, "SET", "ttlkey", "10", "PX", "100")

	time.Sleep(150 * time.Millisecond)

	// Redis lazily deletes expired keys on access, so INCR behaves
	// as if the key never existed and resets it to 1.
	got := mustRun(t, s, "INCR", "ttlkey")
	if got != ":1\r\n" {
		t.Fatalf("INCR expired: want :1 (reset), got %q", got)
	}

	got = mustRun(t, s, "GET", "ttlkey")
	if got != "$1\r\n1\r\n" {
		t.Fatalf("GET after INCR on expired key: want 1, got %q", got)
	}
}

func TestIncrWrongArgCount(t *testing.T) {
	s := newTestServer()
	_, err := run(t, s, "INCR")
	if err == nil {
		t.Fatal("INCR with 0 args: expected error")
	}
	_, err = run(t, s, "INCR", "a", "b")
	if err == nil {
		t.Fatal("INCR with 2 args: expected error")
	}
}

// ── MULTI / EXEC ───────────────────────────────────────────────────────

func TestMultiExecBasic(t *testing.T) {
	s := newTestServer()
	client := &Client{}

	runWithClient(t, s, client, "MULTI")
	runWithClient(t, s, client, "SET", "key", "hello")
	runWithClient(t, s, client, "GET", "key")
	got := runWithClient(t, s, client, "EXEC")

	if !strings.HasPrefix(got, "*2\r\n") {
		t.Fatalf("MULTI EXEC basic: want 2 results, got %q", got)
	}
	if !strings.Contains(got, "+OK\r\n") {
		t.Fatalf("MULTI EXEC basic: want SET OK, got %q", got)
	}
	if !strings.Contains(got, "$5\r\nhello\r\n") {
		t.Fatalf("MULTI EXEC basic: want GET hello, got %q", got)
	}
}

func TestMultiRejectsArgs(t *testing.T) {
	s := newTestServer()
	_, err := run(t, s, "MULTI", "extra")
	if err == nil {
		t.Fatal("MULTI with args: expected error")
	}
}

func TestMultiRejectsNesting(t *testing.T) {
	s := newTestServer()
	client := &Client{}

	runWithClient(t, s, client, "MULTI")
	_, err := runWithClientErr(t, s, client, "MULTI")
	if err == nil {
		t.Fatal("nested MULTI: expected error")
	}

	runWithClient(t, s, client, "EXEC")
}

func TestExecWithoutMulti(t *testing.T) {
	s := newTestServer()
	client := &Client{}

	_, err := runWithClientErr(t, s, client, "EXEC")
	if err == nil {
		t.Fatal("EXEC without MULTI: expected error")
	}
}

func TestMultiExecWithIncr(t *testing.T) {
	s := newTestServer()
	client := &Client{}

	runWithClient(t, s, client, "SET", "counter", "10")
	runWithClient(t, s, client, "MULTI")
	runWithClient(t, s, client, "INCR", "counter")
	runWithClient(t, s, client, "INCR", "counter")
	got := runWithClient(t, s, client, "EXEC")

	if !strings.HasPrefix(got, "*2\r\n") {
		t.Fatalf("MULTI EXEC INCR: want 2 results, got %q", got)
	}

	got = mustRun(t, s, "GET", "counter")
	if got != "$2\r\n12\r\n" {
		t.Fatalf("MULTI EXEC INCR: want 12, got %q", got)
	}
}

// ── WATCH / UNWATCH ────────────────────────────────────────────────────

func TestWatchBasicNoConflict(t *testing.T) {
	s := newTestServer()
	client := &Client{}

	mustRun(t, s, "SET", "k", "1")
	runWithClient(t, s, client, "WATCH", "k")
	runWithClient(t, s, client, "MULTI")
	runWithClient(t, s, client, "SET", "k", "2")
	got := runWithClient(t, s, client, "EXEC")

	// No external modification between WATCH and EXEC — transaction commits.
	if !strings.HasPrefix(got, "*1\r\n") {
		t.Fatalf("WATCH no conflict: want 1 queued result, got %q", got)
	}

	got = mustRun(t, s, "GET", "k")
	if got != "$1\r\n2\r\n" {
		t.Fatalf("WATCH no conflict: want value 2, got %q", got)
	}
}

func TestWatchConflict(t *testing.T) {
	s := newTestServer()
	client := &Client{}

	mustRun(t, s, "SET", "k", "1")

	runWithClient(t, s, client, "WATCH", "k")

	// External modification between WATCH and MULTI/EXEC.
	mustRun(t, s, "SET", "k", "modified")

	runWithClient(t, s, client, "MULTI")
	runWithClient(t, s, client, "SET", "k", "2")
	got := runWithClient(t, s, client, "EXEC")

	// EXEC must abort because the watched key was modified.
	if got != "*-1\r\n" {
		t.Fatalf("WATCH conflict: want nil array, got %q", got)
	}

	got = mustRun(t, s, "GET", "k")
	if got != "$8\r\nmodified\r\n" {
		t.Fatalf("WATCH conflict: want external modification preserved, got %q", got)
	}
}

func TestWatchInsideTransaction(t *testing.T) {
	s := newTestServer()
	client := &Client{}

	runWithClient(t, s, client, "MULTI")
	_, err := runWithClientErr(t, s, client, "WATCH", "k")
	if err == nil {
		t.Fatal("WATCH inside MULTI: expected error")
	}
}

func TestWatchRejectsNoArgs(t *testing.T) {
	s := newTestServer()
	_, err := run(t, s, "WATCH")
	if err == nil {
		t.Fatal("WATCH with 0 args: expected error")
	}
}

func TestUnwatchBasic(t *testing.T) {
	s := newTestServer()
	client := &Client{}

	runWithClient(t, s, client, "WATCH", "k")
	got := runWithClient(t, s, client, "UNWATCH")
	if got != "+OK\r\n" {
		t.Fatalf("UNWATCH: want +OK, got %q", got)
	}

	// Unwatched — modifying k must not abort the transaction.
	mustRun(t, s, "SET", "k", "1")
	runWithClient(t, s, client, "MULTI")
	runWithClient(t, s, client, "SET", "k", "2")
	got = runWithClient(t, s, client, "EXEC")

	if !strings.HasPrefix(got, "*1\r\n") {
		t.Fatalf("UNWATCH then SET: want transaction committed, got %q", got)
	}
}

func TestWatchMultipleKeys(t *testing.T) {
	s := newTestServer()
	client := &Client{}

	mustRun(t, s, "SET", "a", "1")
	mustRun(t, s, "SET", "b", "1")

	runWithClient(t, s, client, "WATCH", "a", "b")

	// Modify only b — transaction aborts because any watched key changed.
	mustRun(t, s, "SET", "b", "modified")

	runWithClient(t, s, client, "MULTI")
	runWithClient(t, s, client, "SET", "a", "2")
	got := runWithClient(t, s, client, "EXEC")

	if got != "*-1\r\n" {
		t.Fatalf("WATCH multi-key: want nil array, got %q", got)
	}

	got = mustRun(t, s, "GET", "a")
	if got != "$1\r\n1\r\n" {
		t.Fatalf("WATCH multi-key: want a unchanged, got %q", got)
	}
	got = mustRun(t, s, "GET", "b")
	if got != "$8\r\nmodified\r\n" {
		t.Fatalf("WATCH multi-key: want b modified, got %q", got)
	}
}

func TestWatchMissingKey(t *testing.T) {
	s := newTestServer()
	client := &Client{}

	// Watching a key that doesn't exist is valid.
	runWithClient(t, s, client, "WATCH", "doesnotexist")
	runWithClient(t, s, client, "MULTI")
	runWithClient(t, s, client, "SET", "doesnotexist", "val")
	got := runWithClient(t, s, client, "EXEC")

	// No external modification — transaction commits.
	if !strings.HasPrefix(got, "*1\r\n") {
		t.Fatalf("WATCH missing key: want transaction committed, got %q", got)
	}
}

func TestUnwatchOnExec(t *testing.T) {
	s := newTestServer()
	client := &Client{}

	mustRun(t, s, "SET", "k", "1")

	runWithClient(t, s, client, "WATCH", "k")
	runWithClient(t, s, client, "MULTI")
	runWithClient(t, s, client, "SET", "k", "2")
	runWithClient(t, s, client, "EXEC")

	// Watches cleared by EXEC — next cycle must start clean.
	runWithClient(t, s, client, "MULTI")
	runWithClient(t, s, client, "SET", "k", "3")
	got := runWithClient(t, s, client, "EXEC")

	if !strings.HasPrefix(got, "*1\r\n") {
		t.Fatalf("EXEC clears watches: want committed, got %q", got)
	}
}

func TestUnwatchOnDiscard(t *testing.T) {
	s := newTestServer()
	client := &Client{}

	mustRun(t, s, "SET", "k", "1")

	runWithClient(t, s, client, "WATCH", "k")
	runWithClient(t, s, client, "MULTI")
	runWithClient(t, s, client, "SET", "k", "2")
	runWithClient(t, s, client, "DISCARD")

	// Watches cleared by DISCARD — next cycle must start clean.
	mustRun(t, s, "SET", "k", "3")
	runWithClient(t, s, client, "MULTI")
	runWithClient(t, s, client, "SET", "k", "4")
	got := runWithClient(t, s, client, "EXEC")

	if !strings.HasPrefix(got, "*1\r\n") {
		t.Fatalf("DISCARD clears watches: want committed, got %q", got)
	}
}

func TestWatchConflictDiscardResume(t *testing.T) {
	s := newTestServer()
	client := &Client{}

	mustRun(t, s, "SET", "k", "1")

	// First cycle: WATCH → conflict → abort.
	runWithClient(t, s, client, "WATCH", "k")
	mustRun(t, s, "SET", "k", "external")

	runWithClient(t, s, client, "MULTI")
	runWithClient(t, s, client, "SET", "k", "queued")
	got := runWithClient(t, s, client, "EXEC")
	if got != "*-1\r\n" {
		t.Fatalf("DISCARD resume: first EXEC want nil array, got %q", got)
	}

	// Second cycle: no watch → commit.
	runWithClient(t, s, client, "MULTI")
	runWithClient(t, s, client, "SET", "k", "final")
	got = runWithClient(t, s, client, "EXEC")
	if !strings.HasPrefix(got, "*1\r\n") {
		t.Fatalf("DISCARD resume: second EXEC want committed, got %q", got)
	}

	got = mustRun(t, s, "GET", "k")
	if got != "$5\r\nfinal\r\n" {
		t.Fatalf("DISCARD resume: want final, got %q", got)
	}
}

func TestUnwatchRejectsArgs(t *testing.T) {
	s := newTestServer()
	_, err := run(t, s, "UNWATCH", "k")
	if err == nil {
		t.Fatal("UNWATCH with arg: expected error")
	}
}
