package db

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetDataQueryPagination_pageSize_zero(t *testing.T) {
	err := GetDataQueryPagination(0, 1, nil, nil)
	assert.Error(t, err)
	assert.Equal(t, "pageSize must be greater than 0", err.Error())
}

func TestRunQueriesInParallel(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		err := RunQueriesInParallel(ctx,
			func() error { return nil },
			func() error { return nil },
		)
		assert.NoError(t, err)
	})

	t.Run("Error in data func", func(t *testing.T) {
		expectedErr := errors.New("data error")
		err := RunQueriesInParallel(ctx,
			func() error { return expectedErr },
			func() error { return nil },
		)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})

	t.Run("Error in count func", func(t *testing.T) {
		expectedErr := errors.New("count error")
		err := RunQueriesInParallel(ctx,
			func() error { return nil },
			func() error { return expectedErr },
		)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})

	t.Run("Context cancellation", func(t *testing.T) {
		cancelCtx, cancel := context.WithCancel(ctx)
		cancel() // Cancel immediately

		err := RunQueriesInParallel(cancelCtx,
			func() error {
				time.Sleep(100 * time.Millisecond)
				return nil
			},
			func() error {
				time.Sleep(100 * time.Millisecond)
				return nil
			},
		)
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})
}
