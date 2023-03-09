package samba

import (
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/urfave/cli/v2"
)

type Credentials struct {
	Username string
	Password string
}

type URL struct {
	Address     string
	Share       string
	Path        string
	Credentials *Credentials
}

func urlFromContext(ctx *cli.Context) (URL, error) {
	if ctx.NArg() == 0 {
		return URL{}, errors.New("missing smb url")
	}

	return newURL(ctx.Args().First())
}

func newURL(str string) (URL, error) {
	u2 := URL{}
	u, err := url.Parse(str)
	if err != nil {
		return u2, fmt.Errorf("smb url error: %v", err)
	}
	if u.Host == "" {
		return u2, fmt.Errorf("smb url has no address")
	}
	if u.Port() == "" {
		u2.Address = u.Host + ":445"
	} else {
		u2.Address = u.Host
	}
	if u.Port() == "" {
		u2.Address = u.Hostname() + ":445"
	}
	if u.User != nil {
		u2.Credentials = &Credentials{
			Username: u.User.Username(),
		}
		if pass, ok := u.User.Password(); ok {
			u2.Credentials.Password = pass
		}
	}

	trimmedPath := strings.TrimPrefix(u.Path, "/")
	parts := strings.Split(trimmedPath, "/")
	if len(parts) > 0 {
		u2.Share = parts[0]
	}
	if len(parts) > 1 {
		u2.Path = path.Join(parts[1:]...)
	}

	return u2, nil
}
