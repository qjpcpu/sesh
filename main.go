package main

import (
	"bufio"
	"fmt"
	"github.com/qjpcpu/sesh/golang.org/x/crypto/ssh/terminal"
	"github.com/qjpcpu/sesh/templ"
	"github.com/qjpcpu/sesh/util"
	"github.com/voxelbrain/goptions"
	"io/ioutil"
	"net"
	"os"
)

type SeshFlags struct {
	Hostfile       string             `goptions:"-f, --host-file, description='HOST_FILE, every host per line'"`
	Hostlist       string             `goptions:"-h, --host-list, description='HOSTS, hosts seperated by comma'"`
	User           string             `goptions:"-u, --user, description='USER, user name'"`
	Password       string             `goptions:"-p, --password, description='PASSWORD'"`
	Port           string             `goptions:"--port, description='PORT'"`
	Keyfile        string             `goptions:"-i, --identity-file, description='ssh auth file'"`
	Cmdfile        string             `goptions:"-c, --command-file, description='CMD_FILE, Command file'"`
	Tmpdir         string             `goptions:"-t, --tmp-directory, description='TMP_DIRECTORY, Specify tmp directory'"`
	Data           []string           `goptions:"-d, --data, description='the name would be replace according -d name=value pair in command or command file. The name format in command should be {{ name }}'"`
	Arguments      string             `goptions:"--exec, description='how to exec script in remote, use {} stands for scriptname'"`
	Parallel       bool               `goptions:"-r, --rapid, description='Parallel execution'"`
	ParallelDegree int                `goptions:"--parallel-degree, description='Parallel degree, default is the size of hosts'"`
	Timeout        int                `goptions:"--timeout, description='ssh connection timeout seconds, default is 5'"`
	Pause          bool               `goptions:"--check, description='Pause after first host done'"`
	Debug          bool               `goptions:"--debug, description='Print the configurations, not perform tasks'"`
	Cmd            goptions.Remainder `goptions:"description='command'"`
	Help           goptions.Help      `goptions:"--help, description='Show this help'"`
	goptions.Verbs
	Sscp struct {
		Src     string `goptions:"-s,--source,obligatory,description='source file'"`
		Destdir string `goptions:"-d, --dest, obligatory,description='destination directory'"`
	} `goptions:"send"`
}

