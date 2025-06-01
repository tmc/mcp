// Command mcptrace-server serves MCP trace files via HTTP with live updates.
// Monitors directories for trace files and provides dynamic web interface.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

var (
	addr      = flag.String("addr", ":8080", "HTTP server address")
	dir       = flag.String("dir", ".", "Directory to monitor for .mcp files")
	verbose   = flag.Bool("v", false, "Verbose output")
	quiet     = flag.Bool("q", false, "Quiet mode")
	open      = flag.Bool("open", false, "Open server in browser on startup")
)

// FileInfo represents an MCP trace file
type FileInfo struct {
	Name    string    `json:"name"`
	Path    string    `json:"path"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"modTime"`
	Lines   int       `json:"lines"`
}

// Server manages the web server and file watching
type Server struct {
	dir      string
	clients  map[chan string]bool
	files    map[string]*FileInfo
	watcher  *fsnotify.Watcher
}

func main() {
	log.SetPrefix("mcptrace-server: ")
	log.SetFlags(0)
	
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Serves MCP trace files via HTTP with live updates.\n")
		fmt.Fprintf(os.Stderr, "Monitors directory for .mcp files and provides dynamic web interface.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *quiet {
		log.SetOutput(os.DevNull)
	}

	// Convert to absolute path
	absDir, err := filepath.Abs(*dir)
	if err != nil {
		log.Fatalf("failed to get absolute path: %v", err)
	}

	// Check if directory exists
	if info, err := os.Stat(absDir); err != nil {
		log.Fatalf("directory does not exist: %v", err)
	} else if !info.IsDir() {
		log.Fatalf("path is not a directory: %s", absDir)
	}

	// Create server
	server := &Server{
		dir:     absDir,
		clients: make(map[chan string]bool),
		files:   make(map[string]*FileInfo),
	}

	// Set up file watcher
	if err := server.setupWatcher(); err != nil {
		log.Fatalf("failed to setup file watcher: %v", err)
	}
	defer server.watcher.Close()

	// Initial scan
	if err := server.scanFiles(); err != nil {
		log.Fatalf("failed to scan files: %v", err)
	}

	// Set up HTTP routes
	http.HandleFunc("/", server.handleIndex)
	http.HandleFunc("/api/files", server.handleFiles)
	http.HandleFunc("/api/file/", server.handleFileContent)
	http.HandleFunc("/view/", server.handleFileView)
	http.HandleFunc("/events", server.handleSSE)

	if *verbose {
		log.Printf("monitoring directory: %s", absDir)
		log.Printf("starting server on %s", *addr)
	}

	// Open browser if requested
	if *open {
		go func() {
			time.Sleep(500 * time.Millisecond) // Give server time to start
			url := fmt.Sprintf("http://localhost%s", *addr)
			if err := openBrowser(url); err != nil {
				log.Printf("failed to open browser: %v", err)
			}
		}()
	}

	log.Fatal(http.ListenAndServe(*addr, nil))
}

func (s *Server) setupWatcher() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	s.watcher = watcher

	// Watch the directory
	if err := watcher.Add(s.dir); err != nil {
		return err
	}

	// Start watching in background
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if strings.HasSuffix(event.Name, ".mcp") {
					if *verbose {
						log.Printf("file event: %s %s", event.Op, event.Name)
					}
					s.handleFileEvent(event)
				}
			case err := <-watcher.Errors:
				if !*quiet {
					log.Printf("watcher error: %v", err)
				}
			}
		}
	}()

	return nil
}

func (s *Server) handleFileEvent(event fsnotify.Event) {
	switch {
	case event.Op&fsnotify.Create == fsnotify.Create:
		s.addFile(event.Name)
		s.broadcast(fmt.Sprintf(`{"type":"create","file":"%s"}`, filepath.Base(event.Name)))
	case event.Op&fsnotify.Write == fsnotify.Write:
		s.updateFile(event.Name)
		s.broadcast(fmt.Sprintf(`{"type":"update","file":"%s"}`, filepath.Base(event.Name)))
	case event.Op&fsnotify.Remove == fsnotify.Remove:
		s.removeFile(event.Name)
		s.broadcast(fmt.Sprintf(`{"type":"remove","file":"%s"}`, filepath.Base(event.Name)))
	}
}

