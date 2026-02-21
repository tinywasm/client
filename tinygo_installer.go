package client

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	tinyGoVersion = "0.33.0"
	installDirName = ".tinywasm"
)

// EnsureTinyGoInstalled verifies if TinyGo is installed and installs it if missing.
// It returns the path to the tinygo executable.
func EnsureTinyGoInstalled() (string, error) {
	// 1. Check if tinygo is already in PATH
	path, err := exec.LookPath("tinygo")
	if err == nil {
		return path, nil
	}

	// 2. Check local installation directory (fallback for tarball install)
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %v", err)
	}

	localInstallDir := filepath.Join(homeDir, installDirName)
	localBin := filepath.Join(localInstallDir, "tinygo", "bin", "tinygo")
	if runtime.GOOS == "windows" {
		localBin += ".exe"
	}

	if _, err := os.Stat(localBin); err == nil {
		return localBin, nil
	}

	// 3. Not found, attempt installation
	fmt.Println("TinyGo not found. Attempting to install...")

	// Check if we are on Debian/Ubuntu (have dpkg)
	if _, err := exec.LookPath("dpkg"); err == nil && runtime.GOOS == "linux" {
		if err := installDebian(); err != nil {
			return "", fmt.Errorf("debian installation failed: %w", err)
		}
		// After installation, check path again
		path, err := exec.LookPath("tinygo")
		if err == nil {
			return path, nil
		}
		return "", fmt.Errorf("installation seemed successful but 'tinygo' not found in PATH: %v", err)
	}

	// Fallback to tarball installation for other Linux distros or if dpkg fails/isn't preferred
	if err := installTinyGo(localInstallDir); err != nil {
		return "", err
	}

	return localBin, nil
}

func installDebian() error {
	arch := runtime.GOARCH
	var debArch string
	switch arch {
	case "amd64":
		debArch = "amd64"
	case "arm", "arm64":
		// TinyGo releases usually ship amd64 and arm64 .deb
		// For arm (32-bit), they ship armhf.deb.
		// runtime.GOARCH "arm" usually means 32-bit ARM.
		if arch == "arm" {
			debArch = "armhf"
		} else {
			debArch = "arm64"
		}
	default:
		return fmt.Errorf("unsupported architecture for .deb installation: %s", arch)
	}

	// URL format: https://github.com/tinygo-org/tinygo/releases/download/v0.33.0/tinygo_0.33.0_amd64.deb
	downloadURL := fmt.Sprintf("https://github.com/tinygo-org/tinygo/releases/download/v%s/tinygo_%s_%s.deb", tinyGoVersion, tinyGoVersion, debArch)

	fmt.Printf("Downloading TinyGo .deb from %s...\n", downloadURL)

	// Create temp file
	tmpFile, err := os.CreateTemp("", "tinygo-*.deb")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Download
	resp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}
	tmpFile.Close() // Close before using in command

	fmt.Println("Installing with sudo dpkg -i...")
	cmd := exec.Command("sudo", "dpkg", "-i", tmpFile.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin // Allow user input for sudo password

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("dpkg installation failed: %v", err)
	}

	return nil
}

func installTinyGo(destDir string) error {
	var downloadURL string

	switch runtime.GOOS {
	case "linux":
		arch := runtime.GOARCH
		var tarArch string
		switch arch {
		case "amd64":
			tarArch = "amd64"
		case "arm64":
			tarArch = "arm64"
		case "arm":
			tarArch = "armhf" // generic arm usually maps to armhf for tinygo releases
		default:
			return fmt.Errorf("unsupported architecture for automatic installation: %s", arch)
		}

		downloadURL = fmt.Sprintf("https://github.com/tinygo-org/tinygo/releases/download/v%s/tinygo%s.%s-%s.tar.gz", tinyGoVersion, tinyGoVersion, runtime.GOOS, tarArch)
		return installLinux(downloadURL, destDir)
	case "darwin":
		return errors.New("automatic installation for macOS is not yet implemented. Please install TinyGo manually: https://tinygo.org/getting-started/install/macos/")
	case "windows":
		return errors.New("automatic installation for Windows is not yet implemented. Please install TinyGo manually: https://tinygo.org/getting-started/install/windows/")
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

func installLinux(url, destDir string) error {
	fmt.Printf("Downloading TinyGo from %s...\n", url)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download TinyGo: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Ensure destination directory exists
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create installation directory: %v", err)
	}

	fmt.Println("Extracting...")
	return untar(destDir, resp.Body)
}

func untar(dst string, r io.Reader) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()

		switch {
		case err == io.EOF:
			return nil
		case err != nil:
			return err
		case header == nil:
			continue
		}

		// Prevent Zip Slip vulnerability
		target := filepath.Join(dst, header.Name)
		if !strings.HasPrefix(target, filepath.Clean(dst)+string(os.PathSeparator)) {
			return fmt.Errorf("tar: invalid file path %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			}
		case tar.TypeReg:
			// Ensure directory exists for file
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}

			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		}
	}
}
