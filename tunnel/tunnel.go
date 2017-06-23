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
	t.Database.Close()
	return t.Connection.Close()
}

//Open open a tunnel
func Open(sshHost string, sshPort int, sshUser string, sshPass string, dbHost string, dbPort int, dbUser string, dbPass string, dbName string) (*Tunnel, error) {
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
