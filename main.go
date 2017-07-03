package main

import (
	"couch2mq/config"
	"couch2mq/couchdb"
	"couch2mq/logger"
	"couch2mq/oc"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"os"
	"runtime/debug"
	"time"

	"github.com/gchaincl/dotsql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/kr/pretty"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}
func getChanges(client *couchdb.Client, dbname string, since string) (*couchdb.Changes, error) {
	//db, err := client.EnsureDB(dbname)
	db, err := client.DB(dbname)
	failOnError(err, "Failed to connect to "+dbname)
	return db.NormalChanges(since)
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
				debug.PrintStack()
				pretty.Println("Recover from error:", r)
			}
		}()
		fn()
	}
	for {
		f()
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
	pretty.Println("Commit transaction", order.Order.OrderInfo.OrderID)
	return tx.Commit()
}

const seqPrefixLen = 20

func handleOrders() {
	lg, err := logger.New("order_seq")
	failOnError(err, "Failed to open database")
	defer lg.Close()
	if (len(os.Args) == 2) && (os.Args[1] == "--init") {
		pretty.Println("Initialize database")
		dot, err := dotsql.LoadFromFile("oc.sql")
		failOnError(err, "Failed to initialize database")
		dot.Exec(lg.DB(), "use-oc")
		dot.Exec(lg.DB(), "set-encoding")
		dot.Exec(lg.DB(), "disable-foreign-key")
		dot.Exec(lg.DB(), "drop-order-discount")
		dot.Exec(lg.DB(), "create-order-discount")
		dot.Exec(lg.DB(), "drop-order-detail")
		dot.Exec(lg.DB(), "create-order-detail")
		dot.Exec(lg.DB(), "drop-order-master")
		dot.Exec(lg.DB(), "create-order-master")
		dot.Exec(lg.DB(), "drop-order-meal-detail")
		dot.Exec(lg.DB(), "create-order-meal-detail")
		dot.Exec(lg.DB(), "drop-order-seq")
		dot.Exec(lg.DB(), "create-order-seq")
		dot.Exec(lg.DB(), "drop-shift-seq")
		dot.Exec(lg.DB(), "create-shift-seq")
		dot.Exec(lg.DB(), "enable-foreign-key")
	}
	seq, err := lg.Seq()
	failOnError(err, "Failed to get latest sequence number")
	err = lg.Clean()
	failOnError(err, "Failed to clean up log")
	for {
		d, _ := time.ParseDuration("5s")
		time.Sleep(d)
		couchcfg := make(map[string]interface{})
		err := config.Get("$.couchdb+", &couchcfg)
		failOnError(err, "Empty CouchDB configuration")
		client, err := couchdb.New(couchcfg["url"].(string), couchcfg["username"].(string), couchcfg["password"].(string))
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
	}
}

func main() {

	forever(handleOrders)
}
