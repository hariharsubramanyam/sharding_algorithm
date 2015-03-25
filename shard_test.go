package shard

import "testing"
import "fmt"

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

func NumMovements(a Config, b Config) int {
	numMovements := 0
	for i := 0; i < len(a.Shards); i++ {
		if a.Shards[i] != b.Shards[i] {
			numMovements++
		}
	}
	return numMovements
}

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

func TestNoChange(t *testing.T) {
	config := MakeConfigFromMap(map[int64][]int{
		1: []int{0, 1, 2, 3},
		2: []int{4, 5, 6},
		3: []int{7, 8, 9}})

	Check(t, config, 0, "Passed TestNoChange...")
}
