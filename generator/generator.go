package generator

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed templates/index.html
var indexTemplate string

//go:embed templates/player.html
var playerTemplate string

// Supported video extensions
var videoExtensions = map[string]bool{
	".mp4":  true,
	".webm": true,
	".mkv":  true,
	".avi":  true,
	".mov":  true,
	".m4v":  true,
	".ogv":  true,
	".3gp":  true,
}

// Extensions that need conversion (not supported by browsers)
var needsConversion = map[string]bool{
	".mkv": true,
	".avi": true,
	".mov": true,
	".wmv": true,
	".flv": true,
}

// Video represents a video file found
type Video struct {
	Name         string // Filename without extension
	FileName     string // Full filename
	RelativePath string // Path relative to root
	Extension    string // File extension
	Directory    string // Parent directory (relative to root)
	PlayerPage   string // Player page filename
}

// Directory represents a directory with videos
type Directory struct {
	Name     string   // Directory name
	Path     string   // Relative path
	Videos   []*Video // Videos in this directory
	Children []*Directory
}

// Generator is responsible for generating HTML files
type Generator struct {
	rootDir     string
	outputDir   string
	customTitle string
	videos      []*Video
	dirTree     map[string][]*Video
	indexTmpl   *template.Template
	playerTmpl  *template.Template
}

// IndexData contains data for the index template
type IndexData struct {
	Title       string
	CurrentPath string
	ParentPath  string
	HasParent   bool
	Directories []DirEntry
	Videos      []*Video
}

// DirEntry represents a directory entry in the listing
type DirEntry struct {
	Name string
	Path string
}

// PlayerData contains data for the player template
type PlayerData struct {
	Title     string
	VideoSrc  string
	VideoType string
	BackLink  string
	VideoName string
	PrevVideo string
	NextVideo string
	HasPrev   bool
	HasNext   bool
}

// New creates a new Generator instance
func New(rootDir string) *Generator {
	return &Generator{
		rootDir:     rootDir,
		outputDir:   rootDir,
		customTitle: "Videos",
		videos:      make([]*Video, 0),
		dirTree:     make(map[string][]*Video),
	}
}

// SetTitle sets the custom title for the root page
func (g *Generator) SetTitle(title string) {
	g.customTitle = title
}

// Generate executes the complete HTML file generation
func (g *Generator) Generate() error {
	// Parse templates
	var err error
	g.indexTmpl, err = template.New("index").Parse(indexTemplate)
	if err != nil {
		return fmt.Errorf("error parsing index template: %w", err)
	}

	g.playerTmpl, err = template.New("player").Parse(playerTemplate)
	if err != nil {
		return fmt.Errorf("error parsing player template: %w", err)
	}

	// Scan videos
	if err := g.scanVideos(); err != nil {
		return fmt.Errorf("error scanning videos: %w", err)
	}

	if len(g.videos) == 0 {
		return fmt.Errorf("no videos found in directory '%s'", g.rootDir)
	}

	fmt.Printf("Found %d videos\n", len(g.videos))

	// Generate index pages
	if err := g.generateIndexPages(); err != nil {
		return fmt.Errorf("error generating index pages: %w", err)
	}

	// Generate player pages
	if err := g.generatePlayerPages(); err != nil {
		return fmt.Errorf("error generating player pages: %w", err)
	}

	fmt.Printf("Files generated in: %s\n", g.outputDir)
	return nil
}

