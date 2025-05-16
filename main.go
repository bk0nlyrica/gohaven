package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	logsDir = "logs"
)

var logBuffer strings.Builder

func writeLog() error {
	now := time.Now()
	logFileName := fmt.Sprintf("%d-%d-%d.log", now.Day(), now.Month(), now.Year())
	logPath := filepath.Join(logsDir, logFileName)

	logBuffer.WriteString(fmt.Sprintf("----------------- %02d:%02d -----------------\n", now.Hour(), now.Minute()))

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}
	defer f.Close()

	if _, err := f.WriteString(logBuffer.String()); err != nil {
		return fmt.Errorf("failed to write to log file: %v", err)
	}

	logBuffer.Reset()
	return nil
}

// detectWindowManager returns the detected window manager ("gnome", "dwm", "i3", or "unknown").
func detectWindowManager() string {
	// Check environment variables
	xdgDesktop := os.Getenv("XDG_CURRENT_DESKTOP")
	desktopSession := os.Getenv("DESKTOP_SESSION")
	if strings.Contains(strings.ToLower(xdgDesktop), "gnome") || strings.Contains(strings.ToLower(desktopSession), "gnome") {
		return "gnome"
	}
	if strings.Contains(strings.ToLower(xdgDesktop), "i3") || strings.Contains(strings.ToLower(desktopSession), "i3") {
		return "i3"
	}
	if strings.Contains(strings.ToLower(xdgDesktop), "dwm") || strings.Contains(strings.ToLower(desktopSession), "dwm") {
		return "dwm"
	}

	// Fallback: Check running processes
	cmd := exec.Command("pgrep", "-l", "dwm")
	output, err := cmd.Output()
	if err == nil && strings.Contains(string(output), "dwm") {
		return "dwm"
	}

	cmd = exec.Command("pgrep", "-l", "i3")
	output, err = cmd.Output()
	if err == nil && strings.Contains(string(output), "i3") {
		return "i3"
	}

	return "unknown"
}

