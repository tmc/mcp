// analyze-deduplication analyzes Apple documentation to identify duplicate/redundant data
package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var (
	frameworkPath = flag.String("framework", "", "Path to framework directory")
	verbose       = flag.Bool("v", false, "Verbose output")
)

type Document struct {
	Identifier    map[string]interface{} `json:"identifier"`
	Kind          string                 `json:"kind"`
	Metadata      map[string]interface{} `json:"metadata"`
	LegalNotices  map[string]interface{} `json:"legalNotices"`
	References    map[string]Reference   `json:"references"`
	Abstract      []interface{}          `json:"abstract"`
	Hierarchy     map[string]interface{} `json:"hierarchy"`
	Variants      []interface{}          `json:"variants"`
	SchemaVersion map[string]interface{} `json:"schemaVersion"`
}

type Reference struct {
	Identifier string                 `json:"identifier"`
	Kind       string                 `json:"kind"`
	Role       string                 `json:"role"`
	Title      string                 `json:"title"`
	Type       string                 `json:"type"`
	URL        string                 `json:"url"`
	Fragments  []interface{}          `json:"fragments"`
	Abstract   []interface{}          `json:"abstract"`
	Extra      map[string]interface{} `json:"-"`
}

type Stats struct {
	TotalFiles       int
	TotalBytes       int64
	TotalReferences  int
	UniqueReferences map[string]int // reference identifier -> count

	// Duplicate tracking
	LegalNoticesHashes  map[string]int // hash -> count
	PlatformHashes      map[string]int // hash -> count
	ModuleHashes        map[string]int // hash -> count
	ReferenceHashes     map[string]int // hash -> count
	FragmentHashes      map[string]int // hash -> count
	SchemaVersionHashes map[string]int // hash -> count
	HierarchyPaths      map[string]int // path -> count

	// Size tracking
	LegalNoticesBytes int64
	PlatformBytes     int64
	ReferenceBytes    int64
	MetadataBytes     int64

	// Actual unique data
	UniqueLegalNotices   map[string]interface{}
	UniquePlatforms      map[string]interface{}
	UniqueModules        map[string]interface{}
	UniqueReferencesData map[string]Reference
	UniqueFragments      map[string]interface{}
	UniqueSchemaVersions map[string]interface{}
}

func main() {
	flag.Parse()

	if *frameworkPath == "" {
		log.Fatal("--framework required")
	}

	stats := &Stats{
		UniqueReferences:     make(map[string]int),
		LegalNoticesHashes:   make(map[string]int),
		PlatformHashes:       make(map[string]int),
		ModuleHashes:         make(map[string]int),
		ReferenceHashes:      make(map[string]int),
		FragmentHashes:       make(map[string]int),
		SchemaVersionHashes:  make(map[string]int),
		HierarchyPaths:       make(map[string]int),
		UniqueLegalNotices:   make(map[string]interface{}),
		UniquePlatforms:      make(map[string]interface{}),
		UniqueModules:        make(map[string]interface{}),
		UniqueReferencesData: make(map[string]Reference),
		UniqueFragments:      make(map[string]interface{}),
		UniqueSchemaVersions: make(map[string]interface{}),
	}

	// Walk all JSON files
	err := filepath.WalkDir(*frameworkPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}

		if *verbose {
			fmt.Printf("Analyzing %s\n", path)
		}

		return analyzeFile(path, stats)
	})

	if err != nil {
		log.Fatal(err)
	}

	// Print report
	printReport(stats)
}

