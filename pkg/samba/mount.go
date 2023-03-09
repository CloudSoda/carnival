package samba

import (
	"net"

	"github.com/hirochachacha/go-smb2"
)

func connect(u URL) (*smb2.Session, error) {
	conn, err := net.Dial("tcp", u.Address)
	if err != nil {
		return nil, err
	}

	d := &smb2.Dialer{}
	if u.Credentials != nil {
		d.Initiator = &smb2.NTLMInitiator{
			User:     u.Credentials.Username,
			Password: u.Credentials.Password,
		}
	} else {
		d.Initiator = &smb2.NTLMInitiator{
			User: "guest",
		}
	}

	srvr, err := d.Dial(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}

	return srvr, nil
}
