package samba

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jcmturner/gokrb5/v8/config"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

// newKrbTestContext builds a *cli.Context with the Kerberos-related flags
// registered and the provided values set.
func newKrbTestContext(t *testing.T, values map[string]string) *cli.Context {
	t.Helper()
	set := flag.NewFlagSet("test", flag.ContinueOnError)
	for _, name := range []string{FlagDomain, FlagRealm, FlagSPN, FlagKeytab, FlagKrb5Conf, FlagCCache} {
		set.String(name, "", "")
	}
	for k, v := range values {
		require.NoError(t, set.Set(k, v))
	}
	return cli.NewContext(nil, set, nil)
}

func TestHostFromAddress(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"server.corp.example.com:445": "server.corp.example.com",
		"server:445":                  "server",
		"server.corp.example.com":     "server.corp.example.com",
		"10.0.0.5:445":                "10.0.0.5",
		"10.0.0.5":                    "10.0.0.5",
	}
	for in, want := range cases {
		require.Equalf(t, want, hostFromAddress(in), "hostFromAddress(%q)", in)
	}
}

func TestDeriveRealm(t *testing.T) {
	testData := []struct {
		name          string
		flags         map[string]string
		userDNSDomain string
		host          string
		expect        string
	}{
		{
			name:   "explicit realm wins and is uppercased",
			flags:  map[string]string{FlagRealm: "corp.example.com", FlagDomain: "other"},
			host:   "server.elsewhere.com",
			expect: "CORP.EXAMPLE.COM",
		},
		{
			name:   "falls back to domain",
			flags:  map[string]string{FlagDomain: "corp.example.com"},
			host:   "server.elsewhere.com",
			expect: "CORP.EXAMPLE.COM",
		},
		{
			name:          "explicit flag beats USERDNSDOMAIN",
			flags:         map[string]string{FlagRealm: "corp.example.com"},
			userDNSDomain: "env.example.com",
			host:          "server.elsewhere.com",
			expect:        "CORP.EXAMPLE.COM",
		},
		{
			name:          "USERDNSDOMAIN beats the server FQDN",
			flags:         map[string]string{},
			userDNSDomain: "corp.example.com",
			host:          "server.elsewhere.com",
			expect:        "CORP.EXAMPLE.COM",
		},
		{
			name:   "derives from fqdn",
			flags:  map[string]string{},
			host:   "server.corp.example.com",
			expect: "CORP.EXAMPLE.COM",
		},
		{
			name:   "bare hostname yields empty",
			flags:  map[string]string{},
			host:   "server",
			expect: "",
		},
		{
			name:   "ip address yields empty",
			flags:  map[string]string{},
			host:   "10.0.0.5",
			expect: "",
		},
	}

	for _, tc := range testData {
		t.Run(tc.name, func(t *testing.T) {
			// t.Setenv (used to isolate USERDNSDOMAIN) forbids t.Parallel.
			t.Setenv("USERDNSDOMAIN", tc.userDNSDomain)
			ctx := newKrbTestContext(t, tc.flags)
			require.Equal(t, tc.expect, deriveRealm(ctx, tc.host))
		})
	}
}

func TestResolveCCachePath(t *testing.T) {
	t.Run("flag wins", func(t *testing.T) {
		t.Setenv("KRB5CCNAME", "FILE:/env/path")
		ctx := newKrbTestContext(t, map[string]string{FlagCCache: "/flag/path"})
		got, err := resolveCCachePath(ctx)
		require.NoError(t, err)
		require.Equal(t, "/flag/path", got)
	})

	t.Run("env with FILE prefix is stripped", func(t *testing.T) {
		t.Setenv("KRB5CCNAME", "FILE:/env/path")
		ctx := newKrbTestContext(t, map[string]string{})
		got, err := resolveCCachePath(ctx)
		require.NoError(t, err)
		require.Equal(t, "/env/path", got)
	})

	t.Run("env without prefix passes through", func(t *testing.T) {
		t.Setenv("KRB5CCNAME", "/env/path")
		ctx := newKrbTestContext(t, map[string]string{})
		got, err := resolveCCachePath(ctx)
		require.NoError(t, err)
		require.Equal(t, "/env/path", got)
	})
}