func analyzeFile(path string, stats *Stats) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	stats.TotalFiles++
	stats.TotalBytes += int64(len(data))

	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}

	// Analyze legalNotices
	if len(doc.LegalNotices) > 0 {
		legalJSON, _ := json.Marshal(doc.LegalNotices)
		hash := hashJSON(legalJSON)
		stats.LegalNoticesHashes[hash]++
		stats.LegalNoticesBytes += int64(len(legalJSON))
		stats.UniqueLegalNotices[hash] = doc.LegalNotices
	}

	// Analyze metadata
	if metadata := doc.Metadata; len(metadata) > 0 {
		metadataJSON, _ := json.Marshal(metadata)
		stats.MetadataBytes += int64(len(metadataJSON))

		// Platforms
		if platforms, ok := metadata["platforms"].([]interface{}); ok {
			platformsJSON, _ := json.Marshal(platforms)
			hash := hashJSON(platformsJSON)
			stats.PlatformHashes[hash]++
			stats.PlatformBytes += int64(len(platformsJSON))
			stats.UniquePlatforms[hash] = platforms
		}

		// Modules
		if modules, ok := metadata["modules"].([]interface{}); ok {
			modulesJSON, _ := json.Marshal(modules)
			hash := hashJSON(modulesJSON)
			stats.ModuleHashes[hash]++
			stats.UniqueModules[hash] = modules
		}

		// Fragments
		if fragments, ok := metadata["fragments"].([]interface{}); ok {
			fragmentsJSON, _ := json.Marshal(fragments)
			hash := hashJSON(fragmentsJSON)
			stats.FragmentHashes[hash]++
			stats.UniqueFragments[hash] = fragments
		}
	}

	// Analyze schemaVersion
	if len(doc.SchemaVersion) > 0 {
		schemaJSON, _ := json.Marshal(doc.SchemaVersion)
		hash := hashJSON(schemaJSON)
		stats.SchemaVersionHashes[hash]++
		stats.UniqueSchemaVersions[hash] = doc.SchemaVersion
	}

	// Analyze hierarchy paths
	if hierarchy := doc.Hierarchy; len(hierarchy) > 0 {
		if paths, ok := hierarchy["paths"].([]interface{}); ok {
			for _, p := range paths {
				pathJSON, _ := json.Marshal(p)
				stats.HierarchyPaths[string(pathJSON)]++
			}
		}
	}

	// Analyze references
	stats.TotalReferences += len(doc.References)
	for id, ref := range doc.References {
		stats.UniqueReferences[id]++

		// Track unique reference data
		refJSON, _ := json.Marshal(ref)
		hash := hashJSON(refJSON)
		stats.ReferenceHashes[hash]++
		stats.ReferenceBytes += int64(len(refJSON))
		stats.UniqueReferencesData[hash] = ref
	}

	return nil
}

func hashJSON(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h[:8]) // Use first 8 bytes for display
}