// scanVideos scans the directory for videos
func (g *Generator) scanVideos() error {
	return filepath.Walk(g.rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") {
			return filepath.SkipDir
		}

		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if !videoExtensions[ext] {
			return nil
		}

		// If format needs conversion, check if MP4 exists
		if needsConversion[ext] {
			mp4Path := strings.TrimSuffix(path, ext) + ".mp4"
			if _, err := os.Stat(mp4Path); err == nil {
				// MP4 version exists, skip this file (MP4 will be processed later)
				return nil
			}
		}

		relPath, err := filepath.Rel(g.rootDir, path)
		if err != nil {
			return err
		}

		dir := filepath.Dir(relPath)
		if dir == "." {
			dir = ""
		}

		video := &Video{
			Name:         strings.TrimSuffix(info.Name(), ext),
			FileName:     info.Name(),
			RelativePath: relPath,
			Extension:    ext,
			Directory:    dir,
			PlayerPage:   g.generatePlayerFileName(relPath),
		}

		g.videos = append(g.videos, video)
		g.dirTree[dir] = append(g.dirTree[dir], video)

		return nil
	})
}

// generatePlayerFileName generates the HTML filename for the player
func (g *Generator) generatePlayerFileName(relPath string) string {
	// Replace path separators and special characters
	name := strings.ReplaceAll(relPath, string(filepath.Separator), "_")
	name = strings.TrimSuffix(name, filepath.Ext(name))
	return "player_" + sanitizeFileName(name) + ".html"
}

// sanitizeFileName removes special characters from filename
func sanitizeFileName(name string) string {
	var result strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			result.WriteRune(r)
		} else if r == ' ' {
			result.WriteRune('_')
		}
	}
	return result.String()
}

// generateIndexPages generates index pages for each directory
func (g *Generator) generateIndexPages() error {
	// Collect all unique directories
	dirs := make(map[string]bool)
	dirs[""] = true // root
	for _, video := range g.videos {
		dir := video.Directory
		for dir != "" {
			dirs[dir] = true
			dir = filepath.Dir(dir)
			if dir == "." {
				dir = ""
			}
		}
	}

	// Generate index page for each directory
	for dir := range dirs {
		if err := g.generateIndexPage(dir); err != nil {
			return err
		}
	}

	return nil
}

// generateIndexPage generates the index page for a specific directory
func (g *Generator) generateIndexPage(dir string) error {
	// Collect immediate subdirectories
	subDirs := make(map[string]bool)
	for d := range g.dirTree {
		if d == dir {
			continue
		}
		// Check if d is a direct child of dir
		if dir == "" {
			// Root: get first component
			parts := strings.Split(d, string(filepath.Separator))
			if len(parts) > 0 {
				subDirs[parts[0]] = true
			}
		} else if strings.HasPrefix(d, dir+string(filepath.Separator)) {
			// Subdirectory: get next component
			rest := strings.TrimPrefix(d, dir+string(filepath.Separator))
			parts := strings.Split(rest, string(filepath.Separator))
			if len(parts) > 0 {
				subDirs[parts[0]] = true
			}
		}
	}

	// Convert to sorted slice
	var directories []DirEntry
	for subDir := range subDirs {
		path := subDir
		if dir != "" {
			path = dir + string(filepath.Separator) + subDir
		}
		directories = append(directories, DirEntry{
			Name: subDir,
			Path: strings.ReplaceAll(path, string(filepath.Separator), "_") + "_index.html",
		})
	}
	sort.Slice(directories, func(i, j int) bool {
		return directories[i].Name < directories[j].Name
	})

	// Sort videos in current directory
	videos := g.dirTree[dir]
	sort.Slice(videos, func(i, j int) bool {
		return videos[i].Name < videos[j].Name
	})

	// Calculate parent directory link
	var parentPath string
	hasParent := dir != ""
	if hasParent {
		parent := filepath.Dir(dir)
		if parent == "." {
			parentPath = "index.html"
		} else {
			parentPath = strings.ReplaceAll(parent, string(filepath.Separator), "_") + "_index.html"
		}
	}

	// Page title
	title := g.customTitle
	if dir != "" {
		title = filepath.Base(dir)
	}

	data := IndexData{
		Title:       title,
		CurrentPath: dir,
		ParentPath:  parentPath,
		HasParent:   hasParent,
		Directories: directories,
		Videos:      videos,
	}

	var buf bytes.Buffer
	if err := g.indexTmpl.Execute(&buf, data); err != nil {
		return err
	}

	// Determine filename
	fileName := "index.html"
	if dir != "" {
		fileName = strings.ReplaceAll(dir, string(filepath.Separator), "_") + "_index.html"
	}

	outputPath := filepath.Join(g.outputDir, fileName)
	return os.WriteFile(outputPath, buf.Bytes(), 0644)
}

