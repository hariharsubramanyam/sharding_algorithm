# Sharding Algorithm

## Context

I'm working on a distributed hashtable in Golang for my distributed systems class. I need to 
implement a sharding algorithm. I've decided to make it a separate repo because:

1. It may come in handy for a future project.
2. It's a fun algorithmic challenge, and I wanted to code it up and write about it.

## Background

Our hashtable must store a set of key-value pairs. The key value pairs may not all fit on a single
server, so we split the key value pairs into **shards**, spread the shards over a set of servers.

Since a server can be a single point of failure, we need some fault tolerance. One way to achieve
this, while preserving strong consistency, is to replace each server with a **group** of servers that
constitute a replicated state machine which agrees on operations using an algorithm like [Raft](https://raftconsensus.github.io/) or
Multi-[Paxos](http://en.wikipedia.org/wiki/Paxos_%28computer_science%29).

So, the goal of sharding is to distribute a set of **shards** evenly over a set of **groups**.

This isn't so bad, but it becomes trickier when groups can join and leave the system.

If a group joins, then it needs some of the shards, so the existing groups each give a few of their
shards to the new group.

If a group leaves, then it evenly distributes its shards over the other groups.

Furthermore, suppose we can move shards programmatically. This can lead to unbalanced sharding 
(ex. one group has many shards, and one group has few). 

A sharding algorithm will perform redistribution of shards. But not all redistributions are good.

Let `maxShards` be the maximum number of shards on any group. Let `minShards` be the minimum number 
of shards on any group. We say that the shards are **balanced** when `maxShards - minShards` is minimal.

Suppose our redistribution algorithm moves a shard from group `G1` to group `G2` (where `G1 != G2`).
We call this a **shard movement**.

So, to get a good redistribution, we want an algorithm which achieves the following:

**Balance the shards using the minimum number of shard movements**

I'll define this problem more formally in the next section. But first, let's define the `Config`
struct.

## Data Structures

Assume each group has a group id, or `gid`, which is an `int64`.

Assume the shards are numbered `0`, ..., `NShards-1`, where `NShards` is the number of shards. Each shard number is an `int`.

We will represent the state of the system using the struct `Config`.

Each `Config` object has two instance variables: `Shards` and `Groups`.

`Shards` is a `[NShards]int64`. If `config.Shards[i] == gid`, this means that the group with group
id = `gid` currently holds shard `i`.

`Groups` is a `map[int64][]string`. If `config.Groups[gid] == serverArr`, this means that the group
with group id = `gid` consists of the servers whose names (ex. IP addresses) in listed in the slice `serverArr`.

Note that I've defined the `Config` structure this way because it resembles what I'm using in the
distributed hashtable.

It will be useful for us to convert the `Config` structure into a map called `shardsForGid`, which 
is `map[int64][]int`. If `shardsForGid[gid] == shards`, this means the group with group id = `gid`
currently holds the shards whose numbers are stored in the `shards` slice.

## Problem: Shard

### Input
A `Config`

### Output
A `Config` where the shards have been balanced using the minimum number of shard movements.

## Algorithm

Let the number of shards be `NShards` and the number of groups be `NGroups`. Notice that in a
balanced configuration, every group must have at least `min = floor(NShards/NGroups)` shards. If
the shards cannot be split evenly (i.e. `NShards % NGroups > 0`), then some groups may have `max =
floor(NShards/NGroups) + 1 = min + 1` shards. Note that if `NShards % NGroups == 0`, then `max = min`

Now create the `shardsForGid` map from the given `Config`. 

Now iterate through each group in `shardsForGid` and give it a designation (by adding it to one of
four lists) as follows:

1. If the group has more than `max` shards, it is a **primary donor**.
2. Otherwise, if the group has exactly `max` shards, it is a **secondary donor**.
3. Otherwise, if the group has less than `min` shards, it is a **recipient**.
4. Otherwise (i.e. the group has exactly `min` shards), it is an **extra recipient**.

Since we know that, after balancing, every group will have at least `min` shards and at most `max`
shards, we can attack this problem in two phases.

### Phase 1: Every Group has at Least `min` Shards

The primary donors donate to the recipients. Here's how that works. The recipients form a line, and
the primary donors form a line facing the recipients' line. Suppose that `P` is the first primary donor and
`R` is the first recipient. We have at least one of these two cases:

1. `P` can donate enough shards such that `R` has `min` shards.
2. `P` can donate enough shards such that `P` has `max` shards.

If only 1 is true, then `P` will donate shards until `R` has `min` shards. At that point, the next 
recipient comes to the front of the line. 

If only 2 is true, then `P` will donate shards until `P` has `max` shards. At that point, `P` will 
join the secondary donors. A new primary donor comes to the front of the line.

If both 1 and 2 are true. Then `P` will donate shards until `P` has `max` shards and `R` has `min` 
shards. At that point, `P` will become a secondary donor. A new primary donor and new recipient will
come to the front of their respective lines.

Now, this process continues until at least one of the following occurs:

1. All the primary donors are now at `max` shards (and have joined the secondary donors).
2. All the recipients are now at `min` shards.

If 2 is not true, then there are still groups with less than `min` shards, so Phase 1 must continue.

So, at this point, the secondary donors form a line in front of the remaining recipients. They do a
process similar to the one the primary donors did, but with two differences:

1. A secondary donor will stop donating when he/she comes down to `min` shards.
2. When a secondary donor finishes donating, he/she joins the extra recipients.

### Phase 2: Every Group has at Most `max` Shards
At this point, is it possible that some group has more than `max` shards?

The recipients cannot, because they all have `min` shards.

The extra recipients cannot, because they all have `min` shards.

The secondary donors cannot, because they all have `max` shards.

Therefore, the primary donors are the only ones who could possible have more than `max` shards.

So, the primary donors must line up for another round of donations. However, they cannot donate to 
the secondary donors, so they will donate to the recipients and extra recipients.

To simplify the donation process, all the extra recipients will now join the recipients.

The primary donors line up, the (now more numerous) recipients line up facing the primary donors. 
The donation process begins again, but with two differences:

1. A primary donor will stop donating when he/she come down to `max` shards. At this point, he/she
will leave his/her line.
2. A recipient will stop receiving when he/she come up to `max` shards. At this point, he/she will 
leave his/her line.

## Runtime and Space
Computing `max` and `min` takes `O(1)` time and `O(1)` space.

Creating `shardsForGrid` takes `O(NShards + NGroups)` time and `O(NShards + NGroups)` space.

Donation from primary donors to recipients (phase 1) takes `O(NGroups)` time (since we are working
with slices, we can move shards from one group to another in constant time in the `shardsForGid` structure). 
Additional space usage is `O(1)`.

Donation from secondary donors to recipients (phase 2) takes `O(NGroups)` time. Additional space usage is
`O(1)`.

Merging the recipients and extra recipients takes `O(1)` time (becuase of slices) and `O(1)` additional space.

Donation from primary donors to recipients (phase 2) takes `O(NGroups)` time. Additional space usage is
`O(1)`.

Converting the resulting `shardsForKey` back into a `Config` takes `O(NShards + NGroups)` time and 
`O(NShards + NGroups)` space.

Thus, the runtime and space usage are both `O(NShards + NGroups)`.

## Correctness

### Shards are Balanced
When phase 1 ends, all the groups will have at least `min` shards. Assume for contradiction
that some group still has less than `min` shards. This group must be a recipient (primary donors 
and secondary donors never fall below `min` shards, and extra recipients have exactly `min` shards 
and don't donate). This means that every primary donor and secondary donor has become an extra 
recipient (and therefore has exactly `min` shards). Thus, every single group now has at most `min` 
shards (and some group has less than `min` shards). But this means the total number of shards is 
less than `NGroups*min <= NShards`, which is a contradiction. Thus, every group must have at least 
`min` shards.

When phase 2 ends, all groups will have at most `max` shards. Assume for contradiction
that some group still has more than `max` shards. This group must be a primary donor (the recipients,
which includes extra recipients, cannot have more than `max` shards and the secondary donors have
exactly `max` shards and did not participate in this round of donations). This meeans that every
recipient (that includes the extra recipients) is now at `max` shards. But this means that every
single group has at least `max` shards (aond some group has more than `max` shards). But this means
that the total number of shards is more than `NGroups*max >= NShards`, which is a contradiction. Thus
every group must have at most `max` shards.

Thus, we now know that the algorithm ensures that every group has between `min` and `max` 
shards (inclusive). 

Now we have two cases.

* `NShards % NGroups > 0`

If this is the case, then a balanced distribution of shards ensures that the maximum difference between
any pair of shards is at least 1 (it cannot be 0 because that would mean that `NShards % NGroups == 0`,
which is a contradiction.

In this case, `max = min + 1`. Since every group must have between `min` and `max` shards, the 
maximum difference in the number of shards between any pair of groups is at most `max - min = 1`, 
as desired.  

Thus, the shards are balanced.

* `NShards % NGroups == 0`

If the shards divide evenly (i.e. `NShards % NGroups == 0`), then in a balanced distribution of 
shards, every group has the same number of shards. So, the maximum difference in the number of 
shards between any pair of groups is exactly 0.

In this case, `max = min`. Since every group must have between `min` and `max` shards, the maximum
difference in the number of shards between any pair of groups is at most `max - min = 0`, as desired.

Thus, the shards are balanced.

In both cases, the shards are balanced.

### Number of Shard Movement is Minimal

I'm getting sleepy and I'm not exactly sure how to approach this proof right now :) I may come 
back to it at a later date. But until then, it might be fun to leave it as an exercise to the reader. ;)
