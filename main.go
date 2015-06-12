package main

import (
	// "fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/jeffail/tunny"
	"github.com/kr/beanstalk"
)

type ReturnIndicator uint

const (
	BeanstalkdDelete  ReturnIndicator = 0
	BeanstalkdRelease ReturnIndicator = 1
	BeanstalkdBury    ReturnIndicator = 2
	BeanstalkdIgnore  ReturnIndicator = 4
)

type BeanstalkdDefaults uint32

const (
	DEFAULT_DELAY    BeanstalkdDefaults = 0    // no delay
	DEFAULT_PRIORITY                    = 1024 // most urgent: 0, least urgent: 4294967295
	DEFAULT_TTR                         = 60   // 1 minute
)

type TunnyInput struct {
	Id   uint64
	Body []byte
}

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	log.Println("==============Running==============\n")

	var wg sync.WaitGroup

	workers := make([]tunny.TunnyWorker, runtime.GOMAXPROCS(runtime.NumCPU())*NUM_WORKERS_MULTIPLIER)
	for i, _ := range workers {
		workers[i] = &(sendSMSWorker{})
	}

	pool, _ := tunny.CreateCustomPool(workers).Open()
	defer pool.Close()

	//Graceful Shutdown code

	signalReceived := false

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	go func() {
		<-sigChan
		signalReceived = true
		log.Println("==============received signal to shut down==============\n")
	}()

	//Display number of go routines
	// go func(signalReceived *bool) {
	// 	for {
	// 		if *signalReceived == true {
	// 			break
	// 		}
	// 		log.Printf("No. of Goroutines: %d", runtime.NumGoroutine())
	// 		time.Sleep(7500 * time.Millisecond)
	// 	}

	// }(&signalReceived)

	//Operate Beanstalkd
	beanstalkConn, err := beanstalk.Dial("tcp", BEANSTALKD_ADDRESS_AND_PORT)
	if err != nil {
		panic(err)
	}
	ts := beanstalk.NewTubeSet(beanstalkConn, "sms")

	for {

		if signalReceived == true {
			break
		}

		id, body, err := ts.Reserve(30 * time.Second)
		if cerr, ok := err.(beanstalk.ConnError); ok && cerr.Err == beanstalk.ErrTimeout {
			// log.Println("timed out")
			continue
		} else if cerr.Err == beanstalk.ErrDeadline {
			log.Println("deadline soon")
			continue
		} else if cerr.Err != nil {
			// Warn/Notify that beanstalkd has broken down!
			log.Print("cerr: %+v", cerr.Err)
			break
		}

		wg.Add(1)

		go func(id uint64, body []byte, wg *sync.WaitGroup) {
			defer func() {
				err := recover()
				if err != nil {
					(*wg).Done()
				}
			}()

			result, err := pool.SendWork(TunnyInput{Id: id, Body: body})
			if err != nil { //ErrWorkerClosed or ErrPoolNotRunning
				// Warn/Notify that Worker has broken down!

				// beanstalkConn.Release(id, uint32(DEFAULT_PRIORITY), 60*time.Second)
				(*wg).Done()
				return
			}

			if result.(ReturnIndicator) == BeanstalkdDelete {
				beanstalkConn.Delete(id)
			} else if result.(ReturnIndicator) == BeanstalkdRelease {
				beanstalkConn.Release(id, uint32(DEFAULT_PRIORITY), 60*time.Second)
			} else if result.(ReturnIndicator) == BeanstalkdBury {
				beanstalkConn.Bury(id, uint32(DEFAULT_PRIORITY))
			} else {
				(*wg).Done()
				return
			}

			(*wg).Done()
		}(id, body, &wg)

	}

	wg.Wait()
	log.Println("==============Terminated==============\n")
}
