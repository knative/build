/*
Copyright 2018 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package entrypoint

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"syscall"

	"go.uber.org/zap"
)

const (
	// InternalErrorCode is what we write to the marker file to
	// indicate that we failed to start the wrapped command
	InternalErrorCode = 127

	// DefaultShouldWaitForPrevStep is the default value for whether the
	// command the entrypoint binary will launch should wait for a "finished"
	// signal from another job.  This allows ordering steps
	DefaultShouldWaitForPrevStep = false

	// DefaultShouldRunPostRun is the default value for whether after the
	// command finishes, it should send a "finished" signal that other waiting
	// jobs might be relying on to begin. This allows ordering steps
	DefaultShouldRunPostRun = false

	// DefaultPreRunFile is the name of the file that a
	// waiting job will be waiting to read before it runs.
	DefaultPreRunFile = "0"

	// DefaultPostRunFile is the name of the file that a
	// finishing job will be write after it successfully completes.
	DefaultPostRunFile = "1"
)

// Run executes the test process then writes the exit code to the marker file.
// This function returns the status code that should be passed to os.Exit().
func (o Options) Run(logger *zap.SugaredLogger) int {
	defer logger.Sync()

	code, err := o.ExecuteProcess()
	if err != nil {
		logger.Errorf("error executing user process: %v", err)
	}
	return code
}

// ExecuteProcess creates the artifact directory then executes the process as
// configured, writing the output to the process log.
func (o Options) ExecuteProcess() (int, error) {
	var commandErr error

	// wait for previous step if specified
	if o.ShouldWaitForPrevStep {
		done := make(chan error)
		go func() {
			done <- o.waitForPrevStep()
		}()
	}

	var arguments []string
	if len(o.Args) > 1 {
		arguments = o.Args[1:]
	}
	command := exec.Command(o.Args[0], arguments...)
	if err := command.Start(); err != nil {
		return InternalErrorCode, fmt.Errorf("could not start the process: %v", err)
	}

	// execute the user specified command
	done := make(chan error)
	go func() {
		done <- command.Wait()
	}()
	select {
	case commandErr = <-done:
	}

	var returnCode int
	if status, ok := command.ProcessState.Sys().(syscall.WaitStatus); ok {
		returnCode = status.ExitStatus()
	} else if commandErr == nil {
		returnCode = 0
	} else {
		returnCode = 1
	}

	if returnCode != 0 {
		commandErr = fmt.Errorf("wrapped process failed: %v", commandErr)
	}

	if o.ShouldRunPostRun {
		o.postRunWriteFile(returnCode)
	}

	return returnCode, commandErr
}

func (o *Options) waitForPrevStep() error {
	// wait for a file to exist that the last step wrote in a mounted shared dir
	for {
		// TODO(aaron-prindle) check for non-zero returnCode only
		// as PreRunFile will have returnCode as it's contents?
		_, err := os.Stat(o.PreRunFile)
		if err == nil {
			break
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func (o *Options) postRunWriteFile(exitCode int) error {
	content := []byte(strconv.Itoa(exitCode))

	err := ioutil.WriteFile(o.PostRunFile, content, os.ModePerm)
	if err != nil {
		return fmt.Errorf("error writing PostRunFile (%s): %v", o.PostRunFile, err)
	}

	return nil
}
