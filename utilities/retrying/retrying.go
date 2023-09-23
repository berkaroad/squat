package retrying

import (
	"time"
)

type ActionFunc func() error
type ActionTResultFunc[TResult any] func() (TResult, error)
type ActionTTResultFunc[T, TResult any] func(status T) (TResult, error)
type FailFunc func(retryCount int, err error) bool

// Retry forever if action returns error.
func RetryForever(action ActionFunc, retryInterval time.Duration, onFailed FailFunc) {
	Retry(action, retryInterval, onFailed, -1)
}

// Retry forever if action returns error, action also returns TResult.
func RetryWithResultForever[TResult any](action ActionTResultFunc[TResult], retryInterval time.Duration, onFailed FailFunc) TResult {
	data, _ := RetryWithResult(action, retryInterval, onFailed, -1)
	return data
}

// Retry forever if action returns error, action with T also returns TResult.
func RetryWithStatusForever[T, TResult any](action ActionTTResultFunc[T, TResult], status T, retryInterval time.Duration, onFailed FailFunc) TResult {
	data, _ := RetryWithStatus(action, status, retryInterval, onFailed, -1)
	return data
}

// Retry again if action returns error.
func Retry(action ActionFunc, retryInterval time.Duration, onFailed FailFunc, maxRetryCount int) error {
	if action == nil {
		panic("param 'action' is null")
	}

	_, err := RetryWithStatus[any, any](func(status any) (any, error) {
		return nil, action()
	}, nil, retryInterval, onFailed, maxRetryCount)
	return err
}

// Retry again if action returns error, action also returns TResult.
func RetryWithResult[TResult any](action ActionTResultFunc[TResult], retryInterval time.Duration, onFailed FailFunc, maxRetryCount int) (TResult, error) {
	if action == nil {
		panic("param 'action' is null")
	}

	return RetryWithStatus[any, TResult](func(status any) (TResult, error) {
		return action()
	}, nil, retryInterval, onFailed, maxRetryCount)
}

// Retry again if action returns error, action with T also returns TResult.
func RetryWithStatus[T, TResult any](action ActionTTResultFunc[T, TResult], status T, retryInterval time.Duration, onFailed FailFunc, maxRetryCount int) (TResult, error) {
	if action == nil {
		panic("param 'action' is null")
	}

	retryCount := 0
	data, err := action(status)
	for err != nil {
		if onFailed != nil {
			if !onFailed(retryCount, err) {
				break
			}
		}

		retryCount++
		if maxRetryCount == 0 || (maxRetryCount > 0 && retryCount > maxRetryCount) {
			break
		}
		time.Sleep(retryInterval)
		data, err = action(status)
	}
	return data, err
}
