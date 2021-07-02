package health

import (
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
	"time"
)

func TestAggregateResult(t *testing.T) {
	// Arrange
	errMsg := "this is an error message"
	testData := map[string]checkStatus{
		"check1": {
			Status:    statusUp,
			Error:     nil,
			Timestamp: time.Now().Add(-5 * time.Minute),
		},
		"check2": {
			Status:    statusWarn,
			Error:     nil,
			Timestamp: time.Now().Add(-3 * time.Minute),
		},
		"check3": {
			Status:    statusDown,
			Error:     &errMsg,
			Timestamp: time.Now().Add(-1 * time.Minute),
		},
	}

	// Act
	result := aggregateStatus(testData)

	// Assert
	assert.Equal(t, statusDown, result.Status)
	assert.Equal(t, true, result.Timestamp.Equal(testData["check1"].Timestamp))
	assert.Equal(t, true, reflect.DeepEqual(testData, result.Checks))
	assert.Nil(t, result.Error)
}
