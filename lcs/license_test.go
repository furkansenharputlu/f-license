package lcs

import (
	"os"

	"github.com/furkansenharputlu/f-license/config"
	"github.com/stretchr/testify/assert"

	"testing"
)

func TestMain(m *testing.M) {
	config.Global.Load("../sample_config.json")
	SampleProduct()
	os.Exit(m.Run())
}

func TestLicense_ApplyProduct(t *testing.T) {
	license := SampleLicense()

	t.Run("License level keys", func(t *testing.T) {
		_ = license.ApplyProduct()

		assert.Equal(t, config.Key{Raw: TestHMACSecret}, license.Keys.HMACSecret)
	})

	// No matter license level keys set, it will use product level keys.
	t.Run("Product level keys", func(t *testing.T) {
		t.Run("License level keys set", func(t *testing.T) {
			license.SetProductName(TestProductName)
			_ = license.ApplyProduct()

			assert.Equal(t, config.Key{Raw: TestProductHMACSecret}, license.Keys.HMACSecret)
		})

		t.Run("License level keys not set", func(t *testing.T) {
			license.Keys = config.Keys{}
			_ = license.ApplyProduct()

			assert.Equal(t, config.Key{Raw: TestProductHMACSecret}, license.Keys.HMACSecret)
		})
	})

	// Keys and alg should fallback to here.
	t.Run("Default keys", func(t *testing.T) {
		license.Keys = config.Keys{}
		license.SetProductName("")
		_ = license.ApplyProduct()

		assert.Equal(t, config.Key{Raw: TestDefaultHMACSecret}, license.Keys.HMACSecret)
	})
}
