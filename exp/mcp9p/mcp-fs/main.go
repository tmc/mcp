// mcp-fs - FUSE filesystem for MCP namespaces (Plan9-style)
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

// NamespaceFS implements a FUSE filesystem for MCP namespaces
type NamespaceFS struct {
	fs.Inode
	
	nsServer string
	client   *http.Client
	cache    map[string]*CacheEntry
	cacheMu  sync.RWMutex
}

// CacheEntry stores cached namespace data
type CacheEntry struct {
	data      interface{}
	timestamp time.Time
}

// NamespaceNode represents a namespace directory
type NamespaceNode struct {
	fs.Inode
	
	nsFS *NamespaceFS
	path string
}

// ServiceNode represents a service file
type ServiceNode struct {
	fs.Inode
	
	nsFS  *NamespaceFS
	path  string
	entry map[string]interface{}
}

// NewNamespaceFS creates a new namespace filesystem
func NewNamespaceFS(nsServer string) *NamespaceFS {
	return &NamespaceFS{
		nsServer: nsServer,
		client:   &http.Client{Timeout: 5 * time.Second},
		cache:    make(map[string]*CacheEntry),
	}
}

// OnAdd is called when the filesystem is mounted
func (nsfs *NamespaceFS) OnAdd(ctx context.Context) {
	// Initialize root node
}

// Lookup looks up a child node
func (nsfs *NamespaceFS) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	return nsfs.lookupPath(ctx, name, out)
}

// Readdir lists directory contents
func (nsfs *NamespaceFS) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	return nsfs.readdir(ctx, "/")
}

// lookupPath looks up a path in the namespace
func (nsfs *NamespaceFS) lookupPath(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	fullPath := "/" + name
	
	// Check cache first
	if cached := nsfs.getCache(fullPath); cached != nil {
		if entry, ok := cached.(map[string]interface{}); ok {
			return nsfs.createNode(name, entry, out), 0
		}
	}
	
	// Lookup in namespace
	resp, err := nsfs.client.Get(nsfs.nsServer + "/lookup?path=" + fullPath)
	if err != nil {
		return nil, syscall.EIO
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == http.StatusNotFound {
		// Try as a directory
		return nsfs.createNamespaceNode(name, fullPath, out), 0
	}
	
	if resp.StatusCode != http.StatusOK {
		return nil, syscall.ENOENT
	}
	
	var entry map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&entry); err != nil {
		return nil, syscall.EIO
	}
	
	nsfs.setCache(fullPath, entry)
	return nsfs.createNode(name, entry, out), 0
}

// createNode creates an inode for an entry
func (nsfs *NamespaceFS) createNode(name string, entry map[string]interface{}, out *fuse.EntryOut) *fs.Inode {
	entryType, _ := entry["type"].(string)
	
	if entryType == "namespace" {
		// Directory node
		node := &NamespaceNode{
			nsFS: nsfs,
			path: "/" + name,
		}
		out.Mode = 0755 | syscall.S_IFDIR
		var ctx = context.Background()
		return nsfs.NewInode(ctx, node, fs.StableAttr{Mode: out.Mode})
	}
	
	// File node
	node := &ServiceNode{
		nsFS:  nsfs,
		path:  "/" + name,
		entry: entry,
	}
	out.Mode = 0644 | syscall.S_IFREG
	out.Size = uint64(len(nsfs.formatEntry(entry)))

	var ctx = context.Background()
	return nsfs.NewInode(ctx, node, fs.StableAttr{Mode: out.Mode})
}

// createNamespaceNode creates a namespace directory node
func (nsfs *NamespaceFS) createNamespaceNode(name, path string, out *fuse.EntryOut) *fs.Inode {
	node := &NamespaceNode{
		nsFS: nsfs,
		path: path,
	}
	out.Mode = 0755 | syscall.S_IFDIR

	var ctx = context.Background()
	return nsfs.NewInode(ctx, node, fs.StableAttr{Mode: out.Mode})
}

// readdir lists directory contents for a path
func (nsfs *NamespaceFS) readdir(ctx context.Context, path string) (fs.DirStream, syscall.Errno) {
	// Check cache
	if cached := nsfs.getCache(path + "/_list"); cached != nil {
		if entries, ok := cached.([]map[string]interface{}); ok {
			return nsfs.entriesToDirStream(entries), 0
		}
	}
	
	// List from namespace
	resp, err := nsfs.client.Get(nsfs.nsServer + "/list?path=" + path)
	if err != nil {
		return nil, syscall.EIO
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, syscall.ENOENT
	}
	
	var entries []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, syscall.EIO
	}
	
	nsfs.setCache(path+"/_list", entries)
	return nsfs.entriesToDirStream(entries), 0
}

