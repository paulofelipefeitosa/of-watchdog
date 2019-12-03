package executor

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"
)

const TestFilepath = "/tmp/restore.log"

var tailTests = []struct {
	FileContent string
	Expected    string
}{
	{"paulo\n", "paulo\n"},
	{"paulo", "paulo"},
	{"", ""},
	{"teste\ntest\npaulo\n", "paulo\n"},
	{"paulo\ntestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetesteteste", "testetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetestetesteteste"},
	{"(00.530077) pie: 18: 18: Restored\n(00.530078) pie: 13: Restoring scheduler params 0.0.0\n(00.530084) pie: 13: 13: Restored\n(00.530090) pie: 12: Restoring scheduler params 0.0.0\n(00.530095) pie: 12: 12: Restored\n(00.530100) pie: 11: Restoring scheduler params 0.0.0\n(00.530105) pie: 11: 11: Restored\n(00.530269) Unlock network\n(00.532123) Restore finished successfully. Resuming tasks.\n(00.533888) Writing stats\n", "(00.533888) Writing stats\n"},
}

func TestReadLastLine(t *testing.T) {
	for _, test := range tailTests {
		err := populate(test.FileContent)
		if err != nil {
			t.Errorf("Error when trying to write data to test file: %v", err.Error())
		}
		result := tail(TestFilepath)
		if reflect.DeepEqual(result, test.Expected) {
			t.Errorf("Tail is incorrect, got: %v, want: %v.", result, test.Expected)
		}
		err = deleteFile()
		if err != nil {
			t.Errorf("Error when trying to delete test file: %v", err.Error())
		}
	}
}

func TestReverse(t *testing.T) {
	data := []byte{0, 1, 2, 3, 4}
	expected := []byte{0, 4, 3, 2, 1}
	reverse(data, 1, 4)
	if !reflect.DeepEqual(data, expected) {
		t.Errorf("Tail is incorrect, got: %v, want: %v.", data, expected)
	}
}

func TestGetRestoreTime(t *testing.T) {
	test := tailTests[5]
	err := populate(test.FileContent)
	if err != nil {
		t.Errorf("Error when trying to write data to test file: %v", err.Error())
	}
	result := getRestoreTime(TestFilepath)
	expected := int64(533888000)
	if result != expected {
		t.Errorf("Tail is incorrect, got: %v, want: %v.", result, expected)
	}
	err = deleteFile()
	if err != nil {
		t.Errorf("Error when trying to delete test file: %v", err.Error())
	}
}

func deleteFile() error {
	return os.Remove(TestFilepath)
}

func populate(data string) error {
	f, err := os.OpenFile(TestFilepath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return errors.New(fmt.Sprintf("Error: cannot open tmp file for test %q", err.Error()))
	}
	if _, err := f.Write([]byte(data)); err != nil {
		f.Close() // ignore error; Write error takes precedence
		return errors.New(fmt.Sprintf("Error: cannot write data to tmp file used in the test %q", err.Error()))
	}
	if err := f.Close(); err != nil {
		return errors.New(fmt.Sprintf("Error: cannot close tmp file used in the test %q", err.Error()))
	}
	return nil
}
