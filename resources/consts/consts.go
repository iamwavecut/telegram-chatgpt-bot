package consts

import (
	"time"
)

const (
	DurationRetryRequest = 5 * time.Second

	IntRetryAttempts = -1

	MinTimeBetweenRequests = 6 * time.Second
)
