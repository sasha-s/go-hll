# HyperLogLog in golang

## What
A go implementation of HypeLogLog data structure with a twist.
See [HyperLogLog in Practice](http://research.google.com/pubs/pub40671.html) paper by Stefan Heule, Marc Nunkesser, Alex Hall.

## Ø-serialization
There is no need to serialize/deserialize hll.
Everything is stored in a byte slice, which can be memory mapped, passed around over th network as is etc.

## Differences from the paper:
* sparse representation. this implementation does exact counting for small sets.
* no compression. HLL of a given precision P uses fixed (8 + 3*2^(P-2), 8 byte header + 6 bits per register) size in bytes.

## Why
I wanted an HLL implementation that is

* simple
* reasonably fast
* (almost) non-allocating
* exact when number of unique elements is small
* memory-mapped file friendly
* well tested (90+% coverage)

## Speed

Benchmark results on my laptop (to give an )
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
