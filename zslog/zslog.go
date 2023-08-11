//go:build go1.21

package zslog

import (
	"context"
	"log/slog"
	"slices"
	"time"

	"github.com/rs/zerolog"
)

var _ slog.Handler = &Handler{}

type Handler struct {
	zlog          zerolog.Logger
	groupAndAttrs []groupAndAttrs
}

func NewHandler(zlog zerolog.Logger) slog.Handler {
	return &Handler{
		zlog:          zlog,
		groupAndAttrs: []groupAndAttrs{{}},
	}
}

type groupAndAttrs struct {
	group string
	attrs []slog.Attr
}

func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return int(level) > int((h.zlog.GetLevel()-1)*4)
}

func (h *Handler) WithGroup(group string) slog.Handler {
	gaa := slices.Clone(h.groupAndAttrs)
	gaa = append(gaa, groupAndAttrs{group: group})
	return &Handler{
		zlog:          h.zlog,
		groupAndAttrs: gaa,
	}
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	gaa := slices.Clone(h.groupAndAttrs)
	gaa[len(gaa)-1].attrs = slices.Concat(gaa[len(gaa)-1].attrs, attrs)
	return &Handler{
		zlog:          h.zlog,
		groupAndAttrs: gaa,
	}
}

func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	z := h.zlog.WithLevel(zerolog.Level(min(max((r.Level/4)+1, -1), 5)))
	z = z.Time(zerolog.TimestampFieldName, r.Time)

	e := zerolog.Dict()
	for i := len(h.groupAndAttrs) - 1; i > 0; i-- {
		var added bool
		for _, a := range h.groupAndAttrs[i].attrs {
			var add bool
			e, add = addAttr(e, a)
			added = added || add
		}
		if i == len(h.groupAndAttrs)-1 {
			r.Attrs(func(a slog.Attr) bool {
				var add bool
				e, add = addAttr(e, a)
				added = added || add
				return true
			})
		}
		if added {
			if i > 1 {
				e = zerolog.Dict().Dict(h.groupAndAttrs[i].group, e)
			} else if i == 1 {
				z.Dict(h.groupAndAttrs[1].group, e)
			}
		}
	}
	if len(h.groupAndAttrs) == 1 {
		r.Attrs(func(a slog.Attr) bool {
			z, _ = addAttr(z, a)
			return true
		})
	}
	for _, a := range h.groupAndAttrs[0].attrs {
		z, _ = addAttr(z, a)
	}

	z.Msg(r.Message)
	return nil
}

func addAttr[Z zlogger[Z]](z Z, attr slog.Attr) (Z, bool) {
	if attr.Equal(slog.Attr{}) {
		return z, false
	}
	val := attr.Value.Resolve()
	switch val.Kind() {
	case slog.KindAny:
		return z.Interface(attr.Key, val.Any()), true
	case slog.KindBool:
		return z.Bool(attr.Key, val.Bool()), true
	case slog.KindDuration:
		return z.Dur(attr.Key, val.Duration()), true
	case slog.KindFloat64:
		return z.Float64(attr.Key, val.Float64()), true
	case slog.KindInt64:
		return z.Int64(attr.Key, val.Int64()), true
	case slog.KindString:
		return z.Str(attr.Key, val.String()), true
	case slog.KindTime:
		return z.Time(attr.Key, val.Time()), true
	case slog.KindUint64:
		return z.Uint64(attr.Key, val.Uint64()), true
	case slog.KindGroup:
		if attr.Key == "" {
			for _, a := range val.Group() {
				z, _ = addAttr(z, a)
			}
			return z, true
		}
		var added bool
		d := zerolog.Dict()
		for _, a := range val.Group() {
			var add bool
			d, add = addAttr[*zerolog.Event](d, a)
			added = added || add
		}
		if added {
			return z.Dict(attr.Key, d), true
		}
		return z, false
	}
	return z, false
}

type zlogger[Z any] interface {
	zerolog.Context | *zerolog.Event
	Interface(string, any) Z
	Bool(string, bool) Z
	Dur(string, time.Duration) Z
	Float64(string, float64) Z
	Int64(string, int64) Z
	Str(string, string) Z
	Time(string, time.Time) Z
	Uint64(string, uint64) Z
	Dict(string, *zerolog.Event) Z
}
