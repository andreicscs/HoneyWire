package main

import (
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	sdk "github.com/honeywire/sdk-go"
	"golang.org/x/sys/unix"
)

func runInit() {
	filesStr := os.Getenv("HW_DECOY_FILES")
	if filesStr == "" {
		log.Println("HW_DECOY_FILES is empty, nothing to provision.")
		return
	}

	files := strings.Split(filesStr, ",")
	validFiles := 0

	for _, f := range files {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}

		clean := filepath.Clean(f)

		// Basic path validation
		if clean != f ||
			!filepath.IsAbs(f) ||
			strings.Contains(f, "..") ||
			strings.Contains(f, "\x00") {
			log.Printf("ERROR: Invalid decoy file path: %s", f)
			continue
		}

		hostPath := filepath.Join("/host", strings.TrimPrefix(f, "/"))

		// Verify intermediate path components are not symlinks
		parts := strings.Split(strings.TrimPrefix(f, "/"), "/")
		current := "/host"

		symlinkFound := false

		for _, p := range parts {
			if p == "" {
				continue
			}

			current = filepath.Join(current, p)

			info, err := os.Lstat(current)
			if err != nil {
				if !os.IsNotExist(err) {
					log.Printf("ERROR: Failed inspecting path %s: %v", current, err)
					symlinkFound = true
				}
				break
			}

			if info.Mode()&os.ModeSymlink != 0 {
				log.Printf("ERROR: Symlink detected in path: %s", current)
				symlinkFound = true
				break
			}
		}

		if symlinkFound {
			continue
		}

		parent := filepath.Dir(hostPath)

		parentInfo, err := os.Lstat(parent)
		if err != nil {
			log.Printf("ERROR: Parent directory inaccessible: %s (%v)", parent, err)
			continue
		}

		if !parentInfo.IsDir() {
			log.Printf("ERROR: Parent path is not a directory: %s", parent)
			continue
		}

		// Check existing file
		fileInfo, err := os.Lstat(hostPath)
		if err == nil {

			if fileInfo.Mode()&os.ModeSymlink != 0 {
				log.Printf("ERROR: Decoy file is a symlink: %s", hostPath)
				continue
			}

			if !fileInfo.Mode().IsRegular() {
				log.Printf("ERROR: Decoy path is not a regular file: %s", hostPath)
				continue
			}

			log.Printf("Decoy file already exists: %s", hostPath)
			validFiles++
			continue
		}

		// Create file if missing
		if os.IsNotExist(err) {

			dirFD, err := unix.Open(
				parent,
				unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC,
				0,
			)

			if err != nil {
				log.Printf("ERROR: Failed opening parent directory %s: %v", parent, err)
				continue
			}

			fd, err := unix.Openat(
				dirFD,
				filepath.Base(hostPath),
				unix.O_CREAT|
					unix.O_EXCL|
					unix.O_WRONLY|
					unix.O_NOFOLLOW|
					unix.O_CLOEXEC,
				0644,
			)

			unix.Close(dirFD)

			if err != nil {
				log.Printf("ERROR: Failed creating decoy file %s: %v", hostPath, err)
				continue
			}

			unix.Close(fd)

			log.Printf("Created decoy file: %s", hostPath)
			validFiles++

		} else {
			log.Printf("ERROR: Failed inspecting file %s: %v", hostPath, err)
		}
	}

	if validFiles == 0 {
		log.Fatalf("FATAL: No valid decoy files could be provisioned.")
	}

	log.Printf("Provisioned %d valid decoy file(s)", validFiles)
}

func reportFileEvent(
	hw *sdk.Sensor,
	trigger string,
	category string,
	action string,
	path string,
) {
	hw.ReportEvent(
		trigger,
		"Local OS",
		filepath.Base(path),
		map[string]any{
			"category": category,
			"action":   action,
			"path":     path,
		},
	)

	log.Printf("[%s] %s -> %s", strings.ToUpper(category), action, path)
}

