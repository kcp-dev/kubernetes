package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/cockroachdb/apd"
)

func main() {
	var process func(input string) string
	switch os.Args[1]{
	case "encode":
		process = func(input string) string {
			rv, err := strconv.ParseInt(input, 10, 64)
			if err != nil {
				panic(err)
			}
			hlc, err := toHybridLogicalClock(rv)
			if err != nil {
				panic(err)
			}
			return hlc.String()
		}
	case "decode":
		process = func(input string) string {
			hlc, cond, err := apd.NewFromString(input)
			if err != nil {
				panic(fmt.Errorf("failed to determine value of logical clock: %v", err))
			}
			if _, err := cond.GoError(apd.DefaultTraps); err != nil {
				panic(fmt.Errorf("failed to determine value of logical clock: %v", err))
			}
			rv, err := toResourceVersion(hlc)
			if err != nil {
				panic(err)
			}
			return strconv.FormatInt(rv, 10)
		}
	}
	for _, arg := range os.Args[2:] {
		fmt.Printf("%s: %s\n", arg, process(arg))
	}
}


const (
	// datum is a time before anyone would have ever used this code, such that we can subtract it from
	// the timestamps in the CockroachDB Hybrid-Logical Clock to reduce the overall amount of data that
	// needs to be encoded, allowing us to pack more bits of the logical part of the clock into the
	// resourceVersion. This is equivalent to the date: 2022-01-01 00:00:00 +0000 UTC
	datum = int64(1640995200000000000)
	// numBits is the number of bits which we will use to pack logical clock values into. This operation
	// is only valid as long as the current time is small enough that the difference between it and the
	// datum leaves this many unused bits inside an int64. For 4 bits, this is valid until:
	// 2040-04-07 23:59:12.303423488 +0000 UTC
	numBits = 4
	// totalBits is the number of bits we have to work with before our int64 overflows
	totalBits = 63
)

func toResourceVersion(hybridLogicalTimestamp *apd.Decimal) (int64, error) {
	var integral, fractional apd.Decimal
	hybridLogicalTimestamp.Modf(&integral, &fractional)

	timestamp, err := integral.Int64()
	if err != nil {
		return 0, fmt.Errorf("failed to convert timestamp to integer: %v", err) // should never happen
	}
	timestamp -= datum
	if timestamp >= 1<<(totalBits-numBits) {
		return 0, fmt.Errorf("nanosecond timestamp in HLC too large to shift: %s (is it past 2040?)", hybridLogicalTimestamp.String())
	}
	timestamp = timestamp << numBits

	if fractional.IsZero() {
		// there is no logical portion to this clock
		return timestamp, nil
	}

	var logicalTimestamp apd.Decimal
	multiplier := apd.New(1, 10)
	condition, err := apd.BaseContext.Mul(&logicalTimestamp, &fractional, multiplier)
	if err != nil {
		return 0, fmt.Errorf("failed to determine value of logical clock: %v", err)
	}
	if _, err := condition.GoError(apd.DefaultTraps); err != nil {
		return 0, fmt.Errorf("failed to determine value of logical clock: %v", err)
	}

	counter, err := logicalTimestamp.Int64()
	if err != nil {
		return 0, fmt.Errorf("failed to convert logical timestamp to counter: %v", err)
	}

	if counter >= 1<<numBits {
		return 0, fmt.Errorf("logical clock in HLC too large to shift into timestamp: %s", hybridLogicalTimestamp.String())
	}

	return timestamp + counter, nil
}

// toHybridLogicalClock is the inverse of toResourceVersion
func toHybridLogicalClock(resourceVersion int64) (*apd.Decimal, error) {
	timestamp := resourceVersion>>numBits + datum
	logical := apd.New(timestamp, 0)

	counter := resourceVersion & ((1 << numBits) - 1) // mask out the lower bits
	if counter == 0 {
		return logical, nil
	}
	fractional := apd.New(counter, -10)

	var hybridLogicalTimestamp apd.Decimal
	condition, err := apd.BaseContext.Add(&hybridLogicalTimestamp, logical, fractional)
	if err != nil {
		return nil, fmt.Errorf("failed to determine value of hybrid logical clock: %v", err)
	}
	if _, err := condition.GoError(apd.DefaultTraps); err != nil {
		return nil, fmt.Errorf("failed to determine value of hybrid logical clock: %v", err)
	}
	return &hybridLogicalTimestamp, nil
}