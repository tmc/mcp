package main

import (
	"archive/tar"
	"flag"
	"fmt"
	"go/build"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/txtar"
)

func main() {
	var (
		packages      = flag.String("packages", "", "Package paths to dump (comma-separated)")
		output        = flag.String("output", "-", "Output file (- for stdout)")
		format        = flag.String("format", "txtar", "Output format: txtar, tar, dir")
		recursive     = flag.Bool("recursive", false, "Include dependencies recursively")
		includeTest   = flag.Bool("include-test", false, "Include test files")
		includeVendor = flag.Bool("include-vendor", false, "Include vendor directory")
		prefix        = flag.String("prefix", "", "Prefix for file paths in archive")
		verbose       = flag.Bool("verbose", false, "Verbose output")
	)
	flag.Parse()

	// Get packages from args or flag
	var pkgPaths []string
	if *packages != "" {
		pkgPaths = strings.Split(*packages, ",")
	}
	if flag.NArg() > 0 {
		pkgPaths = append(pkgPaths, flag.Args()...)
	}

	if len(pkgPaths) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	// Create dumper
	dumper := &PackageDumper{
		Format:        *format,
		Recursive:     *recursive,
		IncludeTest:   *includeTest,
		IncludeVendor: *includeVendor,
		Prefix:        *prefix,
		Verbose:       *verbose,
	}

	// Dump packages
	var result interface{}
	var err error

	switch *format {
	case "txtar":
		result, err = dumper.DumpToTxtar(pkgPaths)
	case "tar":
		result, err = dumper.DumpToTar(pkgPaths)
	case "dir":
		if *output == "-" {
			log.Fatal("Directory format requires output path")
		}
		err = dumper.DumpToDir(pkgPaths, *output)
	default:
		log.Fatalf("Unknown format: %s", *format)
	}

	if err != nil {
		log.Fatalf("Dump failed: %v", err)
	}

	// Write output
	if *format != "dir" {
		if *output == "-" {
			switch v := result.(type) {
			case *txtar.Archive:
				fmt.Print(string(txtar.Format(v)))
			case *tar.Writer:
				// Already written
			}
		} else {
			switch v := result.(type) {
			case *txtar.Archive:
				if err := os.WriteFile(*output, txtar.Format(v), 0644); err != nil {
					log.Fatalf("Failed to write output: %v", err)
				}
			case []byte:
				if err := os.WriteFile(*output, v, 0644); err != nil {
					log.Fatalf("Failed to write output: %v", err)
				}
			}
		}
	}

	if *verbose {
		fmt.Fprintf(os.Stderr, "Successfully dumped %d packages\n", len(pkgPaths))
	}
}

// PackageDumper dumps Go package sources
type PackageDumper struct {
	Format        string
	Recursive     bool
	IncludeTest   bool
	IncludeVendor bool
	Prefix        string
	Verbose       bool

	visited map[string]bool
}

func (d *PackageDumper) DumpToTxtar(packages []string) (*txtar.Archive, error) {
	d.visited = make(map[string]bool)
	archive := &txtar.Archive{}

	for _, pkgPath := range packages {
		if err := d.addPackageToTxtar(archive, pkgPath); err != nil {
			return nil, fmt.Errorf("failed to dump %s: %w", pkgPath, err)
		}
	}

	return archive, nil
}

