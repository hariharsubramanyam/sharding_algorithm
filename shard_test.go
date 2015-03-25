package shard

import "testing"
import "math/rand"
import "fmt"

// MakeConfigFromMap creates a Config object from a map (easier to type during testing).
func MakeConfigFromMap(shardsForGid map[int64][]int) Config {
	config := Config{}
	config.Groups = make(map[int64][]string)
	for gid, shards := range shardsForGid {
		for _, shard := range shards {
			config.Shards[shard] = gid
		}
		config.Groups[gid] = []string{""}
	}
	return config
}

// IsBalanced checks whether the shards are balanced in the given configuration.
func IsBalanced(config Config) bool {
	min := len(config.Shards) / len(config.Groups)
	max := min
	if len(config.Shards)%len(config.Groups) > 0 {
		max = min + 1
	}

	frequencyTable := make(map[int64]int)
	for gid, _ := range config.Groups {
		frequencyTable[gid] = 0
	}
	for _, shard := range config.Shards {
		frequencyTable[shard]++
	}

	for _, numShards := range frequencyTable {
		if numShards != min && numShards != max {
			return false
		}
	}
	return true
}

// NumMovements computes the number of shard movements required to go from Config a to Config b.
func NumMovements(a Config, b Config) int {
	numMovements := 0
	for i := 0; i < len(a.Shards); i++ {
		if a.Shards[i] != b.Shards[i] {
			numMovements++
		}
	}
	return numMovements
}

// Check ensures that the Shard function balances the shards in the minimum number of shard
// movements.
func Check(t *testing.T, config Config, minNumMovements int, success string) {
	newConfig := Shard(config)
	if !IsBalanced(newConfig) {
		t.Fatalf("Unbalanced on", config, "to", newConfig)
	}

	if NumMovements(config, newConfig) != minNumMovements {
		t.Fatalf("Excessive movements on", config, "to", newConfig)
	}

	fmt.Println(success)
}

// Don't move shards if the sharding is already optimal.
func TestNoChange(t *testing.T) {
	config := MakeConfigFromMap(map[int64][]int{
		1: []int{0, 4, 1, 7},
		2: []int{2, 9, 8},
		3: []int{3, 6, 5}})
	Check(t, config, 0, "Passed TestNoChange...")
}

// All the shards are on one group.
func TestAllInOne(t *testing.T) {
	config := MakeConfigFromMap(map[int64][]int{
		1: []int{7, 8, 3, 4, 5, 9, 1, 6, 2, 0},
		2: []int{},
		3: []int{}})
	Check(t, config, 6, "Passed TestAllInOne...")
}

// There are more groups than shards.
func TestMoreGroupsThanShards(t *testing.T) {
	config := MakeConfigFromMap(map[int64][]int{
		1:  []int{9, 6, 2},
		2:  []int{},
		3:  []int{},
		4:  []int{},
		5:  []int{0, 8},
		6:  []int{},
		7:  []int{},
		8:  []int{3, 5, 4, 1},
		9:  []int{},
		10: []int{},
		11: []int{},
		12: []int{},
		13: []int{},
		14: []int{7}})
	Check(t, config, 6, "Passed TestMoreGroupsThanShards...")
}

// There's only one group.
func TestOneGroup(t *testing.T) {
	config := MakeConfigFromMap(map[int64][]int{
		1: []int{1, 8, 3, 5, 7, 0, 2, 4, 6, 9}})
	Check(t, config, 0, "Passed TestOneGroup...")
}

// Every group has a single shard.
func TestAllOne(t *testing.T) {
	config := MakeConfigFromMap(map[int64][]int{
		1:  []int{7},
		2:  []int{8},
		3:  []int{5},
		4:  []int{9},
		5:  []int{4},
		6:  []int{2},
		7:  []int{6},
		8:  []int{0},
		9:  []int{1},
		10: []int{3},
		11: []int{},
		12: []int{},
		13: []int{},
		14: []int{}})
	Check(t, config, 0, "Passed TestAllOne...")
}

// Only a single shard movement is required.
func TestOneOver(t *testing.T) {
	config := MakeConfigFromMap(map[int64][]int{
		1: []int{9, 3, 1, 0, 6, 8},
		2: []int{2, 4, 7, 5}})
	Check(t, config, 1, "Passed TestOneOver...")
}

// Generate a bunch of random configurations and make sure they can all be balanced.
func TestBalanceRandom(t *testing.T) {
	for i := 0; i < 1000; i++ {
		numGroups := 1 + rand.Intn(20)
		config := Config{}
		for shardNum := 0; shardNum < len(config.Shards); shardNum++ {
			config.Shards[shardNum] = int64(1 + rand.Intn(numGroups))
		}
		config.Groups = make(map[int64][]string)
		for groupNum := 0; groupNum < numGroups; groupNum++ {
			config.Groups[int64(groupNum+1)] = []string{}
		}

		newConfig := Shard(config)
		if !IsBalanced(newConfig) {
			t.Fatalf("Unbalanced on", config, "to", newConfig)
		}
	}
	fmt.Println("Passed TestBalanceRandom...")
}

// A new server has joined with no shards.
func TestJoin(t *testing.T) {
	config := MakeConfigFromMap(map[int64][]int{
		1: []int{2, 8, 3, 0, 5, 7, 9, 4},
		2: []int{1, 6},
		3: []int{}})
	Check(t, config, 4, "Passed TestJoin...")
}

// A server just left and gave all its shards to somebody before it disappeared.
func TestLeave(t *testing.T) {
	config := MakeConfigFromMap(map[int64][]int{
		1: []int{8, 0, 3, 1, 7},
		2: []int{5, 6},
		3: []int{2, 4, 9}})
	Check(t, config, 1, "Passed TestLeave...")
}
