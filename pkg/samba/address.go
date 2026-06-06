package samba

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path"
	"strings"

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

// credentialsFromContext reads the username from the -u/--username flag. It does
// not prompt for a password: a username may also be supplied in the URL, so
// prompting is deferred to ensurePassword once the final URL is assembled. A nil
// result means no username was given (an anonymous session).
func credentialsFromContext(ctx *cli.Context) *Credentials {
	if username := ctx.String(FlagUsername); username != "" {
		return &Credentials{Username: username}
	}
	return nil
}

// ensurePassword prompts for a password when the URL carries a username but no
// password yet — regardless of whether the username came from -u or from the URL
// itself. With a Kerberos keytab the secret lives in the keytab file, and an
// anonymous or Kerberos ccache session has no username, so neither prompts.
func ensurePassword(ctx *cli.Context, u *URL) error {
	if u.Credentials == nil || u.Credentials.Username == "" || u.Credentials.Password != "" {
		return nil
	}
	if ctx.Bool(FlagKerberos) && ctx.String(FlagKeytab) != "" {
		return nil
	}

	password, err := promptForPassword()
	if err != nil {
		return err
	}
	u.Credentials.Password = password

	return nil
}

// promptForPassword prompts password from terminal without echoing it
func promptForPassword() (string, error) {
	_, err := fmt.Fprint(os.Stderr, "Password:")
	if err != nil {
		return "", err
	}

	b, err := term.ReadPassword(int(os.Stdin.Fd()))
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

	u, err := newURL(ctx.Args().First(), credentialsFromContext(ctx))
	if err != nil {
		return URL{}, err
	}

	if err := ensurePassword(ctx, &u); err != nil {
		return URL{}, err
	}

	return u, nil
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
		// net.JoinHostPort brackets IPv6 literals (e.g. ::1 -> [::1]:445).
		u2.Address = net.JoinHostPort(u.Hostname(), "445")
	} else {
		u2.Address = u.Host
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
