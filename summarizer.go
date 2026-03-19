package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// Default ignored directories and files
var defaultIgnoreDirs = map[string]bool{
	".git":         true,
	".idea":        true,
	".vscode":      true,
	"vendor":       true,
	"node_modules": true,
	".DS_Store":    true,
	"__pycache__":  true,
	".cache":       true,
	"dist":         true,
	"build":        true,
	"bin":          true,
	"tmp":          true,
}

var defaultIgnoreExts = map[string]bool{
	".exe":  true,
	".bin":  true,
	".so":   true,
	".o":    true,
	".a":    true,
	".dll":  true,
	".dylib": true,
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".gif":  true,
	".ico":  true,
	".svg":  true,
	".pdf":  true,
	".zip":  true,
	".tar":  true,
	".gz":   true,
	".sum":  true,
}

type Summarizer struct {
	rootDir    string
	outputFile string
	ignoreDirs map[string]bool
	ignoreExts map[string]bool
	maxFileSize int64
	out        *os.File
}

func NewSummarizer(rootDir, outputFile string, extraIgnoreDirs, extraIgnoreExts []string, maxSizeKB int64) *Summarizer {
	ignoreDirs := make(map[string]bool)
	for k, v := range defaultIgnoreDirs {
		ignoreDirs[k] = v
	}
	for _, d := range extraIgnoreDirs {
		ignoreDirs[d] = true
	}

	ignoreExts := make(map[string]bool)
	for k, v := range defaultIgnoreExts {
		ignoreExts[k] = v
	}
	for _, e := range extraIgnoreExts {
		ignoreExts[e] = true
	}

	return &Summarizer{
		rootDir:     rootDir,
		outputFile:  outputFile,
		ignoreDirs:  ignoreDirs,
		ignoreExts:  ignoreExts,
		maxFileSize: maxSizeKB * 1024,
	}
}

