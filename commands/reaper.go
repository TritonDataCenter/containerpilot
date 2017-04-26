package commands

import "syscall"

// reapChildren cleans up zombies for a particular process group.
// all our children have unique pgids so we can rely on this to reap
// zombies arising from any children without interfering with other
// children.
func reapChildren(pgid int) {
	go func() {
		for {
			// we need to allow for the possibility that multiple child
			// processes have terminated while one is already being reaped,
			// so we keep trying until there's nothing left
			var wstatus syscall.WaitStatus
			pid, err := syscall.Wait4(-pgid, &wstatus, syscall.WNOHANG, nil)
			switch err {
			case nil:
				if pid > 0 {
					continue
				}
				return
			case syscall.ECHILD:
				return
			case syscall.EINTR:
				continue
			default:
				return
			}
		}
	}()
}
