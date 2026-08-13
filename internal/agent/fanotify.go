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

func NewFanotifyWatcher(cfg FanotifyConfig) *FanotifyWatcher {
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
		unix.O_RDONLY|
			unix.O_LARGEFILE|
			unix.O_NONBLOCK,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"fanotify init: %w",
			err,
		)
	}

	mask := uint64(
		unix.FAN_OPEN |
			unix.FAN_OPEN_EXEC |
			unix.FAN_EVENT_ON_CHILD,
	)

	err = unix.FanotifyMark(
		fd,
		unix.FAN_MARK_ADD,
		mask,
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

		w.readEvents(
			ctx,
			fd,
			watchDir,
			events,
		)
	}()

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
			switch err {
			case unix.EAGAIN:
				select {
				case <-ctx.Done():
					return

				case <-time.After(100 * time.Millisecond):
					continue
				}

			case unix.EINTR:
				continue

			case unix.EBADF:
				return

			default:
				fmt.Printf(
					"fanotify read error: %v\n",
					err,
				)

				return
			}
		}

		if n == 0 {
			continue
		}

		offset := 0

		for offset < n {
			event := (*unix.FanotifyEventMetadata)(
				unsafe.Pointer(&buffer[offset]),
			)

			if event.Event_len == 0 {
				break
			}

			if offset+int(event.Event_len) > n {
				break
			}

			if event.Vers != unix.FANOTIFY_METADATA_VERSION {
				fmt.Printf(
					"unsupported fanotify metadata version: %d\n",
					event.Vers,
				)

				return
			}

			if event.Metadata_len <
				uint16(unsafe.Sizeof(unix.FanotifyEventMetadata{})) {
				offset += int(event.Event_len)
				continue
			}

			if event.Mask&unix.FAN_Q_OVERFLOW != 0 {
				offset += int(event.Event_len)
				continue
			}

			if event.Fd >= 0 {
				w.handleEvent(
					ctx,
					event,
					watchDir,
					events,
				)

				_ = unix.Close(int(event.Fd))
			}

			offset += int(event.Event_len)
		}
	}
}

func (w *FanotifyWatcher) handleEvent(
	ctx context.Context,
	event *unix.FanotifyEventMetadata,
	watchDir string,
	events chan<- domain.Event,
) {
	path := resolveEventPath(
		int(event.Fd),
	)

	if path == "" {
		return
	}

	if !isWatchedScript(
		path,
		watchDir,
	) {
		return
	}

	action := eventAction(event.Mask)

	if !action.IsValid() {
		return
	}

	userName := resolveUser(event.Pid)

	domainEvent := domain.Event{
		User:      userName,
		Script:    path,
		Action:    action,
		CreatedAt: time.Now().UTC(),
	}

	select {
	case events <- domainEvent:

	case <-ctx.Done():
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

	if strings.HasSuffix(path, " (deleted)") {
		path = strings.TrimSuffix(
			path,
			" (deleted)",
		)
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
