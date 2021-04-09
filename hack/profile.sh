#!/bin/bash

time=$(date +%s)
duration=10
host="http://localhost:6060"

name="${1:-profile}"
echo $name-$time

curl "${host}/debug/pprof/profile?seconds=${duration}" > $name-$time.profile.pprof
curl "${host}/debug/pprof/block?seconds=${duration}" > $name-$time.block.pprof
curl "${host}/debug/pprof/heap?seconds=${duration}" > $name-$time.heap.pprof
curl "${host}/debug/pprof/allocs?seconds=${duration}" > $name-$time.allocs.pprof
curl "${host}/debug/pprof/trace?seconds=${duration}" > $name-$time.trace.pprof
