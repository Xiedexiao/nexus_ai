package common

import (
	"bytes"
	"image/png"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSliderCaptchaProducesTwoImages(t *testing.T) {
	bg, piece, x, _, id, err := NewSliderCaptcha()
	require.NoError(t, err)
	require.NotEmpty(t, id)
	assert.NotZero(t, x)

	bgImg, err := png.Decode(bytes.NewReader(bg))
	require.NoError(t, err)
	assert.Equal(t, SliderCaptchaWidth, bgImg.Bounds().Dx())
	assert.Equal(t, SliderCaptchaHeight, bgImg.Bounds().Dy())

	pieceImg, err := png.Decode(bytes.NewReader(piece))
	require.NoError(t, err)
	assert.Equal(t, SliderCaptchaPiece, pieceImg.Bounds().Dx())

	// The hole must actually be punched: a darkened patch exists near x.
	found := false
	for dy := 20; dy < 60 && !found; dy++ {
		for dx := 0; dx < SliderCaptchaPiece; dx++ {
			r, g, _, _ := bgImg.At(x+dx, dy).RGBA()
			if r>>8 < 120 && g>>8 < 120 {
				found = true
				break
			}
		}
	}
	assert.True(t, found, "background must contain a darkened puzzle hole")
}

func TestSliderCaptchaVerifyLifecycle(t *testing.T) {
	_, _, x, _, id, err := NewSliderCaptcha()
	require.NoError(t, err)

	t.Run("exact answer passes", func(t *testing.T) {
		assert.True(t, VerifySliderCaptcha(id, x, 500))
	})

	t.Run("challenge is single-use", func(t *testing.T) {
		assert.False(t, VerifySliderCaptcha(id, x, 500), "replaying the same id must fail")
	})
}

func TestSliderCaptchaVerifyBounds(t *testing.T) {
	t.Run("within tolerance passes", func(t *testing.T) {
		_, _, gx, _, gid, gerr := NewSliderCaptcha()
		require.NoError(t, gerr)
		assert.True(t, VerifySliderCaptcha(gid, gx-SliderCaptchaTolerancePx, 500))
	})
}

func TestSliderCaptchaRejectsInstantDrag(t *testing.T) {
	_, _, x, _, id, err := NewSliderCaptcha()
	require.NoError(t, err)
	assert.False(t, VerifySliderCaptcha(id, x, 10), "a near-instant drag is a bot signal")
}

func TestSliderCaptchaRejectsWrongX(t *testing.T) {
	_, _, x, _, id, err := NewSliderCaptcha()
	require.NoError(t, err)
	assert.False(t, VerifySliderCaptcha(id, x+SliderCaptchaTolerancePx+15, 500))
}

func TestSliderCaptchaUnknownIdFails(t *testing.T) {
	assert.False(t, VerifySliderCaptcha("no-such-id", 100, 500))
}

func TestSliderCaptchaImagesDifferEachTime(t *testing.T) {
	bg1, _, _, _, _, err := NewSliderCaptcha()
	require.NoError(t, err)
	bg2, _, _, _, _, err := NewSliderCaptcha()
	require.NoError(t, err)
	assert.False(t, bytes.Equal(bg1, bg2), "each challenge must draw a unique background")
}

func TestSliderCaptchaConcurrentGeneration(t *testing.T) {
	var wg sync.WaitGroup
	errs := make(chan error, 16)
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, _, _, _, err := NewSliderCaptcha()
			if err != nil {
				errs <- err
				return
			}
			_, _, _, _, id2, err := NewSliderCaptcha()
			if err == nil {
				VerifySliderCaptcha(id2, 0, 1000)
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}
