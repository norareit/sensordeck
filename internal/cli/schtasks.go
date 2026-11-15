package cli

import "os/user"

// runSchtasks runs schtasks.exe with the arguments.
func runSchtasks(arguments ...string) (string, string, error) {
	return runProgram("schtasks.exe", arguments...)
}

// currentUserIdentifier gives the security identifier of the user on Windows, which is the Uid that os/user gives.
func currentUserIdentifier() (string, error) {
	current, err := user.Current()
	if err != nil {
		return "", err
	}
	return current.Uid, nil
}
