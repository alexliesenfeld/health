package health

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetAuthWithResult(t *testing.T) {
	expected := true
	assert.Equal(t, &expected, getAuthResult(withAuthResult(context.Background(), expected)))
}

func TestGetAuthNoResult(t *testing.T) {
	assert.Nil(t, getAuthResult(context.Background()))
}