func printReport(stats *Stats) {
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("DEDUPLICATION ANALYSIS REPORT")
	fmt.Println(strings.Repeat("=", 80))

	// Overall stats
	fmt.Printf("\nOVERALL STATISTICS:\n")
	fmt.Printf("  Total Files:       %d\n", stats.TotalFiles)
	fmt.Printf("  Total Size:        %s (%.2f MB)\n", formatBytes(stats.TotalBytes), float64(stats.TotalBytes)/1024/1024)
	fmt.Printf("  Total References:  %d\n", stats.TotalReferences)
	fmt.Printf("  Avg File Size:     %s\n", formatBytes(stats.TotalBytes/int64(stats.TotalFiles)))

	// Legal Notices
	fmt.Printf("\nLEGAL NOTICES:\n")
	fmt.Printf("  Total Occurrences: %d\n", stats.TotalFiles)
	fmt.Printf("  Unique Versions:   %d\n", len(stats.LegalNoticesHashes))
	fmt.Printf("  Total Bytes:       %s\n", formatBytes(stats.LegalNoticesBytes))
	if len(stats.LegalNoticesHashes) > 0 {
		// Most common
		_, count := findMostCommon(stats.LegalNoticesHashes)
		fmt.Printf("  Most Common:       %d occurrences (%.1f%%)\n", count, float64(count)/float64(stats.TotalFiles)*100)
		// Dedup savings
		avgSize := stats.LegalNoticesBytes / int64(stats.TotalFiles)
		saved := avgSize * int64(stats.TotalFiles-len(stats.LegalNoticesHashes))
		fmt.Printf("  Dedup Savings:     %s (%.1f%%)\n", formatBytes(saved), float64(saved)/float64(stats.LegalNoticesBytes)*100)

		if *verbose && len(stats.UniqueLegalNotices) > 0 {
			for hash, legal := range stats.UniqueLegalNotices {
				fmt.Printf("    %s: %d occurrences\n", hash, stats.LegalNoticesHashes[hash])
				legalJSON, _ := json.MarshalIndent(legal, "      ", "  ")
				fmt.Printf("      %s\n", legalJSON)
				break // Just show one example
			}
		}
	}

	// Platform Information
	fmt.Printf("\nPLATFORM INFORMATION:\n")
	platformOccurrences := 0
	for _, count := range stats.PlatformHashes {
		platformOccurrences += count
	}
	fmt.Printf("  Total Occurrences: %d\n", platformOccurrences)
	fmt.Printf("  Unique Versions:   %d\n", len(stats.PlatformHashes))
	fmt.Printf("  Total Bytes:       %s\n", formatBytes(stats.PlatformBytes))
	if len(stats.PlatformHashes) > 0 {
		_, count := findMostCommon(stats.PlatformHashes)
		fmt.Printf("  Most Common:       %d occurrences (%.1f%%)\n", count, float64(count)/float64(platformOccurrences)*100)
		avgSize := stats.PlatformBytes / int64(platformOccurrences)
		saved := avgSize * int64(platformOccurrences-len(stats.PlatformHashes))
		fmt.Printf("  Dedup Savings:     %s (%.1f%%)\n", formatBytes(saved), float64(saved)/float64(stats.PlatformBytes)*100)

		if *verbose {
			fmt.Printf("\n  Top 5 Platform Configurations:\n")
			printTopN(stats.PlatformHashes, 5)
		}
	}

	// Schema Version
	fmt.Printf("\nSCHEMA VERSION:\n")
	schemaOccurrences := 0
	for _, count := range stats.SchemaVersionHashes {
		schemaOccurrences += count
	}
	fmt.Printf("  Total Occurrences: %d\n", schemaOccurrences)
	fmt.Printf("  Unique Versions:   %d\n", len(stats.SchemaVersionHashes))
	if len(stats.SchemaVersionHashes) > 0 && *verbose {
		for hash, schema := range stats.UniqueSchemaVersions {
			schemaJSON, _ := json.MarshalIndent(schema, "    ", "  ")
			fmt.Printf("    %s (%d occurrences):\n%s\n", hash, stats.SchemaVersionHashes[hash], schemaJSON)
		}
	}

	// Modules
	fmt.Printf("\nMODULES:\n")
	moduleOccurrences := 0
	for _, count := range stats.ModuleHashes {
		moduleOccurrences += count
	}
	fmt.Printf("  Total Occurrences: %d\n", moduleOccurrences)
	fmt.Printf("  Unique Versions:   %d\n", len(stats.ModuleHashes))
	if len(stats.ModuleHashes) > 0 && *verbose {
		fmt.Printf("\n  Unique Modules:\n")
		for hash, module := range stats.UniqueModules {
			moduleJSON, _ := json.Marshal(module)
			fmt.Printf("    %s (%d occurrences): %s\n", hash, stats.ModuleHashes[hash], moduleJSON)
		}
	}

	// References
	fmt.Printf("\nREFERENCES:\n")
	fmt.Printf("  Total References:       %d\n", stats.TotalReferences)
	fmt.Printf("  Unique Reference IDs:   %d\n", len(stats.UniqueReferences))
	fmt.Printf("  Unique Reference Data:  %d\n", len(stats.ReferenceHashes))
	fmt.Printf("  Total Reference Bytes:  %s\n", formatBytes(stats.ReferenceBytes))
	fmt.Printf("  Avg References/File:    %.1f\n", float64(stats.TotalReferences)/float64(stats.TotalFiles))

	// Cross-file reference sharing
	crossFileRefs := 0
	for _, count := range stats.UniqueReferences {
		if count > 1 {
			crossFileRefs++
		}
	}
	fmt.Printf("  Cross-File References:  %d (%.1f%%)\n", crossFileRefs, float64(crossFileRefs)/float64(len(stats.UniqueReferences))*100)

	// Most referenced
	if *verbose {
		fmt.Printf("\n  Top 10 Most Referenced:\n")
		type refCount struct {
			id    string
			count int
		}
		var refs []refCount
		for id, count := range stats.UniqueReferences {
			refs = append(refs, refCount{id, count})
		}
		sort.Slice(refs, func(i, j int) bool {
			return refs[i].count > refs[j].count
		})
		for i := 0; i < 10 && i < len(refs); i++ {
			fmt.Printf("    %3d: %s\n", refs[i].count, refs[i].id)
		}
	}

	// Deduplication potential for references
	if len(stats.ReferenceHashes) > 0 {
		avgRefSize := stats.ReferenceBytes / int64(stats.TotalReferences)
		uniqueRefBytes := avgRefSize * int64(len(stats.ReferenceHashes))
		saved := stats.ReferenceBytes - uniqueRefBytes
		fmt.Printf("  Reference Dedup Savings: %s (%.1f%%)\n", formatBytes(saved), float64(saved)/float64(stats.ReferenceBytes)*100)
	}

	// Hierarchy paths
	fmt.Printf("\nHIERARCHY PATHS:\n")
	fmt.Printf("  Unique Paths:      %d\n", len(stats.HierarchyPaths))
	if *verbose && len(stats.HierarchyPaths) > 0 {
		fmt.Printf("\n  Top 10 Most Common Paths:\n")
		printTopN(stats.HierarchyPaths, 10)
	}

	// Overall deduplication potential
	fmt.Print("\n" + strings.Repeat("=", 80))
	fmt.Print("\nDEDUPLICATION POTENTIAL SUMMARY:\n")
	fmt.Print(strings.Repeat("=", 80) + "\n")

	totalDedup := int64(0)

	// Legal notices
	if len(stats.LegalNoticesHashes) > 0 {
		avgSize := stats.LegalNoticesBytes / int64(stats.TotalFiles)
		saved := avgSize * int64(stats.TotalFiles-len(stats.LegalNoticesHashes))
		totalDedup += saved
		fmt.Printf("  Legal Notices:     %s\n", formatBytes(saved))
	}

	// Platforms
	if len(stats.PlatformHashes) > 0 {
		platformOccurrences := 0
		for _, count := range stats.PlatformHashes {
			platformOccurrences += count
		}
		avgSize := stats.PlatformBytes / int64(platformOccurrences)
		saved := avgSize * int64(platformOccurrences-len(stats.PlatformHashes))
		totalDedup += saved
		fmt.Printf("  Platform Info:     %s\n", formatBytes(saved))
	}

	// References
	if len(stats.ReferenceHashes) > 0 {
		avgRefSize := stats.ReferenceBytes / int64(stats.TotalReferences)
		uniqueRefBytes := avgRefSize * int64(len(stats.ReferenceHashes))
		saved := stats.ReferenceBytes - uniqueRefBytes
		totalDedup += saved
		fmt.Printf("  References:        %s\n", formatBytes(saved))
	}

	fmt.Printf("  %s\n", strings.Repeat("-", 40))
	fmt.Printf("  Total Savings:     %s (%.1f%% of total)\n", formatBytes(totalDedup), float64(totalDedup)/float64(stats.TotalBytes)*100)
	fmt.Printf("  Original Size:     %s\n", formatBytes(stats.TotalBytes))
	fmt.Printf("  Deduplicated Size: %s\n", formatBytes(stats.TotalBytes-totalDedup))
	fmt.Printf("  Compression Ratio: %.2fx\n", float64(stats.TotalBytes)/float64(stats.TotalBytes-totalDedup))

	fmt.Print("\n" + strings.Repeat("=", 80) + "\n")
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func findMostCommon(m map[string]int) (string, int) {
	var mostCommonHash string
	var mostCommonCount int
	for hash, count := range m {
		if count > mostCommonCount {
			mostCommonHash = hash
			mostCommonCount = count
		}
	}
	return mostCommonHash, mostCommonCount
}

func printTopN(m map[string]int, n int) {
	type item struct {
		hash  string
		count int
	}
	var items []item
	for hash, count := range m {
		items = append(items, item{hash, count})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].count > items[j].count
	})
	for i := 0; i < n && i < len(items); i++ {
		fmt.Printf("    %s: %d occurrences\n", items[i].hash, items[i].count)
	}
}