func (s *Summarizer) Run() error {
	var err error
	s.out, err = os.Create(s.outputFile)
	if err != nil {
		return fmt.Errorf("cannot create output file: %w", err)
	}
	defer s.out.Close()

	absRoot, err := filepath.Abs(s.rootDir)
	if err != nil {
		return fmt.Errorf("cannot resolve root dir: %w", err)
	}

	s.writeLine("╔══════════════════════════════════════════════════════════════╗")
	s.writeLine("║              PROJECT SUMMARY                                 ║")
	s.writeLine("╚══════════════════════════════════════════════════════════════╝")
	s.writeLine("")
	s.writeLine(fmt.Sprintf("Project root: %s", absRoot))
	s.writeLine("")

	// Section 1: File hierarchy
	s.writeLine("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	s.writeLine("  FILE HIERARCHY")
	s.writeLine("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	s.writeLine("")
	if err := s.writeHierarchy(absRoot, "", true); err != nil {
		return err
	}

	// Section 2: File contents
	s.writeLine("")
	s.writeLine("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	s.writeLine("  FILE CONTENTS")
	s.writeLine("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	s.writeLine("")
	if err := s.writeContents(absRoot, absRoot); err != nil {
		return err
	}

	fmt.Printf("✓ Summary written to: %s\n", s.outputFile)
	return nil
}

func (s *Summarizer) writeHierarchy(dir, prefix string, isRoot bool) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	// Filter entries
	var filtered []os.DirEntry
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			if s.ignoreDirs[name] {
				continue
			}
		} else {
			ext := strings.ToLower(filepath.Ext(name))
			if s.ignoreExts[ext] {
				continue
			}
			// Skip the output file itself
			if filepath.Join(dir, name) == s.outputFile {
				continue
			}
		}
		filtered = append(filtered, e)
	}

	for i, entry := range filtered {
		isLast := i == len(filtered)-1
		connector := "├── "
		childPrefix := prefix + "│   "
		if isLast {
			connector = "└── "
			childPrefix = prefix + "    "
		}

		if isRoot {
			connector = "├── "
			childPrefix = "│   "
			if isLast {
				connector = "└── "
				childPrefix = "    "
			}
		}

		if entry.IsDir() {
			s.writeLine(fmt.Sprintf("%s%s%s/", prefix, connector, entry.Name()))
			if err := s.writeHierarchy(filepath.Join(dir, entry.Name()), childPrefix, false); err != nil {
				return err
			}
		} else {
			info, _ := entry.Info()
			size := ""
			if info != nil {
				size = formatSize(info.Size())
			}
			ext := filepath.Ext(entry.Name())
			if ext == "" {
				ext = "(no ext)"
			}
			s.writeLine(fmt.Sprintf("%s%s%s  [%s, %s]", prefix, connector, entry.Name(), ext, size))
		}
	}
	return nil
}

func (s *Summarizer) writeContents(absRoot, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		name := entry.Name()
		fullPath := filepath.Join(dir, name)

		if entry.IsDir() {
			if s.ignoreDirs[name] {
				continue
			}
			if err := s.writeContents(absRoot, fullPath); err != nil {
				return err
			}
			continue
		}

		ext := strings.ToLower(filepath.Ext(name))
		if s.ignoreExts[ext] {
			continue
		}
		if fullPath == s.outputFile {
			continue
		}

		relPath, _ := filepath.Rel(absRoot, fullPath)

		info, err := os.Stat(fullPath)
		if err != nil {
			continue
		}

		s.writeLine(fmt.Sprintf("┌─ FILE: %s", relPath))
		s.writeLine(fmt.Sprintf("│  Extension : %s", func() string {
			if ext == "" {
				return "(no extension)"
			}
			return ext
		}()))
		s.writeLine(fmt.Sprintf("│  Size      : %s", formatSize(info.Size())))

		if s.maxFileSize > 0 && info.Size() > s.maxFileSize {
			s.writeLine(fmt.Sprintf("│  [SKIPPED: file too large (> %s)]", formatSize(s.maxFileSize)))
			s.writeLine("└" + strings.Repeat("─", 62))
			s.writeLine("")
			continue
		}

		data, err := os.ReadFile(fullPath)
		if err != nil {
			s.writeLine(fmt.Sprintf("│  [ERROR reading file: %v]", err))
			s.writeLine("└" + strings.Repeat("─", 62))
			s.writeLine("")
			continue
		}

		if !utf8.Valid(data) {
			s.writeLine("│  [BINARY FILE — content skipped]")
			s.writeLine("└" + strings.Repeat("─", 62))
			s.writeLine("")
			continue
		}

		s.writeLine("│")
		content := string(data)
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			s.writeLine("│  " + line)
		}
		s.writeLine("│")
		s.writeLine("└" + strings.Repeat("─", 62))
		s.writeLine("")
	}
	return nil
}

func (s *Summarizer) writeLine(line string) {
	fmt.Fprintln(s.out, line)
}

func formatSize(bytes int64) string {
	switch {
	case bytes < 1024:
		return fmt.Sprintf("%d B", bytes)
	case bytes < 1024*1024:
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	default:
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
	}
}

func main() {
	var (
		rootDir     = flag.String("root", ".", "Root directory of the project")
		outputFile  = flag.String("out", "project_summary.txt", "Output file path")
		ignoreDirs  = flag.String("ignore-dirs", "", "Comma-separated extra directories to ignore")
		ignoreExts  = flag.String("ignore-exts", "", "Comma-separated extra extensions to ignore (e.g. .log,.tmp)")
		maxSizeKB   = flag.Int64("max-size", 500, "Max file size to include in KB (0 = unlimited)")
	)
	flag.Parse()

	var extraDirs, extraExts []string
	if *ignoreDirs != "" {
		for _, d := range strings.Split(*ignoreDirs, ",") {
			extraDirs = append(extraDirs, strings.TrimSpace(d))
		}
	}
	if *ignoreExts != "" {
		for _, e := range strings.Split(*ignoreExts, ",") {
			ext := strings.TrimSpace(e)
			if !strings.HasPrefix(ext, ".") {
				ext = "." + ext
			}
			extraExts = append(extraExts, ext)
		}
	}

	s := NewSummarizer(*rootDir, *outputFile, extraDirs, extraExts, *maxSizeKB)
	if err := s.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