func TestCCachePathFromName(t *testing.T) {
	t.Parallel()

	t.Run("supported and bare values resolve to a path", func(t *testing.T) {
		t.Parallel()
		cases := map[string]string{
			"FILE:/tmp/krb5cc_1000": "/tmp/krb5cc_1000",
			"file:/tmp/krb5cc_1000": "/tmp/krb5cc_1000",   // type names are case-insensitive
			"/tmp/krb5cc_1000":      "/tmp/krb5cc_1000",   // no prefix -> bare path
			`C:\Users\me\krb5cc`:    `C:\Users\me\krb5cc`, // Windows path, not a cache type
		}
		for in, want := range cases {
			got, err := ccachePathFromName(in)
			require.NoErrorf(t, err, "ccachePathFromName(%q)", in)
			require.Equalf(t, want, got, "ccachePathFromName(%q)", in)
		}
	})

	t.Run("unsupported cache types error", func(t *testing.T) {
		t.Parallel()
		for _, name := range []string{"KEYRING:persistent:1000", "KCM:", "DIR:/run/cc", "API:foo", "MEMORY:abc", "MSLSA:"} {
			_, err := ccachePathFromName(name)
			require.Errorf(t, err, "expected error for %q", name)
			require.Contains(t, err.Error(), "not supported")
		}
	})
}

func TestDefaultCCachePath(t *testing.T) {
	t.Parallel()

	got, err := defaultCCachePath(1000)
	require.NoError(t, err)
	require.Equal(t, "/tmp/krb5cc_1000", got)

	// A negative uid (os.Getuid returns -1 on Windows) has no usable default.
	_, err = defaultCCachePath(-1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Windows")
}

func TestLoadKrb5Config(t *testing.T) {
	t.Run("explicit missing --krb5conf errors", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "nope.conf")
		ctx := newKrbTestContext(t, map[string]string{FlagKrb5Conf: missing})
		_, err := loadKrb5Config(ctx, "CORP.EXAMPLE.COM")
		require.Error(t, err)
		require.Contains(t, err.Error(), "does not exist")
	})

	t.Run("missing KRB5_CONFIG path errors", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "nope.conf")
		t.Setenv("KRB5_CONFIG", missing)
		ctx := newKrbTestContext(t, map[string]string{})
		_, err := loadKrb5Config(ctx, "CORP.EXAMPLE.COM")
		require.Error(t, err)
		require.Contains(t, err.Error(), "does not exist")
	})

	t.Run("explicit existing path loads", func(t *testing.T) {
		p := filepath.Join(t.TempDir(), "krb5.conf")
		require.NoError(t, os.WriteFile(p, []byte(synthesizedKrb5Conf("CORP.EXAMPLE.COM")), 0o600))
		ctx := newKrbTestContext(t, map[string]string{FlagKrb5Conf: p})
		cfg, err := loadKrb5Config(ctx, "")
		require.NoError(t, err)
		require.Equal(t, "CORP.EXAMPLE.COM", cfg.LibDefaults.DefaultRealm)
	})
}

func TestSynthesizedKrb5Conf(t *testing.T) {
	t.Parallel()

	conf := synthesizedKrb5Conf("CORP.EXAMPLE.COM")
	require.Contains(t, conf, "default_realm = CORP.EXAMPLE.COM")
	require.Contains(t, conf, "dns_lookup_kdc = true")

	// It must be parseable by gokrb5 and produce a usable config.
	cfg, err := config.NewFromString(conf)
	require.NoError(t, err)
	require.Equal(t, "CORP.EXAMPLE.COM", cfg.LibDefaults.DefaultRealm)
	require.True(t, cfg.LibDefaults.DNSLookupKDC)
}

func TestFriendlyKrbError(t *testing.T) {
	t.Parallel()

	require.Nil(t, friendlyKrbError(nil))

	skew := friendlyKrbError(fmt.Errorf("KRB_AP_ERR_SKEW: clock difference"))
	require.Contains(t, skew.Error(), "sync system time")

	// Unrecognized errors are returned unchanged.
	other := fmt.Errorf("some unrelated failure")
	require.Equal(t, other.Error(), friendlyKrbError(other).Error())
}
