package zslog

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"os"
	"testing"
	"testing/slogtest"

	"github.com/rs/zerolog"
)

func TestSlog(t *testing.T) {
	buf := new(bytes.Buffer)
	z := zerolog.New(buf)
	h := NewHandler(z)
	err := slogtest.TestHandler(h, func() []map[string]any {
		d := json.NewDecoder(buf)
		var res []map[string]any
		for d.More() {
			var out map[string]any
			err := d.Decode(&out)
			if err != nil {
				t.Errorf("decode line: %v", err)
			}
			res = append(res, out)
		}
		return res
	})
	if err != nil {
		t.Error(err)
	}
}

func ExampleNewHandler() {
	l := slog.New(NewHandler(zerolog.New(os.Stdout)))
	l.Info("hello", "a", "aa")
	l.With("a", "aa").Info("hello", "b", "bb")
	l.WithGroup("1").With("a", "aa").Info("hello")
	l.WithGroup("1").Info("hello", "a", "aa")
	l.With("a", "aa").WithGroup("1").Info("hello", "b", "bb")
	l.With("a", "aa").WithGroup("1").With("b", "bb").Info("hello")
	l.With("a", "aa").WithGroup("1").WithGroup("2").With("b", "bb").Info("hello")
	l.WithGroup("1").WithGroup("2").WithGroup("3").Error("oops", "a", "aa")
	l.WithGroup("1").With("a", "aa").WithGroup("2").With("b", "bb", slog.Group("c", "d", "dd")).WithGroup("3").Info("hello again")
	l.WithGroup("1").WithGroup("2").Info("hello")
	l.With(slog.Group("1", slog.Group("2", slog.Group("3"))), "a", "aa").Info("hello")
}
