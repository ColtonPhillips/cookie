package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fsnotify/fsnotify"
)

var (
	varPattern     = regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_]*)\s*=\s*\[(.*)\]$`) // Single-line variable pattern
	multilineStart = regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_]*)\s*=\s*\[\[\[`)    // Multi-line start pattern
	subPattern     = regexp.MustCompile(`<\$(\$\$\$)?([a-zA-Z0-9_.]+)>`)             // Variable substitution pattern
)

func main() {
	// Initialize the file watcher
	watcher, err := fsnotify.NewWatcher()
	check(err)
	defer watcher.Close()

	// Watch the current directory for file changes
	err = watcher.Add(".")
	check(err)
	err = watcher.Add("src")
	check(err)

	// Run the precompiler immediately when the program starts
	runPrecompiler()

	// Keep watching for file changes
	for {
		select {
		case event := <-watcher.Events:
			if event.Name == "main.igm" {
				continue
			}

			// If a file is modified, re-run the precompiler
			if event.Op&fsnotify.Write == fsnotify.Write {
				fmt.Println("Detected file change:", event.Name)
				runPrecompiler()
			}
		case err := <-watcher.Errors:
			fmt.Println("Watcher error:", err)
		}
	}
}

// runPrecompiler runs the precompiler logic
func runPrecompiler() {
	cookieFiles := []string{}
	fileContents := map[string]string{}
	variables := map[string]string{}
	var mainFile string

	// Walk through the current directory to find relevant files
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			fmt.Printf("Error walking path: %s\n", path)
			return err
		}

		// If it's a directory, continue walking, if it's a file, process it
		if d.IsDir() {
			return nil
		}

		base := filepath.Base(path)
		if strings.Contains(base, ".cookie.") {
			cookieFiles = append(cookieFiles, path)
		} else if strings.HasSuffix(path, ".css") {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			fileContents[filepath.Base(path)] = string(content)
		}
		return nil
	})
	check(err)

	// Load variables from cookie files
	for _, file := range cookieFiles {
		vars, isMain, err := parseCookieFile(file)
		check(err)
		for k, v := range vars {
			variables[k] = v
		}
		if isMain {
			mainFile = file
		}
	}

	if mainFile == "" {
		panic("No main cookie.x file found!")
	}

	// Compile the main file using loaded variables and file contents
	compiled, err := compileMain(mainFile, variables, fileContents)
	check(err)
	check(os.WriteFile("build/main.igm", []byte(compiled), 0644))
	fmt.Println("Compiled main.igm successfully!")
}

// parseCookieFile processes the cookie files to extract variables
func parseCookieFile(path string) (map[string]string, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, false, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	variables := map[string]string{}
	var currentVar string
	var multiline []string
	isMultiline := false
	lineNumber := 0
	isMain := false

	for scanner.Scan() {
		line := scanner.Text()
		lineNumber++
		if lineNumber == 1 && line == "Let's make a game!" {
			isMain = true
		}
		if isMultiline {
			if line == `]]]` { // End of multi-line variable
				variables[currentVar] = strings.Join(multiline, "\n")
				isMultiline = false
				multiline = nil
				currentVar = ""
			} else {
				multiline = append(multiline, line)
			}
			continue
		}

		// Check for single-line variables
		if match := varPattern.FindStringSubmatch(line); match != nil {
			variables[match[1]] = match[2]
			continue
		}

		// Check for multi-line start
		if match := multilineStart.FindStringSubmatch(line); match != nil {
			isMultiline = true
			currentVar = match[1]
			multiline = []string{}
			continue
		}
	}

	return variables, isMain, scanner.Err()
}

// compileMain processes the main cookie file and resolves variables
func compileMain(path string, vars map[string]string, files map[string]string) (string, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(bytes), "\n")
	for i := range lines {
		lines[i] = resolveVars(lines[i], vars, files, map[string]bool{})
	}
	return strings.Join(lines, "\n"), nil
}

// resolveVars resolves the variables within the string
func resolveVars(s string, vars, files map[string]string, seen map[string]bool) string {
	return subPattern.ReplaceAllStringFunc(s, func(match string) string {
		parts := subPattern.FindStringSubmatch(match)
		key := parts[2]
		_, isFile := files[key]
		if isFile {
			// If it's a file, replace with its contents
			if val, ok := files[key]; ok {
				return val
			}
			return fmt.Sprintf("<missing file: %s>", key)
		}
		// Resolve non-file variables
		if seen[key] {
			return fmt.Sprintf("<circular %s>", key)
		}
		val, ok := vars[key]
		if !ok {
			return fmt.Sprintf("<missing %s>", key)
		}
		seen[key] = true
		defer delete(seen, key)
		return resolveVars(val, vars, files, seen)
	})
}

// check is a utility function to handle errors
func check(err error) {
	if err != nil {
		panic(err)
	}
}
