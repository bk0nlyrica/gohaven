# Wallhaven CLI

A simple Go CLI tool to download random wallpapers from Wallhaven.cc and set them as your desktop background on Linux. Supports GNOME, DWM, and i3.

## Features

- Downloads random wallpapers from Wallhaven.cc.
- Saves to a user-specified directory (defaults to `./Pictures`).
- Sets wallpaper using `gsettings` (GNOME) or `feh`  for Window Managers (DWM/i3/Sway).
- Logs actions to daily log files in `logs/`.

## Prerequisites

- Go: Install from go.dev.
- feh: For DWM/i3 (`sudo apt install feh` on Debian/Ubuntu).
- Go dependency: `go get github.com/PuerkitoBio/goquery`.

## Installation

1. Clone the repo:

   ```bash
   git clone https://github.com/bk0nlyrica/wallhaven.git
   cd wallhaven
   ```
2. Build:

   ```bash
   go build -o wallhaven
   ```

## Usage

Run `./wallhaven`, enter a directory (or press Enter for `./Pictures`), and choose to keep or skip wallpapers.

## License

MIT License.