// Command mcpspy records MCP interactions.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/tmc/mcp/internal/mcpspy"
	"github.com/tmc/mcp/internal/mcpspy/web"
	"golang.org/x/term"
)

var (
	outFile         = flag.String("f", "", "output recording file")
	verbose         = flag.Bool("v", false, "verbose mode: print interactions to stderr")
	veryVerbose     = flag.Bool("vv", false, "very verbose mode: print raw stdin/stdout/stderr to stderr")
	useAppend       = flag.Bool("a", false, "append to existing file instead of overwriting")
	noCopyStderr    = flag.Bool("no-stderr", false, "do not copy stderr from command")
	prettyJSON      = flag.Bool("pretty", false, "pretty-print JSON output")
	pipeMode        = flag.Bool("pipe", false, "explicitly enable pipe mode (for testing)")
	indentLevel     = flag.Int("indent", 0, "indentation level for output (useful when piping multiple mcpspy commands)")
	indentChar      = flag.String("indent-char", "  ", "character(s) to use for indentation")
	parentPID       = flag.Int("parent-pid", 0, "parent mcpspy process ID (set automatically in pipelines)")
	passThrough     = flag.Bool("pass-through", false, "pass JSON through unmodified (no mcp- prefix/suffix)")
	autoIndent      = flag.Bool("auto-indent", false, "automatically determine indentation level based on pipeline depth")
	forceUnbuffered = flag.Bool("unbuffered", false, "force unbuffered output (useful for real-time piping)")
	quiet           = flag.Bool("q", false, "quiet mode: suppress log messages")
	listenUI        = flag.Bool("l", false, "enable the embedded web UI")
	httpAddr        = flag.String("http", "127.0.0.1:0", "HTTP bind address for the UI")
	openBrowser     = flag.Bool("open", false, "open the UI in the browser")
	nameFlag        = flag.String("name", "", "optional human-readable instance name")
	sessionFlag     = flag.String("session", "", "session identifier shared across related mcpspy processes")
	specFile        = flag.String("spec-file", "", "output .mcpspec file (defaults to a sidecar next to the recording)")
)

