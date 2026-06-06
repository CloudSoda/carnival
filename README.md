# carnival

A tool for testing connectivity to SMB shares. With it you can:

### Calculate the MD5 hash of a file
```
$ carnival -u user md5 smb://address/sharename/path/to/file.txt
Password:
```

### Copy a file from an SMB share to a local destination
```
$ carnival -u user cp smb://address/sharename/file.bin .
Password:
# OR rename it at the destination
$ carnival -u user cp smb://address/sharename/file.bin file.binary
Password:
```
The `cp` command will tell you how long the transfer took and the average transfer speed.

### List the files in a directory
```
# List the files at the root of the share
$ carnival -u user ls smb://address/sharename/
Password:

# List the files in the 'Games' directory of the share
$ carnival -u user ls smb://address/sharename/Games
Password:
```

### Print the names of publicly visible SMB shares
```
$ carnival -u user shares smb://address
Password:
```

If the username is set through the `-u/--username` flag, the password will be prompted

If no username is set, an anonymous session will be attempted

## Authentication

By default carnival authenticates with **NTLM**. Pass `-k/--kerberos` on any
command to use **Kerberos** instead.

### Kerberos

Kerberos is stricter than NTLM, so a few things must be in place:

- **Use the server's FQDN, not an IP.** The service principal carnival requests
  is `cifs/<server-fqdn>`, and the KDC will only issue a ticket if that name
  matches what it has registered. Override it with `--spn` if needed.
- **Clocks must be in sync.** The client and KDC must agree on the time to
  within ~5 minutes, otherwise authentication fails with a clock-skew error.
- **A realm must be known.** It is taken from `--realm`, then `--domain`, then
  the domain portion of the server's FQDN. Realms are upper-case
  (e.g. `CORP.EXAMPLE.COM`).
- **A `krb5.conf` is used if present** (`--krb5conf`, else `$KRB5_CONFIG`, else
  `/etc/krb5.conf`). If none is found, carnival falls back to a minimal
  DNS-discovery configuration, which usually works for Active Directory.

You can provide credentials in three ways:

```
# 1. Reuse an existing ticket obtained with kinit (no password needed).
#    Honors $KRB5CCNAME, otherwise /tmp/krb5cc_<uid>; override with --ccache.
$ kinit user@CORP.EXAMPLE.COM
$ carnival -k ls smb://server.corp.example.com/sharename/

# 2. Username + password prompt.
$ carnival -k -u user ls smb://server.corp.example.com/sharename/
Password:

# 3. A keytab file (no password prompt).
$ carnival -k -u user --keytab /path/to/user.keytab ls smb://server.corp.example.com/sharename/
```

For an MIT KDC you will typically need a `/etc/krb5.conf` describing the realm's
KDCs. For Active Directory the DNS-discovery fallback is usually sufficient, and
the realm is the AD domain in upper-case.
