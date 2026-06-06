package samba

import (
	"fmt"
	"net"
	"net/netip"
	"os"
	"strings"

	"github.com/cloudsoda/go-smb2"
	"github.com/jcmturner/gokrb5/v8/client"
	"github.com/jcmturner/gokrb5/v8/config"
	"github.com/jcmturner/gokrb5/v8/credentials"
	"github.com/jcmturner/gokrb5/v8/keytab"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

// defaultKrb5ConfPath is the conventional location of the system Kerberos
// configuration on unix-like systems.
const defaultKrb5ConfPath = "/etc/krb5.conf"

// newKrb5Initiator builds a go-smb2 Kerberos initiator from the user-supplied
// flags and the target URL. It picks a credential source in the following order
// of precedence:
//
//  1. a keytab file (--keytab), for non-interactive / service-account use
//  2. a username + password (the standard prompt flow), doing an AS exchange
//  3. an existing credential cache (kinit tickets), the zero-config default
//
// Whenever possible it degrades gracefully: a missing krb5.conf is tolerated by
// synthesizing a minimal, DNS-discovery config (which is what most Active
// Directory environments need), and common Kerberos failures are translated
// into actionable messages.
func newKrb5Initiator(ctx *cli.Context, u URL) (*smb2.Krb5Initiator, error) {
	host := hostFromAddress(u.Address)

	spn := ctx.String(FlagSPN)
	if spn == "" {
		spn = "cifs/" + host
	}

	if _, err := netip.ParseAddr(host); err == nil {
		log.Warn().Msgf("'%s' is an IP address; Kerberos normally requires the server's FQDN so the SPN (%s) matches what the KDC has registered. Pass --%s or use a hostname if authentication fails.", host, spn, FlagSPN)
	}

	// Best realm guess available before any krb5.conf is read; used to
	// synthesize a DNS-discovery config when no krb5.conf exists.
	realm := deriveRealm(ctx, host)

	cfg, err := loadKrb5Config(ctx, realm)
	if err != nil {
		return nil, err
	}
	// A krb5.conf default_realm reflects the user's account realm and is more
	// authoritative than guessing from the server's FQDN, so prefer it whenever
	// the realm wasn't given explicitly via --realm/--domain. (When we
	// synthesized the config above, its default_realm is just `realm`, so this
	// is a harmless no-op in that case.)
	if explicitRealm(ctx) == "" && cfg.LibDefaults.DefaultRealm != "" {
		realm = cfg.LibDefaults.DefaultRealm
	}

	cl, err := newKrb5Client(ctx, u, cfg, realm)
	if err != nil {
		return nil, err
	}

	return &smb2.Krb5Initiator{
		Client:    cl,
		TargetSPN: spn,
	}, nil
}

// newKrb5Client constructs and logs in a gokrb5 client using whichever
// credential source the user provided.
func newKrb5Client(ctx *cli.Context, u URL, cfg *config.Config, realm string) (*client.Client, error) {
	username := ""
	if u.Credentials != nil {
		username = u.Credentials.Username
	}

	// 1. keytab
	if ktPath := ctx.String(FlagKeytab); ktPath != "" {
		if username == "" {
			return nil, fmt.Errorf("--%s requires a username (use -u/--%s or include it in the smb url)", FlagKeytab, FlagUsername)
		}
		if realm == "" {
			return nil, errRealmRequired()
		}
		kt, err := keytab.Load(ktPath)
		if err != nil {
			return nil, fmt.Errorf("loading keytab '%s': %w", ktPath, err)
		}
		cl := client.NewWithKeytab(username, realm, kt, cfg)
		if err := cl.Login(); err != nil {
			return nil, friendlyKrbError(fmt.Errorf("kerberos login with keytab failed: %w", err))
		}
		return cl, nil
	}

	// 2. username + password
	if username != "" {
		if realm == "" {
			return nil, errRealmRequired()
		}
		password := ""
		if u.Credentials != nil {
			password = u.Credentials.Password
		}
		cl := client.NewWithPassword(username, realm, password, cfg)
		if err := cl.Login(); err != nil {
			return nil, friendlyKrbError(fmt.Errorf("kerberos login for %s@%s failed: %w", username, realm, err))
		}
		return cl, nil
	}

	// 3. existing credential cache (kinit)
	ccPath, err := resolveCCachePath(ctx)
	if err != nil {
		return nil, err
	}
	cc, err := credentials.LoadCCache(ccPath)
	if err != nil {
		return nil, fmt.Errorf("no Kerberos credentials provided and the credential cache at '%s' could not be read: %w\n"+
			"Run 'kinit' to obtain a ticket, or pass -u/--%s (for a password) or --%s", ccPath, err, FlagUsername, FlagKeytab)
	}
	cl, err := client.NewFromCCache(cc, cfg)
	if err != nil {
		return nil, fmt.Errorf("creating Kerberos client from credential cache '%s': %w", ccPath, err)
	}
	return cl, nil
}

// hostFromAddress strips the port from an "host:port" address, returning just
// the host. If there is no port it returns the address unchanged.
func hostFromAddress(address string) string {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return address
	}
	return host
}

