package upload

import "testing"

func TestAsUploader_nil(t *testing.T) {
	if AsUploader(nil) != nil {
		t.Fatal()
	}
}
