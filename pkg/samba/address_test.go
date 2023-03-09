package samba

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewURL(t *testing.T) {
	t.Parallel()

	testData := []struct {
		input    string
		expected URL
	}{
		{
			input: "smb://address/share/path",
			expected: URL{
				Address: "address:445",
				Share:   "share",
				Path:    "path",
			},
		},
		{
			input: "smb://host.tld:3622/share",
			expected: URL{
				Address: "host.tld:3622",
				Share:   "share",
			},
		},
		{
			input: "smb://user:foo@address.com/myshare/path/to/file.txt",
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

			actual, err := newURL(td.input)
			require.NoError(t, err)
			require.Equal(t, td.expected, actual)
		})
	}
}
