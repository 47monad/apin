package apin_test

import (
	"testing"

	"github.com/47monad/apin"
	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
	// act
	app, err := apin.Load(apin.WithAppDir("writer"))

	// assert
	assert.NoError(t, err)
	assert.Equal(t, app.GetConfig().Name, "TestWriterApp")
}