// generatePlayerPages generates player pages for each video
func (g *Generator) generatePlayerPages() error {
	for i, video := range g.videos {
		if err := g.generatePlayerPage(video, i); err != nil {
			return err
		}
	}
	return nil
}

// generatePlayerPage generates the player page for a specific video
func (g *Generator) generatePlayerPage(video *Video, index int) error {
	// Build URL-encoded path preserving directory separators
	// URL encode each path segment individually
	parts := strings.Split(video.RelativePath, string(filepath.Separator))
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	videoSrc := strings.Join(parts, "/")

	// Determine MIME type
	mimeTypes := map[string]string{
		".mp4":  "video/mp4",
		".webm": "video/webm",
		".mkv":  "video/x-matroska",
		".avi":  "video/x-msvideo",
		".mov":  "video/quicktime",
		".m4v":  "video/x-m4v",
		".ogv":  "video/ogg",
		".3gp":  "video/3gpp",
	}

	// Back link
	backLink := "index.html"
	if video.Directory != "" {
		backLink = strings.ReplaceAll(video.Directory, string(filepath.Separator), "_") + "_index.html"
	}

	// Navigation between videos in the same directory
	var prevVideo, nextVideo string
	var hasPrev, hasNext bool

	dirVideos := g.dirTree[video.Directory]
	for j, v := range dirVideos {
		if v.RelativePath == video.RelativePath {
			if j > 0 {
				hasPrev = true
				prevVideo = dirVideos[j-1].PlayerPage
			}
			if j < len(dirVideos)-1 {
				hasNext = true
				nextVideo = dirVideos[j+1].PlayerPage
			}
			break
		}
	}

	data := PlayerData{
		Title:     video.Name,
		VideoSrc:  videoSrc,
		VideoType: mimeTypes[video.Extension],
		BackLink:  backLink,
		VideoName: video.FileName,
		PrevVideo: prevVideo,
		NextVideo: nextVideo,
		HasPrev:   hasPrev,
		HasNext:   hasNext,
	}

	var buf bytes.Buffer
	if err := g.playerTmpl.Execute(&buf, data); err != nil {
		return err
	}

	outputPath := filepath.Join(g.outputDir, video.PlayerPage)
	return os.WriteFile(outputPath, buf.Bytes(), 0644)
}

