package utils

import (
	"os"
	"syscall"
	"jfrog-credential-provider/internal/logger"
)

func GetLock(logs *logger.Logger, file *os.File, lockType int) error {
	logs.Debug("Attempting to acquire lock on file: " + file.Name())
	err := syscall.Flock(int(file.Fd()), lockType)
	if err != nil {
		logs.Error("Failed to acquire lock: " + err.Error())
		return err
	}
	return nil
}

func ReleaseLock(logs *logger.Logger, file *os.File) error {
	logs.Debug("Attempting to release lock on file: " + file.Name())
	err := syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
	if err != nil {
		logs.Error("Failed to release lock: " + err.Error())
		return err
	}
	return nil
}
