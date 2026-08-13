package agent

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/rezexell/bashtt/internal/domain"
	"golang.org/x/sys/unix"
)

type FanotifyConfig struct {
	WatchDir string
}

type FanotifyWatcher struct {
	cfg FanotifyConfig
}

func NewFanotifyWatcher(
	cfg FanotifyConfig,
) *FanotifyWatcher {
	return &FanotifyWatcher{
		cfg: cfg,
	}
}

func (w *FanotifyWatcher) Watch(
	ctx context.Context,
) (<-chan domain.Event, error) {
	if w.cfg.WatchDir == "" {
		return nil, fmt.Errorf("watch directory is empty")
	}

	info, err := os.Stat(w.cfg.WatchDir)
	if err != nil {
		return nil, fmt.Errorf(
			"stat watch directory: %w",
			err,
		)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf(
			"watch path %q is not a directory",
			w.cfg.WatchDir,
		)
	}

	watchDir, err := filepath.Abs(w.cfg.WatchDir)
	if err != nil {
		return nil, fmt.Errorf(
			"resolve watch directory: %w",
			err,
		)
	}

	fd, err := unix.FanotifyInit(
		unix.FAN_CLASS_NOTIF|unix.FAN_CLOEXEC,
		unix.O_RDONLY|unix.O_LARGEFILE,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"fanotify init: %w",
			err,
		)
	}

	err = unix.FanotifyMark(
		fd,
		unix.FAN_MARK_ADD,
		unix.FAN_OPEN|unix.FAN_OPEN_EXEC,
		unix.AT_FDCWD,
		watchDir,
	)
	if err != nil {
		_ = unix.Close(fd)

		return nil, fmt.Errorf(
			"fanotify mark %q: %w",
			watchDir,
			err,
		)
	}

	events := make(chan domain.Event)

	go func() {
		defer close(events)
		defer unix.Close(fd)

		<-ctx.Done()
	}()

	go w.readEvents(
		ctx,
		fd,
		watchDir,
		events,
	)

	return events, nil
}

func (w *FanotifyWatcher) readEvents(
	ctx context.Context,
	fd int,
	watchDir string,
	events chan<- domain.Event,
) {
	buffer := make([]byte, 4096)

	for {
		select {
		case <-ctx.Done():
			return

		default:
		}

		n, err := unix.Read(fd, buffer)
		if err != nil {
			if err == unix.EINTR {
				continue
			}

			if err == unix.EBADF {
				return
			}

			return
		}

		offset := 0

		for offset < n {
			event := (*unix.FanotifyEventMetadata)(
				unsafe.Pointer(&buffer[offset]),
			)

			if event.Event_len == 0 {
				break
			}

			if event.Vers != unix.FANOTIFY_METADATA_VERSION {
				return
			}

			if event.Mask&unix.FAN_Q_OVERFLOW != 0 {
				offset += int(event.Event_len)
				continue
			}

			if fd >= 0 {
				path := resolveEventPath(
					int(fd),
				)

				if isWatchedScript(
					path,
					watchDir,
				) {
					action := eventAction(event.Mask)

					if action.IsValid() {
						userName := resolveUser(
							event.Pid,
						)

						event := domain.Event{
							User:      userName,
							Script:    path,
							Action:    action,
							CreatedAt: time.Now().UTC(),
						}

						select {
						case events <- event:
						case <-ctx.Done():
							_ = unix.Close(
								int(fd),
							)

							return
						}
					}
				}

				_ = unix.Close(
					int(fd),
				)
			}

			offset += int(event.Event_len)
		}
	}
}

func resolveEventPath(fd int) string {
	path, err := os.Readlink(
		fmt.Sprintf(
			"/proc/self/fd/%d",
			fd,
		),
	)
	if err != nil {
		return ""
	}

	return path
}

func isWatchedScript(
	path string,
	watchDir string,
) bool {
	if path == "" {
		return false
	}

	path = filepath.Clean(path)
	watchDir = filepath.Clean(watchDir)

	prefix := watchDir + string(os.PathSeparator)

	if !strings.HasPrefix(path, prefix) {
		return false
	}

	return strings.HasSuffix(path, ".sh")
}

func eventAction(mask uint64) domain.EventAction {
	if mask&unix.FAN_OPEN_EXEC != 0 {
		return domain.EventExecute
	}

	if mask&unix.FAN_OPEN != 0 {
		return domain.EventOpen
	}

	return ""
}

func resolveUser(pid int32) string {
	if pid <= 0 {
		return ""
	}

	data, err := os.ReadFile(
		fmt.Sprintf(
			"/proc/%d/status",
			pid,
		),
	)
	if err != nil {
		return ""
	}

	uid := parseUID(data)
	if uid < 0 {
		return ""
	}

	u, err := user.LookupId(
		strconv.Itoa(uid),
	)
	if err != nil {
		return strconv.Itoa(uid)
	}

	return u.Username
}

func parseUID(data []byte) int {
	lines := strings.Split(
		string(data),
		"\n",
	)

	for _, line := range lines {
		if !strings.HasPrefix(line, "Uid:") {
			continue
		}

		fields := strings.Fields(line)

		if len(fields) < 2 {
			return -1
		}

		uid, err := strconv.Atoi(fields[1])
		if err != nil {
			return -1
		}

		return uid
	}

	return -1
}
