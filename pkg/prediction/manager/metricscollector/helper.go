package metricscollector

import (
	"fmt"
	"regexp"
	"strconv"
)

const (
	MemoryKiB = "Ki"
	MemoryMiB = "Mi"
	MemoryGiB = "Gi"
	MemoryB   = "B"
	CPUns     = "n"
	CPUus     = "u"
	CPUms     = "m"
)

// Return CPU xxx core
// Return Memory xxx Bytes
func ParseResourceUsage(input, types string) (float64, string, error) {
	re := regexp.MustCompile(`^([0-9.]+)([A-Za-z]*)$`)
	matches := re.FindStringSubmatch(input)

	if len(matches) < 2 {
		return 0, "", fmt.Errorf("failed to parse resources")
	}
	numberPart := matches[1]

	number, err := strconv.ParseFloat(numberPart, 64)
	if err != nil {
		return 0, "", fmt.Errorf("failed to parse number part: %v", err)
	}

	if len(matches) == 3 {
		letterPart := matches[2]
		switch letterPart {
		case MemoryB:
			number = (number)
		case MemoryKiB:
			number = (number * 1024)
		case MemoryGiB:
			number = number * 1024 * 1024 * 1024
		case CPUns:
			number = (number / 1000 / 1000 / 1000)
		case CPUus:
			number = (number / 1000 / 1000)
		case MemoryMiB:
			number = number * 1024 * 1024
		case CPUms:
			number = (number / 1000)
		default:
			return 0, "", fmt.Errorf("invalid metrics info %s", input)
		}
		return number, letterPart, nil
	} else {
		switch types {
		case "memory":
			number = (number)
		case "cpu":
			number = (number)
		default:
			return 0, "", fmt.Errorf("invalid metrics info %s", input)
		}
	}
	return number, "", nil
}
