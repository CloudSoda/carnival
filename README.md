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

If the `-u/--username` flag is set, you will be prompted for the password.

If no username is provided, carnival will attempt to authenticate anonymously.
