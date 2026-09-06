package mongoinitr_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/47monad/apin/initrs/mongoinitr"
	"go.mongodb.org/mongo-driver/v2/x/mongo/driver/topology"
)

func TestNewPreservesPingError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := mongoinitr.New(ctx, mongoinitr.Opts().SetURI(
		"mongodb://127.0.0.1:1/?serverSelectionTimeoutMS=20",
	))
	if err == nil {
		t.Fatal("New() error = nil, want a server selection error")
	}

	var selectionErr topology.ServerSelectionError
	if !errors.As(err, &selectionErr) {
		t.Fatalf("errors.As(%T, *topology.ServerSelectionError) = false: %v", err, err)
	}
}
