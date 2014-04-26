sexy-ssh
========

Sexy ssh makes batch ssh tasks sexy!

## Install

	git clone git@github.com:qjpcpu/sexy-ssh.git
	cd sexy-ssh
	export GOPATH=`pwd`
	go install main
	mv bin/{main,sesh}
	
The executable file `bin/sesh` is all you need.

## Usage

#### hosts

Use with host file:

	sesh -f host-file -u user -p password 'echo hello'
	
The host-file's format is simple, one host per line, like:

	www.host1.com
	www.host2.com

If the hosts can be counted, you can use `-h` to specify remote hosts, seperated by `,`:

	sesh -h www.host1.com,www.host2.com -u user -p password 'echo hello'

#### Authorization
Use `-p` to specify password, but it's better to use rsa authorization with `-k`:

	sesh -f host-file -u user -k ~/.ssh/id_rsa 'echo hello'
	# Or, just use nothing for authorization, sesh use ~/.ssh/id_rsa as default
	sesh -f host-file -u user 'echo hello'

And, if the user of remote host is same as current user, we can just drop `-u` flag:

	sesh -f host-file 'echo hello'

#### Configuration file

You can put commonly used user and rsa file in `~/.seshrc`, which is a json file:

	{ "User":"jason","Keyfile":"/path/to/rsa"}
	
Sesh would use this file as preference, so you can input less:

	sesh -f host-file 'echo hello'
	
#### Parallel

Sesh would execute job for each host serially by default, swith `-parallel` on for parallel execution:

	sesh -f host-file -parallel 'echo hello'
	
#### Save output

Sesh would print remote output to screen by default, but you can save output to file
	
	sesh -f host-file -o result 'echo hello'
	
#### Check for sure

If you want have a check after the first host's job done, you can use `-check`, when  first job done, you would auto logon the first host, if everything is fine, press `Ctrl+\`to return and continue.

	sesh -f host-file -check 'touch new-file'
	
#### Execute script

If you want execute many commands on remote host, you would find it's hard to use `ssh` command to accomplish that, so you can put these comands into a file, for example:

```bash get-user-process.cmd
user=`whoami`
pstree $user|grep -vE '^ |^$'|awk -F "---" '{print $1}'
```
Use `-c` to specity command file:

	sesh -f host-list -c get-user-process.cmd
	
#### Help

	sesh -help