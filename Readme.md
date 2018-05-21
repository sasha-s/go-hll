# HyperLogLog in golang. [Docs](https://godoc.org/github.com/sasha-s/go-hll). [![Build Status](https://travis-ci.org/sasha-s/go-hll.svg?branch=master)](https://travis-ci.org/sasha-s/go-hll) [![codecov](https://codecov.io/gh/sasha-s/go-hll/branch/master/graph/badge.svg)](https://codecov.io/gh/sasha-s/go-hll)
## What
A go implementation of HypeLogLog data structure with a twist.
See [HyperLogLog in Practice](http://research.google.com/pubs/pub40671.html) paper by Stefan Heule, Marc Nunkesser, Alex Hall.

## Ø-serialization
There is no need to serialize/deserialize hll.
Everything is stored in a byte slice, which can be memory mapped, passed around over the network as is etc.

## Differences from the paper:
* sparse representation. this implementation does exact counting for small sets.
* fixed memory usage (even for empty HLL). HLL of a given precision P uses fixed (8 + 3*2^(P-2), 8 byte header + 6 bits per register) size in bytes.
* thresholds are tuned. different from [Sub-Algorithm Threshold](https://docs.google.com/document/d/1gyjfMHy43U9OWBXxfaeG-3MjGzejW1dlpyMwEYAAWEI/view?fullscreen#heading=h.nd379k1fxnux).

## Why
I wanted an HLL implementation that is

* simple
* reasonably fast
* (almost) non-allocating
* exact when number of unique elements is small
* memory-mapped file friendly
* well tested (90+% coverage)

## Usage
Get go-hll:
```sh
go get github.com/sasha-s/go-hll
```

Use it:
```go
s, err := SizeByP(16)
if err != nil {
	log.Panicln(err)
}
h := make(HLL, s)
...
for _, x := range []string{"alpha", "beta"} {
	h.Add(siphash.Hash(2, 57, []byte(x)))
}
log.Println(h.EstimateCardinality())
```

Use good hash (otherwise accuracy would be poor). Some options:

* [MurmurHash3](https://github.com/spaolacci/murmur3)
* [HighwayHash](https://github.com/dgryski/go-highway)
* [Siphash](https://github.com/dchest/siphash)
* [SpookyHash](https://github.com/dgryski/go-spooky)

## Speed

Benchmark results on my MacBook Pro (Mid 2014).

```
Add-8            9.68ns ± 1%
Estimate-8       27.3µs ± 1%
Merge-8          38.0µs ± 1%

AddDense-8       6.73ns ± 3%
MergeDense-8     37.9µs ± 1%
EstimateDense-8  22.9µs ± 1%
Sort-8            108µs ± 1%
AddSparse-8      10.2ns ± 3%
```

Merge/Estimate etc. are done for P=14.

## Other implementations (in no particular order)

* [HLL++ by Micha Gorelick @mynameisfiber](https://github.com/mynameisfiber/gohll)
* [Probably (has regular HLL) by Dustin Sallings @dustin](https://github.com/dustin/go-probably)
* [HLL++ by lytrics](https://github.com/lytics/hll)
* [HLL/HLL++ by Clark DuVall @clarkduvall](https://github.com/clarkduvall/hyperloglog)
* [Redis has regular HLL](http://download.redis.io/redis-stable/src/hyperloglog.c). Also see [blogpost](http://antirez.com/news/75).
* [hllpp by Muir Manders @muirrn/retailnext](https://github.com/retailnext/hllpp)
