# carnival

A tool for testing connectivity to SMB shares. With it you can:

### Calculate the MD5 hash of a file
```
carnival md5 smb://user:password@address/sharename/path/to/file.txt
```

### Copy a file from an SMB share to a local destination
```
carnival cp smb://user:password@address/sharename/file.bin .
# OR rename it at the destination
carnival cp smb://user:password@address/sharename/file.bin file.binary
```
The `cp` command will tell you how long the transfer took and the average transfer speed.

### List the files in a directory
```
# List the files at the root of the share
carnival ls smb://user:password@address/sharename/

# List the fiels in the 'Games' directory of the share
carnival ls smb://user:password@address/sharename/Games
```

### Print the names of publicly visible SMB shares
```
carnival shares smb://user:password@address
```

If a username and password are not included in the SMB url, carnival will authenticate as `guest` with an empty password.
