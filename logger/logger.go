package logger

import (
	"couch2mq/tunnel"
	"database/sql"
	"strconv"
	"strings"

	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

//Logger holds the handle of database
type Logger struct {
	ssh   *tunnel.Tunnel
	db    *sql.DB
	table string
}

func seq2index(seq string) (int, error) {
	s := strings.SplitN(seq, "-", 2)
	return strconv.Atoi(s[0])
}

//New creates log database and create sequence table
func New(tbl string) (*Logger, error) {
	//for production
	//t, err := tunnel.Open("54.223.176.133", 22, "kenlin", "thomas123", "sm12g5l9d32eyun.cjwa2zciaejp.rds.cn-north-1.amazonaws.com.cn", 3306, "keithyau", "thomas123", "oc")
	//for testing
	t, err := tunnel.Open("54.223.176.133", 22, "kenlin", "thomas123", "sparkpadgptest.cjwa2zciaejp.rds.cn-north-1.amazonaws.com.cn", 3306, "keithyau", "thomas123", "oc")
	//for local development
	//d, err := sql.Open("mysql", "root@tcp(127.0.0.1:3306)/oc")
	if err == nil {
		log := Logger{
			ssh:   t,
			db:    t.Database,
			table: tbl,
		}
		return &log, nil
	}
	return nil, err
}

//DB return the database handle
func (log *Logger) DB() *sql.DB {
	return log.db
}

//Close closes the log database
func (log *Logger) Close() error {
	return log.ssh.Close()
}

//Clean clears sequence table except the latest one
func (log *Logger) Clean() error {
	mid, err := log.MaxID()
	if err == nil {
		_, err := log.db.Exec(fmt.Sprintf(`DELETE FROM %s WHERE id < ?`, log.table), mid)
		return err
	}
	return err
}

//Count returns the count of records in sequence
func (log *Logger) Count() (int, error) {
	rows, err := log.db.Query(fmt.Sprintf("SELECT COUNT(*) FROM %s", log.table))
	if err == nil {
		defer rows.Close()
		if rows.Next() {
			cn := 0
			err = rows.Scan(&cn)
			if err == nil {
				return cn, nil
			}
		}
	}
	return 0, err
}

//MaxID return the max id in sequence
func (log *Logger) MaxID() (int, error) {
	cn, err := log.Count()
	if err == nil {
		if cn == 0 {
			return 1, nil
		}
	}
	rows, err := log.db.Query(fmt.Sprintf("SELECT MAX(id) FROM %s", log.table))
	if err == nil {
		defer rows.Close()
		if rows.Next() {
			id := 1
			err = rows.Scan(&id)
			if err == nil {
				return id, nil
			}
		}
	}
	return 1, err
}

//Seq retrieves the latest sequence number
func (log *Logger) Seq() (string, error) {
	cn, err := log.Count()
	if err == nil {
		if cn == 0 {
			return "", nil
		}
	}
	mid, err := log.MaxID()
	if err == nil {
		rows, err := log.db.Query(fmt.Sprintf("SELECT seq FROM %s where id=?", log.table), mid)
		if err == nil {
			defer rows.Close()
			if rows.Next() {
				seq := ""
				err = rows.Scan(&seq)
				if err == nil {
					return seq, nil
				}
			}
		}
	}
	return "", nil
}

//Update updates the lastest sequence number
func (log *Logger) Update(seq string, docid string, inerr error) error {
	stmt, err := log.db.Prepare(fmt.Sprintf(`INSERT INTO %s(id, seq, docid, error) VALUES(?,?,?,?)`, log.table))
	if err == nil {
		id, _ := seq2index(seq)
		if inerr == nil {
			_, err = stmt.Exec(id, seq, docid, "nil")
		} else {
			_, err = stmt.Exec(id, seq, docid, inerr.Error())
		}

	}
	return err
}
