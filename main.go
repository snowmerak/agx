package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	cfg, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if len(os.Args) > 1 {
		subcommand := os.Args[1]
		switch subcommand {
		case "init":
			runInit(cfg)
			return
		case "list":
			runList(cfg)
			return
		case "remove":
			if len(os.Args) < 3 {
				fmt.Fprintln(os.Stderr, "Usage: agx remove <directory_path_or_conversation_id>")
				os.Exit(1)
			}
			runRemove(cfg, os.Args[2])
			return
		case "help", "-h", "--help":
			printUsage()
			return
		default:
			// If not a built-in subcommand, treat the arguments as a prompt
			prompt := strings.Join(os.Args[1:], " ")
			runPrompt(cfg, prompt)
			return
		}
	}

	// No arguments provided: interactive mode (resume only)
	runInteractive(cfg)
}

func printUsage() {
	fmt.Println("agx - A CLI wrapper for agy managing conversations per directory")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  agx                 Start or resume the interactive agy session in the current directory")
	fmt.Println("  agx \"prompt\"        Run a single prompt non-interactively using the current directory's conversation")
	fmt.Println("  agx init            Initialize a new conversation mapping for the current directory")
	fmt.Println("  agx list            List all active directory-to-conversation mapping pairs")
	fmt.Println("  agx remove <query>  Remove a mapping pair by directory path or conversation ID")
	fmt.Println("  agx help            Show this help message")
}

func runList(cfg *Config) {
	if len(cfg.Mappings) == 0 {
		fmt.Println("No active directory-to-conversation mappings found.")
		return
	}
	fmt.Println("Active mappings:")
	for dir, id := range cfg.Mappings {
		fmt.Printf("  %s -> %s\n", dir, id)
	}
}

func runRemove(cfg *Config, query string) {
	// Try matching as canonicalized directory path
	canonicalQuery, err := canonicalizePath(query)
	if err == nil {
		if _, exists := cfg.Mappings[canonicalQuery]; exists {
			delete(cfg.Mappings, canonicalQuery)
			if err := SaveConfig(cfg); err != nil {
				fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Removed mapping for directory: %s\n", canonicalQuery)
			return
		}
	}

	// Try matching as conversation ID or exact directory match
	removed := false
	for dir, id := range cfg.Mappings {
		if id == query || dir == query || strings.ToLower(dir) == strings.ToLower(query) {
			delete(cfg.Mappings, dir)
			removed = true
		}
	}

	if removed {
		if err := SaveConfig(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Removed mapping matching: %s\n", query)
	} else {
		fmt.Printf("No mapping found matching: %s\n", query)
	}
}

func runPrompt(cfg *Config, prompt string) {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting current directory: %v\n", err)
		os.Exit(1)
	}

	canonicalDir, err := canonicalizePath(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error canonicalizing path: %v\n", err)
		os.Exit(1)
	}

	conversationID, exists := cfg.Mappings[canonicalDir]
	if !exists {
		fmt.Fprintln(os.Stderr, "No conversation mapping found for this directory. Please initialize it by running: agx init")
		os.Exit(1)
	}

	transcriptPath, err := getTranscriptPath(conversationID)
	var startLine int
	if err == nil {
		startLine, _ = countTranscriptLines(transcriptPath)
	}

	// Run the single prompt non-interactively using the mapped conversation ID
	agyPath, err := exec.LookPath("agy")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: 'agy' executable not found in PATH.")
		os.Exit(1)
	}

	cmd := exec.Command(agyPath, "--conversation="+conversationID, "-p", prompt)
	cmd.Stdin = os.Stdin
	cmd.Stdout = io.Discard
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running agy: %v\n", err)
		os.Exit(1)
	}

	if err := printNewResponses(transcriptPath, startLine); err != nil {
		_ = printNewResponses(transcriptPath, 0)
	}
}

type TranscriptLine struct {
	StepIndex int    `json:"step_index"`
	Source    string `json:"source"`
	Type      string `json:"type"`
	Content   string `json:"content"`
}

func getTranscriptPath(conversationID string) (string, error) {
	brainDir, err := getBrainDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(brainDir, conversationID, ".system_generated", "logs", "transcript.jsonl"), nil
}

func countTranscriptLines(path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	count := 0
	for scanner.Scan() {
		count++
	}
	return count, scanner.Err()
}

