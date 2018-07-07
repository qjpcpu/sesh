package util

import (
	"fmt"
	"github.com/qjpcpu/sesh/batch"
	"github.com/qjpcpu/sesh/cowsay"
	"github.com/qjpcpu/sesh/dircat"
	"github.com/qjpcpu/sesh/golang.org/x/crypto/ssh/terminal"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"time"
)

type TaskArgs struct {
	User     string
	Password string
	Keyfile  string
	Port     string
	Cmd      string
	CmdArgs  string
	Timeout  int
	// parallel run
	Output    io.Writer
	ErrOutput io.Writer
	// scp
	Parallel bool
	Source   string
	Destdir  string
}

func GirlSay(content ...interface{}) string {
	return cowsay.Format(fmt.Sprint(content))
}

// Hook for per task state changed
func report(output io.Writer, prefix, host string, color bool) {
	if color {
		output.Write([]byte(fmt.Sprintf("\033[33m%s========== %s ==========\033[0m\n", prefix, host)))
	} else {
		output.Write([]byte(fmt.Sprintf("%s========== %s ==========\n", prefix, host)))
	}
}

func SerialRun(config TaskArgs, raw_host_arr []string, start, end int) error {
	host_arr := raw_host_arr[start:end]
	user := config.User
	pwd := config.Password
	keyfile := config.Keyfile
	cmd := config.Cmd
	args := config.CmdArgs
	timeout := config.Timeout
	port := config.Port
	// Format command
	cmd = format_cmd(cmd, args)
	printer := config.Output
	err_printer := config.ErrOutput

	BuildKnownHosts(host_arr)

	wg := new(sync.WaitGroup)

	for index, h := range host_arr {
		task := batch.NewTask(h, user, pwd, keyfile, cmd, port, printer, err_printer, wg)
		task.Timeout = timeout
		report(err_printer, fmt.Sprintf("%d/%d ", index+1+start, len(raw_host_arr)), task.Host, true)
		task.Work()
	}
	wg.Wait()
	return nil
}

func ParallelRun(config TaskArgs, raw_host_arr []string, start, end int, tmpdir string) error {
	host_arr := raw_host_arr[start:end]
	user := config.User
	pwd := config.Password
	keyfile := config.Keyfile
	cmd := config.Cmd
	args := config.CmdArgs
	timeout := config.Timeout
	port := config.Port
	cmd = format_cmd(cmd, args)
	printer := config.Output
	err_printer := config.ErrOutput

	// Create master, the master is used to manage go routines
	wg := new(sync.WaitGroup)
	// Setup tmp directory for tmp files
	dir := fmt.Sprintf("%s/.task.%d", tmpdir, time.Now().Nanosecond())
	if err := os.Mkdir(dir, os.ModeDir|os.ModePerm); err != nil {
		return err
	}

	// Listen interrupt and kill signal, clear tmp files before exit.
	intqueue := make(chan os.Signal, 1)
	signal.Notify(intqueue, os.Interrupt, os.Kill)
	// If got interrupt or kill signal, delete tmp directory first, then exit with 1
	go func() {
		<-intqueue
		os.RemoveAll(dir)
		os.Exit(1)
	}()
	// If the complete all the tasks normlly, stop listenning signals and remove tmp directory
	defer func() {
		signal.Stop(intqueue)
		os.RemoveAll(dir)
	}()

	BuildKnownHosts(host_arr)

	// Create tmp file for every host, then executes.
	var tmpfiles []*os.File
	for _, h := range host_arr {
		file, _ := os.Create(fmt.Sprintf("%s/%s", dir, h))
		err_file, _ := os.Create(fmt.Sprintf("%s/%s.err", dir, h))
		tmpfiles = append(tmpfiles, file, err_file)
		task := batch.NewTask(h, user, pwd, keyfile, cmd, port, file, err_file, wg)
		task.Timeout = timeout
		go task.Work()
	}

	// show realtime view for each host
	var dc *dircat.DirCat
	if terminal.IsTerminal(1) {
		wlist := []string{}
		for _, h := range host_arr {
			wlist = append(wlist, fmt.Sprintf("%s/%s", dir, h))
		}
		dc, _ = dircat.Init(wlist...)
		go dc.Start()
	}
	// When a host is ready and request for continue, the master would echo CONTINUE for response to allow host to run
	wg.Wait()

	if terminal.IsTerminal(1) {
		dc.Stop()
	}
	// close tmp files
	for _, f := range tmpfiles {
		f.Close()
	}
	// Merge all the hosts' output to the output file
	for _, h := range host_arr {
		report(os.Stderr, "", h, true)
		// copy err output first
		err_fn := fmt.Sprintf("%s/%s.err", dir, h)
		err_src, _ := os.Open(err_fn)
		io.Copy(err_printer, err_src)
		err_src.Close()
		// copy output then
		fn := fmt.Sprintf("%s/%s", dir, h)
		src, _ := os.Open(fn)
		io.Copy(printer, src)
		src.Close()
		// remove tmp file
		os.Remove(err_fn)
		os.Remove(fn)
	}
	return nil
}
func Interact(config TaskArgs, host string) {
	wg := new(sync.WaitGroup)
	task := batch.NewTask(
		host,
		config.User,
		config.Password,
		config.Keyfile,
		config.Cmd,
		config.Port,
		config.Output,
		config.ErrOutput,
		wg,
	)
	task.SysLogin()
}

func ScpRun(config TaskArgs, host_arr []string) error {
	user := config.User
	parallel := config.Parallel
	src := config.Source
	dest := config.Destdir
	if !strings.HasSuffix(dest, "/") {
		dest += "/"
	}
	wg := new(sync.WaitGroup)
	BuildKnownHosts(host_arr)
	for _, h := range host_arr {
		wg.Add(1)
		taskF := func(host string) {
			defer wg.Done()
			cmdstr := fmt.Sprintf(`rsync -azh %s %s@%s:%s`, src, user, host, dest)
			cmd := exec.Command("/bin/bash", "-c", cmdstr)
			fmt.Fprintf(os.Stderr, "Start sync to  %s......\n", host)
			if err := cmd.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "Sync to %s:%s fail:%v\n", host, dest, err)
			} else {
				fmt.Fprintf(os.Stderr, "Sync to %s:%s OK\n", host, dest)
			}
		}
		if parallel {
			go taskF(h)
		} else {
			taskF(h)
		}
	}

	wg.Wait()
	return nil
}

func BuildKnownHosts(host_arr []string) {
	filename := os.Getenv("HOME") + "/.ssh/known_hosts"
	dirname := os.Getenv("HOME") + "/.ssh"
	content_bytes, _ := ioutil.ReadFile(filename)
	content := string(content_bytes)
	var not_exists []string
	for _, host := range host_arr {
		if !strings.Contains(content, host) {
			not_exists = append(not_exists, host)
		}
	}
	if len(not_exists) == 0 {
		return
	}
	cmdstr := fmt.Sprintf(`
[ ! -e %s ] && mkdir -p %s && chmod 700 %s
[ ! -e %s ] && touch %s && chmod 644 %s
ssh-keyscan %s >> %s
`, dirname, dirname, dirname,
		filename, filename, filename,
		strings.Join(not_exists, " "), filename)
	cmd := exec.Command("/bin/sh", "-c", cmdstr)
	cmd.Run()
}
