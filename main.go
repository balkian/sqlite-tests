package main

import (
	"bufio"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"runtime/pprof"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var lastChecked, started time.Time
var lastProgress int64
var curSpeed, avgSpeed float64

func updateStatus(lines, current, total int64) {
	// fmt.Printf("\033[0;0H")
	// os.Stdout.Write([]byte("\033[F")) //back to previous line
	// os.Stdout.Write([]byte("\033[K")) //clear line
	if !lastChecked.IsZero() {
		curSpeed = float64(current-lastProgress) / time.Since(lastChecked).Seconds()
	}
	avgSpeed = float64((current)) / time.Since(started).Seconds()
	lastChecked = time.Now()
	lastProgress = current

	fmt.Printf("%10d/%-10d Bytes (%7.3f%%) %d lines - %.2f Bps (avg. %.2f Bps)\n",
		current,
		total,
		100*float32(current)/float32(total),
		lines,
		curSpeed,
		avgSpeed)

}

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

func main() {
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	filename := flag.Args()[0]

	var pos int64

	f, err := os.Open(filename)
	checkErr(err)

	stat, err := f.Stat()
	checkErr(err)
	total := stat.Size()

	started = time.Now()
	users := make(chan []string, 1000)
	wg := sync.WaitGroup{}

	var counter int64

	timer := time.NewTicker(time.Second)
	doneChan := make(chan bool)
	stopMetrics := make(chan bool)

	go func() {
		for {
			select {
			case <-timer.C:
				updateStatus(counter, pos, total)
			case <-stopMetrics:
				updateStatus(counter, pos, total)
				doneChan <- true
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(f)

		scanLines := func(data []byte, atEOF bool) (advance int, token []byte, err error) {
			advance, token, err = bufio.ScanLines(data, atEOF)
			pos += int64(advance)
			return
		}
		scanner.Split(scanLines)
		var line string
		var tokens []string
		for scanner.Scan() {
			counter++
			line = scanner.Text()
			tokens = strings.Split(line, "\t")
			if len(tokens) != 2 {
				fmt.Println("Wrong line: ", line, len(line))
				continue
			}
			users <- tokens[:2]
		}
		close(users)
		fmt.Println("Done with file")
	}()

	commitTimer := time.NewTicker(30 * time.Second)
	defer commitTimer.Stop()
	wg.Add(1)
	go func() {
		bname := path.Base(filename)
		defer wg.Done()

		db, err := sql.Open("sqlite3", fmt.Sprintf("./%s.go.db", bname))
		checkErr(err)
		db.SetMaxOpenConns(1)
		_, err = db.Exec("PRAGMA journal_mode=MEMORY;")
		checkErr(err)

		_, err = db.Exec("CREATE TABLE IF NOT EXISTS followers (user int, follower int)")
		checkErr(err)
		_, err = db.Exec("CREATE INDEX IF NOT EXISTS followers_user ON followers(user)")
		checkErr(err)
		_, err = db.Exec("CREATE INDEX IF NOT EXISTS followers_follower ON followers(follower)")
		checkErr(err)
		_, err = db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS followers_unique ON followers(user, follower)")
		checkErr(err)

		// # Using transactions
		insertStmt, err := db.Prepare("INSERT OR IGNORE INTO followers VALUES (?, ?)")
		checkErr(err)
		tx, err := db.Begin()
		checkErr(err)
		var pair []string
		var ok bool
		txst := tx.Stmt(insertStmt)
	loop:
		for {
			select {
			case pair, ok = <-users:
				if !ok {
					fmt.Println("No more pairs")
					break loop
				}
				_, err = txst.Exec(pair[0], pair[1])
				checkErr(err)
			case <-commitTimer.C:
				tx.Commit()
				tx, err = db.Begin()
				checkErr(err)
				txst = tx.Stmt(insertStmt)
			}
		}

		err = tx.Commit()
		checkErr(err)

		db.Close()
		fmt.Println("Done writing SQL")
	}()

	wg.Wait()

	fmt.Println("Stopping timer")
	timer.Stop()
	stopMetrics <- true

	<-doneChan // Wait for metrics to finish

}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