func main() {

	// Init mode
	if len(os.Args) > 1 && os.Args[1] == "init" {
		runInit()
		return
	}

	// Initialize SDK
	hw, err := sdk.NewSensor()
	if err != nil {
		log.Fatalf("FATAL: Failed to initialize sensor: %v", err)
	}

	hw.SetTestPayload(
		"decoy_file_tampered",
		"Wizard Firedrill",
		"Mock Canary File",
		map[string]any{
			"test_message": "Wizard triggered a synthetic event firedrill.",
			"category":     "tamper",
			"action":       "Decoy file modified",
			"path":         "/canaries/mock_passwords.txt",
		},
	)

	// Test mode
	if hw.TestMode {
		if hw.RunTestMode() {
			log.Println("Test mode completed successfully.")
			os.Exit(0)
		}

		log.Println("Test mode failed to contact Hub.")
		os.Exit(1)
	}

	// Start sensor
	if err := hw.Start(); err != nil {
		log.Fatalf("FATAL: Failed to sync with Hub: %v", err)
	}

	defer hw.Stop()

	alertOnOpen := os.Getenv("HW_ALERT_ON_OPEN") == "true"

	decoyFilesStr := os.Getenv("HW_DECOY_FILES")
	if decoyFilesStr == "" {
		log.Println("HW_DECOY_FILES is empty. Nothing to monitor.")
		select {}
	}

	fd, err := unix.InotifyInit()
	if err != nil {
		log.Fatalf("Failed to initialize inotify: %v", err)
	}

	defer unix.Close(fd)

	watchMap := make(map[int]string)

	// Default low-noise tamper detection
	mask := uint32(
		unix.IN_MODIFY |
			unix.IN_ATTRIB |
			unix.IN_DELETE_SELF |
			unix.IN_MOVE_SELF |
			unix.IN_CLOSE_WRITE,
	)

	// Optional access detection
	if alertOnOpen {
		mask |= unix.IN_OPEN | unix.IN_ACCESS
	}

	log.Println("")
	log.Println("=== HoneyWire File Canary ===")
	log.Println("")

	log.Println("[Detection Modes]")
	log.Println("- Tamper Detection: ENABLED")

	if alertOnOpen {
		log.Println("- Access Detection: ENABLED")
	} else {
		log.Println("- Access Detection: DISABLED")
	}

	log.Println("")
	log.Println("[Monitored Files]")

	files := strings.Split(decoyFilesStr, ",")

	for _, f := range files {

		f = strings.TrimSpace(f)

		if f == "" {
			continue
		}

		targetPath := filepath.Join(
			"/canaries",
			strings.TrimPrefix(f, "/"),
		)

		info, err := os.Lstat(targetPath)
		if err != nil {
			log.Printf("WARNING: Canary file missing: %s", targetPath)
			continue
		}

		if !info.Mode().IsRegular() {
			log.Printf("WARNING: Canary target is not a regular file: %s", targetPath)
			continue
		}

		wd, err := unix.InotifyAddWatch(fd, targetPath, mask)
		if err != nil {
			log.Printf("WARNING: Failed monitoring %s: %v", targetPath, err)
			continue
		}

		watchMap[int(wd)] = f

		log.Printf("- %s", f)
	}

	if len(watchMap) == 0 {
		log.Fatalf("FATAL: No valid canary files mounted.")
	}

	log.Println("")
	log.Println("File Canary active. Listening for events...")

	// Graceful shutdown handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {

		var buf [unix.SizeofInotifyEvent * 4096]byte

		for {

			n, err := unix.Read(fd, buf[:])

			if err != nil {

				if err == syscall.EINTR {
					continue
				}

				log.Printf("inotify read error: %v", err)
				return
			}

			var offset uint32

			for offset <= uint32(n-unix.SizeofInotifyEvent) {

				event := (*unix.InotifyEvent)(
					unsafe.Pointer(&buf[offset]),
				)

				fileName, ok := watchMap[int(event.Wd)]

				if ok {

					// Tamper events
					if event.Mask&unix.IN_MODIFY != 0 {
						reportFileEvent(
							hw,
							"decoy_file_tampered",
							"tamper",
							"Decoy file modified",
							fileName,
						)
					}

					if event.Mask&unix.IN_CLOSE_WRITE != 0 {
						reportFileEvent(
							hw,
							"decoy_file_tampered",
							"tamper",
							"Decoy file modified",
							fileName,
						)
					}

					if event.Mask&unix.IN_ATTRIB != 0 {
						reportFileEvent(
							hw,
							"decoy_file_tampered",
							"tamper",
							"File attributes changed",
							fileName,
						)
					}

					if event.Mask&unix.IN_DELETE_SELF != 0 {

						reportFileEvent(
							hw,
							"decoy_file_tampered",
							"tamper",
							"Decoy file deleted",
							fileName,
						)

						log.Fatalf(
							"FATAL: Monitored file deleted: %s",
							fileName,
						)
					}

					if event.Mask&unix.IN_MOVE_SELF != 0 {

						reportFileEvent(
							hw,
							"decoy_file_tampered",
							"tamper",
							"Decoy file moved",
							fileName,
						)

						log.Fatalf(
							"FATAL: Monitored file moved: %s",
							fileName,
						)
					}

					// Optional access events
					if alertOnOpen {

						if event.Mask&unix.IN_OPEN != 0 {
							reportFileEvent(
								hw,
								"decoy_file_accessed",
								"access",
								"Decoy file opened",
								fileName,
							)
						}

						if event.Mask&unix.IN_ACCESS != 0 {
							reportFileEvent(
								hw,
								"decoy_file_accessed",
								"access",
								"Decoy file read",
								fileName,
							)
						}
					}
				}

				offset += unix.SizeofInotifyEvent + event.Len
			}
		}
	}()

	<-sigChan

	log.Println("Shutting down File Canary...")
}