func main() {
	doubleDashIndex := -1
	for i, arg := range os.Args {
		if arg == "--" {
			doubleDashIndex = i
			break
		}
	}

	log.SetPrefix("mcpspy: ")
	log.SetFlags(0)

	flagArgs := os.Args[1:]
	var cmdArgs []string
	if doubleDashIndex > 0 {
		flagArgs = os.Args[1:doubleDashIndex]
		cmdArgs = os.Args[doubleDashIndex+1:]
		original := os.Args
		os.Args = append([]string{os.Args[0]}, flagArgs...)
		defer func() { os.Args = original }()
	}

	flag.Parse()

	if *quiet {
		log.SetOutput(io.Discard)
	}
	if *openBrowser && !*listenUI {
		log.Fatal("-open requires -l")
	}
	if *sessionFlag == "" {
		*sessionFlag = mcpspy.NewSessionID()
	}

	var command []string
	if len(cmdArgs) > 0 {
		command = cmdArgs
	} else if len(flag.Args()) > 0 {
		command = flag.Args()
	}
	if *outFile == "" {
		*outFile = generateFilename(command)
	}
	if *specFile == "" {
		*specFile = mcpspy.SpecFilenameFor(*outFile)
	}

	writer, err := newSafeWriter(*outFile, *useAppend)
	if err != nil {
		log.Fatal(err)
	}
	defer writer.Close()

	if *verbose || *veryVerbose {
		mode := "overwriting"
		if *useAppend {
			mode = "appending to"
		}
		fmt.Fprintf(os.Stderr, "mcpspy: %s %s\n", mode, *outFile)
	}

	recorder := mcpspy.New(writer, mcpspy.Options{
		PrettyJSON:  *prettyJSON,
		PassThrough: *passThrough,
		BufferSize:  4096,
		Name:        *nameFlag,
		SessionID:   *sessionFlag,
	})
	stopVerbose := startVerboseOutput(recorder)
	defer stopVerbose()
	specTracker := mcpspy.NewSpecTracker(recorder, mcpspy.SpecOptions{
		Path: *specFile,
		Name: *nameFlag,
	})
	defer specTracker.Close()

	pipeDepth := 0
	inPipeline := false
	if *autoIndent {
		pipeDepth, inPipeline = detectPipeline()
		if inPipeline {
			*indentLevel = pipeDepth
		}
	}
	if len(command) > 0 && *autoIndent {
		command = modifyArgsForChild(command, *sessionFlag)
	}

	cwd, _ := os.Getwd()
	runtimeRegistry, err := mcpspy.NewRuntime(recorder, mcpspy.RuntimeOptions{
		Name:            *nameFlag,
		SessionID:       *sessionFlag,
		ParentMCPSpyPID: *parentPID,
		Command:         command,
		OutputFile:      *outFile,
		CWD:             cwd,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer runtimeRegistry.Close()

	uiServer := web.New(recorder, specTracker, runtimeRegistry, web.Options{
		Addr:       *httpAddr,
		OutputFile: *outFile,
	})
	if err := runtimeRegistry.Start(uiServer.Start); err != nil {
		log.Fatal(err)
	}

	if *listenUI {
		url, err := uiServer.Start()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Fprintf(os.Stderr, "mcpspy: ui %s\n", url)
		if *openBrowser {
			if err := web.OpenBrowser(url); err != nil {
				log.Printf("open browser: %v", err)
			}
		}
	}
	defer uiServer.Close()

	startUI := func() {
		url, err := uiServer.Start()
		if err != nil {
			log.Printf("start ui: %v", err)
			return
		}
		fmt.Fprintf(os.Stderr, "mcpspy: ui %s\n", url)
	}

	if len(command) == 0 || *pipeMode {
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, syscall.SIGUSR1)
		defer signal.Stop(signalChan)
		go func() {
			for sig := range signalChan {
				if sig == syscall.SIGUSR1 {
					startUI()
				}
			}
		}()
		stdout := io.Writer(os.Stdout)
		if *passThrough && *indentLevel > 0 {
			stdout = &indentWriter{
				dst:       stdout,
				indent:    strings.Repeat(*indentChar, *indentLevel),
				forceSync: *forceUnbuffered,
			}
		}
		if err := handlePipeMode(recorder, stdout); err != nil {
			log.Fatal(err)
		}
		return
	}

	cmd := exec.Command(command[0], command[1:]...)
	cmdStdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}
	cmdStdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	if *veryVerbose {
		cmdStderr, err := cmd.StderrPipe()
		if err != nil {
			log.Fatal(err)
		}
		go streamStderr(cmdStderr)
	} else if !*noCopyStderr {
		cmd.Stderr = os.Stderr
	}
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	signalChan := make(chan os.Signal, 4)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGUSR1)
	defer signal.Stop(signalChan)
	go func() {
		for sig := range signalChan {
			if sig == syscall.SIGUSR1 {
				startUI()
				continue
			}
			if cmd.Process != nil {
				_ = cmd.Process.Signal(sig)
			}
		}
	}()

	stdout := io.Writer(os.Stdout)
	if *passThrough && *indentLevel > 0 {
		stdout = &indentWriter{
			dst:       stdout,
			indent:    strings.Repeat(*indentChar, *indentLevel),
			forceSync: *forceUnbuffered,
		}
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(cmdStdin, recorder.Reader("recv", os.Stdin))
		_ = cmdStdin.Close()
	}()
	go func() {
		defer wg.Done()
		_, _ = io.Copy(recorder.Writer("send", stdout), cmdStdout)
	}()
	wg.Wait()

	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		log.Fatal(err)
	}
}

// safeWriter provides a thread-safe writer that reopens the file if it disappears.
type safeWriter struct {
	mu       sync.Mutex
	filename string
	file     *os.File
	buf      *bufio.Writer
}

func newSafeWriter(filename string, appendMode bool) (*safeWriter, error) {
	flags := os.O_CREATE | os.O_WRONLY
	if appendMode {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}
	f, err := os.OpenFile(filename, flags, 0644)
	if err != nil {
		return nil, err
	}
	return &safeWriter{
		filename: filename,
		file:     f,
		buf:      bufio.NewWriter(f),
	}, nil
}

func (w *safeWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if _, err := os.Stat(w.filename); os.IsNotExist(err) {
		if err := w.reopen(); err != nil {
			return 0, err
		}
	}
	n, err := w.buf.Write(p)
	if err != nil {
		return n, err
	}
	if err := w.buf.Flush(); err != nil {
		return n, err
	}
	return n, nil
}

