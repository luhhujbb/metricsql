package metricsql

import (
	"reflect"
	"testing"
)

func TestIsSpecialIntegerPrefix(t *testing.T) {
	f := func(s string, resultExpected bool) {
		t.Helper()
		result := isSpecialIntegerPrefix(s)
		if result != resultExpected {
			t.Fatalf("unexpected result for isSpecialIntegerPrefix(%q); got %v; want %v", s, result, resultExpected)
		}
	}
	f("", false)
	f("1", false)
	f("0", false)

	// octal numbers
	f("03", true)
	f("0o1", true)
	f("0O12", true)

	// binary numbers
	f("0b1110", true)
	f("0B0", true)

	// hex number
	f("0x1ffa", true)
	f("0X4", true)
}

func TestUnescapeIdent(t *testing.T) {
	f := func(s, resultExpected string) {
		t.Helper()
		result := unescapeIdent(s)
		if result != resultExpected {
			t.Fatalf("unexpected result for unescapeIdent(%q); got %q; want %q", s, result, resultExpected)
		}
	}
	f("", "")
	f("a", "a")
	f("\\", "")
	f(`\\`, `\`)
	f(`\foo\-bar`, `foo-bar`)
	f(`a\\\\b\"c\d`, `a\\b"cd`)
	f(`foo.bar:baz_123`, `foo.bar:baz_123`)
	f(`foo\ bar`, `foo bar`)
	f(`\x21`, `!`)
	f(`\xeDfoo\x2Fbar\-\xqw\x`, "\xedfoo\x2fbar-xqwx")
	f(`\п\р\и\в\е\т123`, "привет123")
}

func TestAppendEscapedIdent(t *testing.T) {
	f := func(s, resultExpected string) {
		t.Helper()
		result := appendEscapedIdent(nil, s)
		if string(result) != resultExpected {
			t.Fatalf("unexpected result for appendEscapedIdent(%q); got %q; want %q", s, result, resultExpected)
		}
	}
	f(`a`, `a`)
	f(`a.b:c_23`, `a.b:c_23`)
	f(`a b-cd+dd\`, `a\ b\-cd\+dd\\`)
	f("a\x1E\x20\xee", `a\x1e\ \xee`)
	f("\x2e\x2e", `\x2e.`)
	f("привет123", `\п\р\и\в\е\т123`)
}

func TestScanIdent(t *testing.T) {
	f := func(s, resultExpected string) {
		t.Helper()
		result := scanIdent(s)
		if result != resultExpected {
			t.Fatalf("unexpected result for scanIdent(%q): got %q; want %q", s, result, resultExpected)
		}
	}
	f("a", "a")
	f("foo.bar:baz_123", "foo.bar:baz_123")
	f("a+b", "a")
	f("foo()", "foo")
	f(`a\-b+c`, `a\-b`)
	f(`a\ b\\\ c\`, `a\ b\\\ c\`)
	f(`\п\р\и\в\е\т123`, `\п\р\и\в\е\т123`)
}

func TestLexerNextPrev(t *testing.T) {
	var lex lexer
	lex.Init("foo bar baz")
	if lex.Token != "" {
		t.Fatalf("unexpected token got: %q; want %q", lex.Token, "")
	}
	if err := lex.Next(); err != nil {
		t.Fatalf("unexpeted error: %s", err)
	}
	if lex.Token != "foo" {
		t.Fatalf("unexpected token got: %q; want %q", lex.Token, "foo")
	}

	// Rewind before the first item.
	lex.Prev()
	if lex.Token != "" {
		t.Fatalf("unexpected token got: %q; want %q", lex.Token, "")
	}
	if err := lex.Next(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if lex.Token != "foo" {
		t.Fatalf("unexpected token got: %q; want %q", lex.Token, "foo")
	}
	if err := lex.Next(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if lex.Token != "bar" {
		t.Fatalf("unexpected token got: %q; want %q", lex.Token, "bar")
	}

	// Rewind to the first item.
	lex.Prev()
	if lex.Token != "foo" {
		t.Fatalf("unexpected token got: %q; want %q", lex.Token, "foo")
	}
	if err := lex.Next(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if lex.Token != "bar" {
		t.Fatalf("unexpected token got: %q; want %q", lex.Token, "bar")
	}
	if err := lex.Next(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if lex.Token != "baz" {
		t.Fatalf("unexpected token got: %q; want %q", lex.Token, "baz")
	}

	// Go beyond the token stream.
	if err := lex.Next(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if lex.Token != "" {
		t.Fatalf("unexpected token got: %q; want %q", lex.Token, "")
	}
	if !isEOF(lex.Token) {
		t.Fatalf("expecting eof")
	}
	lex.Prev()
	if lex.Token != "baz" {
		t.Fatalf("unexpected token got: %q; want %q", lex.Token, "baz")
	}

	// Go multiple times lex.Next() beyond token stream.
	if err := lex.Next(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if lex.Token != "" {
		t.Fatalf("unexpected token got: %q; want %q", lex.Token, "")
	}
	if !isEOF(lex.Token) {
		t.Fatalf("expecting eof")
	}
	if err := lex.Next(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if lex.Token != "" {
		t.Fatalf("unexpected token got: %q; want %q", lex.Token, "")
	}
	if !isEOF(lex.Token) {
		t.Fatalf("expecting eof")
	}
	lex.Prev()
	if lex.Token != "" {
		t.Fatalf("unexpected token got: %q; want %q", lex.Token, "")
	}
	if !isEOF(lex.Token) {
		t.Fatalf("expecting eof")
	}
}

func TestLexerSuccess(t *testing.T) {
	var s string
	var expectedTokens []string

	// An empty string
	s = ""
	expectedTokens = nil
	testLexerSuccess(t, s, expectedTokens)

	// String with whitespace
	s = "  \n\t\r "
	expectedTokens = nil
	testLexerSuccess(t, s, expectedTokens)

	// Just metric name
	s = "metric"
	expectedTokens = []string{"metric"}
	testLexerSuccess(t, s, expectedTokens)

	// Metric name with spec chars
	s = ":foo.bar_"
	expectedTokens = []string{":foo.bar_"}
	testLexerSuccess(t, s, expectedTokens)

	// Metric name with window
	s = "metric[5m]  "
	expectedTokens = []string{"metric", "[", "5m", "]"}
	testLexerSuccess(t, s, expectedTokens)

	// Metric name with tag filters
	s = `  metric:12.34{a="foo", b != "bar", c=~ "x.+y", d !~ "zzz"}`
	expectedTokens = []string{`metric:12.34`, `{`, `a`, `=`, `"foo"`, `,`, `b`, `!=`, `"bar"`, `,`, `c`, `=~`, `"x.+y"`, `,`, `d`, `!~`, `"zzz"`, `}`}
	testLexerSuccess(t, s, expectedTokens)

	// Metric name with offset
	s = `   metric offset 10d   `
	expectedTokens = []string{`metric`, `offset`, `10d`}
	testLexerSuccess(t, s, expectedTokens)

	// Func call
	s = `sum  (  metric{x="y"  }  [5m] offset 10h)`
	expectedTokens = []string{`sum`, `(`, `metric`, `{`, `x`, `=`, `"y"`, `}`, `[`, `5m`, `]`, `offset`, `10h`, `)`}
	testLexerSuccess(t, s, expectedTokens)

	// Binary op
	s = `a+b or c % d and e unless f`
	expectedTokens = []string{`a`, `+`, `b`, `or`, `c`, `%`, `d`, `and`, `e`, `unless`, `f`}
	testLexerSuccess(t, s, expectedTokens)

	// Numbers
	s = `3+1.2-.23+4.5e5-78e-6+1.24e+45-NaN+Inf`
	expectedTokens = []string{`3`, `+`, `1.2`, `-`, `.23`, `+`, `4.5e5`, `-`, `78e-6`, `+`, `1.24e+45`, `-`, `NaN`, `+`, `Inf`}
	testLexerSuccess(t, s, expectedTokens)

	s = `12.34 * 0X34 + 0b11 + 0O77`
	expectedTokens = []string{`12.34`, `*`, `0X34`, `+`, `0b11`, `+`, `0O77`}
	testLexerSuccess(t, s, expectedTokens)

	// Strings
	s = `""''` + "``" + `"\\"  '\\'  "\"" '\''"\\\"\\"`
	expectedTokens = []string{`""`, `''`, "``", `"\\"`, `'\\'`, `"\""`, `'\''`, `"\\\"\\"`}
	testLexerSuccess(t, s, expectedTokens)

	// Various durations
	s = `m offset 123h`
	expectedTokens = []string{`m`, `offset`, `123h`}
	testLexerSuccess(t, s, expectedTokens)

	s = `m offset -1.23w-5h34.5m - 123`
	expectedTokens = []string{`m`, `offset`, `-`, `1.23w-5h34.5m`, `-`, `123`}
	testLexerSuccess(t, s, expectedTokens)

	s = "   `foo\\\\\\`бар`  "
	expectedTokens = []string{"`foo\\\\\\`бар`"}
	testLexerSuccess(t, s, expectedTokens)

	s = `# comment # sdf
		foobar # comment
		baz
		# yet another comment`
	expectedTokens = []string{"foobar", "baz"}
	testLexerSuccess(t, s, expectedTokens)
}

func testLexerSuccess(t *testing.T, s string, expectedTokens []string) {
	t.Helper()

	var lex lexer
	lex.Init(s)

	var tokens []string
	for {
		if err := lex.Next(); err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if isEOF(lex.Token) {
			break
		}
		tokens = append(tokens, lex.Token)
	}
	if !reflect.DeepEqual(tokens, expectedTokens) {
		t.Fatalf("unexected tokens\ngot\n%q\nwant\n%q", tokens, expectedTokens)
	}
}

func TestLexerError(t *testing.T) {
	// Invalid identifier
	testLexerError(t, ".foo")

	// Incomplete string
	testLexerError(t, `"foobar`)
	testLexerError(t, `'`)
	testLexerError(t, "`")

	// Unrecognized char
	testLexerError(t, "тест")

	// Invalid numbers
	testLexerError(t, `.`)
	testLexerError(t, `123.`)
	testLexerError(t, `12e`)
	testLexerError(t, `1.2e`)
	testLexerError(t, `1.2E+`)
	testLexerError(t, `1.2E-`)
}

func testLexerError(t *testing.T, s string) {
	t.Helper()

	var lex lexer
	lex.Init(s)
	for {
		if err := lex.Next(); err != nil {
			// Expected error
			break
		}
		if isEOF(lex.Token) {
			t.Fatalf("expecting error during parse")
		}
	}

	// Try calling Next again. It must return error.
	if err := lex.Next(); err == nil {
		t.Fatalf("expecting non-nil error")
	}
}

func TestPositiveDurationSuccess(t *testing.T) {
	f := func(s string, step, expectedD int64) {
		t.Helper()
		d, err := PositiveDurationValue(s, step)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if d != expectedD {
			t.Fatalf("unexpected duration; got %d; want %d", d, expectedD)
		}
	}

	// Integer durations
	f("123ms", 42, 123)
	f("123s", 42, 123*1000)
	f("123m", 42, 123*60*1000)
	f("1h", 42, 1*60*60*1000)
	f("2d", 42, 2*24*60*60*1000)
	f("3w", 42, 3*7*24*60*60*1000)
	f("4y", 42, 4*365*24*60*60*1000)
	f("1i", 42*1000, 42*1000)
	f("3i", 42, 3*42)

	// Float durations
	f("123.45ms", 42, 123)
	f("0.234s", 42, 234)
	f("1.5s", 42, 1.5*1000)
	f("1.5m", 42, 1.5*60*1000)
	f("1.2h", 42, 1.2*60*60*1000)
	f("1.1d", 42, 1.1*24*60*60*1000)
	f("1.1w", 42, 1.1*7*24*60*60*1000)
	f("1.3y", 42, 1.3*365*24*60*60*1000)
	f("0.1i", 12340, 0.1*12340)
}

func TestPositiveDurationError(t *testing.T) {
	f := func(s string) {
		t.Helper()
		d, err := PositiveDurationValue(s, 42)
		if err == nil {
			t.Fatalf("expecting non-nil error for duration %q", s)
		}
		if d != 0 {
			t.Fatalf("expecting zero duration; got %d", d)
		}
	}
	f("")
	f("foo")
	f("m")
	f("12")
	f("1.23")
	f("1.23mm")
	f("123q")
	f("-123s")

	// Too big duration
	f("10000000000y")
}

func TestDurationSuccess(t *testing.T) {
	f := func(s string, step, expectedD int64) {
		t.Helper()
		d, err := DurationValue(s, step)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}
		if d != expectedD {
			t.Fatalf("unexpected duration; got %d; want %d", d, expectedD)
		}
	}

	// Integer durations
	f("123ms", 42, 123)
	f("-123ms", 42, -123)
	f("123s", 42, 123*1000)
	f("-123s", 42, -123*1000)
	f("123m", 42, 123*60*1000)
	f("1h", 42, 1*60*60*1000)
	f("2d", 42, 2*24*60*60*1000)
	f("3w", 42, 3*7*24*60*60*1000)
	f("4y", 42, 4*365*24*60*60*1000)
	f("1i", 42*1000, 42*1000)
	f("3i", 42, 3*42)
	f("-3i", 42, -3*42)
	f("1m34s24ms", 42, 94024)
	f("1m-34s24ms", 42, 25976)
	f("-1m34s24ms", 42, -94024)
	f("-1m-34s24ms", 42, -94024)

	// Float durations
	f("34.54ms", 42, 34)
	f("-34.34ms", 42, -34)
	f("0.234s", 42, 234)
	f("-0.234s", 42, -234)
	f("1.5s", 42, 1.5*1000)
	f("1.5m", 42, 1.5*60*1000)
	f("1.2h", 42, 1.2*60*60*1000)
	f("1.1d", 42, 1.1*24*60*60*1000)
	f("1.1w", 42, 1.1*7*24*60*60*1000)
	f("1.3y", 42, 1.3*365*24*60*60*1000)
	f("-1.3y", 42, -1.3*365*24*60*60*1000)
	f("0.1i", 12340, 0.1*12340)
	f("1.5m3.4s2.4ms", 42, 93402)
	f("-1.5m3.4s2.4ms", 42, -93402)
}

func TestDurationError(t *testing.T) {
	f := func(s string) {
		t.Helper()
		d, err := DurationValue(s, 42)
		if err == nil {
			t.Fatalf("expecting non-nil error for duration %q", s)
		}
		if d != 0 {
			t.Fatalf("expecting zero duration; got %d", d)
		}
	}
	f("")
	f("foo")
	f("m")
	f("12")
	f("1.23")
	f("1.23mm")
	f("123q")
	f("-123q")
}
