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
		if clean != f || !filepath.IsAbs(f) || strings.Contains(f, "..") || strings.Contains(f, "\x00") {
			log.Printf("ERROR: Path %s contains invalid characters or traversal attempts", f)
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
					log.Printf("ERROR: Cannot inspect path %s: %v", current, err)
					symlinkFound = true
				}
				break // Rest of the path doesn't exist yet, we can create the file
			}
			if info.Mode()&os.ModeSymlink != 0 {
				log.Printf("ERROR: Symlink detected at %s", current)
				symlinkFound = true
				break
			}
		}

		if symlinkFound {
			continue
		}

		parent := filepath.Dir(hostPath)

		pInfo, err := os.Lstat(parent)
		if err != nil {
			log.Printf("ERROR: Parent directory %s does not exist or inaccessible: %v", parent, err)
			continue
		}
		if !pInfo.IsDir() {
			log.Printf("ERROR: Parent %s is not a directory", parent)
			continue
		}

		fInfo, err := os.Lstat(hostPath)
		if err == nil {
			if fInfo.Mode()&os.ModeSymlink != 0 {
				log.Printf("ERROR: File %s is a symlink", hostPath)
				continue
			}
			if !fInfo.Mode().IsRegular() {
				log.Printf("ERROR: File %s is not a regular file", hostPath)
				continue
			}
			log.Printf("File %s already exists, leaving untouched", hostPath)
			validFiles++
			continue
		}

		if os.IsNotExist(err) {
			dirFD, err := unix.Open(parent, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
			if err != nil {
				log.Printf("ERROR: Failed to open parent directory %s: %v", parent, err)
				continue
			}
			fd, err := unix.Openat(dirFD, filepath.Base(hostPath), unix.O_CREAT|unix.O_EXCL|unix.O_WRONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0644)
			unix.Close(dirFD)
			if err != nil {
				log.Printf("ERROR: Failed to create file %s: %v", hostPath, err)
				continue
			}
			unix.Close(fd)
			log.Printf("Created missing decoy file: %s", hostPath)
			validFiles++
		} else {
			log.Printf("ERROR: Failed to stat file %s: %v", hostPath, err)
			continue
		}
	}

	if validFiles == 0 {
		log.Fatalf("FATAL: No valid decoy files could be provisioned.")
	}
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "init" {
		runInit()
		return
	}

	hw, err := sdk.NewSensor()
	if err != nil {
		log.Fatalf("FATAL: Failed to initialize sensor: %v", err)
	}

	if hw.TestMode {
		if hw.RunTestMode() {
			log.Println("✅ Test mode complete. Exiting gracefully.")
			os.Exit(0)
		}
		log.Println("❌ Test mode failed to contact Hub.")
		os.Exit(1)
	}

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

	mask := uint32(unix.IN_MODIFY | unix.IN_ATTRIB | unix.IN_DELETE_SELF | unix.IN_MOVE_SELF | unix.IN_CLOSE_WRITE)
	if alertOnOpen {
		mask |= unix.IN_OPEN | unix.IN_ACCESS
	}

	log.Println("\n[MODE]")
	log.Println("Tamper Detection: ENABLED")
	if alertOnOpen {
		log.Println("Access Detection: ENABLED")
	} else {
		log.Println("Access Detection: DISABLED")
	}

	log.Println("\n[OK] Monitoring:")

	files := strings.Split(decoyFilesStr, ",")
	for _, f := range files {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}

		targetPath := filepath.Join("/canaries", strings.TrimPrefix(f, "/"))

		_, err := os.Lstat(targetPath)
		if err != nil {
			log.Printf("WARNING: configured canary file missing or inaccessible: %s", targetPath)
			continue
		}

		wd, err := unix.InotifyAddWatch(fd, targetPath, mask)
		if err != nil {
			log.Printf("WARNING: Failed to add watch for %s: %v", targetPath, err)
			continue
		}
		watchMap[int(wd)] = f // Map watch descriptor back to the original host path
		log.Printf("- %s", targetPath)
	}

	if len(watchMap) == 0 {
		log.Fatalf("FATAL: no valid canary files mounted")
	}

	log.Println("\nFile Canary active. Listening for tamper events...")

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
				event := (*unix.InotifyEvent)(unsafe.Pointer(&buf[offset]))

				fileName, ok := watchMap[int(event.Wd)]
				if ok {
					// Independent checks properly handle combined mask events
					if event.Mask&unix.IN_MODIFY != 0 {
						hw.ReportEvent(hw.Severity, "decoy_file_tampered", "Local OS", fileName, map[string]any{"action": "File Modified (IN_MODIFY)", "file_name": fileName, "path": fileName})
					}
					if event.Mask&unix.IN_ATTRIB != 0 {
						hw.ReportEvent(hw.Severity, "decoy_file_tampered", "Local OS", fileName, map[string]any{"action": "Attributes Changed (IN_ATTRIB)", "file_name": fileName, "path": fileName})
					}
					if event.Mask&unix.IN_CLOSE_WRITE != 0 {
						hw.ReportEvent(hw.Severity, "decoy_file_tampered", "Local OS", fileName, map[string]any{"action": "File Written and Closed (IN_CLOSE_WRITE)", "file_name": fileName, "path": fileName})
					}
					if event.Mask&unix.IN_DELETE_SELF != 0 {
						hw.ReportEvent(hw.Severity, "decoy_file_tampered", "Local OS", fileName, map[string]any{"action": "File Deleted (IN_DELETE_SELF)", "file_name": fileName, "path": fileName})
						log.Fatalf("FATAL: Monitored file %s was deleted. Sensor exiting to trigger restart and reprovisioning.", fileName)
					}
					if event.Mask&unix.IN_MOVE_SELF != 0 {
						hw.ReportEvent(hw.Severity, "decoy_file_tampered", "Local OS", fileName, map[string]any{"action": "File Moved (IN_MOVE_SELF)", "file_name": fileName, "path": fileName})
						log.Fatalf("FATAL: Monitored file %s was moved. Sensor exiting to trigger restart and reprovisioning.", fileName)
					}
					if alertOnOpen {
						if event.Mask&unix.IN_OPEN != 0 {
							hw.ReportEvent(hw.Severity, "decoy_file_accessed", "Local OS", fileName, map[string]any{"action": "File Opened (IN_OPEN)", "file_name": fileName, "path": fileName})
						}
						if event.Mask&unix.IN_ACCESS != 0 {
							hw.ReportEvent(hw.Severity, "decoy_file_accessed", "Local OS", fileName, map[string]any{"action": "File Read (IN_ACCESS)", "file_name": fileName, "path": fileName})
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