// entriesToDirStream converts entries to directory stream
func (nsfs *NamespaceFS) entriesToDirStream(entries []map[string]interface{}) fs.DirStream {
	r := make([]fuse.DirEntry, 0, len(entries))
	
	for _, entry := range entries {
		name, _ := entry["name"].(string)
		entryType, _ := entry["type"].(string)
		
		var mode uint32
		if entryType == "namespace" {
			mode = syscall.S_IFDIR
		} else {
			mode = syscall.S_IFREG
		}
		
		r = append(r, fuse.DirEntry{
			Name: name,
			Mode: mode,
		})
	}
	
	return fs.NewListDirStream(r)
}

// formatEntry formats an entry as text
func (nsfs *NamespaceFS) formatEntry(entry map[string]interface{}) string {
	data, _ := json.MarshalIndent(entry, "", "  ")
	return string(data) + "\n"
}

// Cache management

func (nsfs *NamespaceFS) getCache(key string) interface{} {
	nsfs.cacheMu.RLock()
	defer nsfs.cacheMu.RUnlock()
	
	if entry, ok := nsfs.cache[key]; ok {
		if time.Since(entry.timestamp) < 5*time.Second {
			return entry.data
		}
	}
	return nil
}

func (nsfs *NamespaceFS) setCache(key string, data interface{}) {
	nsfs.cacheMu.Lock()
	defer nsfs.cacheMu.Unlock()
	
	nsfs.cache[key] = &CacheEntry{
		data:      data,
		timestamp: time.Now(),
	}
}

// NamespaceNode methods

func (n *NamespaceNode) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	fullPath := path.Join(n.path, name)
	return n.nsFS.lookupPath(ctx, strings.TrimPrefix(fullPath, "/"), out)
}

func (n *NamespaceNode) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	return n.nsFS.readdir(ctx, n.path)
}

// ServiceNode methods

func (n *ServiceNode) Open(ctx context.Context, flags uint32) (fs.FileHandle, uint32, syscall.Errno) {
	return nil, fuse.FOPEN_KEEP_CACHE, 0
}

func (n *ServiceNode) Read(ctx context.Context, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	content := []byte(n.nsFS.formatEntry(n.entry))
	
	if off >= int64(len(content)) {
		return fuse.ReadResultData(nil), 0
	}
	
	end := off + int64(len(dest))
	if end > int64(len(content)) {
		end = int64(len(content))
	}
	
	return fuse.ReadResultData(content[off:end]), 0
}

func (n *ServiceNode) Getattr(ctx context.Context, out *fuse.AttrOut) syscall.Errno {
	content := n.nsFS.formatEntry(n.entry)
	out.Size = uint64(len(content))
	out.Mode = 0644
	return 0
}

// Special files

type SpecialFile struct {
	fs.Inode
	nsFS    *NamespaceFS
	content func() string
}

func (f *SpecialFile) Open(ctx context.Context, flags uint32) (fs.FileHandle, uint32, syscall.Errno) {
	return nil, fuse.FOPEN_KEEP_CACHE, 0
}

func (f *SpecialFile) Read(ctx context.Context, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	content := []byte(f.content())
	
	if off >= int64(len(content)) {
		return fuse.ReadResultData(nil), 0
	}
	
	end := off + int64(len(dest))
	if end > int64(len(content)) {
		end = int64(len(content))
	}
	
	return fuse.ReadResultData(content[off:end]), 0
}

func (f *SpecialFile) Getattr(ctx context.Context, out *fuse.AttrOut) syscall.Errno {
	content := f.content()
	out.Size = uint64(len(content))
	out.Mode = 0444
	return 0
}

func main() {
	var (
		mountpoint = flag.String("mount", "", "Mount point")
		nsServer   = flag.String("ns", "http://localhost:9000", "Namespace server URL")
		debug      = flag.Bool("debug", false, "Enable debug output")
	)
	flag.Parse()
	
	if *mountpoint == "" {
		log.Fatal("Mount point required")
	}
	
	// Create namespace filesystem
	nsFS := NewNamespaceFS(*nsServer)
	
	// Mount options
	opts := &fs.Options{
		MountOptions: fuse.MountOptions{
			Debug: *debug,
			Name:  "mcpfs",
		},
	}
	
	// Create and mount filesystem
	server, err := fs.Mount(*mountpoint, nsFS, opts)
	if err != nil {
		log.Fatalf("Mount failed: %v", err)
	}
	
	log.Printf("Mounted MCP namespace at %s", *mountpoint)
	log.Printf("Namespace server: %s", *nsServer)
	
	// Serve filesystem
	server.Wait()
}