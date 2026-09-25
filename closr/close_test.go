package closr_test

import (
	"context"
	"errors"
	"testing"

	"github.com/47monad/apin/closr"
)

// stubCloser records the context it was closed with and returns a canned
// result.
type stubCloser struct {
	err       error
	calls     int
	gotCtx    context.Context
	ctxWasNil bool
}

func (s *stubCloser) Close(ctx context.Context) error {
	s.calls++
	s.gotCtx = ctx
	s.ctxWasNil = ctx == nil
	return s.err
}

func TestClose(t *testing.T) {
	t.Run("returns closer error", func(t *testing.T) {
		want := errors.New("close failed")
		closer := &stubCloser{err: want}

		if err := closr.Close(closer); !errors.Is(err, want) {
			t.Fatalf("Close() error = %v, want %v", err, want)
		}
		if closer.calls != 1 {
			t.Errorf("Close() called %d times, want 1", closer.calls)
		}
		if closer.ctxWasNil {
			t.Error("Close() passed a nil context, want context.Background()")
		}
	})

	t.Run("passes WithContext", func(t *testing.T) {
		type ctxKey struct{}
		ctx := context.WithValue(context.Background(), ctxKey{}, "value")
		closer := &stubCloser{}

		if err := closr.Close(closer, closr.WithContext(ctx)); err != nil {
			t.Fatalf("Close() error = %v, want nil", err)
		}
		if closer.gotCtx != ctx {
			t.Error("Close() did not pass the context from WithContext")
		}
	})
}

func TestMustClose(t *testing.T) {
	t.Run("returns normally on success", func(t *testing.T) {
		closer := &stubCloser{}

		closr.MustClose(closer)

		if closer.calls != 1 {
			t.Errorf("MustClose() called %d times, want 1", closer.calls)
		}
	})

	t.Run("panics on error", func(t *testing.T) {
		want := errors.New("close failed")
		closer := &stubCloser{err: want}

		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("MustClose() did not panic on close error")
			}
			err, ok := r.(error)
			if !ok {
				t.Fatalf("MustClose() panicked with %T (%v), want error", r, r)
			}
			if !errors.Is(err, want) {
				t.Errorf("MustClose() panicked with %v, want %v", err, want)
			}
		}()

		closr.MustClose(closer)
	})
}
