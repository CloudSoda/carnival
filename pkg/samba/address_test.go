package samba

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewURL(t *testing.T) {
	t.Parallel()

	testData := []struct {
		inputURL string
		creds    *Credentials
		expected URL
	}{
		{
			inputURL: "smb://address/share/path",
			expected: URL{
				Address: "address:445",
				Share:   "share",
				Path:    "path",
			},
		},
		{
			inputURL: "smb://address/share/path",
			creds: &Credentials{
				Username: "foo",
				Password: "bar",
			},
			expected: URL{
				Address: "address:445",
				Share:   "share",
				Path:    "path",
				Credentials: &Credentials{
					Username: "foo",
					Password: "bar",
				},
			},
		},
		{
			inputURL: "smb://host.tld:3622/share",
			expected: URL{
				Address: "host.tld:3622",
				Share:   "share",
			},
		},
		{
			inputURL: "smb://user:foo@address.com/myshare/path/to/file.txt",
			expected: URL{
				Address: "address.com:445",
				Share:   "myshare",
				Path:    "path/to/file.txt",
				Credentials: &Credentials{
					Username: "user",
					Password: "foo",
				},
			},
		},
	}

	for i, td := range testData {
		td := td
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Parallel()

			actual, err := newURL(td.inputURL, td.creds)
			require.NoError(t, err)
			require.Equal(t, td.expected.Address, actual.Address)
			require.Equal(t, td.expected.Share, actual.Share)
			require.Equal(t, td.expected.Path, actual.Path)
			if td.expected.Credentials != nil {
				require.NotNil(t, actual.Credentials)
				require.Equal(t, td.expected.Credentials.Username, actual.Credentials.Username)
				require.Equal(t, td.expected.Credentials.Password, actual.Credentials.Password)
			}
		})
	}
}