func printNewResponses(path string, startLine int) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// First pass to count total lines in the file
	totalLines := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		totalLines++
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	// Reset file offset
	if _, err := file.Seek(0, 0); err != nil {
		return err
	}

	// If the file has been recreated/truncated and has fewer lines, reset startLine to 0
	if totalLines < startLine {
		startLine = 0
	}

	scanner = bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		if lineNum <= startLine {
			continue
		}
		var line TranscriptLine
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			continue
		}
		if line.Source == "MODEL" && line.Content != "" {
			fmt.Println(line.Content)
		}
	}
	return scanner.Err()
}

func runInteractive(cfg *Config) {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting current working directory: %v\n", err)
		os.Exit(1)
	}

	canonicalDir, err := canonicalizePath(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error canonicalizing path: %v\n", err)
		os.Exit(1)
	}

	conversationID, exists := cfg.Mappings[canonicalDir]
	if !exists {
		fmt.Fprintln(os.Stderr, "No conversation mapping found for this directory. Please initialize it by running: agx init")
		os.Exit(1)
	}

	// Resume existing conversation interactively
	agyPath, err := exec.LookPath("agy")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: 'agy' executable not found in PATH.")
		os.Exit(1)
	}

	fmt.Printf("Resuming conversation: %s\n", conversationID)
	cmd := exec.Command(agyPath, "--conversation="+conversationID)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running agy: %v\n", err)
		os.Exit(1)
	}
}

func runInit(cfg *Config) {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting current working directory: %v\n", err)
		os.Exit(1)
	}

	canonicalDir, err := canonicalizePath(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error canonicalizing path: %v\n", err)
		os.Exit(1)
	}

	if _, exists := cfg.Mappings[canonicalDir]; exists {
		fmt.Println("This directory is already mapped to a conversation ID.")
		fmt.Println("To re-initialize, please remove the mapping first by running: agx remove")
		os.Exit(0)
	}

	initializeConversation(cfg, canonicalDir)
}

func getBrainDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".gemini", "antigravity-cli", "brain"), nil
}

func scanBrainDir(brainDir string) (map[string]os.FileInfo, error) {
	entries := make(map[string]os.FileInfo)
	files, err := os.ReadDir(brainDir)
	if err != nil {
		if os.IsNotExist(err) {
			return entries, nil
		}
		return nil, err
	}
	for _, f := range files {
		if f.IsDir() {
			info, err := f.Info()
			if err == nil {
				entries[f.Name()] = info
			}
		}
	}
	return entries, nil
}

func findNewConversationID(before, after map[string]os.FileInfo) string {
	var newDirs []string
	for name := range after {
		if _, exists := before[name]; !exists {
			newDirs = append(newDirs, name)
		}
	}

	if len(newDirs) == 1 {
		return newDirs[0]
	}

	// Fallback: Return the most recently modified directory in the 'after' set
	var newestName string
	var newestTime time.Time
	for name, info := range after {
		if info.ModTime().After(newestTime) {
			newestTime = info.ModTime()
			newestName = name
		}
	}
	return newestName
}

func initializeConversation(cfg *Config, canonicalDir string) {
	brainDir, err := getBrainDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error determining brain directory: %v\n", err)
		os.Exit(1)
	}

	// Scan brain directory before running agy
	before, err := scanBrainDir(brainDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning brain directory before execution: %v\n", err)
		os.Exit(1)
	}

	agyPath, err := exec.LookPath("agy")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: 'agy' executable not found in PATH.")
		os.Exit(1)
	}

	// Run initial session interactively with the pre-configured system prompt
	cmd := exec.Command(agyPath, "-i", cfg.SystemPrompt)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running agy interactive initialization: %v\n", err)
		os.Exit(1)
	}

	// Scan brain directory after running agy
	after, err := scanBrainDir(brainDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning brain directory after execution: %v\n", err)
		os.Exit(1)
	}

	conversationID := findNewConversationID(before, after)
	if conversationID == "" {
		fmt.Fprintln(os.Stderr, "Could not determine the new conversation ID.")
		os.Exit(1)
	}

	// Record mapping and save
	cfg.Mappings[canonicalDir] = conversationID
	if err := SaveConfig(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving conversation mapping: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nSaved conversation mapping: %s -> %s\n", canonicalDir, conversationID)
}
