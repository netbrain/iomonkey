package iomonkey

import (
	"fmt"
	"github.com/s-urbaniak/uevent"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
)

type AutoMounter struct {
	mutex     sync.RWMutex
	Mounts    map[string]*Mount
	listeners []chan *MountEvent
	quit      chan bool
}

type MountEvent struct {
	Mount *Mount
	Error error
}

type Mount struct {
	Src    string
	Target string
	Fs     string
}

// Creates a new automounter which responsibility is to automatically mount device partitions that is notified through
// the kernels netlink socket
func NewAutoMounter() (*AutoMounter, error) {
	a := &AutoMounter{
		Mounts:    make(map[string]*Mount),
		listeners: make([]chan *MountEvent, 0),
		quit:      make(chan bool, 1),
	}

	r, err := uevent.NewReader()
	if err != nil {
		return nil, fmt.Errorf("Could not create netlink listener, err: %s", err)
	}

	go a.ueventListener(r)

	return a, nil
}

func (a *AutoMounter) ueventListener(r io.ReadCloser) {
	dec := uevent.NewDecoder(r)

	for {
		evt, err := dec.Decode()
		if err != nil {
			a.notify(&MountEvent{Error: err})
			return
		}

		select {
		case <-a.quit:
			return
		default:
			if evt.Action == "add" && evt.Subsystem == "block" && evt.Vars["DEVTYPE"] == "partition" {
				devName := evt.Vars["DEVNAME"]
				src := "/dev/" + devName
				target := "/mnt/" + devName

				mount, err := a.mount(src, target)
				if err == nil {
					a.mutex.Lock()
					a.Mounts[devName] = mount
					a.mutex.Unlock()
				}

				a.notify(&MountEvent{
					Mount: mount,
					Error: err,
				})
			}
		}
	}
}

//mounts a physical device partition
func (a *AutoMounter) mount(src, target string) (*Mount, error) {
	if err := os.MkdirAll(target, 0755); err != nil {
		return nil, fmt.Errorf("Could not create mount directory '%s', err: %s", target, err)
	}

	cmd := exec.Command("bash", "-c", "eval $(blkid "+src+" | awk '{print $4}') && echo -n $TYPE")
	result, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("Could not evaluate filesystem type of device '%s': %s\n", src, err)
	}
	fs := string(result)
	if err := exec.Command("mount", "-t", fs, src, target).Run(); err != nil {
		return nil, fmt.Errorf("Could not mount '%s' to '%s' using fs '%s': %s\n", src, target, fs, err)
	}

	return &Mount{
		Src:    src,
		Target: target,
		Fs:     fs,
	}, nil
}

//unmounts a mounted device
func (a *AutoMounter) unmount(mount *Mount) error {
	if err := exec.Command("umount", mount.Src).Run(); err != nil {
		return err
	}

	return syscall.Rmdir(mount.Target)
}

//notifies mount listeners of a new mount attempt
func (a *AutoMounter) notify(event *MountEvent) {
	for _, listener := range a.listeners {
		listener <- event
	}
}

//Listen creates a listener channel wich outputs mount events
func (a *AutoMounter) Listen() <-chan *MountEvent {
	ch := make(chan *MountEvent)
	a.listeners = append(a.listeners, ch)
	return ch
}

//Close unmounts all mounted directories and removes created mount directories.
func (a *AutoMounter) Close() error {
	a.quit <- true

	defer func() {
		for _, listener := range a.listeners {
			close(listener)
		}
	}()

	a.mutex.RLock()
	defer a.mutex.RUnlock()
	for _, mount := range a.Mounts {
		if err := a.unmount(mount); err != nil {
			return err
		}
	}
	return nil
}