func setWallpaper(wallpaperFileName string, picturesDir string) error {
	wallpaperPath := filepath.Join(picturesDir, wallpaperFileName)
	// Get absolute path
	absPath, err := filepath.Abs(wallpaperPath)
	if err != nil {
		logBuffer.WriteString(fmt.Sprintf("Error getting absolute path: %v\n", err))
		if err := writeLog(); err != nil {
			log.Printf("Failed to write log: %v", err)
		}
		return fmt.Errorf("failed to get absolute path: %v", err)
	}

	wm := detectWindowManager()
	logBuffer.WriteString(fmt.Sprintf("Detected window manager: %s\n", wm))

	switch wm {
	case "gnome":
		// Set wallpaper using gsettings
		command := fmt.Sprintf("gsettings set org.gnome.desktop.background picture-uri file://%s", absPath)
		fmt.Println("running:", command)
		logBuffer.WriteString(fmt.Sprintf("%s\n", command))

		cmd := exec.Command("gsettings", "set", "org.gnome.desktop.background", "picture-uri", fmt.Sprintf("file://%s", absPath))
		stdout, stderr := new(strings.Builder), new(strings.Builder)
		cmd.Stdout = stdout
		cmd.Stderr = stderr

		if err := cmd.Run(); err != nil {
			logBuffer.WriteString(fmt.Sprintf("Error: %v\n", err))
			if err := writeLog(); err != nil {
				log.Printf("Failed to write log: %v", err)
			}
			return fmt.Errorf("failed to set wallpaper: %v", err)
		}

		if stderr.String() != "" {
			logBuffer.WriteString(fmt.Sprintf("stderr: %s\n", stderr.String()))
			if err := writeLog(); err != nil {
				log.Printf("Failed to write log: %v", err)
			}
			return fmt.Errorf("gsettings stderr: %s", stderr.String())
		}

		fmt.Println(stdout.String())
		logBuffer.WriteString(fmt.Sprintf("stdout: %s\n", stdout.String()))
	case "dwm", "i3":
		// Check if feh is installed
		if _, err := exec.LookPath("feh"); err != nil {
			logBuffer.WriteString("Error: feh not found. Please install feh to set wallpapers in DWM or i3.\n")
			if err := writeLog(); err != nil {
				log.Printf("Failed to write log: %v", err)
			}
			return fmt.Errorf("feh not found: %v", err)
		}

		// Set wallpaper using feh
		command := fmt.Sprintf("feh --bg-scale %s", absPath)
		fmt.Println("running:", command)
		logBuffer.WriteString(fmt.Sprintf("%s\n", command))

		cmd := exec.Command("feh", "--bg-scale", absPath)
		stdout, stderr := new(strings.Builder), new(strings.Builder)
		cmd.Stdout = stdout
		cmd.Stderr = stderr

		if err := cmd.Run(); err != nil {
			logBuffer.WriteString(fmt.Sprintf("Error: %v\n", err))
			if err := writeLog(); err != nil {
				log.Printf("Failed to write log: %v", err)
			}
			return fmt.Errorf("failed to set wallpaper with feh: %v", err)
		}

		if stderr.String() != "" {
			logBuffer.WriteString(fmt.Sprintf("stderr: %s\n", stderr.String()))
			if err := writeLog(); err != nil {
				log.Printf("Failed to write log: %v", err)
			}
			return fmt.Errorf("feh stderr: %s", stderr.String())
		}

		fmt.Println(stdout.String())
		logBuffer.WriteString(fmt.Sprintf("stdout: %s\n", stdout.String()))

		// For i3, optionally add to config for persistence
		if wm == "i3" {
			configPath := filepath.Join(os.Getenv("HOME"), ".config", "i3", "config")
			if _, err := os.Stat(configPath); err == nil {
				// Append feh command to i3 config if not already present
				configContent, err := os.ReadFile(configPath)
				if err != nil {
					logBuffer.WriteString(fmt.Sprintf("Warning: could not read i3 config: %v\n", err))
				} else if !strings.Contains(string(configContent), fmt.Sprintf("feh --bg-scale %s", absPath)) {
					f, err := os.OpenFile(configPath, os.O_APPEND|os.O_WRONLY, 0644)
					if err != nil {
						logBuffer.WriteString(fmt.Sprintf("Warning: could not append to i3 config: %v\n", err))
					} else {
						defer f.Close()
						fehLine := fmt.Sprintf("\nexec --no-startup-id feh --bg-scale %s\n", absPath)
						if _, err := f.WriteString(fehLine); err != nil {
							logBuffer.WriteString(fmt.Sprintf("Warning: could not write to i3 config: %v\n", err))
						} else {
							logBuffer.WriteString("Added feh command to i3 config for persistence\n")
						}
					}
				}
			}
		}
	default:
		logBuffer.WriteString("Error: unknown window manager. Supported: gnome, dwm, i3.\n")
		if err := writeLog(); err != nil {
			log.Printf("Failed to write log: %v", err)
		}
		return fmt.Errorf("unknown window manager: %s", wm)
	}

	logBuffer.WriteString("Wallpaper Set\n")
	fmt.Println("Wallpaper Set")

	return writeLog()
}

func downloadWallpaper(url string, picturesDir string) error {
	// Create HTTP client with headers to mimic a browser
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/115.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch wallpaper page: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to parse HTML: %v", err)
	}

	// Try multiple selectors for the image
	var src string
	var exists bool

	// Primary selector (original)
	src, exists = doc.Find(".scrollbox img").Attr("data-cfsrc")
	if !exists {
		// Fallback 1: Check #wallpaper img
		src, exists = doc.Find("#wallpaper").Attr("src")
	}
	if !exists {
		// Fallback 2: Check img#showcase-wallpaper
		src, exists = doc.Find("img#showcase-wallpaper").Attr("src")
	}
	if !exists {
		// Debug: Print all img tags to inspect
		fmt.Println("No image source found. Dumping img tags:")
		logBuffer.WriteString("No image source found. Dumping img tags:\n")
		doc.Find("img").Each(func(i int, s *goquery.Selection) {
			imgSrc, _ := s.Attr("src")
			imgDataSrc, _ := s.Attr("data-cfsrc")
			fmt.Printf("img %d: src=%s, data-cfsrc=%s\n", i, imgSrc, imgDataSrc)
			logBuffer.WriteString(fmt.Sprintf("img %d: src=%s, data-cfsrc=%s\n", i, imgSrc, imgDataSrc))
		})
		return fmt.Errorf("wallpaper image source not found")
	}

	wallpaperID, exists := doc.Find(".scrollbox img").Attr("data-wallpaper-id")
	if !exists {
		wallpaperID = "unknown"
	}

	wallpaperFileName := fmt.Sprintf("%d-%s.png", time.Now().UnixMilli(), wallpaperID)
	filePath := filepath.Join(picturesDir, wallpaperFileName)

	// Download the image
	imgReq, err := http.NewRequest("GET", src, nil)
	if err != nil {
		return fmt.Errorf("failed to create image request: %v", err)
	}
	imgReq.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/115.0")

	imgResp, err := client.Do(imgReq)
	if err != nil {
		return fmt.Errorf("failed to download wallpaper: %v", err)
	}
	defer imgResp.Body.Close()

	if imgResp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected image download status code: %d", imgResp.StatusCode)
	}

	fmt.Println("Starting Download")
	logBuffer.WriteString("Starting Download\n")

	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create image file: %v", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, imgResp.Body); err != nil {
		return fmt.Errorf("failed to save image: %v", err)
	}

	fmt.Println("Download done")
	logBuffer.WriteString("Download done\n")

	return setWallpaper(wallpaperFileName, picturesDir)
}