func (d *PackageDumper) addPackageToTxtar(archive *txtar.Archive, pkgPath string) error {
	if d.visited[pkgPath] {
		return nil
	}
	d.visited[pkgPath] = true

	// Load package
	pkg, err := build.Import(pkgPath, ".", 0)
	if err != nil {
		return fmt.Errorf("failed to import package: %w", err)
	}

	if d.Verbose {
		fmt.Fprintf(os.Stderr, "Dumping package: %s\n", pkgPath)
	}

	// Add source files
	for _, file := range pkg.GoFiles {
		if err := d.addFileToTxtar(archive, pkg.Dir, file, pkgPath); err != nil {
			return err
		}
	}

	// Add test files if requested
	if d.IncludeTest {
		for _, file := range pkg.TestGoFiles {
			if err := d.addFileToTxtar(archive, pkg.Dir, file, pkgPath); err != nil {
				return err
			}
		}
		for _, file := range pkg.XTestGoFiles {
			if err := d.addFileToTxtar(archive, pkg.Dir, file, pkgPath); err != nil {
				return err
			}
		}
	}

	// Recurse into dependencies if requested
	if d.Recursive {
		for _, imp := range pkg.Imports {
			if d.shouldIncludeImport(imp) {
				if err := d.addPackageToTxtar(archive, imp); err != nil {
					return err
				}
			}
		}
		if d.IncludeTest {
			for _, imp := range pkg.TestImports {
				if d.shouldIncludeImport(imp) {
					if err := d.addPackageToTxtar(archive, imp); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

func (d *PackageDumper) addFileToTxtar(archive *txtar.Archive, dir, file, pkgPath string) error {
	fullPath := filepath.Join(dir, file)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", fullPath, err)
	}

	// Determine archive path
	archivePath := file
	if d.Prefix != "" {
		archivePath = filepath.Join(d.Prefix, pkgPath, file)
	} else {
		archivePath = filepath.Join(pkgPath, file)
	}

	archive.Files = append(archive.Files, txtar.File{
		Name: archivePath,
		Data: content,
	})

	return nil
}

func (d *PackageDumper) shouldIncludeImport(imp string) bool {
	// Skip standard library unless it's a sub-package
	if !strings.Contains(imp, ".") && !strings.Contains(imp, "/") {
		return false
	}

	// Skip vendor unless requested
	if strings.Contains(imp, "/vendor/") && !d.IncludeVendor {
		return false
	}

	return true
}

func (d *PackageDumper) DumpToTar(packages []string) ([]byte, error) {
	d.visited = make(map[string]bool)

	// Create tar writer
	var buf strings.Builder
	tw := tar.NewWriter(&buf)
	defer tw.Close()

	for _, pkgPath := range packages {
		if err := d.addPackageToTar(tw, pkgPath); err != nil {
			return nil, fmt.Errorf("failed to dump %s: %w", pkgPath, err)
		}
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}

	return []byte(buf.String()), nil
}

func (d *PackageDumper) addPackageToTar(tw *tar.Writer, pkgPath string) error {
	if d.visited[pkgPath] {
		return nil
	}
	d.visited[pkgPath] = true

	// Load package
	pkg, err := build.Import(pkgPath, ".", 0)
	if err != nil {
		return fmt.Errorf("failed to import package: %w", err)
	}

	// Add files to tar
	files := append([]string{}, pkg.GoFiles...)
	if d.IncludeTest {
		files = append(files, pkg.TestGoFiles...)
		files = append(files, pkg.XTestGoFiles...)
	}

	for _, file := range files {
		fullPath := filepath.Join(pkg.Dir, file)
		info, err := os.Stat(fullPath)
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}

		// Set name in archive
		archivePath := file
		if d.Prefix != "" {
			archivePath = filepath.Join(d.Prefix, pkgPath, file)
		} else {
			archivePath = filepath.Join(pkgPath, file)
		}
		header.Name = archivePath

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		content, err := os.ReadFile(fullPath)
		if err != nil {
			return err
		}

		if _, err := tw.Write(content); err != nil {
			return err
		}
	}

	// Recurse if needed
	if d.Recursive {
		for _, imp := range pkg.Imports {
			if d.shouldIncludeImport(imp) {
				if err := d.addPackageToTar(tw, imp); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (d *PackageDumper) DumpToDir(packages []string, outputDir string) error {
	d.visited = make(map[string]bool)

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	for _, pkgPath := range packages {
		if err := d.addPackageToDir(outputDir, pkgPath); err != nil {
			return fmt.Errorf("failed to dump %s: %w", pkgPath, err)
		}
	}

	return nil
}

func (d *PackageDumper) addPackageToDir(outputDir, pkgPath string) error {
	if d.visited[pkgPath] {
		return nil
	}
	d.visited[pkgPath] = true

	// Load package
	pkg, err := build.Import(pkgPath, ".", 0)
	if err != nil {
		return fmt.Errorf("failed to import package: %w", err)
	}

	// Create package directory
	pkgDir := filepath.Join(outputDir, pkgPath)
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		return err
	}

	// Copy files
	files := append([]string{}, pkg.GoFiles...)
	if d.IncludeTest {
		files = append(files, pkg.TestGoFiles...)
		files = append(files, pkg.XTestGoFiles...)
	}

	for _, file := range files {
		src := filepath.Join(pkg.Dir, file)
		dst := filepath.Join(pkgDir, file)

		if err := copyFile(src, dst); err != nil {
			return err
		}
	}

	// Recurse if needed
	if d.Recursive {
		for _, imp := range pkg.Imports {
			if d.shouldIncludeImport(imp) {
				if err := d.addPackageToDir(outputDir, imp); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `gopackdump - Dump Go package sources

Usage:
  gopackdump [options] package...

Examples:
  gopackdump fmt                           # Dump fmt package
  gopackdump -format=tar -o pkg.tar myapp  # Create tar archive
  gopackdump -recursive github.com/pkg/...  # Dump with dependencies
  gopackdump -format=dir -o ./dump pkg     # Extract to directory

Options:
`)
		flag.PrintDefaults()
	}
}
