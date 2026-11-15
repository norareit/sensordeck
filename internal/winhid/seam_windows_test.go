package winhid

import (
	"errors"
	"io/fs"
	"testing"

	"golang.org/x/sys/windows"
)

func TestEveryProcedureOfTheSeamResolves(test *testing.T) {
	for _, procedure := range proceduresOfTheSeam {
		if err := procedure.Find(); err != nil {
			test.Errorf("%s: %v", procedure.Name, err)
		}
	}
}

func TestTheListingOfTheSystemGivesNoErrorAndNoEmptyPath(test *testing.T) {
	collections, err := List(Windows())
	if err != nil {
		test.Fatal(err)
	}
	for _, collection := range collections {
		if collection.Path == "" {
			test.Error("a collection has an empty path")
		}
	}
}

func TestTheCodesOfTheWordingAreThoseOfWindows(test *testing.T) {
	for _, pair := range [][2]error{
		{errorAccessDenied, windows.ERROR_ACCESS_DENIED},
		{errorSharingViolation, windows.ERROR_SHARING_VIOLATION},
		{errorOperationAborted, windows.ERROR_OPERATION_ABORTED},
	} {
		if pair[0] != pair[1] {
			test.Errorf("%v is not %v", pair[0], pair[1])
		}
	}
}

func TestTheOpeningErrorsMapToTheErrorsOfTheStandardLibrary(test *testing.T) {
	for _, testCase := range []struct {
		code error
		want error
	}{
		{windows.ERROR_ACCESS_DENIED, fs.ErrPermission},
		{windows.ERROR_FILE_NOT_FOUND, fs.ErrNotExist},
		{windows.ERROR_PATH_NOT_FOUND, fs.ErrNotExist},
	} {
		if !errors.Is(openError("p", testCase.code), testCase.want) {
			test.Errorf("%v is not %v", testCase.code, testCase.want)
		}
	}
}