func (s *Server) scanFiles() error {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".mcp") {
			fullPath := filepath.Join(s.dir, entry.Name())
			s.addFile(fullPath)
		}
	}

	if *verbose {
		log.Printf("found %d .mcp files", len(s.files))
	}

	return nil
}

func (s *Server) addFile(path string) {
	info, err := s.getFileInfo(path)
	if err != nil {
		if *verbose {
			log.Printf("failed to get file info for %s: %v", path, err)
		}
		return
	}
	s.files[path] = info
}

func (s *Server) updateFile(path string) {
	if _, exists := s.files[path]; exists {
		s.addFile(path) // Just re-add to update info
	}
}

func (s *Server) removeFile(path string) {
	delete(s.files, path)
}

func (s *Server) getFileInfo(path string) (*FileInfo, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	// Count lines
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Count(string(content), "\n")

	return &FileInfo{
		Name:    filepath.Base(path),
		Path:    path,
		Size:    stat.Size(),
		ModTime: stat.ModTime(),
		Lines:   lines,
	}, nil
}

func (s *Server) broadcast(message string) {
	for client := range s.clients {
		select {
		case client <- message:
		default:
			// Client buffer full, remove it
			close(client)
			delete(s.clients, client)
		}
	}
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	tmpl := `<!DOCTYPE html>
<html>
<head>
    <title>MCP Trace Server</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Arial, sans-serif; margin: 20px; background: #f8f9fa; }
        .header { background: white; padding: 20px; border-radius: 8px; margin-bottom: 20px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .file-list { list-style: none; padding: 0; }
        .file-item { 
            padding: 15px; 
            border: 1px solid #ddd; 
            margin: 10px 0; 
            border-radius: 8px;
            background: white;
            box-shadow: 0 1px 3px rgba(0,0,0,0.1);
            transition: all 0.2s;
        }
        .file-item:hover { box-shadow: 0 2px 8px rgba(0,0,0,0.15); }
        .file-name { font-weight: bold; color: #0066cc; font-size: 1.1em; margin-bottom: 5px; }
        .file-meta { font-size: 0.9em; color: #666; margin-bottom: 10px; }
        .file-actions { display: flex; gap: 10px; }
        .btn { 
            padding: 6px 12px; 
            border: 1px solid #ddd; 
            border-radius: 4px; 
            text-decoration: none; 
            font-size: 0.9em;
            transition: all 0.2s;
        }
        .btn:hover { background: #f0f0f0; }
        .btn-primary { background: #0066cc; color: white; border-color: #0066cc; }
        .btn-primary:hover { background: #0056b3; }
        .status { 
            position: fixed; 
            top: 10px; 
            right: 10px; 
            padding: 8px 15px; 
            background: #4CAF50; 
            color: white; 
            border-radius: 4px; 
            font-size: 0.9em;
        }
        .new-file { animation: highlight-green 2s; }
        .updated-file { animation: highlight-yellow 2s; }
        @keyframes highlight-green { 
            0% { background: #e8f5e8; border-color: #4CAF50; }
            100% { background: white; border-color: #ddd; }
        }
        @keyframes highlight-yellow { 
            0% { background: #fff3cd; border-color: #ffc107; }
            100% { background: white; border-color: #ddd; }
        }
        .stats { display: flex; gap: 20px; font-size: 0.9em; color: #666; }
    </style>
</head>
<body>
    <div class="status" id="status">Connected</div>
    <div class="header">
        <h1>MCP Trace Server</h1>
        <p>Monitoring: <code>{{.Dir}}</code></p>
        <div class="stats">
            <span><strong>Files:</strong> <span id="fileCount">0</span></span>
            <span><strong>Last Updated:</strong> <span id="lastUpdate">-</span></span>
        </div>
    </div>
    <ul class="file-list" id="fileList">
        <li>Loading...</li>
    </ul>

    <script>
        const fileList = document.getElementById('fileList');
        const status = document.getElementById('status');
        const fileCount = document.getElementById('fileCount');
        const lastUpdate = document.getElementById('lastUpdate');
        let files = {};

        // Fetch initial file list
        fetch('/api/files')
            .then(r => r.json())
            .then(data => {
                files = {};
                data.forEach(f => files[f.name] = f);
                updateFileList();
            });

        // Set up SSE for live updates
        const eventSource = new EventSource('/events');
        
        eventSource.onopen = () => {
            status.textContent = 'Connected';
            status.style.background = '#4CAF50';
        };
        
        eventSource.onerror = () => {
            status.textContent = 'Disconnected';
            status.style.background = '#f44336';
        };
        
        eventSource.onmessage = (event) => {
            const data = JSON.parse(event.data);
            if (data.type === 'ping') return;
            handleFileEvent(data);
        };

        function handleFileEvent(event) {
            if (event.type === 'remove') {
                delete files[event.file];
            } else {
                // Refresh file info for create/update
                fetch('/api/file/' + event.file)
                    .then(r => r.json())
                    .then(fileInfo => {
                        files[event.file] = fileInfo;
                        updateFileList();
                        
                        // Highlight based on event type
                        const item = document.getElementById('file-' + event.file.replace(/[^a-zA-Z0-9]/g, '-'));
                        if (item) {
                            item.classList.add(event.type === 'create' ? 'new-file' : 'updated-file');
                            setTimeout(() => {
                                item.classList.remove('new-file', 'updated-file');
                            }, 2000);
                        }
                    });
                return;
            }
            updateFileList();
        }

        function updateFileList() {
            const fileArray = Object.values(files).sort((a, b) => b.modTime.localeCompare(a.modTime));
            fileCount.textContent = fileArray.length;
            lastUpdate.textContent = fileArray.length > 0 ? new Date(fileArray[0].modTime).toLocaleString() : '-';
            
            if (fileArray.length === 0) {
                fileList.innerHTML = '<li style="padding: 20px; text-align: center; color: #666;">No .mcp files found</li>';
                return;
            }

            fileList.innerHTML = fileArray.map(file => {
                const safeId = file.name.replace(/[^a-zA-Z0-9]/g, '-');
                return '<li class="file-item" id="file-' + safeId + '">' +
                    '<div class="file-name">' + file.name + '</div>' +
                    '<div class="file-meta">' +
                        'Size: ' + formatBytes(file.size) + ' | ' +
                        'Lines: ' + file.lines + ' | ' +
                        'Modified: ' + new Date(file.modTime).toLocaleString() +
                    '</div>' +
                    '<div class="file-actions">' +
                        '<a href="/view/' + encodeURIComponent(file.name) + '" class="btn btn-primary">View</a>' +
                        '<a href="/api/file/' + encodeURIComponent(file.name) + '?raw=1" class="btn">Download</a>' +
                    '</div>' +
                '</li>';
            }).join('');
        }

        function formatBytes(bytes) {
            if (bytes === 0) return '0 B';
            const k = 1024;
            const sizes = ['B', 'KB', 'MB', 'GB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
        }
    </script>
</body>
</html>`

	t := template.Must(template.New("index").Parse(tmpl))
	t.Execute(w, struct{ Dir string }{Dir: s.dir})
}

