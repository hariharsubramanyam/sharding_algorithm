# Sharding Algorithm

## Context

I'm working on a distributed hashtable in Golang for my distributed systems class. I need to 
implement a sharding algorithm. I've decided to make it a separate repo because it may come in handy
for future projects.

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
