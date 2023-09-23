package retrying

import (
	"errors"
	"testing"
	"time"
)

func TestRetryForever(t *testing.T) {
	t.Run("retry forever until no error", func(t *testing.T) {
		executeCount := 0
		RetryForever(func() error {
			executeCount++
			if executeCount == 10 {
				return nil
			}
			return errors.New("fail")
		}, time.Millisecond*10, func(retryCount int, err error) bool {
			if retryCount != executeCount-1 {
				t.Errorf("retryCount: expected = %d, actual = %d", executeCount-1, retryCount)
			}
			return true
		})
	})
}

func TestRetryWithResultForever(t *testing.T) {
	t.Run("retry forever until no error", func(t *testing.T) {
		executeCount := 0
		RetryWithResultForever[string](func() (string, error) {
			executeCount++
			if executeCount == 10 {
				return "hello", nil
			}
			return "", errors.New("fail")
		}, time.Millisecond*10, func(retryCount int, err error) bool {
			if retryCount != executeCount-1 {
				t.Errorf("retryCount: expected = %d, actual = %d", executeCount-1, retryCount)
			}
			return true
		})
	})
}

func TestRetryWithStatusForever(t *testing.T) {
	t.Run("retry forever until no error", func(t *testing.T) {
		executeCount := 0
		RetryWithStatusForever[string, string](func(status string) (string, error) {
			executeCount++
			if executeCount == 10 {
				return "hello", nil
			}
			return "", errors.New("fail")
		}, "status", time.Millisecond*10, func(retryCount int, err error) bool {
			if retryCount != executeCount-1 {
				t.Errorf("retryCount: expected = %d, actual = %d", executeCount-1, retryCount)
			}
			return true
		})
	})
}

func TestRetry(t *testing.T) {
	t.Run("action is null should panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("action is null, but not panic")
			}
		}()
		Retry(nil, time.Millisecond*10, nil, 10)
	})

	t.Run("exit after maxRetryCount is 10", func(t *testing.T) {
		executeCount := 0
		Retry(func() error {
			executeCount++
			return errors.New("fail")
		}, time.Millisecond*10, func(retryCount int, err error) bool {
			if retryCount != executeCount-1 {
				t.Errorf("retryCount: expected = %d, actual = %d", executeCount-1, retryCount)
			}
			return true
		}, 10)
	})

	t.Run("exit after maxRetryCount is 0", func(t *testing.T) {
		executeCount := 0
		Retry(func() error {
			executeCount++
			return errors.New("fail")
		}, time.Millisecond*10, func(retryCount int, err error) bool {
			if retryCount != executeCount-1 {
				t.Errorf("retryCount: expected = %d, actual = %d", executeCount-1, retryCount)
			}
			return true
		}, 0)
	})

	t.Run("retry forever until no error", func(t *testing.T) {
		executeCount := 0
		Retry(func() error {
			executeCount++
			if executeCount == 10 {
				return nil
			}
			return errors.New("fail")
		}, time.Millisecond*10, func(retryCount int, err error) bool {
			if retryCount != executeCount-1 {
				t.Errorf("retryCount: expected = %d, actual = %d", executeCount-1, retryCount)
			}
			return true
		}, -1)
	})
}

func TestRetryWithResult(t *testing.T) {
	t.Run("action is null should panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("action is null, but not panic")
			}
		}()
		RetryWithResult[string](nil, time.Millisecond*10, nil, 10)
	})

	t.Run("exit after maxRetryCount is 10", func(t *testing.T) {
		executeCount := 0
		RetryWithResult[string](func() (string, error) {
			executeCount++
			return "", errors.New("fail")
		}, time.Millisecond*10, func(retryCount int, err error) bool {
			if retryCount != executeCount-1 {
				t.Errorf("retryCount: expected = %d, actual = %d", executeCount-1, retryCount)
			}
			return true
		}, 10)
	})

	t.Run("exit after maxRetryCount is 0", func(t *testing.T) {
		executeCount := 0
		RetryWithResult[string](func() (string, error) {
			executeCount++
			return "", errors.New("fail")
		}, time.Millisecond*10, func(retryCount int, err error) bool {
			if retryCount != executeCount-1 {
				t.Errorf("retryCount: expected = %d, actual = %d", executeCount-1, retryCount)
			}
			return true
		}, 0)
	})

	t.Run("retry forever until no error", func(t *testing.T) {
		executeCount := 0
		RetryWithResult[string](func() (string, error) {
			executeCount++
			if executeCount == 10 {
				return "hello", nil
			}
			return "", errors.New("fail")
		}, time.Millisecond*10, func(retryCount int, err error) bool {
			if retryCount != executeCount-1 {
				t.Errorf("retryCount: expected = %d, actual = %d", executeCount-1, retryCount)
			}
			return true
		}, -1)
	})
}

func TestRetryWithStatus(t *testing.T) {
	t.Run("action is null should panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("action is null, but not panic")
			}
		}()
		RetryWithStatus[string, string](nil, "status", time.Millisecond*10, nil, 10)
	})

	t.Run("exit after maxRetryCount is 10", func(t *testing.T) {
		executeCount := 0
		RetryWithStatus[string, string](func(status string) (string, error) {
			executeCount++
			return "", errors.New("fail")
		}, "status", time.Millisecond*10, func(retryCount int, err error) bool {
			if retryCount != executeCount-1 {
				t.Errorf("retryCount: expected = %d, actual = %d", executeCount-1, retryCount)
			}
			return true
		}, 10)
	})

	t.Run("exit after maxRetryCount is 0", func(t *testing.T) {
		executeCount := 0
		RetryWithStatus[string, string](func(status string) (string, error) {
			executeCount++
			return "", errors.New("fail")
		}, "status", time.Millisecond*10, func(retryCount int, err error) bool {
			if retryCount != executeCount-1 {
				t.Errorf("retryCount: expected = %d, actual = %d", executeCount-1, retryCount)
			}
			return true
		}, 0)
	})

	t.Run("retry forever until no error", func(t *testing.T) {
		executeCount := 0
		RetryWithStatus[string, string](func(status string) (string, error) {
			executeCount++
			if executeCount == 10 {
				return "hello", nil
			}
			return "", errors.New("fail")
		}, "status", time.Millisecond*10, func(retryCount int, err error) bool {
			if retryCount != executeCount-1 {
				t.Errorf("retryCount: expected = %d, actual = %d", executeCount-1, retryCount)
			}
			return true
		}, -1)
	})
}
