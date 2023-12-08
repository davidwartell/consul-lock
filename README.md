# golang Consul lock command line demo

I was exploring using consul as a distributed lock, but ended up going a different direction and using a cluster with a 
consistent hash-ring to control access to data structures in a high write throughput application. Along the way I ended
up writing this sample golang code to understand consul locking in golang and locking overhead to estimate how it would
scale in my particular use case.  Sharing this as a functional example of consul locking in golang for others in hopes it 
may benefit them.

### Implementation note:
My intended use of the distribute lock was for very high thoughput writes (> 1 Million write ops/s) most with different 
unique ids/locks, so I did not want to leave defunct keys in consul storage for unused locks. This implementation works
to not leave unused keys on consul. This is a complication your application may potentially avoid.

### Dependencies
* local running consul instance with no access control for development
* golang
* I'm using MacOS. Makefile functionality may vary on windoz.

### Build and run
```shell
make run
```

### Installing and running consul on Docker
```shell
make pull-consul
make start-consul
```
