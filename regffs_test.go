package regffs

import (
	"io/fs"
	"log"
	"os"
	"strings"
	"testing"
	"testing/fstest"
)

func TestNew(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name     string
		args     args
		sub      string
		expected string
	}{
		{"Test NTUSER.DAT", args{"testdata/NTUSER.DAT"}, "AppEvents/Schemes", "Apps/.Default/SystemNotification"},
		{"Test SAM", args{"testdata/SAM"}, "SAM/Domains/Account", "Users/000001F4"},
		{"Test SOFTWARE", args{"testdata/SOFTWARE"}, "Microsoft/Windows/CurrentVersion/Explorer/Desktop", "NameSpace"},
		{"Test SYSTEM", args{"testdata/SYSTEM"}, "ControlSet001/Control/ComputerName/ComputerName", "ComputerName"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := os.Open(tt.args.name)
			if err != nil {
				log.Fatal(err)
			}
			defer f.Close()

			var fsys fs.FS
			fsys, err = New(f)
			if err != nil {
				t.Error(err)
			}

			if tt.sub != "" {
				fsys, err = fs.Sub(fsys, tt.sub)
			}

			err = fstest.TestFS(fsys, tt.expected)
			if err != nil {
				t.Error(err)
			}

		})
	}
}

func TestRead(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name     string
		args     args
		filename string
		expected string
	}{
		{"Read ComputerName", args{"testdata/SYSTEM"}, "ControlSet001/Control/ComputerName/ComputerName/ComputerName", "WKS-WIN732BITA"},
		{"Read CodePage", args{"testdata/SYSTEM"}, "ControlSet001/Control/Nls/CodePage/ACP", "1252"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := os.Open(tt.args.name)
			if err != nil {
				log.Fatal(err)
			}
			defer f.Close()

			var fsys fs.FS
			fsys, err = New(f)
			if err != nil {
				t.Error(err)
			}

			b, err := fs.ReadFile(fsys, tt.filename)
			if err != nil {
				t.Fatal(err)
			}
			s, _ := DecodeRegSz(b)
			s = strings.TrimRight(s, "\x00")
			if s != tt.expected {
				t.Errorf("Wrong read, got '%x', want '%x'", s, tt.expected)
			}
		})
	}
}
