package shard

const NShards = 10

type Config struct {
	Shards [NShards]int64
	Groups map[int64][]string
}

func ToMap(config Config) *map[int64][]int {
	shardsForGid := make(map[int64][]int)
	for gid := range config.Groups {
		shardsForGid[gid] = make([]int, 0)
	}
	for shard, gid := range config.Shards {
		shardsForGid[gid] = append(shardsForGid[gid], shard)
	}
	return &shardsForGid
}

func ToShards(shardsForGid *map[int64][]int) [NShards]int64 {
	var shards [NShards]int64
	for gid, gidShards := range *shardsForGid {
		for _, shard := range gidShards {
			shards[shard] = gid
		}
	}
	return shards
}
