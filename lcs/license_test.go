package lcs

import (
	"os"

	"github.com/furkansenharputlu/f-license/config"
	"github.com/stretchr/testify/assert"

	"testing"
)

func TestMain(m *testing.M) {
	config.Global.Load("../sample_config.json")
	SampleApp()
	os.Exit(m.Run())
}

func TestLicense_ApplyApp(t *testing.T) {
	license := SampleLicense()

	t.Run("License level keys", func(t *testing.T) {
		_ = license.ApplyApp()

		assert.Equal(t, config.Key{Raw: TestHMACSecret}, license.Keys.HMACSecret)
	})

	// No matter license level keys set, it will use app level keys.
	t.Run("App level keys", func(t *testing.T) {
		t.Run("License level keys set", func(t *testing.T) {
			license.SetAppName(TestAppName)
			_ = license.ApplyApp()

			assert.Equal(t, config.Key{Raw: TestAppHMACSecret}, license.Keys.HMACSecret)
		})

		t.Run("License level keys not set", func(t *testing.T) {
			license.Keys = config.Keys{}
			_ = license.ApplyApp()

			assert.Equal(t, config.Key{Raw: TestAppHMACSecret}, license.Keys.HMACSecret)
		})
	})

	// Keys and alg should fallback to here.
	t.Run("Default keys", func(t *testing.T) {
		license.Keys = config.Keys{}
		license.SetAppName("")
		_ = license.ApplyApp()

		assert.Equal(t, config.Key{Raw: TestDefaultHMACSecret}, license.Keys.HMACSecret)
	})
}
