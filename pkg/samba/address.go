package samba

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"strings"
	"syscall"

	"github.com/urfave/cli/v2"
	"golang.org/x/term"
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

// credentialsFromContext gets username from cli context and if set, it prompts the password from the terminal.
// If the flag is not set, then empty credentials will be returned (for anonymous session).
func credentialsFromContext(ctx *cli.Context) (*Credentials, error) {
	creds := &Credentials{}

	if username := ctx.String(FlagUsername); username != "" {
		password, err := promptForPassword()
		if err != nil {
			return nil, err
		}

		creds.Username = username
		creds.Password = password
	}

	return creds, nil
}

// promptForPassword prompts password from terminal without echoing it
func promptForPassword() (string, error) {
	_, err := fmt.Fprint(os.Stderr, "Password:")
	if err != nil {
		return "", err
	}

	b, err := term.ReadPassword(syscall.Stdin)
	if err != nil {
		return "", err
	}

	_, _ = fmt.Fprintln(os.Stderr)

	return string(b), nil
}

func urlFromContext(ctx *cli.Context) (URL, error) {
	if ctx.NArg() == 0 {
		return URL{}, errors.New("missing smb url")
	}

	creds, err := credentialsFromContext(ctx)
	if err != nil {
		return URL{}, err
	}

	return newURL(ctx.Args().First(), creds)
}

func newURL(str string, creds *Credentials) (URL, error) {
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

	if creds != nil {
		u2.Credentials = creds
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
