package utils

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"
)

const (
	snowflakeNodeBits     int64 = 10
	snowflakeSequenceBits int64 = 12

	snowflakeMaxNodeID  int64 = (1 << snowflakeNodeBits) - 1
	snowflakeSequenceMS int64 = (1 << snowflakeSequenceBits) - 1

	snowflakeNodeShift      int64 = snowflakeSequenceBits
	snowflakeTimestampShift int64 = snowflakeSequenceBits + snowflakeNodeBits

	// Custom epoch: 2025-01-01T00:00:00Z
	snowflakeEpochMillis int64 = 1735689600000
)

type snowflakeGenerator struct {
	mu            sync.Mutex
	nodeID        int64
	lastTimestamp int64
	sequence      int64
}

var defaultSnowflakeGenerator = &snowflakeGenerator{
	nodeID: getSnowflakeNodeID(),
}

func getSnowflakeNodeID() int64 {
	raw := os.Getenv("SNOWFLAKE_NODE_ID")
	if raw == "" {
		return 1
	}

	nodeID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || nodeID < 0 || nodeID > snowflakeMaxNodeID {
		return 1
	}

	return nodeID
}

func currentMillis() int64 {
	return time.Now().UnixMilli()
}

func (g *snowflakeGenerator) nextID() int64 {
	g.mu.Lock()
	defer g.mu.Unlock()

	currentTS := currentMillis()
	if currentTS < g.lastTimestamp {
		// Guard against clock drift by pinning to last seen timestamp.
		currentTS = g.lastTimestamp
	}

	if currentTS == g.lastTimestamp {
		g.sequence = (g.sequence + 1) & snowflakeSequenceMS
		if g.sequence == 0 {
			for currentTS <= g.lastTimestamp {
				currentTS = currentMillis()
			}
		}
	} else {
		g.sequence = 0
	}

	g.lastTimestamp = currentTS

	return ((currentTS - snowflakeEpochMillis) << snowflakeTimestampShift) |
		(g.nodeID << snowflakeNodeShift) |
		g.sequence
}

func GenerateSnowflakeID() int64 {
	return defaultSnowflakeGenerator.nextID()
}

func GenerateSnowflakeIDString() string {
	return strconv.FormatInt(GenerateSnowflakeID(), 10)
}

func ParseSnowflakeID(id string) (int64, error) {
	parsedID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid id: %s", id)
	}

	if parsedID <= 0 {
		return 0, fmt.Errorf("invalid id: %s", id)
	}

	return parsedID, nil
}
