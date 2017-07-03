package tunnel

import (
	"database/sql"
	"fmt"
	"net"

	"github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/ssh"
)

type viaSSHDialer struct {
	client *ssh.Client
}

//Dial see document of mysql driver
func (d *viaSSHDialer) Dial(addr string) (net.Conn, error) {
	return d.client.Dial("tcp", addr)
}

// Tunnel is a mysql connection via SSH tunnel
type Tunnel struct {
	Connection *ssh.Client
	Database   *sql.DB
}

//Close close a tunnel
func (t *Tunnel) Close() error {
	if t.Connection != nil {
		t.Database.Close()
		return t.Connection.Close()
	}
	return t.Database.Close()
}

//Open open a tunnel
func Open(dbHost string, dbPort int, dbUser string, dbPass string, dbName string) (*Tunnel, error) {
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@mysql+tcp(%s:%d)/%s", dbUser, dbPass, dbHost, dbPort, dbName))
	if err == nil {
		t := Tunnel{
			Database:   db,
			Connection: nil,
		}
		return &t, nil
	}
	return nil, err
}

//OpenSSH open a tunnel
func OpenSSH(sshHost string, sshPort int, sshUser string, sshPass string, dbHost string, dbPort int, dbUser string, dbPass string, dbName string) (*Tunnel, error) {
	sshConfig := &ssh.ClientConfig{
		User: sshUser,
		Auth: []ssh.AuthMethod{
			ssh.Password(sshPass),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	sshcon, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", sshHost, sshPort), sshConfig)
	if err == nil {
		mysql.RegisterDial("mysql+tcp", (&viaSSHDialer{sshcon}).Dial)
		db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@mysql+tcp(%s:%d)/%s", dbUser, dbPass, dbHost, dbPort, dbName))
		if err == nil {
			t := Tunnel{
				Database:   db,
				Connection: sshcon,
			}
			return &t, nil
		}
		sshcon.Close()
	}
	return nil, err
}
