package shard

const NShards = 10

type Config struct {
	Shards [NShards]int64
	Groups map[int64][]string
}

// ToMap takes a Config and produces a map where the key = gid, value = shards owned by group
// with id = gid.
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

// ToShards takes a map where key = gid, value = shards owned by group with id = gid, and produces
// an array shards where shards[i] = gid means shard i is owned by the group with id = gid.
func ToShards(shardsForGid *map[int64][]int) [NShards]int64 {
	var shards [NShards]int64
	for gid, gidShards := range *shardsForGid {
		for _, shard := range gidShards {
			shards[shard] = gid
		}
	}
	return shards
}

func DeepCopyConfig(config *Config) *Config {
	newConfig := &Config{}
	for i, gid := range config.Shards {
		newConfig.Shards[i] = gid
	}

	newConfig.Groups = make(map[int64][]string)
	for gid, servers := range config.Groups {
		newConfig.Groups[gid] = make([]string, len(servers))
		for _, server := range servers {
			newConfig.Groups[gid] = append(newConfig.Groups[gid], server)
		}
	}

	return newConfig
}

// Shard will create a new config that balances the shards using the minimum number of shard movements.
func Shard(config Config) Config {
	shardsForGidPtr := ToMap(config)
	shardsForGid := *shardsForGidPtr

	min := len(config.Shards) / len(config.Groups)
	max := min
	if len(config.Shards)%len(config.Groups) != 0 {
		max = min + 1
	}

	// Phase 1: Ensure that nobody has < min shards.

	// primaryDonors are the groups with > max shards.
	// secondaryDonors are the groups with > min shards.
	// recipients are the groups with < min shards.
	primaryDonors := make([]int64, 0)
	secondaryDonors := make([]int64, 0)
	recipients := make([]int64, 0)
	extraRecipients := make([]int64, 0)

	for gid, shards := range shardsForGid {
		if len(shards) > max {
			primaryDonors = append(primaryDonors, gid)
		} else if len(shards) > min {
			secondaryDonors = append(secondaryDonors, gid)
		} else if len(shards) < min {
			recipients = append(recipients, gid)
		} else {
			extraRecipients = append(extraRecipients, gid)
		}
	}

	// The primary donors donate to the recipients.
	currDonor := 0
	currRecipient := 0

	var numExcess int
	var numNeeded int
	var donor int64
	var recipient int64
	for currDonor < len(primaryDonors) && currRecipient < len(recipients) {
		donor = primaryDonors[currDonor]
		recipient = recipients[currRecipient]
		numExcess = len(shardsForGid[donor]) - max
		numNeeded = min - len(shardsForGid[recipient])

		if numExcess > numNeeded {
			shardsForGid[recipient] = append(shardsForGid[recipient], (shardsForGid[donor])[:numNeeded]...)
			shardsForGid[donor] = shardsForGid[donor][numNeeded:]
			currRecipient++
		} else {
			shardsForGid[recipient] = append(shardsForGid[recipient], (shardsForGid[donor])[:numExcess]...)
			shardsForGid[donor] = shardsForGid[donor][numExcess:]
			secondaryDonors = append(secondaryDonors, donor)
			currDonor++
		}
	}

	// Now the secondary donors donate to the recipients.
	currSecondaryDonor := 0

	for currSecondaryDonor < len(secondaryDonors) && currRecipient < len(recipients) {
		donor = secondaryDonors[currSecondaryDonor]
		recipient = recipients[currRecipient]
		numExcess = len(shardsForGid[donor]) - min
		numNeeded = min - len(shardsForGid[recipient])
		if numExcess > numNeeded {
			shardsForGid[recipient] = append(shardsForGid[recipient], (shardsForGid[donor])[:numNeeded]...)
			shardsForGid[donor] = shardsForGid[donor][numNeeded:]
			currRecipient++
		} else {
			shardsForGid[recipient] = append(shardsForGid[recipient], (shardsForGid[donor])[:numExcess]...)
			shardsForGid[donor] = shardsForGid[donor][numExcess:]
			extraRecipients = append(extraRecipients, donor)
			currSecondaryDonor++
		}
	}

	// Phase 2: Ensure that nobody has more than max shards.

	// Now nobody has < min shards.
	// Some primary donors may have > max shards.
	// Everybody in recipients has exactly min shards.
	// Everybody in extraRecipients has exactly min shards.
	// Everybody in secondaryDonors[:currSecondaryDonor] is now in extraRecipients.
	// Everybody in secondaryDonors[currSecondaryDonor:] has max shards.
	// Thus, the only people who can accept shards from the primaryDonors are the recipients and
	// extraRecipients. So, let's combine them.
	recipients = append(recipients, extraRecipients...)
	currRecipient = 0

	for currDonor < len(primaryDonors) {
		donor = primaryDonors[currDonor]
		recipient = extraRecipients[currRecipient]
		numExcess = len(shardsForGid[donor]) - max
		numNeeded = max - len(shardsForGid[recipient])
		if numExcess > numNeeded {
			shardsForGid[recipient] = append(shardsForGid[recipient], (shardsForGid[donor])[:numNeeded]...)
			shardsForGid[donor] = shardsForGid[donor][numNeeded:]
			currRecipient++
		} else {
			shardsForGid[recipient] = append(shardsForGid[recipient], (shardsForGid[donor])[:numExcess]...)
			shardsForGid[donor] = shardsForGid[donor][numExcess:]
			currDonor++
		}
	}

	// Now we've finally redistributed the shards. Let's reconstruct the config.
	newConfig := DeepCopyConfig(&config)
	for gid, shards := range shardsForGid {
		for _, shard := range shards {
			newConfig.Shards[shard] = gid
		}
	}

	return *newConfig
}
