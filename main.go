package main

import (
	"context"
	"fmt"
	"github.com/davidwartell/go-commons-drw/onecontext"
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const lockKeyPathDelim = "/"
const LockNameSpaceTest = "test1"

const concurrency = 5

func main() {
	// warm the client
	if _, err := getInstance().getClient(); err != nil {
		fmt.Printf("error getting client: %v\n", err)
		os.Exit(1)
	}

	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go runLockTest(ctx, &wg, i)
	}

	var signalWg sync.WaitGroup
	signalWg.Add(1)
	go watchSignals(ctx, &signalWg, cancel)

	wg.Wait()
	cancel()
	signalWg.Wait()

	fmt.Println("exited")
	os.Exit(0)
}

func watchSignals(ctx context.Context, wg *sync.WaitGroup, cancel context.CancelFunc) {
	defer wg.Done()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		fmt.Printf("exiting received signal %s\n", sig.String())
		cancel()
		return
	case <-ctx.Done():
		return
	}
}

type service struct {
	config     *api.Config
	client     *api.Client
	clientLock sync.RWMutex
}

var instance *service
var once sync.Once

func getInstance() *service {
	once.Do(func() {
		instance = &service{
			config: &api.Config{Address: "127.0.0.1:8500"},
		}
	})
	return instance
}

func (s *service) getClient() (client *api.Client, err error) {
	s.clientLock.RLock()
	if s.client != nil {
		client = s.client
		s.clientLock.RUnlock()
		return
	}
	s.clientLock.RUnlock()
	return s.newClient()
}

func (s *service) newClient() (client *api.Client, err error) {
	s.clientLock.Lock()
	defer s.clientLock.Unlock()
	if s.client != nil {
		client = s.client
		return
	}
	if s.client, err = api.NewClient(&api.Config{Address: "127.0.0.1:8500"}); err != nil {
		err = errors.Errorf("error creating consul client: %v", err)
		return
	}
	client = s.client
	return
}

func runLockTest(clientCtx context.Context, wg *sync.WaitGroup, id int) {
	defer wg.Done()

	lockCtx, lockCancel := context.WithCancel(clientCtx)
	defer lockCancel()

	mergedCtx, _ := onecontext.Merge(clientCtx, lockCtx)

	lockAcquireStart := time.Now()
	var client *api.Client
	var err error
	if client, err = getInstance().getClient(); err != nil {
		return
	}

	writeOpts := new(api.WriteOptions)
	var sessionPtr *string
	var session string
	if session, _, err = client.Session().Create(
		&api.SessionEntry{
			Behavior:  api.SessionBehaviorDelete,
			TTL:       "10s",
			LockDelay: 1 * time.Millisecond,
		},
		writeOpts.WithContext(clientCtx),
	); err != nil {
		fmt.Printf("%d: error creating consul session: %v\n", id, err)
		return
	}
	sessionPtr = &session
	defer func() {
		if sessionPtr != nil {
			if _, err = client.Session().Destroy(*sessionPtr, nil); err != nil {
				fmt.Printf("%d: error destroying consul session: %v\n", id, err)
			}
		}
	}()

	opts := &api.LockOptions{
		Key:     TestLockKey("1"),
		Session: session,
	}
	var lock *api.Lock
	if lock, err = client.LockOpts(opts); err != nil {
		fmt.Printf("%d: error creating lock handle: %v\n", id, err)
		return
	}

	// stopCh is used to signal the lock to stop trying to acquire the lock
	stopCh := make(chan struct{})
	go func() {
		defer func() {
			close(stopCh)
			fmt.Printf("%d: stopCh closed\n", id)
		}()
		select {
		case <-lockCtx.Done():
			// lock is done do nothing
			return
		case <-clientCtx.Done():
			fmt.Printf("%d: clientCtx cancelled notifying stop lock\n", id)
			// will close stopCh on exit
			return
		}
	}()

	var lockChan <-chan struct{}
	if lockChan, err = lock.Lock(stopCh); err != nil {
		fmt.Printf("%d: failed to acquire lock: %v\n", id, err)
		return
	} else if mergedCtx.Err() == nil {
		fmt.Printf("%d acquired lock in %v\n", id, time.Since(lockAcquireStart))
	} else {
		fmt.Printf("%d lock acquire interrupted %v\n", id, time.Since(lockAcquireStart))
	}

	// cancel the work if the lock is lost
	go func() {
		<-lockChan
		fmt.Printf("%d: lockChan closed\n", id)
		lockCancel()
		return
	}()

	if mergedCtx.Err() == nil {
		// do some work
		doSleep(mergedCtx)
	}

	// if the lock was cancelled
	if lockCtx.Err() == nil {
		lockReleaseStart := time.Now()
		//if err = lock.Unlock(); err != nil {
		//	fmt.Printf("error lock already unlocked in %d: %v", id, err)
		//	return
		//}
		if _, err = client.Session().Destroy(*sessionPtr, nil); err != nil {
			fmt.Printf("%d: error unlocking (destroying session): %v", id, err)
			return
		}
		sessionPtr = nil
		fmt.Printf("%d: released lock in %v\n", id, time.Since(lockReleaseStart))
	} else {
		fmt.Printf("%d: lock cancelled\n", id)
	}
}

func doSleep(ctx context.Context) {
	select {
	case <-ctx.Done(): //context cancelled
		return
	case <-time.After(3 * time.Second): //timeout
		return
	}
}

func TestLockKey(id string) string {
	return LockNameSpaceTest + lockKeyPathDelim + id
}
