package util

import (
	"fmt"
	"github.com/qjpcpu/sesh/batch"
	"github.com/qjpcpu/sesh/cowsay"
	"github.com/qjpcpu/sesh/dircat"
	cfg "github.com/qjpcpu/sesh/goconf.googlecode.com/hg"
	"github.com/qjpcpu/sesh/golang.org/x/crypto/ssh/terminal"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"time"
)

func Gettaskrc() (conf map[string]map[string]string, err error) {
	conf = make(map[string]map[string]string)
	fn := os.Getenv("HOME") + "/.seshrc"
	if _, err = os.Stat(fn); os.IsNotExist(err) {
		return conf, err
	}
	if c, err := cfg.ReadConfigFile(fn); err != nil {
		return conf, err
	} else {
		sections := c.GetSections()
		for _, sec := range sections {
			conf[sec] = make(map[string]string)
			if user, err := c.GetString(sec, "user"); err == nil {
				conf[sec]["user"] = user
			}
			if keyfile, err := c.GetString(sec, "keyfile"); err == nil {
				conf[sec]["keyfile"] = keyfile
			}
			if password, err := c.GetString(sec, "password"); err == nil {
				conf[sec]["password"] = password
			}
		}
		return conf, err
	}
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

func SerialRun(config map[string]interface{}, raw_host_arr []string, start, end int) error {
	host_arr := raw_host_arr[start:end]
	user, _ := config["User"].(string)
	pwd, _ := config["Password"].(string)
	keyfile, _ := config["Keyfile"].(string)
	cmd, _ := config["Cmd"].(string)
	args, _ := config["Args"].(string)
	timeout, _ := config["Timeout"].(int)
	// Format command
	cmd = format_cmd(cmd, args)
	printer, _ := config["Output"].(io.Writer)
	err_printer, _ := config["Errout"].(io.Writer)

	wg := new(sync.WaitGroup)

	for index, h := range host_arr {
		task := batch.NewTask(h, user, pwd, keyfile, cmd, printer, err_printer, wg)
		task.Timeout = timeout
		report(err_printer, fmt.Sprintf("%d/%d ", index+1+start, len(raw_host_arr)), task.Host, true)
		task.Work()
	}
	wg.Wait()
	return nil
}

func ParallelRun(config map[string]interface{}, raw_host_arr []string, start, end int, tmpdir string) error {
	host_arr := raw_host_arr[start:end]
	user, _ := config["User"].(string)
	pwd, _ := config["Password"].(string)
	keyfile, _ := config["Keyfile"].(string)
	cmd, _ := config["Cmd"].(string)
	args, _ := config["Args"].(string)
	timeout, _ := config["Timeout"].(int)
	cmd = format_cmd(cmd, args)
	printer, _ := config["Output"].(io.Writer)
	err_printer, _ := config["Errout"].(io.Writer)

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

	// Create tmp file for every host, then executes.
	var tmpfiles []*os.File
	for _, h := range host_arr {
		file, _ := os.Create(fmt.Sprintf("%s/%s", dir, h))
		err_file, _ := os.Create(fmt.Sprintf("%s/%s.err", dir, h))
		tmpfiles = append(tmpfiles, file, err_file)
		task := batch.NewTask(h, user, pwd, keyfile, cmd, file, err_file, wg)
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
func Interact(config map[string]interface{}, host string) {
	user, _ := config["User"].(string)
	pwd, _ := config["Password"].(string)
	keyfile, _ := config["Keyfile"].(string)
	cmd, _ := config["Cmd"].(string)
	printer, _ := config["Output"].(io.Writer)
	err_printer, _ := config["Errout"].(io.Writer)

	wg := new(sync.WaitGroup)
	task := batch.NewTask(host, user, pwd, keyfile, cmd, printer, err_printer, wg)
	task.SysLogin()
}

func ScpRun(config map[string]interface{}, host_arr []string) error {
	user, _ := config["User"].(string)
	//keyfile, _ := config["Keyfile"].(string)
	parallel, ok := config["Parallel"].(bool)
	src, _ := config["Source"].(string)
	dest, _ := config["Destdir"].(string)
	if !strings.HasSuffix(dest, "/") {
		dest += "/"
	}
	wg := new(sync.WaitGroup)
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
		if ok && parallel {
			go taskF(h)
		} else {
			taskF(h)
		}
	}

	wg.Wait()
	return nil
}
