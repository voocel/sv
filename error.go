package main

import "fmt"

type SVError struct {
	Message string
	Type    string
}

func (e *SVError) Error() string {
	return e.Message
}

func NewError(msg string) error {
	return &SVError{Message: msg, Type: "error"}
}

func NewWarning(msg string) error {
	return &SVError{Message: msg, Type: "warning"}
}

func NewInfo(msg string) error {
	return &SVError{Message: msg, Type: "info"}
}

func ErrTagEmpty() error {
	return NewError("tag is empty")
}

func ErrURLEmpty() error {
	return NewError("download URL is empty")
}

func ErrLocalNotExist() error {
	return NewInfo("local version does not exist")
}

func ErrVersionInUse(version string) error {
	return NewWarning(fmt.Sprintf("version %s is in use, please switch to another version before uninstalling", version))
}

func ErrNoVersionsAvailable() error {
	return NewInfo("no available versions locally to select, you can use -r for remote versions")
}

func ErrLatestVersionFailed() error {
	return NewError("get latest version information failed, please try again")
}

func ErrAlreadyLatest(version string) error {
	return NewInfo(fmt.Sprintf("you already have the latest version of SV (%s)", version))
}

func ErrChecksumMismatch() error {
	return NewError("file checksum does not match the computed checksum")
}

func ErrUnsupportedAlgorithm() error {
	return NewError("unsupported checksum algorithm")
}

func ErrUnsupportedCommand() error {
	return NewError("unsupported command")
}

func PrintError(err error) {
	if svErr, ok := err.(*SVError); ok {
		switch svErr.Type {
		case "error":
			PrintRed(svErr.Message)
		case "warning":
			PrintYellow(svErr.Message)
		case "info":
			PrintBlue(svErr.Message)
		default:
			PrintRed(svErr.Message)
		}
	} else {
		PrintRed(err.Error())
	}
}
