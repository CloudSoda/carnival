package main

import (
	"fmt"
	"os"
	"runtime"

	"cloudsoda.dev/carnival/pkg/samba"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

// kerberosOnly returns a flag Action that rejects the flag unless Kerberos
// authentication (--kerberos) is enabled.
func kerberosOnly(flag string) func(*cli.Context, string) error {
	return func(ctx *cli.Context, _ string) error {
		if !ctx.Bool(samba.FlagKerberos) {
			return fmt.Errorf("--%s may only be used with kerberos authentication (--%s)", flag, samba.FlagKerberos)
		}
		return nil
	}
}

func main() {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = zerolog.New(zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: "2006-01-02T15:04:05",
	}).With().Timestamp().Logger()

	prog := cli.NewApp()
	prog.Name = "Carnival"
	prog.HelpName = "carnival"
	prog.Usage = "For testing connectivity to SMB shares"
	prog.Description = "Carnival authenticates with NTLM by default. Pass -k/--kerberos to use\n" +
		"Kerberos instead. Kerberos has a few requirements:\n" +
		"\n" +
		"  * The server must be addressed by its fully-qualified domain name (FQDN),\n" +
		"    not an IP, so the service principal (cifs/<fqdn>) matches the KDC.\n" +
		"  * The client and KDC clocks must be in sync (within ~5 minutes).\n" +
		"  * You must supply credentials in one of three ways:\n" +
		"      - a username (-u) and the password prompt,\n" +
		"      - a keytab file (--keytab) together with -u, or\n" +
		"      - an existing file ticket cache from 'kinit' (honors KRB5CCNAME).\n" +
		"\n" +
		"The realm is taken from --realm, then --domain, then a krb5.conf\n" +
		"default_realm, then the server's domain (and %USERDNSDOMAIN% on Windows).\n" +
		"If no krb5.conf is found (--krb5conf / KRB5_CONFIG / /etc/krb5.conf), a\n" +
		"minimal DNS-discovery config is used, which usually works for Active Directory.\n" +
		"\n" +
		"On Windows there is no single sign-on: carnival cannot read the LSA ticket\n" +
		"cache, so use -u (with the password prompt) or a keytab. A 'kinit' cache\n" +
		"works only if it is a file cache pointed to by KRB5CCNAME or --ccache."
	prog.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:  samba.FlagDomain,
			Usage: "the user account's `DOMAIN` (used as the Kerberos realm if --realm is unset)",
		},
		&cli.BoolFlag{
			Aliases: []string{"k"},
			Name:    samba.FlagKerberos,
			Usage:   "authenticate with Kerberos instead of NTLM",
		},
		&cli.StringFlag{
			Name:   samba.FlagKeytab,
			Usage:  "Kerberos only: authenticate using the keytab at `PATH` (requires -u)",
			Action: kerberosOnly(samba.FlagKeytab),
		},
		&cli.BoolFlag{
			Name:  samba.FlagMapchars,
			Usage: "use the equivalent of the samba 'mapchars' option",
		},
		&cli.BoolFlag{
			Name:  samba.FlagMapposix,
			Usage: "use the equivalent of the samba 'mapposix' option",
		},
		&cli.StringFlag{
			Name:   samba.FlagRealm,
			Usage:  "Kerberos only: the `REALM` to authenticate against (e.g. CORP.EXAMPLE.COM)",
			Action: kerberosOnly(samba.FlagRealm),
		},
		&cli.StringFlag{
			Name:   samba.FlagSPN,
			Usage:  "Kerberos only: override the target service principal name (default cifs/<server-fqdn>)",
			Action: kerberosOnly(samba.FlagSPN),
		},
		&cli.StringFlag{
			Name:   samba.FlagKrb5Conf,
			Usage:  "Kerberos only: `PATH` to a krb5.conf (default: $KRB5_CONFIG or /etc/krb5.conf)",
			Action: kerberosOnly(samba.FlagKrb5Conf),
		},
		&cli.StringFlag{
			Name:   samba.FlagCCache,
			Usage:  "Kerberos only: `PATH` to a credential cache (default: $KRB5CCNAME or /tmp/krb5cc_<uid>)",
			Action: kerberosOnly(samba.FlagCCache),
		},
		&cli.StringFlag{
			Aliases: []string{"u"},
			Name:    samba.FlagUsername,
			Usage:   "use the given username, the program will prompt for password",
		},
	}

	prog.Commands = []*cli.Command{
		{
			Name:      "cp",
			Usage:     "copy a file from a samba share to a local destination",
			UsageText: "carnival cp [smburl] [destination]",
			Action:    samba.Copy,
		},
		// {
		// 	Name:      "cpto",
		// 	Usage:     "copy a file to a samba share from a local source",
		// 	UsageText: "carnival cpto [fromlocal] [tosmburl]",
		// 	Action:    samba.CopyTo,
		// },
		{
			Name:      "help",
			Usage:     "show the app or a command's help text",
			UsageText: "carnival help [command]",
			Action:    prog.Action, // the default action is help
		},
		{
			Name:      "ls",
			Usage:     "list the contents of a directory",
			UsageText: "carnival ls [smburl]",
			Action:    samba.List,
		},
		{
			Name:      "md5",
			Usage:     "calculate the md5 hash of a file",
			UsageText: "carnival md5 [smburl]",
			Action:    samba.MD5,
		},
		{
			Name:      "sd",
			Usage:     "display the security descriptor of a file in string format",
			UsageText: "carnival sd [smburl]",
			Action:    samba.Sd,
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "pretty",
					Usage: "when set, will output the security descriptor indented",
				},
			},
		},
		{
			Name:      "shares",
			Usage:     "list the publicly visible shares",
			UsageText: "carnival shares [smburl]",
			Action:    samba.Shares,
		},
	}

	if err := prog.Run(os.Args); err != nil {
		if runtime.GOOS != "windows" {
			fmt.Fprintln(os.Stderr, err)
		} else {
			// cmd & powershell automatically write a newline
			// after the program terminates
			fmt.Fprint(os.Stderr, err)
		}
		os.Exit(1)
	}
}
