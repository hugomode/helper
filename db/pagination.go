package db

import (
	"context"
	"errors"
	"sync"

	"github.com/hugomode/helper/logger"
	"gorm.io/gorm"
)

// GetDataQueryPagination applies standard GORM pagination (LIMIT and OFFSET) to a query.
func GetDataQueryPagination(pageSize, pageNumber uint, dest any, tx *gorm.DB) error {
	if pageSize == 0 {
		return errors.New("pageSize must be greater than 0")
	}
	if pageNumber == 0 {
		pageNumber = 1
	}

	offset := int((pageNumber - 1) * pageSize)
	query := tx.Offset(offset).Limit(int(pageSize))

	return RunOneQuery(dest, query)
}

// RunOneQuery executes a GORM query and handles the record not found error.
func RunOneQuery(dest any, tx *gorm.DB) error {
	err := tx.Find(dest).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		logger.Log.Sugar().Errorf("Error executing query: %v", err)
		return err
	}
	return nil
}

// RunQueriesInParallel executes two query functions in parallel (typically data and count) using goroutines.
// It supports context cancellation and collects the first error encountered.
func RunQueriesInParallel(
	ctx context.Context,
	queryDataFunc func() error,
	queryCountFunc func() error,
) error {
	var wg sync.WaitGroup
	errs := make(chan error, 2)

	wg.Add(2)

	go func() {
		defer wg.Done()
		if err := queryDataFunc(); err != nil {
			errs <- err
		}
	}()

	go func() {
		defer wg.Done()
		if err := queryCountFunc(); err != nil {
			errs <- err
		}
	}()

	// Usamos un canal para detectar si el contexto se cancela prematuramente
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		close(errs)
	}

	// Recolectamos el primer error encontrado, si hubiera
	for err := range errs {
		if err != nil {
			return err
		}
	}

	return nil
}