// generateStylesheet generates the CSS file
func (g *Generator) generateStylesheet() error {
	css := `/* vsite - Stylesheet */
:root {
  --bg-primary: #0f0f0f;
  --bg-secondary: #1a1a1a;
  --bg-tertiary: #252525;
  --text-primary: #ffffff;
  --text-secondary: #a0a0a0;
  --accent: #6366f1;
  --accent-hover: #818cf8;
  --border: #333333;
  --shadow: rgba(0, 0, 0, 0.5);
}

* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  background: var(--bg-primary);
  color: var(--text-primary);
  min-height: 100vh;
  line-height: 1.6;
}

.container {
  max-width: 1400px;
  margin: 0 auto;
  padding: 2rem;
}

/* Header */
.header {
  display: flex;
  align-items: center;
  gap: 1rem;
  margin-bottom: 2rem;
  padding-bottom: 1.5rem;
  border-bottom: 1px solid var(--border);
}

.back-button {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.625rem 1rem;
  background: var(--bg-tertiary);
  color: var(--text-primary);
  text-decoration: none;
  border-radius: 8px;
  font-size: 0.875rem;
  font-weight: 500;
  transition: all 0.2s ease;
}

.back-button:hover {
  background: var(--accent);
}

.header h1 {
  font-size: 1.75rem;
  font-weight: 600;
  background: linear-gradient(135deg, var(--text-primary), var(--accent));
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

/* Grid de Vídeos */
.section-title {
  font-size: 1rem;
  font-weight: 600;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  margin-bottom: 1rem;
}

.directories-grid, .videos-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 1.25rem;
  margin-bottom: 2.5rem;
}

.videos-grid {
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
}

/* Cards de Diretório */
.dir-card {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 1rem 1.25rem;
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: 12px;
  text-decoration: none;
  color: var(--text-primary);
  transition: all 0.2s ease;
}

.dir-card:hover {
  background: var(--bg-tertiary);
  border-color: var(--accent);
  transform: translateY(-2px);
}

.dir-icon {
  font-size: 1.5rem;
}

.dir-name {
  font-weight: 500;
  font-size: 0.9375rem;
}

/* Cards de Vídeo */
.video-card {
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: 12px;
  overflow: hidden;
  text-decoration: none;
  color: var(--text-primary);
  transition: all 0.25s ease;
}

.video-card:hover {
  border-color: var(--accent);
  transform: translateY(-4px);
  box-shadow: 0 12px 40px var(--shadow);
}

.video-thumbnail {
  position: relative;
  aspect-ratio: 16/9;
  background: var(--bg-tertiary);
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}

.video-thumbnail::before {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(135deg, rgba(99, 102, 241, 0.1), transparent);
}

.play-icon {
  position: relative;
  z-index: 1;
  width: 56px;
  height: 56px;
  background: rgba(255, 255, 255, 0.1);
  backdrop-filter: blur(8px);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
}

.video-card:hover .play-icon {
  background: var(--accent);
  transform: scale(1.1);
}

.play-icon svg {
  width: 24px;
  height: 24px;
  fill: white;
  margin-left: 2px;
}

.video-info {
  padding: 1rem;
}

.video-title {
  font-weight: 500;
  font-size: 0.9375rem;
  margin-bottom: 0.25rem;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.video-meta {
  font-size: 0.8125rem;
  color: var(--text-secondary);
  text-transform: uppercase;
}

/* Player Page */
.player-wrapper {
  max-width: 1200px;
  margin: 0 auto;
}

.video-player {
  width: 100%;
  background: #000;
  border-radius: 12px;
  overflow: hidden;
  box-shadow: 0 20px 60px var(--shadow);
}

.video-player video {
  width: 100%;
  display: block;
}

.player-controls {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 1.5rem;
  padding: 1rem;
  background: var(--bg-secondary);
  border-radius: 12px;
}

.nav-buttons {
  display: flex;
  gap: 0.75rem;
}

.nav-button {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.625rem 1.25rem;
  background: var(--bg-tertiary);
  color: var(--text-primary);
  text-decoration: none;
  border-radius: 8px;
  font-size: 0.875rem;
  font-weight: 500;
  transition: all 0.2s ease;
}

.nav-button:hover {
  background: var(--accent);
}

.nav-button.disabled {
  opacity: 0.3;
  pointer-events: none;
}

.video-filename {
  font-size: 0.875rem;
  color: var(--text-secondary);
}

/* Empty State */
.empty-state {
  text-align: center;
  padding: 4rem 2rem;
  color: var(--text-secondary);
}

.empty-state svg {
  width: 64px;
  height: 64px;
  margin-bottom: 1rem;
  opacity: 0.5;
}

/* Responsive */
@media (max-width: 768px) {
  .container {
    padding: 1rem;
  }

  .header {
    flex-direction: column;
    align-items: flex-start;
    gap: 0.75rem;
  }

  .header h1 {
    font-size: 1.5rem;
  }

  .videos-grid {
    grid-template-columns: 1fr;
  }

  .player-controls {
    flex-direction: column;
    gap: 1rem;
  }
}
`
	return os.WriteFile(filepath.Join(g.outputDir, "style.css"), []byte(css), 0644)
}

