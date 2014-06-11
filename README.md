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

Sesh also can get hosts from Stdin:
    
    echo host1,host2 | sesh -u user -p password 'echo hello'
    cat host-file | sesh -u user -p password 'echo hello'

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

Sesh would execute job for each host serially by default, swith `-r` on for parallel execution:

	sesh -f host-file -r 'echo hello'
	
Sesh would execute all task by parallel,if you want control the parallel degree, you could use `--parallel-degree`.

Forexample, if you want execute every 2 hosts by paraallel:

    sesh -f host-file -r --parallel-degree 2 'echo hello'

#### Save output

Sesh would print remote output to screen by default, but you can save output to file
	
	sesh -f host-file -o result 'echo hello'
	
#### Check for sure

If you want have a check after the first host's job done, you can use `--check`, when  first job done, you would auto logon the first host, if everything is fine, press `Ctrl+D`to return and continue.

	sesh -f host-file --check 'touch new-file'
	
#### Execute script

If you want execute many commands on remote host, you would find it's hard to use `ssh` command to accomplish that, so you can put these comands into a file, for example `keygen.cmd`:

	if [ ! -e "~/.ssh/id_rsa" ];then
        ssh-keygen -t rsa -N "" -f ~/.ssh/id_rsa
	fi

Use `-c` to specity command file:

	sesh -f host-list -c keygen.cmd
	
#### Command template

You can embedded parameter in command or command file with `{{ .name }}`, then invoke sesh with `-d`, for example, there is a command file `enter-today-dir.cmd`:

	cd ~/{{ .date }}/logs && pwd

then, we can use sesh like this:

	sesh -f hosts -d date=$(date +%Y%m%d) -c enter-today-dir.cmd
	
You can also use argument parse for inline commands:

	sesh -f hosts -d who=jason 'echo {{ .who }} is sexy'
	
Or you can invoke script(the first line must start with `#!`) with arguments in normal way:

This is a ruby script `x.rb`:

    #!/usr/bin/ruby
    puts ARGV
    
Now the sesh would be:
	sesh -f hosts -c x.rb --args "hello"

#### Embedded command template

And sesh also support embedded template, for example, there is two command template files:

File who.cmd

	{{define "who"}}
	whoami
	{{end}}
	
File main.cmd

	name=$({{ template "who" }})
	echo "Now ${name} is in $(pwd)"

Then we can use sesh like this:

	sesh -f hosts -c main.cmd -c who.cmd
	# The output is:
	# Now jason is in /home/jason
	
The main template `main.cmd` invoke the embedded template `who.cmd`. Use `{{define "XXX"}} ....{{end}}` to define template `XXX`, and then use `{{template "XXX"}}` to invoke template. By default, the `-d` parameters can't be seen in subtemplate, if you want deliver parameters into subtemplate, you should use:

	{{ template "XXX" . }}
	

#### Help

	sesh -help
