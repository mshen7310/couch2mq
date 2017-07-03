package logger

import (
	"couch2mq/config"
	"couch2mq/tunnel"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
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
	sql := make(map[string]interface{})
	err := config.Get("$.mysql+", &sql)
	if err == nil {
		if s, ok := sql["ssh"]; ok {
			ssh := s.(map[string]interface{})
			t, err := tunnel.OpenSSH(
				ssh["host"].(string),
				int(ssh["port"].(float64)),
				ssh["username"].(string),
				ssh["password"].(string),
				sql["host"].(string),
				int(sql["port"].(float64)),
				sql["username"].(string),
				sql["password"].(string),
				sql["database"].(string))
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
		t, err := tunnel.Open(
			sql["host"].(string),
			int(sql["port"].(float64)),
			sql["username"].(string),
			sql["password"].(string),
			sql["database"].(string))
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
	panic("There is no available mysql configuration")
}

//DB return the database handle
func (log *Logger) DB() *sql.DB {
	return log.db
}

//Close closes the log database
func (log *Logger) Close() error {
	return log.ssh.Close()
}

//Truncate clean up table
func (log *Logger) Truncate() error {
	_, err := log.db.Exec(fmt.Sprintf(`TRUNCATE TABLE %s`, log.table))
	return err
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
			return err
		}
		_, err = stmt.Exec(id, seq, docid, inerr.Error())
		return err
	}
	return err
}
