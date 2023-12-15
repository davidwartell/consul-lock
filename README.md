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

### Example output
```shell
$ go run .                                                                                                     ✔  INSERT
0 acquired lock in 26.115708ms
0: lockChan closed
0: released lock in 12.319667ms
0: stopCh closed
1 acquired lock in 3.043962416s
1: released lock in 4.455917ms
1: stopCh closed
1: lockChan closed
3 acquired lock in 6.054300458s
3: released lock in 12.602375ms
3: stopCh closed
3: lockChan closed
4 acquired lock in 9.07517475s
4: lockChan closed
4: stopCh closed
4: released lock in 3.808833ms
2 acquired lock in 12.083049166s
2: lockChan closed
2: released lock in 4.186667ms
exited
2: stopCh closed
```