// Clean removes all generated HTML files
func (g *Generator) Clean() (int, error) {
	count := 0

	// Patterns of generated files
	patterns := []string{
		"index.html",
		"style.css",
		"*_index.html",
		"player_*.html",
	}

	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(g.rootDir, pattern))
		if err != nil {
			return count, err
		}
		for _, match := range matches {
			if err := os.Remove(match); err != nil {
				return count, fmt.Errorf("error removing %s: %w", match, err)
			}
			fmt.Printf("Removed: %s\n", filepath.Base(match))
			count++
		}
	}

	return count, nil
}

// CleanConverted removes MP4 files that were converted from other formats
// (i.e., MP4 files that have a corresponding original file like .avi, .mkv, etc)
func (g *Generator) CleanConverted() (int, error) {
	count := 0

	fmt.Println("Searching for converted files...")

	err := filepath.Walk(g.rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if it's an MP4 file
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".mp4" {
			return nil
		}

		// Check if there's an original file (avi, mkv, mov, etc)
		basePath := strings.TrimSuffix(path, ext)
		originalExtensions := []string{".avi", ".mkv", ".mov", ".wmv", ".flv"}

		for _, origExt := range originalExtensions {
			originalPath := basePath + origExt
			if _, err := os.Stat(originalPath); err == nil {
				// Original exists, this MP4 was converted - remove it
				if err := os.Remove(path); err != nil {
					return fmt.Errorf("error removing %s: %w", path, err)
				}
				fmt.Printf("Removed: %s (original: %s)\n", filepath.Base(path), filepath.Base(originalPath))
				count++
				break
			}
		}

		return nil
	})

	if err != nil {
		return count, err
	}

	return count, nil
}

// CleanOriginal removes original files (avi, mkv, etc) that have been converted to MP4
// (i.e., original files that have a corresponding MP4 file)
func (g *Generator) CleanOriginal() (int, error) {
	count := 0

	fmt.Println("Searching for original files that have been converted...")

	originalExtensions := []string{".avi", ".mkv", ".mov", ".wmv", ".flv"}

	err := filepath.Walk(g.rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))

		// Check if it's an original format
		isOriginal := false
		for _, origExt := range originalExtensions {
			if ext == origExt {
				isOriginal = true
				break
			}
		}

		if !isOriginal {
			return nil
		}

		// Check if there's a corresponding MP4 file
		mp4Path := strings.TrimSuffix(path, ext) + ".mp4"
		if _, err := os.Stat(mp4Path); err == nil {
			// MP4 exists, this original was converted - remove it
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("error removing %s: %w", path, err)
			}
			fmt.Printf("Removed: %s (converted: %s)\n", filepath.Base(path), filepath.Base(mp4Path))
			count++
		}

		return nil
	})

	if err != nil {
		return count, err
	}

	return count, nil
}

