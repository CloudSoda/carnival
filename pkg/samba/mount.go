package samba

import (
	"fmt"
	"net"

	"github.com/cloudsoda/go-smb2"
)

func parseOptions(strOpts []string) ([]smb2.MountOption, error) {
	var mos []smb2.MountOption
	for _, so := range strOpts {
		switch so {
		case "mapchars":
			mos = append(mos, smb2.WithMapChars())
		case "mapposix":
			mos = append(mos, smb2.WithMapPosix())
		default:
			return nil, fmt.Errorf("unknown mount options: %v", so)
		}
	}

	return mos, nil
}

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