// deriveRealm derives the Kerberos account realm without consulting krb5.conf,
// in order of authority: an explicit --realm/--domain, the %USERDNSDOMAIN% of a
// domain-joined Windows host (which is the account's DNS domain, i.e. its
// realm), and finally the domain portion of the target's FQDN. Realms are
// conventionally uppercase. An empty string is returned when nothing can be
// derived (e.g. a bare hostname or IP off-domain), in which case the caller may
// still fall back to the config's default_realm.
//
// Note the FQDN fallback yields the *server's* domain, which only equals the
// account realm in a single-domain setup; multi-domain users should pass
// --realm/--domain (or have a krb5.conf default_realm).
func deriveRealm(ctx *cli.Context, host string) string {
	if r := explicitRealm(ctx); r != "" {
		return r
	}
	// Domain-joined Windows exposes the logged-in account's DNS domain here.
	if r := os.Getenv("USERDNSDOMAIN"); r != "" {
		return strings.ToUpper(r)
	}
	return guessRealmFromHost(host)
}

// explicitRealm returns the realm the user named explicitly, preferring --realm
// over --domain, uppercased. It returns "" when neither flag is set.
func explicitRealm(ctx *cli.Context) string {
	if r := ctx.String(FlagRealm); r != "" {
		return strings.ToUpper(r)
	}
	if d := ctx.String(FlagDomain); d != "" {
		return strings.ToUpper(d)
	}
	return ""
}

// guessRealmFromHost derives a realm from the domain portion of an FQDN:
// host.corp.example.com -> CORP.EXAMPLE.COM. It returns "" for a bare hostname
// or an IP address.
func guessRealmFromHost(host string) string {
	if net.ParseIP(host) == nil {
		if i := strings.IndexByte(host, '.'); i >= 0 && i < len(host)-1 {
			return strings.ToUpper(host[i+1:])
		}
	}
	return ""
}

// resolveCCachePath determines which credential cache to read, honoring (in
// order) the --ccache flag, the KRB5CCNAME environment variable, and finally
// the conventional /tmp/krb5cc_<uid> location. It returns an error when the
// requested cache type cannot be read (gokrb5 only supports file caches) or
// when no default location can be determined (e.g. on Windows).
func resolveCCachePath(ctx *cli.Context) (string, error) {
	if p := ctx.String(FlagCCache); p != "" {
		return p, nil
	}
	if env := os.Getenv("KRB5CCNAME"); env != "" {
		return ccachePathFromName(env)
	}
	return defaultCCachePath(os.Getuid())
}

// ccachePathFromName resolves a KRB5CCNAME value to a file path. KRB5CCNAME may
// carry a cache-type prefix such as "FILE:/path", "KEYRING:...", "KCM:" etc.
// gokrb5 can only read file caches, so a "FILE:" prefix is stripped, a value
// without a recognized type prefix is treated as a bare file path (this also
// covers Windows paths like C:\... whose "C" is not a known cache type), and a
// recognized-but-unsupported type yields an actionable error.
func ccachePathFromName(name string) (string, error) {
	if i := strings.IndexByte(name, ':'); i > 0 {
		switch strings.ToUpper(name[:i]) {
		case "FILE":
			return name[i+1:], nil
		case "DIR", "KEYRING", "KCM", "API", "MEMORY", "MSLSA":
			return "", fmt.Errorf("the KRB5CCNAME credential cache type %q is not supported; carnival can only read file-based caches\n"+
				"obtain a file ticket cache (e.g. 'kinit -c FILE:/tmp/krb5cc') or pass --%s with a file path", name[:i], FlagCCache)
		}
	}
	return name, nil
}

// defaultCCachePath returns the conventional /tmp/krb5cc_<uid> file cache for
// the given uid. A negative uid (os.Getuid returns -1 on Windows) means there
// is no usable default — notably on domain-joined Windows hosts, where tickets
// live in the LSA cache that gokrb5 cannot read.
func defaultCCachePath(uid int) (string, error) {
	if uid < 0 {
		return "", fmt.Errorf("could not determine a default Kerberos credential cache location\n"+
			"this is expected on Windows, where carnival cannot read the LSA ticket cache; "+
			"provide credentials via --%s (a file cache), -u/--%s, or --%s", FlagCCache, FlagUsername, FlagKeytab)
	}
	return fmt.Sprintf("/tmp/krb5cc_%d", uid), nil
}

