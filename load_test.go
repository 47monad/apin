package apin_test

import (
	"testing"

	"github.com/47monad/apin"
	"github.com/stretchr/testify/assert"
)

func TestLoad(t *testing.T) {
	// act
	app, err := apin.InitWithZaal("./config/writer/main.cue", "./config/writer/.env")

	// assert
	assert.NoError(t, err)
	assert.Equal(t, app.GetConfig().HTTP.Servers["main"].Port, 8888)
	assert.Equal(t, app.GetConfig().GRPC.Servers["main"].Port, 9999)
	assert.True(t, app.GetConfig().GRPC.Servers["main"].Features.Logging)
}
