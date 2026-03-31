package main

import (
	"fmt"
	"os"
	"time"
)

// supervise launches a detached "station _supervise <id>" child process so
// that restart-on-crash survives the short-lived "station run" command exiting.
// Both the Windows-native and WSL2 paths lose the goroutine when the launcher
// exits; a detached child process avoids that entirely.
func supervise(rec *ContainerRecord) int {
	self, err := os.Executable()
	if err != nil {
		supervisorLog(rec.ID, "resolve binary: %v — supervisor not started", err)
		return 0
	}
	pid, err := startDetachedProcess(self, []string{"_supervise", rec.ID})
	if err != nil {
		supervisorLog(rec.ID, "start supervisor daemon: %v", err)
		return 0
	}
	return pid
}

// cmdSuperviseDaemon is the restart loop that runs in the detached child.
// It loads the container record, waits for the PID to die, and restarts
// unless the record was removed (cmdStop) or the policy was cleared.
func cmdSuperviseDaemon(id string) {
	rec, err := loadRecord(id)
	if err != nil {
		supervisorLog(id, "load record: %v — aborting supervisor", err)
		return
	}
	consecutiveFails := 0
	for {
		waitPIDDeath(rec.PID)

		// If cmdStop removed the record or cleared the policy, stop.
		current, err := loadRecord(rec.ID)
		if err != nil || current.RestartPolicy != "always" {
			return
		}

		consecutiveFails++
		if consecutiveFails > 1 {
			// Exponential back-off starting at 2 s for the second consecutive
			// failure; first failure restarts immediately.
			backoff := time.Duration(consecutiveFails-1) * 2 * time.Second
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
			supervisorLog(rec.ID, "restart #%d — backing off %s", consecutiveFails, backoff)
			time.Sleep(backoff)
		} else {
			supervisorLog(rec.ID, "process exited — restarting")
		}

		lf, err := os.OpenFile(logPath(rec.ID), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			supervisorLog(rec.ID, "open log: %v — stopping supervisor", err)
			return
		}
		newPID, err := doSpawn(current, false, lf)
		_ = lf.Close()
		if err != nil {
			supervisorLog(rec.ID, "restart failed: %v — stopping supervisor", err)
			return
		}

		current.PID = newPID
		if err := saveRecord(current); err != nil {
			supervisorLog(rec.ID, "save record: %v", err)
		}
		rec = current
		supervisorLog(rec.ID, "restarted as pid %d", newPID)
		consecutiveFails = 0
	}
}

// waitPIDDeath blocks, polling every second, until pid is no longer alive.
func waitPIDDeath(pid int) {
	for pidAlive(pid) {
		time.Sleep(time.Second)
	}
}

func supervisorLog(id, format string, args ...any) {
	msg := fmt.Sprintf("[supervisor] "+format+"\n", args...)
	fmt.Fprint(os.Stderr, msg)
	// Mirror to the container log so it's visible in 'station logs <id>'.
	if lf, err := os.OpenFile(logPath(id), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		fmt.Fprint(lf, msg)
		lf.Close()
	}
}

