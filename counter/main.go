package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/bitly/go-nsq"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var (
	fatalErr   error
	counts     map[string]int
	countsLock sync.Mutex
)

const updateDuration = 1 * time.Second

func fatal(e error) {
	fmt.Println(e)
	flag.PrintDefaults()
	fatalErr = e
}

func main() {
	defer func() {
		if fatalErr != nil {
			os.Exit(1)
		}
	}()

	log.Println("Connecting to the database...")
	db, err := mgo.Dial("localhost")
	if err != nil {
		fatal(err)
		return
	}
	defer func() {
		log.Println("Closing database connection...")
		db.Close()
	}()
	pollData := db.DB("ballots").C("polls")

	log.Println("Connecting to nsq")
	q, err := nsq.NewConsumer("votes", "counter", nsq.NewConfig())
	if err != nil {
		fatal(err)
		return
	}

	q.AddHandler(nsq.HandlerFunc(func(m *nsq.Message) error {
		countsLock.Lock()
		defer countsLock.Unlock()
		if counts == nil {
			counts = make(map[string]int)
		}
		vote := string(m.Body)
		counts[vote]++
		return nil
	}))

	if err := q.ConnectToNSQLookupd("localhost:4161"); err != nil {
		fatal(err)
		return
	}

	ticker := time.NewTicker(updateDuration)
	termChan := make(chan os.Signal)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	for {
		select {
		case <-ticker.C:
			doCount(&countsLock, &counts, pollData)
		case <-termChan:
			ticker.Stop()
			q.Stop()
		case <-q.StopChan:
			// finished
			return
		}
	}
}

func doCount(countsLock *sync.Mutex, counts *map[string]int, pollData *mgo.Collection) {
	countsLock.Lock()
	defer countsLock.Unlock()
	if len(*counts) == 0 {
		log.Println("No new votes, skipping database update")
		return
	}
	log.Println("Updating database...")
	log.Println(*counts)
	ok := true
	for option, count := range *counts {
		selector := bson.M{"options": bson.M{"$in": []string{option}}}
		update := bson.M{"$inc": bson.M{"results." + option: count}}
		if _, err := pollData.UpdateAll(selector, update); err != nil {
			log.Println("Failed to update:", err)
			ok = false
		}
	}

	if ok {
		log.Println("Finished updating database...")
		*counts = nil //reset counts
	}
}