func (w *safeWriter) reopen() error {
	if w.file != nil {
		_ = w.buf.Flush()
		_ = w.file.Close()
	}
	f, err := os.OpenFile(w.filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	w.file = f
	w.buf = bufio.NewWriter(f)
	return nil
}

func (w *safeWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	_ = w.buf.Flush()
	err := w.file.Close()
	w.file = nil
	w.buf = nil
	return err
}

type indentWriter struct {
	dst       io.Writer
	indent    string
	forceSync bool
}

func (w *indentWriter) Write(p []byte) (int, error) {
	text := string(p)
	if text == "" {
		return w.dst.Write(p)
	}
	lines := strings.SplitAfter(text, "\n")
	var b strings.Builder
	for _, line := range lines {
		if line == "" {
			continue
		}
		if strings.HasSuffix(line, "\n") {
			b.WriteString(w.indent)
			b.WriteString(strings.TrimSuffix(line, "\n"))
			b.WriteByte('\n')
		} else {
			b.WriteString(w.indent)
			b.WriteString(line)
		}
	}
	n, err := w.dst.Write([]byte(b.String()))
	if w.forceSync {
		if f, ok := w.dst.(*os.File); ok {
			_ = f.Sync()
		}
	}
	if n > len(p) {
		n = len(p)
	}
	return n, err
}

func startVerboseOutput(recorder *mcpspy.Recorder) func() {
	if !*verbose && !*veryVerbose {
		return func() {}
	}
	ch, cancel := recorder.Subscribe()
	done := make(chan struct{})
	go func() {
		defer close(done)
		for ev := range ch {
			line := string(ev.Formatted)
			switch ev.Direction {
			case "recv":
				if *verbose {
					prefix := strings.Repeat(*indentChar, *indentLevel)
					fmt.Fprintf(os.Stderr, "\033[32m%s%s\033[0m\n", prefix, line)
				}
				if *veryVerbose {
					indent := strings.Repeat(*indentChar, *indentLevel)
					fmt.Fprintf(os.Stderr, "%s#> %s\n", indent, ev.Raw)
				}
			case "send":
				if *verbose {
					fmt.Fprintf(os.Stderr, "\033[34m%s\033[0m\n", line)
				}
				if *veryVerbose {
					indent := strings.Repeat(*indentChar, *indentLevel)
					fmt.Fprintf(os.Stderr, "%s#< %s\n", indent, ev.Raw)
				}
			}
		}
	}()
	return func() {
		cancel()
		<-done
	}
}

func streamStderr(r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		fmt.Fprintf(os.Stderr, "#! %s\n", scanner.Text())
	}
}

func generateFilename(args []string) string {
	if len(args) == 0 {
		return "mcpspy.0.mcp"
	}
	parts := []string{filepath.Base(args[0])}
	for i := 1; i < len(args) && i <= 3; i++ {
		arg := filepath.Base(args[i])
		arg = strings.ReplaceAll(arg, "/", "-")
		arg = strings.ReplaceAll(arg, "\\", "-")
		arg = strings.ReplaceAll(arg, ":", "-")
		parts = append(parts, arg)
	}
	base := strings.Join(parts, "-")
	for i := 0; ; i++ {
		filename := fmt.Sprintf("%s.%d.mcp", base, i)
		f, err := os.OpenFile(filename, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err == nil {
			_ = f.Close()
			_ = os.Remove(filename)
			return filename
		}
		if !os.IsExist(err) {
			break
		}
	}
	return fmt.Sprintf("%s.%d.mcp", base, time.Now().Unix())
}

func detectPipeline() (int, bool) {
	stdinIsPipe := !term.IsTerminal(int(os.Stdin.Fd()))
	stdoutIsPipe := !term.IsTerminal(int(os.Stdout.Fd()))
	inPipeline := stdinIsPipe || stdoutIsPipe
	pipeDepth := 0
	if *parentPID > 0 {
		pipeDepth = 1
	} else if inPipeline {
		if stdinIsPipe {
			pipeDepth++
		}
		if stdoutIsPipe {
			pipeDepth++
		}
	}
	return pipeDepth, inPipeline
}

func modifyArgsForChild(args []string, session string) []string {
	currentPID := os.Getpid()
	var out []string
	inserted := false
	for _, arg := range args {
		out = append(out, arg)
		if inserted {
			continue
		}
		if strings.Contains(filepath.Base(arg), "mcpspy") {
			out = append(out, fmt.Sprintf("-parent-pid=%d", currentPID), fmt.Sprintf("-session=%s", session))
			if *autoIndent {
				out = append(out, "-auto-indent")
			}
			inserted = true
		}
	}
	return out
}
