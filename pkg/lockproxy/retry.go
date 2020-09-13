package lockproxy

import (
	"time"

	"github.com/cenkalti/backoff"
)

func GetExponentialBackOff(initialInterval time.Duration, maxElapsedTime time.Duration) *backoff.ExponentialBackOff {
	randomizationFactor := backoff.DefaultRandomizationFactor
	multiplier := backoff.DefaultMultiplier
	maxInterval := backoff.DefaultMaxInterval
	clock := backoff.SystemClock

	if maxInterval > maxElapsedTime {
		maxInterval = maxElapsedTime
	}
	if initialInterval > maxElapsedTime {
		initialInterval = maxElapsedTime
	}

	b := &backoff.ExponentialBackOff{
		InitialInterval:     initialInterval,
		RandomizationFactor: randomizationFactor,
		Multiplier:          multiplier,
		MaxInterval:         maxInterval,
		MaxElapsedTime:      maxElapsedTime,
		Clock:               clock,
	}
	b.Reset()

	return b
}
