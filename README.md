sexy-ssh
========

Sexy ssh makes batch ssh tasks sexy!

## Install

### Build from source

	git clone git@github.com:qjpcpu/sexy-ssh.git
	cd sexy-ssh
	export GOPATH=`pwd`
	go install sesh
	
The executable file `bin/sesh` is all you need.

### Download binary file

    go get -u github.com/qjpcpu/sesh

## Usage

#### hosts

Use with host file:

	sesh -f host-file -u user -p password 'echo hello'
	
The host-file's format is simple, one host per line, like:

	www.example-host00.com
	www.example-host01.com

Output is:

![output](https://raw.githubusercontent.com/qjpcpu/sexy-ssh/master/screen_shoot/serial_exec.png)

If the hosts can be counted, you can use `-h` to specify remote hosts, seperated by `,`:

	sesh -h www.host1.com,www.host2.com -u user -p password 'echo hello'

Sesh also can get hosts from Stdin:
    
    echo host1,host2 | sesh -u user -p password 'echo hello'
    cat host-file | sesh -u user -p password 'echo hello'

#### Authorization
Use `-p` to specify password, but it's better to use rsa authorization with `-i`:

	sesh -f host-file -u user -i ~/.ssh/id_rsa 'echo hello'
	# Or, just use nothing for authorization, sesh use ~/.ssh/id_rsa as default
	sesh -f host-file -u user 'echo hello'

And, if the user of remote host is same as current user, we can just drop `-u` flag:

	sesh -f host-file 'echo hello'

> build key auth quickly: sesh -f host-file -u user -c @auth.cmd

#### Timeout

The default connection timeout is 5 seconds, it can be changed by `--timeout`:

    sesh -f host-file  --timeout 1 'echo hello'

#### Parallel

Sesh would execute job for each host serially by default, swith `-r` on for parallel execution:

	sesh -f host-file -r 'echo hello'
	
Sesh would execute all task by parallel,if you want control the parallel degree, you could use `--parallel-degree`.

Output is:

![output](https://raw.githubusercontent.com/qjpcpu/sexy-ssh/master/screen_shoot/realtime_output.png)

Forexample, if you want execute every 2 hosts by paraallel:

    sesh -f host-file -r --parallel-degree 2 'echo hello'

	
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
	
If the script needs arguments, the `--exec` might help.
Note: the first symbol `{}`	represents the script file name.

    sesh -f host-list  -c myscript.cmd  --exec 'sh {} hello'

the content of myscript.cmd is `echo "say $1"`, so the output would be `say hello`.

If the script starts with shibang `#!`, forexample, a ruby file `test.rb` is:

    #!/usr/bin/ruby
    puts "say "+ARGV.to_s

You can run sesh like this:

    sesh -f host-list -c test.rb --exec '{} hello'

#### Command template

You can embedded parameter in command or command file with `{{ name }}`, then invoke sesh with `-d`, for example, there is a command file `enter-today-dir.cmd`:

	cd ~/{{ date }}/logs && pwd

then, we can use sesh like this:

	sesh -f hosts -d date=$(date +%Y%m%d) -c enter-today-dir.cmd
	
You can also use argument parse for inline commands:

	sesh -f hosts -d who=jason 'echo {{ who }} is sexy'
	
Or you can invoke script(the first line must start with `#!`) with arguments in normal way:

This is a ruby script `x.rb`:

    #!/usr/bin/ruby
    puts ARGV
    
Now the sesh would be:

	sesh -f hosts -c x.rb --exec "{} hello"


#### Remote cp(scp)

Sesh also support scp

    sesh -f hosts send -s srcfile -d /remote/directory


#### Help

	sesh --help
