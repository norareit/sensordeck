package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	// stopWait is how long a stop of the background run waits for it to end before the task is ended with schtasks.
	stopWait = 30 * time.Second
	// stopCheck is how often a stop checks whether the background run ended.
	stopCheck = 200 * time.Millisecond
)

// cacheDirectory gives the cache directory of the user, which holds the task file and the log.
func (current *session) cacheDirectory() (string, error) {
	directory, err := current.env.userCacheDir()
	if err != nil {
		return "", fmt.Errorf("cannot find the cache directory: %w", err)
	}
	return directory, nil
}

// schtasks makes one call and gives its standard output. A failure names the call and gives the standard error of the
// call as its cause, or the error of the call when it wrote none.
func (current *session) schtasks(arguments ...string) (string, error) {
	stdout, stderr, err := current.env.schtasks(arguments...)
	if err != nil {
		return "", callFailure("schtasks", arguments, stderr, err)
	}
	return stdout, nil
}

// startupTaskInstall writes the task file, stops a background run that runs, and creates and starts the task.
func (current *session) startupTaskInstall(named string) error {
	cacheDirectory, err := current.cacheDirectory()
	if err != nil {
		return err
	}
	taskFile := taskFilePath(cacheDirectory)
	executable, err := current.executablePath()
	if err != nil {
		return err
	}
	config, err := current.serviceConfigPath(named)
	if err != nil {
		return err
	}
	user, err := current.env.currentUser()
	if err != nil {
		return fmt.Errorf("cannot find the user: %w", err)
	}
	content, err := taskFileContent(user, executable, config)
	if err != nil {
		return err
	}
	if err := current.checkConfig(config); err != nil {
		return err
	}
	if err := current.stopBackgroundRun(); err != nil {
		return err
	}
	if err := writeTaskFile(taskFile, content); err != nil {
		return err
	}
	current.say("wrote " + taskFile)
	for _, arguments := range [][]string{{"/Create", "/TN", taskName, "/XML", taskFile, "/F"}, {"/Run", "/TN", taskName}} {
		if _, err := current.schtasks(arguments...); err != nil {
			return err
		}
	}
	current.say("the startup task " + taskName + " is created and started")
	return nil
}

// writeTaskFile writes the task file, creating its directory and replacing a file that exists.
func writeTaskFile(taskFile string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(taskFile), 0o755); err != nil {
		return fmt.Errorf("cannot write the task file %s: %w", taskFile, innerError(err))
	}
	if err := os.WriteFile(taskFile, content, 0o644); err != nil {
		return fmt.Errorf("cannot write the task file %s: %w", taskFile, innerError(err))
	}
	return nil
}

// startupTaskUninstall stops a background run that runs, deletes the task and removes the task file.
func (current *session) startupTaskUninstall() error {
	cacheDirectory, err := current.cacheDirectory()
	if err != nil {
		return err
	}
	taskFile := taskFilePath(cacheDirectory)
	present, err := installed("task file", taskFile)
	if err != nil {
		return err
	}
	if !present {
		current.say("the startup task " + taskName + " is not installed")
		return nil
	}
	if err := current.stopBackgroundRun(); err != nil {
		return err
	}
	if _, err := current.schtasks("/Delete", "/TN", taskName, "/F"); err != nil {
		return err
	}
	if err := os.Remove(taskFile); err != nil {
		return fmt.Errorf("cannot remove the task file %s: %w", taskFile, innerError(err))
	}
	current.say("the startup task " + taskName + " is stopped and removed")
	return nil
}

// startupTaskStatus writes whether the task is installed, enabled and running, and where the log is.
func (current *session) startupTaskStatus() error {
	cacheDirectory, err := current.cacheDirectory()
	if err != nil {
		return err
	}
	taskFile := taskFilePath(cacheDirectory)
	present, err := installed("task file", taskFile)
	if err != nil {
		return err
	}
	if !present {
		current.say("installed: no")
		return nil
	}
	output, err := current.schtasks("/Query", "/TN", taskName, "/XML")
	if err != nil {
		return err
	}
	for _, line := range taskStatusLines(taskFile, taskEnabled(output), current.backgroundRunState(), logPath(cacheDirectory)) {
		current.say(line)
	}
	return nil
}

// backgroundRunState is running when a background run holds the stop event, stopped when none does, and unknown when
// the check failed.
func (current *session) backgroundRunState() string {
	found, err := current.env.findBackgroundRun()
	switch {
	case err != nil:
		return "unknown"
	case found:
		return "running"
	}
	return "stopped"
}

// stopBackgroundRun signals the stop event of a background run that runs and waits for it to end, and ends the task
// with schtasks when it has not ended within the time.
func (current *session) stopBackgroundRun() error {
	running, err := current.env.findBackgroundRun()
	if err != nil || !running {
		return err
	}
	if err := current.env.signalBackgroundRun(); err != nil {
		return err
	}
	deadline := current.env.now().Add(stopWait)
	for current.env.now().Before(deadline) {
		current.env.wait(context.Background(), stopCheck)
		running, err = current.env.findBackgroundRun()
		if err != nil || !running {
			return err
		}
	}
	if _, err := current.schtasks("/End", "/TN", taskName); err != nil {
		return err
	}
	current.say("sensordeck run did not stop within 30 seconds and was ended with schtasks /End")
	return nil
}
