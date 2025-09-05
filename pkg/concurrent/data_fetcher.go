package concurrent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/helloworlde/miwifi-exporter/internal/models"
)

// DataFetcher handles concurrent data fetching from router
type DataFetcher struct {
	timeout      time.Duration
	maxRetries   int
	retryDelay   time.Duration
}

// NewDataFetcher creates a new data fetcher
func NewDataFetcher(timeout time.Duration, maxRetries int, retryDelay time.Duration) *DataFetcher {
	return &DataFetcher{
		timeout:    timeout,
		maxRetries: maxRetries,
		retryDelay: retryDelay,
	}
}

// FetchData fetches all router data concurrently
func (df *DataFetcher) FetchData(ctx context.Context, client RouterClient) (*RouterData, error) {
	ctx, cancel := context.WithTimeout(ctx, df.timeout)
	defer cancel()
	
	// Create tasks for concurrent execution
	tasks := []Task{
		{
			ID: 0,
			Work: func() (interface{}, error) {
				return df.fetchWithRetry(ctx, func() (interface{}, error) {
					return client.GetSystemStatus(ctx)
				})
			},
		},
		{
			ID: 1,
			Work: func() (interface{}, error) {
				return df.fetchWithRetry(ctx, func() (interface{}, error) {
					return client.GetDeviceList(ctx)
				})
			},
		},
		{
			ID: 2,
			Work: func() (interface{}, error) {
				return df.fetchWithRetry(ctx, func() (interface{}, error) {
					return client.GetWanInfo(ctx)
				})
			},
		},
		{
			ID: 3,
			Work: func() (interface{}, error) {
				return df.fetchWithRetry(ctx, func() (interface{}, error) {
					return client.GetWifiDetails(ctx)
				})
			},
		},
	}
	
	// Execute tasks concurrently
	results, err := ExecuteWithTimeout(ctx, tasks, df.timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data concurrently: %w", err)
	}
	
	// Process results
	data := &RouterData{}
	var firstError error
	
	for _, result := range results {
		if result.Error != nil {
			if firstError == nil {
				firstError = result.Error
			}
			continue
		}
		
		switch result.ID {
		case 0:
			if status, ok := result.Value.(*models.SystemStatus); ok {
				data.SystemStatus = status
			}
		case 1:
			if devices, ok := result.Value.(*models.DeviceList); ok {
				data.DeviceList = devices
			}
		case 2:
			if wan, ok := result.Value.(*models.WanInfo); ok {
				data.WanInfo = wan
			}
		case 3:
			if wifi, ok := result.Value.(*models.WifiDetailAll); ok {
				data.WifiDetails = wifi
			}
		}
	}
	
	if firstError != nil {
		return data, fmt.Errorf("some data fetches failed: %w", firstError)
	}
	
	return data, nil
}

// fetchWithRetry fetches data with retry logic
func (df *DataFetcher) fetchWithRetry(ctx context.Context, fetchFunc func() (interface{}, error)) (interface{}, error) {
	var lastError error
	
	for i := 0; i < df.maxRetries; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			result, err := fetchFunc()
			if err == nil {
				return result, nil
			}
			
			lastError = err
			
			// If this is the last retry, don't wait
			if i == df.maxRetries-1 {
				break
			}
			
			// Wait before retrying
			select {
			case <-time.After(df.retryDelay):
				continue
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}
	
	return nil, lastError
}

// RouterClient defines the interface for router data fetching
type RouterClient interface {
	GetSystemStatus(ctx context.Context) (*models.SystemStatus, error)
	GetDeviceList(ctx context.Context) (*models.DeviceList, error)
	GetWanInfo(ctx context.Context) (*models.WanInfo, error)
	GetWifiDetails(ctx context.Context) (*models.WifiDetailAll, error)
}

// RouterData contains all router data
type RouterData struct {
	SystemStatus *models.SystemStatus
	DeviceList   *models.DeviceList
	WanInfo      *models.WanInfo
	WifiDetails  *models.WifiDetailAll
}

// FetchResult represents the result of a fetch operation
type FetchResult struct {
	Data      *RouterData
	Duration  time.Duration
	TimedOut  bool
	Errors    []error
}

// TimedFetch performs a timed fetch operation
func (df *DataFetcher) TimedFetch(ctx context.Context, client RouterClient) *FetchResult {
	start := time.Now()
	
	data, err := df.FetchData(ctx, client)
	duration := time.Since(start)
	
	result := &FetchResult{
		Data:     data,
		Duration: duration,
		TimedOut: false,
	}
	
	if err != nil {
		result.Errors = []error{err}
		if ctx.Err() == context.DeadlineExceeded {
			result.TimedOut = true
		}
	}
	
	return result
}

// ParallelFetcher handles parallel fetching with progress tracking
type ParallelFetcher struct {
	fetcher    *DataFetcher
	progress   *FetchProgress
	mu         sync.Mutex
}

// FetchProgress tracks fetch progress
type FetchProgress struct {
	Started     time.Time
	Completed   time.Time
	TotalTasks  int
	CompletedTasks int
	FailedTasks int
	CurrentTask string
}

// NewParallelFetcher creates a new parallel fetcher
func NewParallelFetcher(timeout time.Duration, maxRetries int, retryDelay time.Duration) *ParallelFetcher {
	return &ParallelFetcher{
		fetcher:  NewDataFetcher(timeout, maxRetries, retryDelay),
		progress: &FetchProgress{},
	}
}

// FetchWithProgress fetches data with progress tracking
func (pf *ParallelFetcher) FetchWithProgress(ctx context.Context, client RouterClient) (*FetchResult, *FetchProgress) {
	pf.progress.Started = time.Now()
	pf.progress.TotalTasks = 4
	
	result := pf.fetcher.TimedFetch(ctx, client)
	pf.progress.Completed = time.Now()
	
	// Update progress based on result
	if result.Data != nil {
		completed := 0
		if result.Data.SystemStatus != nil {
			completed++
		}
		if result.Data.DeviceList != nil {
			completed++
		}
		if result.Data.WanInfo != nil {
			completed++
		}
		if result.Data.WifiDetails != nil {
			completed++
		}
		pf.progress.CompletedTasks = completed
		pf.progress.FailedTasks = 4 - completed
	}
	
	return result, pf.progress
}