// ConvertVideos converts incompatible videos to MP4 using ffmpeg
func (g *Generator) ConvertVideos(useGPU bool) error {
	// Check if ffmpeg is installed
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return fmt.Errorf("ffmpeg not found. Install with:\n  Debian/Ubuntu: sudo apt install ffmpeg\n  Fedora/RHEL:   sudo dnf install ffmpeg")
	}

	// If using GPU, check requirements
	if useGPU {
		if err := g.checkNvidiaGPU(); err != nil {
			return err
		}
		fmt.Println("NVIDIA GPU detected, using NVENC for conversion")
	}

	fmt.Println("Searching for videos to convert...")

	var toConvert []string

	// Scan directory for videos that need conversion
	err := filepath.Walk(g.rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if !needsConversion[ext] {
			return nil
		}

		// Check if MP4 version already exists
		mp4Path := strings.TrimSuffix(path, ext) + ".mp4"
		if _, err := os.Stat(mp4Path); err == nil {
			// MP4 already exists, skip
			return nil
		}

		toConvert = append(toConvert, path)
		return nil
	})

	if err != nil {
		return err
	}

	if len(toConvert) == 0 {
		fmt.Println("No videos need conversion.")
		return nil
	}

	fmt.Printf("Found %d videos to convert\n", len(toConvert))

	for i, videoPath := range toConvert {
		ext := filepath.Ext(videoPath)
		mp4Path := strings.TrimSuffix(videoPath, ext) + ".mp4"
		fileName := filepath.Base(videoPath)

		fmt.Printf("[%d/%d] Converting: %s\n", i+1, len(toConvert), fileName)

		// Build ffmpeg command
		var cmd *exec.Cmd
		if useGPU {
			// Use NVIDIA NVENC
			cmd = exec.Command("ffmpeg",
				"-hwaccel", "cuda",
				"-hwaccel_output_format", "cuda",
				"-i", videoPath,
				"-c:v", "h264_nvenc",
				"-preset", "p4",
				"-cq", "23",
				"-c:a", "aac",
				"-b:a", "128k",
				"-movflags", "+faststart",
				"-y",
				mp4Path,
			)
		} else {
			// Use CPU (libx264)
			cmd = exec.Command("ffmpeg",
				"-i", videoPath,
				"-c:v", "libx264",
				"-preset", "fast",
				"-crf", "22",
				"-c:a", "aac",
				"-b:a", "128k",
				"-movflags", "+faststart",
				"-y",
				mp4Path,
			)
		}

		// Capture stderr to show progress
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			fmt.Printf("  Warning: Error converting %s: %v\n", fileName, err)
			// Remove partial file if exists
			os.Remove(mp4Path)
			continue
		}

		fmt.Printf("  Done: %s\n", filepath.Base(mp4Path))
	}

	fmt.Println("Conversion completed!")
	return nil
}

// checkNvidiaGPU checks if NVIDIA GPU is available and ffmpeg has NVENC support
func (g *Generator) checkNvidiaGPU() error {
	// Check if nvidia-smi is available
	if _, err := exec.LookPath("nvidia-smi"); err != nil {
		return fmt.Errorf("NVIDIA GPU not detected.\n\nRequirements for --gpu:\n  1. NVIDIA driver installed (nvidia-smi must work)\n  2. ffmpeg with NVENC support\n\nDriver installation:\n  Debian/Ubuntu: sudo apt install nvidia-driver-535\n  Fedora/RHEL:   sudo dnf install akmod-nvidia")
	}

	// Check if GPU is working
	cmd := exec.Command("nvidia-smi", "--query-gpu=name", "--format=csv,noheader")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("Error querying NVIDIA GPU: %v\n\nCheck if driver is installed correctly.", err)
	}

	gpuName := strings.TrimSpace(string(output))
	if gpuName == "" {
		return fmt.Errorf("No NVIDIA GPU found.")
	}

	fmt.Printf("GPU detected: %s\n", gpuName)

	// Check if ffmpeg has NVENC support
	cmd = exec.Command("ffmpeg", "-hide_banner", "-encoders")
	output, err = cmd.Output()
	if err != nil {
		return fmt.Errorf("Error checking ffmpeg encoders: %v", err)
	}

	if !strings.Contains(string(output), "h264_nvenc") {
		return fmt.Errorf("ffmpeg does not have NVENC support.\n\nffmpeg needs to be compiled with NVENC support.\n\nInstallation:\n  Debian/Ubuntu: sudo apt install ffmpeg\n  Fedora/RHEL:   sudo dnf install ffmpeg --allowerasing\n\nIf the problem persists, you may need to install ffmpeg\nfrom a repository that includes NVENC support (e.g., RPM Fusion).")
	}

	return nil
}