// loadKrb5Config loads the Kerberos configuration. It honors --krb5conf, then
// the KRB5_CONFIG environment variable, then /etc/krb5.conf. If none of those
// exist it synthesizes a minimal configuration that relies on DNS SRV discovery
// to locate the KDC — the common case for Active Directory — provided a realm
// is known.
func loadKrb5Config(ctx *cli.Context, realm string) (*config.Config, error) {
	path := ctx.String(FlagKrb5Conf)
	explicit := path != ""
	if path == "" {
		path = os.Getenv("KRB5_CONFIG")
		explicit = path != ""
	}
	if path == "" {
		path = defaultKrb5ConfPath
	}

	if _, err := os.Stat(path); err == nil {
		cfg, err := config.Load(path)
		if err != nil {
			return nil, fmt.Errorf("loading krb5 config '%s': %w", path, err)
		}
		return cfg, nil
	} else if !os.IsNotExist(err) {
		// A real error (permissions, etc.) — surface it rather than silently
		// falling back.
		return nil, fmt.Errorf("reading krb5 config '%s': %w", path, err)
	} else if explicit {
		// The user pointed us at a specific file (via --krb5conf or
		// KRB5_CONFIG); a missing one is a mistake to report, not a reason to
		// silently guess a different configuration.
		return nil, fmt.Errorf("krb5 config '%s' does not exist (set via --%s or KRB5_CONFIG)", path, FlagKrb5Conf)
	}

	// No config file: synthesize one that uses DNS discovery.
	if realm == "" {
		return nil, fmt.Errorf("no krb5 configuration found at '%s' and no realm could be determined; "+
			"pass --%s (and/or --%s) or provide a krb5.conf via --%s or KRB5_CONFIG", path, FlagRealm, FlagDomain, FlagKrb5Conf)
	}
	log.Warn().Msgf("no krb5 configuration found at '%s'; using DNS-based KDC discovery for realm %s (typical for Active Directory)", path, realm)

	cfg, err := config.NewFromString(synthesizedKrb5Conf(realm))
	if err != nil {
		return nil, fmt.Errorf("building fallback krb5 configuration: %w", err)
	}
	return cfg, nil
}

// synthesizedKrb5Conf returns a minimal krb5.conf body that enables DNS SRV
// lookups for the given realm. (dns_lookup_realm is deliberately omitted: gokrb5
// resolves realms only from [domain_realm]/default_realm, never via DNS, so the
// option would be a no-op here.)
func synthesizedKrb5Conf(realm string) string {
	return fmt.Sprintf("[libdefaults]\n"+
		"  default_realm = %s\n"+
		"  dns_lookup_kdc = true\n", realm)
}

func errRealmRequired() error {
	return fmt.Errorf("could not determine the Kerberos realm; pass --%s (e.g. --%s CORP.EXAMPLE.COM), --%s, or use a fully-qualified server name", FlagRealm, FlagRealm, FlagDomain)
}

// friendlyKrbError augments common, opaque Kerberos errors with a hint about
// the likely cause. The underlying error is always preserved.
func friendlyKrbError(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	for _, c := range []struct{ fragment, hint string }{
		{"Clock skew", "the client and KDC clocks differ too much; sync system time (e.g. via NTP) and retry"},
		{"KRB_AP_ERR_SKEW", "the client and KDC clocks differ too much; sync system time (e.g. via NTP) and retry"},
		{"KDC_ERR_PREAUTH_FAILED", "pre-authentication failed; check the username, password and realm"},
		{"Preauthentication failed", "pre-authentication failed; check the username, password and realm"},
		{"KDC_ERR_C_PRINCIPAL_UNKNOWN", "the KDC does not know this user; check the username and realm"},
		{"KDC_ERR_S_PRINCIPAL_UNKNOWN", "the KDC has no service principal for this server; check the SPN/--spn and that you used the server's FQDN"},
		{"KDC_ERR_ETYPE_NOSUPP", "no common encryption type with the KDC; check the supported enctypes in krb5.conf"},
	} {
		if strings.Contains(msg, c.fragment) {
			return fmt.Errorf("%w\nhint: %s", err, c.hint)
		}
	}
	// Network-level failures reaching the KDC.
	if strings.Contains(msg, "connection refused") || strings.Contains(msg, "no such host") || strings.Contains(msg, "i/o timeout") || strings.Contains(msg, "lookup") {
		return fmt.Errorf("%w\nhint: could not reach the KDC; verify the realm's KDC addresses (krb5.conf) or DNS SRV records", err)
	}
	return err
}
