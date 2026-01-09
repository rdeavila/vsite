package main

import (
	"fmt"
	"os"
	"strings"

	"vsite/generator"
)

const version = "1.5.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	args := os.Args[1:]
	var rootDir string
	var title string
	var cleanMode bool
	var cleanConvertedMode bool
	var convertMode bool
	var useGPU bool

	// Parse arguments
	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch arg {
		case "-h", "--help":
			printUsage()
			os.Exit(0)
		case "-v", "--version":
			fmt.Printf("vsite v%s\n", version)
			os.Exit(0)
		case "-c", "--clean":
			cleanMode = true
		case "--clean-converted":
			cleanConvertedMode = true
		case "--convert":
			convertMode = true
		case "--gpu":
			useGPU = true
		case "-t", "--title":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "Error: --title requires a value.")
				os.Exit(1)
			}
			i++
			title = args[i]
		default:
			if !strings.HasPrefix(arg, "-") {
				rootDir = arg
			} else {
				fmt.Fprintf(os.Stderr, "Error: Unknown option '%s'\n", arg)
				os.Exit(1)
			}
		}
	}

	if rootDir == "" {
		fmt.Fprintln(os.Stderr, "Error: Directory not specified.")
		printUsage()
		os.Exit(1)
	}

	if err := validateDirectory(rootDir); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	gen := generator.New(rootDir)

	if cleanMode {
		count, err := gen.Clean()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error cleaning files: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Done! %d files removed.\n", count)
		os.Exit(0)
	}

	if cleanConvertedMode {
		count, err := gen.CleanConverted()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error cleaning converted files: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Done! %d converted files removed.\n", count)
		os.Exit(0)
	}

	// Convert videos if requested
	if convertMode {
		if err := gen.ConvertVideos(useGPU); err != nil {
			fmt.Fprintf(os.Stderr, "Error converting videos: %v\n", err)
			os.Exit(1)
		}
	}

	// Set custom title
	if title != "" {
		gen.SetTitle(title)
	}

	if err := gen.Generate(); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating HTML: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Done! HTML files generated successfully.")
}

func validateDirectory(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("error: directory '%s' does not exist", dir)
		}
		return fmt.Errorf("error accessing directory: %v", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("error: '%s' is not a directory", dir)
	}
	return nil
}

func printUsage() {
	fmt.Println(`vsite - Static HTML video gallery generator

Usage:
  vsite [options] <directory>
  vsite --clean <directory>
  vsite --clean-converted <directory>

Description:
  Scans the specified directory and subdirectories for video files,
  generating static HTML pages with:
  - Video listing with thumbnails
  - Video player for each file

Arguments:
  <directory>          Root directory containing video files

Options:
  -t, --title <text>   Sets the title of the main page (default: "Videos")
  --convert            Converts incompatible videos (avi, mkv) to MP4
  --gpu                Uses NVIDIA GPU (NVENC) for faster conversion
                       Requires: NVIDIA driver and ffmpeg with NVENC support
  -c, --clean          Removes all generated HTML files from the directory
  --clean-converted    Removes converted MP4 files (keeps original avi, mkv, etc)
  -h, --help           Shows this help
  -v, --version        Shows version

Video formats supported by browsers:
  OK  mp4, webm, ogv   Play natively
  NO  avi, mkv, mov    Require conversion (use --convert)

External dependencies:
  The --convert option requires ffmpeg installed on the system:
    Debian/Ubuntu:  sudo apt install ffmpeg
    Fedora/RHEL:    sudo dnf install ffmpeg

  The --gpu option additionally requires:
    - NVIDIA driver installed (nvidia-smi must work)
    - ffmpeg compiled with NVENC support

Examples:
  vsite /path/to/videos
  vsite --title "My Collection" /path/to/videos
  vsite --convert /path/to/videos
  vsite --convert --gpu /path/to/videos
  vsite --clean /path/to/videos
  vsite --clean-converted /path/to/videos`)
}
