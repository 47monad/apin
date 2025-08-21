package pginitr_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/47monad/apin/initrs/pginitr"
	"github.com/47monad/zaal"
)

func TestInit(t *testing.T) {
	config := &zaal.PostgresConfig{
		URI:      "postgres://localhost:5432/settings",
		Username: "postgres",
		Password: "123456789",
	}

	shell, err := pginitr.New(context.Background(), pginitr.Opts().WithConfig(config))
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("connection done")
		shell.Conn.Close(context.Background())
	}
}
