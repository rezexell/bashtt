package agent

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/rezexell/bashtt/internal/domain"
	"golang.org/x/sys/unix"
)

const (
	openDebounce = 100 * time.Millisecond

	executeRememberDuration = 500 * time.Millisecond
)

type FanotifyConfig struct {
	WatchDir string
}

type FanotifyWatcher struct {
	cfg FanotifyConfig

	mu sync.Mutex

	pendingOpen   map[pendingOpen]*time.Timer
	recentExecute map[eventKey]time.Time
}

type eventKey struct {
	pid  int32
	path string
}

type pendingOpen struct {
	pid  int32
	path string
	user string
}

func NewFanotifyWatcher(cfg FanotifyConfig) *FanotifyWatcher {
	return &FanotifyWatcher{
		cfg: cfg,

		pendingOpen: make(
			map[pendingOpen]*time.Timer,
		),

		recentExecute: make(
			map[eventKey]time.Time,
		),
	}
}

func (w *FanotifyWatcher) Watch(
	ctx context.Context,
) (<-chan domain.Event, error) {
	if w.cfg.WatchDir == "" {
		return nil, fmt.Errorf(
			"watch directory is empty",
		)
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

	watchDir, err := filepath.Abs(
		w.cfg.WatchDir,
	)
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

		w.cleanup()
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

		n, err := unix.Read(
			fd,
			buffer,
		)

		if err != nil {
			switch err {
			case unix.EAGAIN:
				select {
				case <-ctx.Done():
					return

				case <-time.After(
					100 * time.Millisecond,
				):
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
			if n-offset < int(
				unsafe.Sizeof(
					unix.FanotifyEventMetadata{},
				),
			) {
				break
			}

			event := (*unix.FanotifyEventMetadata)(
				unsafe.Pointer(
					&buffer[offset],
				),
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
				uint16(
					unsafe.Sizeof(
						unix.FanotifyEventMetadata{},
					),
				) {
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

				_ = unix.Close(
					int(event.Fd),
				)
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

	fmt.Printf(
		"fanotify: pid=%d mask=%#x path=%s\n",
		event.Pid,
		event.Mask,
		path,
	)

	userName := resolveUser(event.Pid)

	if userName == "" {
		userName = resolveAgentUser()
	}

	if event.Mask&unix.FAN_OPEN_EXEC != 0 {
		w.rememberExecute(
			event.Pid,
			path,
		)

		w.cancelPendingOpen(
			event.Pid,
			path,
		)

		w.sendEvent(
			ctx,
			events,
			userName,
			path,
			domain.EventExecute,
		)

		return
	}

	if event.Mask&unix.FAN_OPEN != 0 {
		if w.wasRecentlyExecuted(
			event.Pid,
			path,
		) {
			return
		}

		key := pendingOpen{
			pid:  event.Pid,
			path: path,
			user: userName,
		}

		w.scheduleOpen(
			ctx,
			events,
			key,
		)
	}
}

func (w *FanotifyWatcher) rememberExecute(
	pid int32,
	path string,
) {
	w.mu.Lock()

	key := eventKey{
		pid:  pid,
		path: path,
	}

	w.recentExecute[key] = time.Now()

	w.mu.Unlock()

	time.AfterFunc(
		executeRememberDuration,
		func() {
			w.mu.Lock()
			defer w.mu.Unlock()

			t, ok := w.recentExecute[key]
			if !ok {
				return
			}

			if time.Since(t) >=
				executeRememberDuration {
				delete(
					w.recentExecute,
					key,
				)
			}
		},
	)
}

func (w *FanotifyWatcher) wasRecentlyExecuted(
	pid int32,
	path string,
) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	key := eventKey{
		pid:  pid,
		path: path,
	}

	t, ok := w.recentExecute[key]
	if !ok {
		return false
	}

	if time.Since(t) >
		executeRememberDuration {
		delete(
			w.recentExecute,
			key,
		)

		return false
	}

	return true
}

func (w *FanotifyWatcher) scheduleOpen(
	ctx context.Context,
	events chan<- domain.Event,
	key pendingOpen,
) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for existing := range w.pendingOpen {
		if existing.pid == key.pid &&
			existing.path == key.path {
			return
		}
	}

	timer := time.AfterFunc(
		openDebounce,
		func() {
			w.firePendingOpen(
				ctx,
				events,
				key,
			)
		},
	)

	w.pendingOpen[key] = timer
}

func (w *FanotifyWatcher) firePendingOpen(
	ctx context.Context,
	events chan<- domain.Event,
	key pendingOpen,
) {
	w.mu.Lock()

	var found bool

	for existing, timer := range w.pendingOpen {
		if existing.pid != key.pid ||
			existing.path != key.path {
			continue
		}

		delete(
			w.pendingOpen,
			existing,
		)

		timer.Stop()

		found = true

		break
	}

	w.mu.Unlock()

	if !found {
		return
	}
	if w.wasRecentlyExecuted(
		key.pid,
		key.path,
	) {
		return
	}

	w.sendEvent(
		ctx,
		events,
		key.user,
		key.path,
		domain.EventOpen,
	)
}

func (w *FanotifyWatcher) cancelPendingOpen(
	pid int32,
	path string,
) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for existing, timer := range w.pendingOpen {
		if existing.pid != pid ||
			existing.path != path {
			continue
		}

		delete(
			w.pendingOpen,
			existing,
		)

		timer.Stop()
	}
}

func (w *FanotifyWatcher) cleanup() {
	w.mu.Lock()
	defer w.mu.Unlock()

	for key, timer := range w.pendingOpen {
		timer.Stop()

		delete(
			w.pendingOpen,
			key,
		)
	}

	clear(w.recentExecute)
}

func (w *FanotifyWatcher) sendEvent(
	ctx context.Context,
	events chan<- domain.Event,
	userName string,
	path string,
	action domain.EventAction,
) {
	if !action.IsValid() {
		return
	}

	event := domain.Event{
		User:      userName,
		Script:    path,
		Action:    action,
		CreatedAt: time.Now().UTC(),
	}

	select {
	case events <- event:

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

	if strings.HasSuffix(
		path,
		" (deleted)",
	) {
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

	prefix := watchDir +
		string(os.PathSeparator)

	if !strings.HasPrefix(
		path,
		prefix,
	) {
		return false
	}

	return strings.HasSuffix(
		path,
		".sh",
	)
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
		if !strings.HasPrefix(
			line,
			"Uid:",
		) {
			continue
		}

		fields := strings.Fields(line)

		if len(fields) < 2 {
			return -1
		}

		uid, err := strconv.Atoi(
			fields[1],
		)
		if err != nil {
			return -1
		}

		return uid
	}

	return -1
}

func resolveAgentUser() string {
	u, err := user.Current()
	if err != nil {
		return ""
	}

	return u.Username
}