func fetchRandomWallpaperURL() (string, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://wallhaven.cc/random", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/115.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch random page: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to parse random page HTML: %v", err)
	}

	src, exists := doc.Find(".thumb a").First().Attr("href")
	if !exists {
		return "", fmt.Errorf("wallpaper random URL not found")
	}

	return src, nil
}

func main() {
	fmt.Println("Wallhaven Download Started")

	// Prompt for picture directory path
	fmt.Print("Enter the directory path to store wallpapers (e.g., /home/user/Pictures): ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	picturesDir := strings.TrimSpace(scanner.Text())

	if picturesDir == "" {
		fmt.Println("No directory provided, using default: ./Pictures")
		logBuffer.WriteString("No directory provided, using default: ./Pictures\n")
		picturesDir = "./Pictures"
	}

	// Create logs and pictures directories if they don't exist
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		log.Fatalf("Failed to create logs directory: %v", err)
	}
	if err := os.MkdirAll(picturesDir, 0755); err != nil {
		log.Fatalf("Failed to create pictures directory: %v", err)
	}

	// Log the chosen directory
	fmt.Printf("Using pictures directory: %s\n", picturesDir)
	logBuffer.WriteString(fmt.Sprintf("Using pictures directory: %s\n", picturesDir))

	for {
		// Fetch random wallpaper URL
		src, err := fetchRandomWallpaperURL()
		if err != nil {
			fmt.Printf("Error fetching random wallpaper URL: %v\n", err)
			logBuffer.WriteString(fmt.Sprintf("Error fetching random wallpaper URL: %v\n", err))
			if err := writeLog(); err != nil {
				log.Printf("Failed to write log: %v", err)
			}
			time.Sleep(2 * time.Second) // Wait before retrying
			continue
		}

		fmt.Println(src)
		logBuffer.WriteString(fmt.Sprintf("%s\n", src))

		// Download and set wallpaper
		if err := downloadWallpaper(src, picturesDir); err != nil {
			fmt.Printf("Error downloading wallpaper: %v\n", err)
			logBuffer.WriteString(fmt.Sprintf("Error downloading wallpaper: %v\n", err))
			if err := writeLog(); err != nil {
				log.Printf("Failed to write log: %v", err)
			}
			time.Sleep(2 * time.Second) // Wait before retrying
			continue
		}

		// Prompt user to keep or get next wallpaper
		fmt.Print("Do you like this wallpaper? (y/n): ")
		scanner.Scan()
		response := strings.ToLower(strings.TrimSpace(scanner.Text()))

		if response == "y" {
			fmt.Println("Keeping this wallpaper. Exiting.")
			logBuffer.WriteString("User chose to keep wallpaper. Exiting.\n")
			if err := writeLog(); err != nil {
				log.Printf("Failed to write log: %v", err)
			}
			break
		} else {
			fmt.Println("Fetching next wallpaper...")
			logBuffer.WriteString("User chose next wallpaper.\n")
			if err := writeLog(); err != nil {
				log.Printf("Failed to write log: %v", err)
			}
		}
	}

	// Print current working directory
	wd, err := os.Getwd()
	if err != nil {
		log.Printf("Failed to get working directory: %v", err)
	}
	fmt.Println(wd)
}