func (s *Server) handleFiles(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	var fileList []*FileInfo
	for _, file := range s.files {
		fileList = append(fileList, file)
	}
	
	// Sort by modification time (newest first)
	sort.Slice(fileList, func(i, j int) bool {
		return fileList[i].ModTime.After(fileList[j].ModTime)
	})
	
	json.NewEncoder(w).Encode(fileList)
}

func (s *Server) handleFileContent(w http.ResponseWriter, r *http.Request) {
	filename := strings.TrimPrefix(r.URL.Path, "/api/file/")
	
	var file *FileInfo
	for _, f := range s.files {
		if f.Name == filename {
			file = f
			break
		}
	}
	
	if file == nil {
		http.NotFound(w, r)
		return
	}
	
	// If raw parameter is set, serve the file directly
	if r.URL.Query().Get("raw") == "1" {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
		http.ServeFile(w, r, file.Path)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(file)
}

func (s *Server) handleFileView(w http.ResponseWriter, r *http.Request) {
	filename := strings.TrimPrefix(r.URL.Path, "/view/")
	
	var file *FileInfo
	for _, f := range s.files {
		if f.Name == filename {
			file = f
			break
		}
	}
	
	if file == nil {
		http.NotFound(w, r)
		return
	}
	
	// Use mcptrace-to-html to generate the view
	// For now, serve a simple HTML page with the file content
	content, err := os.ReadFile(file.Path)
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}
	
	tmpl := `<!DOCTYPE html>
<html>
<head>
    <title>{{.Filename}} - MCP Trace</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Arial, sans-serif; margin: 20px; background: #f8f9fa; }
        .header { background: white; padding: 20px; border-radius: 8px; margin-bottom: 20px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .content { background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        pre { background: #f8f9fa; padding: 15px; border-radius: 4px; overflow-x: auto; }
        .back-link { display: inline-block; margin-bottom: 10px; text-decoration: none; color: #0066cc; }
        .back-link:hover { text-decoration: underline; }
    </style>
</head>
<body>
    <div class="header">
        <a href="/" class="back-link">← Back to file list</a>
        <h1>{{.Filename}}</h1>
        <p><strong>Size:</strong> {{.Size}} bytes | <strong>Lines:</strong> {{.Lines}} | <strong>Modified:</strong> {{.ModTime}}</p>
    </div>
    <div class="content">
        <pre>{{.Content}}</pre>
    </div>
</body>
</html>`

	data := struct {
		Filename string
		Size     int64
		Lines    int
		ModTime  string
		Content  string
	}{
		Filename: filename,
		Size:     file.Size,
		Lines:    file.Lines,
		ModTime:  file.ModTime.Format("2006-01-02 15:04:05"),
		Content:  string(content),
	}
	
	t := template.Must(template.New("view").Parse(tmpl))
	t.Execute(w, data)
}

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create channel for this client
	client := make(chan string, 10)
	s.clients[client] = true

	// Clean up when client disconnects
	defer func() {
		delete(s.clients, client)
		close(client)
	}()

	// Send keep-alive pings
	ping := time.NewTicker(30 * time.Second)
	defer ping.Stop()

	for {
		select {
		case message := <-client:
			fmt.Fprintf(w, "data: %s\n\n", message)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		case <-ping.C:
			fmt.Fprintf(w, "data: {\"type\":\"ping\"}\n\n")
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		case <-r.Context().Done():
			return
		}
	}
}

// openBrowser opens the specified URL in the default browser
func openBrowser(url string) error {
	var cmd string
	var args []string

	switch {
	case os.Getenv("WSL_DISTRO_NAME") != "":
		// Windows Subsystem for Linux
		cmd = "cmd.exe"
		args = []string{"/c", "start", url}
	case strings.Contains(os.Getenv("PATH"), "/mnt/c"):
		// WSL detection fallback
		cmd = "cmd.exe"
		args = []string{"/c", "start", url}
	default:
		// Native platform detection
		switch {
		case fileExists("/usr/bin/xdg-open"):
			cmd = "xdg-open"
			args = []string{url}
		case fileExists("/usr/bin/open"):
			cmd = "open"
			args = []string{url}
		case fileExists("/System/Library/CoreServices/Finder.app"):
			cmd = "open"
			args = []string{url}
		default:
			return fmt.Errorf("unsupported platform")
		}
	}

	return exec.Command(cmd, args...).Start()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}