func main() {
	options := SeshFlags{
		Debug:          false,
		Pause:          false,
		Tmpdir:         ".",
		Parallel:       false,
		ParallelDegree: 0,
	}
	goptions.ParseAndFail(&options)

	//timeout
	if options.Timeout < 1 {
		options.Timeout = 5
	}
	// get hosts
	var host_arr []string
	if options.Hostfile != "" {
		if buf, err := ioutil.ReadFile(options.Hostfile); err != nil {
			fmt.Fprintln(os.Stderr, "\033[31mFailed to read host from file!\033[0m")
			return
		} else {
			hoststr := string(buf)
			host_arr = parseHostsFromString(hoststr)
		}
	} else if options.Hostlist != "" {
		host_arr = parseHostsFromString(options.Hostlist)
	} else {
		if terminal.IsTerminal(0) {
			fmt.Fprintln(os.Stderr, "\033[33mPlease input hosts, seperated by LINE SEPERATOR, press Ctrl+D to finish input:\033[0m")
		}
		buf, _ := ioutil.ReadAll(os.Stdin)
		host_arr = parseHostsFromString(string(buf))
	}

	// get user
	if options.User == "" {
		options.User = os.Getenv("USER")
	}
	// get  key file
	if options.Keyfile == "" {
		if options.Keyfile == "" {
			options.Keyfile = os.Getenv("HOME") + "/.ssh/id_rsa"
		}
		if _, err := os.Stat(options.Keyfile); os.IsNotExist(err) {
			if options.Password == "" {
				if os.Getenv("SSH_AUTH_SOCK") == "" {
					fmt.Fprintln(os.Stderr, "\033[31mKey file "+options.Keyfile+" not found!\033[0m")
					return
				}
				if _, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err != nil {
					fmt.Fprintln(os.Stderr, "\033[31mKey file "+options.Keyfile+" not found!\033[0m")
					return
				}
			} else {
				options.Keyfile = ""
			}
		}
	}

	// Scp file
	if options.Sscp.Src != "" && options.Sscp.Destdir != "" {
		taskArgs := util.TaskArgs{
			User:     options.User,
			Password: options.Password,
			Port:     options.Port,
			Keyfile:  options.Keyfile,
			Source:   options.Sscp.Src,
			Destdir:  options.Sscp.Destdir,
			Timeout:  options.Timeout,
			Parallel: options.Parallel,
		}
		if err := util.ScpRun(taskArgs, host_arr); err != nil {
			fmt.Fprintf(os.Stderr, "\033[31mCopy faild! %v\033[0m\n", err)
		} else {
			fmt.Fprintln(os.Stderr, "\033[32mFinished!\033[0m")
		}
		return
	}

	//check command
	if len(options.Cmd) == 0 && len(options.Cmdfile) == 0 {
		fmt.Fprintln(os.Stderr, "\033[31mPlese specify command you want execute.\033[0m")
		return
	}
	if options.Cmdfile == string(util.AuthCmdFile) {
		if c, err := util.AuthCmdFile.Get(); err == nil {
			options.Cmd = []string{c}
			options.Cmdfile = ""
		}
	}
	// parse command template
	cmd := ""
	if options.Cmdfile != "" {
		if _, err := os.Stat(options.Cmdfile); os.IsNotExist(err) {
			fmt.Fprintln(os.Stderr, "\033[31mCommand file "+options.Cmdfile+" not found!\033[0m")
			return
		}
		if o, err := templ.ParseFromFiles(options.Cmdfile, parseData(options.Data)); err != nil {
			fmt.Fprintf(os.Stderr, "\033[31mParse command file failed!\033[0m\n%v\n", err)
			return
		} else {
			cmd = o
		}
	} else {
		// join commands
		for _, v := range options.Cmd {
			cmd = cmd + v + " "
		}
		if o, err := templ.ParseFromString(cmd, parseData(options.Data)); err != nil {
			fmt.Fprintf(os.Stderr, "\033[31mParse command failed!\033[0m\n%v\n", err)
			return
		} else {
			cmd = o
		}
	}
	if _, err := os.Stat(options.Tmpdir); os.IsNotExist(err) && options.Parallel {
		fmt.Fprintln(os.Stderr, "\033[31mTemporary directory "+options.Tmpdir+" is not exist!\033[0m")
		return
	}

	// Begin to run
	config := util.TaskArgs{
		User:      options.User,
		Password:  options.Password,
		Keyfile:   options.Keyfile,
		Port:      options.Port,
		Cmd:       cmd,
		CmdArgs:   options.Arguments,
		Output:    os.Stdout,
		ErrOutput: os.Stderr,
		Timeout:   options.Timeout,
	}
	if options.Debug {
		printDebugInfo(options, host_arr, cmd)
		return
	}
	host_offset := 0
	if options.Pause {
		util.SerialRun(config, host_arr, host_offset, 1)
		fmt.Fprintf(os.Stderr, "The task on \033[33m%s\033[0m has done.\nPress any key to auto login \033[33m%s\033[0m to have a check...", host_arr[0], host_arr[0])
		reader := bufio.NewReader(os.Stdin)
		reader.ReadString('\n')
		util.Interact(config, host_arr[0])
		fmt.Fprintf(os.Stderr, "\n\033[32mCheck completed! Press any key to acomplish the left tasks.\033[0m")
		reader = bufio.NewReader(os.Stdin)
		reader.ReadString('\n')
		host_offset = 1
	}
	if options.Parallel {
		fmt.Fprintln(os.Stderr, util.GirlSay("  Please wait me for a moment, Baby!  "))
		end := len(host_arr)
		if options.ParallelDegree < 1 || options.ParallelDegree > (end-host_offset) {
			options.ParallelDegree = end - host_offset
		}
		for {
			to := host_offset + options.ParallelDegree
			if to > end {
				to = end
			}
			if host_offset >= to {
				break
			}
			util.ParallelRun(config, host_arr, host_offset, to, options.Tmpdir)
			host_offset += options.ParallelDegree
		}
	} else {
		util.SerialRun(config, host_arr, host_offset, len(host_arr))
	}

	fmt.Fprintln(os.Stderr, "\033[32mFinished!\033[0m")
}
