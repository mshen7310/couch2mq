package main

import (
	"couch2mq/couchdb"
	"couch2mq/logger"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"couch2mq/oc"

	_ "github.com/go-sql-driver/mysql"
	"github.com/kr/pretty"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
		panic(fmt.Sprintf("%s: %s", msg, err))
	}
}
func getChanges(client *couchdb.Client, dbname string, since string) (*couchdb.Changes, error) {
	//db, err := client.EnsureDB(dbname)
	db, err := client.DB(dbname)
	failOnError(err, "Failed to connect to "+dbname)
	return db.Changes(since)
}

func panicOnError(err error) {
	if err != nil {
		panic(err)
	}
}
func forever(fn func()) {
	f := func() {
		defer func() {
			if r := recover(); r != nil {
				pretty.Println("Recover from error:", r)
			}
		}()
		fn()
	}
	for {
		f()
		d, _ := time.ParseDuration("5s")
		time.Sleep(d)
	}
}
func doOrder(db *sql.DB, order oc.OrderJSON) error {
	statements := order.Do(db)
	tx, err := db.Begin()
	failOnError(err, "Failed to begine transaction")
	defer tx.Rollback()
	for _, stmt := range statements {
		_, err := tx.Exec(stmt)
		failOnError(err, "Failed to exec "+stmt)
	}
	pretty.Println("Commit transaction", order.OrderID)
	return tx.Commit()
}

const seqPrefixLen = 20

func handleOrders() {
	lg, err := logger.New("order_seq")
	failOnError(err, "Failed to open database")
	defer lg.Close()
	seq, err := lg.Seq()
	failOnError(err, "Failed to get latest sequence number")
	err = lg.Clean()
	failOnError(err, "Failed to clean up log")
	forever(func() {
		client, err := couchdb.New("http://couchdb-cloud.gtdx.liansuola.com", "ymeng", "111111")
		failOnError(err, "Failed to connect to CouchDB")
		ch, err := getChanges(client, "orders", seq)
		failOnError(err, "Failed to get changes of orders")
		for _, c := range ch.Results {
			var dst oc.OrderJSON
			err = json.Unmarshal(c.Doc, &dst)
			if (err != nil) || (dst.Order.OrderInfo.OrderID == "") {
				seq = string(c.Seq)
				pretty.Println("Cannot unmarshal doc", c.ID, seq[:seqPrefixLen])
				if err == nil {
					err = lg.Update(seq, c.ID, errors.New("Wrong JSON format"))
					pretty.Println("Wrong JSON format", seq[:seqPrefixLen])
				} else {
					pretty.Println(err.Error(), seq[:seqPrefixLen])
					err = lg.Update(seq, c.ID, err)
				}
				if err != nil {
					pretty.Println(err.Error(), seq[:seqPrefixLen])
				}
				continue
			}
			//pretty.Println(dst)
			err = doOrder(lg.DB(), dst)
			if err == nil {
				seq = string(c.Seq)
				pretty.Println("Handle doc successfully", c.ID, seq[:seqPrefixLen])
				err = lg.Update(seq, c.ID, errors.New("Success"))
				if err != nil {
					pretty.Println(err, seq[:seqPrefixLen])
				}
			}
		}
	})
}

func main() {
	//	conn, err := amqp.Dial("amqp://guest:guest@mq.liansuola.com:5672/")
	//	failOnError(err, "Failed to connect to RabbitMQ")
	//	defer conn.Close()

	//	db, err := oc.Open("mysql", "root@tcp(127.0.0.1:3306)/oc")
	//	failOnError(err, "Failed to connect to MySQL")
	//	defer db.Close()
	handleOrders()
}
