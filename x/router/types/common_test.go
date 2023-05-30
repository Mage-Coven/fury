package types_test

import (
	"os"
	"testing"

	"github.com/mage-coven/fury/app"
)

func TestMain(m *testing.M) {
	app.SetSDKConfig()
	os.Exit(m.Run())
}
