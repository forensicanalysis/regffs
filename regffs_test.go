package regffs

import (
	"log"
	"os"
	"testing"
	"testing/fstest"
)

func TestNew(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name string
		args args
	}{
		{"TestFS", args{"testdata/NTUSER.DAT"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := os.Open(tt.args.name)
			if err != nil {
				log.Fatal(err)
			}
			got := New(f)

			err = fstest.TestFS(got, "AppEvents/Schemes/Apps/.Default/SystemNotification")
			if err != nil {
				t.Error(err)
			}
		})
	}
}
