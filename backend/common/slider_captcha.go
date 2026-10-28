package common

import (
	"crypto/rand"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"math"
	mrand "math/rand"
	"sync"
	"time"
)

// Slider captcha: the backend draws a procedural background with a punched
// puzzle-hole and a matching puzzle piece, the user drags the piece to fill
// the hole, and the answer is the hole's X coordinate. Everything is generated
// in-process with x/image primitives — no external images, no copyright, and
// every challenge looks different.

const (
	SliderCaptchaWidth       = 320
	SliderCaptchaHeight      = 160
	SliderCaptchaPiece       = 44
	SliderCaptchaMinX        = SliderCaptchaPiece + 20
	SliderCaptchaMaxX        = SliderCaptchaWidth - SliderCaptchaPiece - 10
	SliderCaptchaTolerancePx = 5
	// SliderCaptchaMinDurationMs rejects instant drags; a human takes at
	// least a couple hundred milliseconds to see the target and move.
	SliderCaptchaMinDurationMs = 180
	SliderCaptchaTTLSecs       = 180
	sliderCaptchaStoreMax      = 10000
)

type sliderChallenge struct {
	x         int
	createdAt time.Time
}

var (
	sliderCaptchaMu     sync.Mutex
	sliderCaptchaStore  = make(map[string]sliderChallenge)
	sliderCaptchaRNG    = mrand.New(mrand.NewSource(time.Now().UnixNano()))
	sliderCaptchaRNGMut sync.Mutex
)

func sliderRandomSeed() int64 {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return int64(binary.LittleEndian.Uint64(b[:]))
}

func sliderRand(n *mrand.Rand, min, max int) int {
	if max <= min {
		return min
	}
	return min + n.Intn(max-min+1)
}

// NewSliderCaptcha draws one challenge and returns the base image, the puzzle
// piece cut from it, the hole X (server-side secret), the hole Y (safe to
// publish — only X is the answer), and the challenge id.
func NewSliderCaptcha() (bg []byte, piece []byte, x, y int, id string, err error) {
	sliderCaptchaRNGMut.Lock()
	n := sliderCaptchaRNG
	x = sliderRand(n, SliderCaptchaMinX, SliderCaptchaMaxX)
	y = sliderRand(n, 24, SliderCaptchaHeight-SliderCaptchaPiece-24)
	hue := sliderRand(n, 0, 359)
	style := sliderRand(n, 0, 2)
	sliderCaptchaRNGMut.Unlock()

	img := drawSliderBackground(n, hue, style)
	pieceImg := cutSliderPiece(img, x, y)
	punchSliderHole(img, x, y)

	if id == "" {
		id = GetRandomString(32)
	}
	if bg, err = pngEncode(img); err != nil {
		return nil, nil, 0, 0, "", err
	}
	if piece, err = pngEncode(pieceImg); err != nil {
		return nil, nil, 0, 0, "", err
	}

	sliderCaptchaMu.Lock()
	if len(sliderCaptchaStore) >= sliderCaptchaStoreMax {
		sliderCaptchaSweepLocked(time.Now())
	}
	sliderCaptchaStore[id] = sliderChallenge{x: x, createdAt: time.Now()}
	sliderCaptchaMu.Unlock()
	return bg, piece, x, y, id, nil
}

// VerifySliderCaptcha consumes the challenge once: correct X within tolerance,
// drag duration above the human floor, and idempotent on replay.
func VerifySliderCaptcha(id string, x int, durationMs int64) bool {
	sliderCaptchaMu.Lock()
	ch, ok := sliderCaptchaStore[id]
	if ok {
		delete(sliderCaptchaStore, id)
	}
	sliderCaptchaMu.Unlock()
	if !ok {
		return false
	}
	if time.Since(ch.createdAt) > SliderCaptchaTTLSecs*time.Second {
		return false
	}
	if durationMs < SliderCaptchaMinDurationMs {
		return false
	}
	dx := ch.x - x
	if dx < 0 {
		dx = -dx
	}
	return dx <= SliderCaptchaTolerancePx
}

// sliderCaptchaSweepLocked drops expired challenges; caller holds the lock.
func sliderCaptchaSweepLocked(now time.Time) {
	cutoff := now.Add(-SliderCaptchaTTLSecs * time.Second)
	for id, ch := range sliderCaptchaStore {
		if ch.createdAt.Before(cutoff) {
			delete(sliderCaptchaStore, id)
		}
	}
}

// CleanupSliderCaptcha is called periodically from housekeeping.
func CleanupSliderCaptcha() {
	sliderCaptchaMu.Lock()
	defer sliderCaptchaMu.Unlock()
	sliderCaptchaSweepLocked(time.Now())
}

func pngEncode(img image.Image) ([]byte, error) {
	var buf binaryWriter
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.b, nil
}

type binaryWriter struct{ b []byte }

func (w *binaryWriter) Write(p []byte) (int, error) {
	w.b = append(w.b, p...)
	return len(p), nil
}

// drawSliderBackground paints a deterministic procedural scene: vertical
// gradient plus translucent circles and triangles in a hue family. Shapes make
// the hole position non-guessable from the texture alone.
func drawSliderBackground(n *mrand.Rand, hue int, style int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, SliderCaptchaWidth, SliderCaptchaHeight))
	baseR, baseG, baseB := hslToRGB(float64(hue), 0.42, 0.62)
	for y := 0; y < SliderCaptchaHeight; y++ {
		t := float64(y) / float64(SliderCaptchaHeight)
		l := 0.62 + 0.22*t
		r, g, b := hslToRGB(float64(hue), 0.42, l)
		for x := 0; x < SliderCaptchaWidth; x++ {
			img.Set(x, y, color.RGBA{uint8(r), uint8(g), uint8(b), 255})
		}
		_ = baseR
		_ = baseG
		_ = baseB
	}
	// Decorative shapes in darker/lighter variants of the same hue.
	shapes := sliderRand(n, 6, 10)
	for i := 0; i < shapes; i++ {
		cx := sliderRand(n, 10, SliderCaptchaWidth-10)
		cy := sliderRand(n, 10, SliderCaptchaHeight-10)
		rad := sliderRand(n, 14, 46)
		l := 0.35 + float64(sliderRand(n, 0, 4))/10
		r, g, b := hslToRGB(float64((hue+sliderRand(n, -30, 30)+360)%360), 0.45, l)
		c := color.RGBA{uint8(r), uint8(g), uint8(b), 255}
		switch style {
		case 0:
			fillCircle(img, cx, cy, rad, c)
		case 1:
			fillTriangle(img, cx, cy, rad, c)
		default:
			if i%2 == 0 {
				fillCircle(img, cx, cy, rad, c)
			} else {
				fillRect(img, cx-rad, cy-rad/2, rad*2, rad, c)
			}
		}
	}
	// Grain noise so the background cannot be template-matched from a cached
	// copy of a previous challenge.
	for i := 0; i < SliderCaptchaWidth*SliderCaptchaHeight/18; i++ {
		x, y := sliderRand(n, 0, SliderCaptchaWidth-1), sliderRand(n, 0, SliderCaptchaHeight-1)
		v := uint8(sliderRand(n, 0, 255))
		c := img.RGBAAt(x, y)
		img.SetRGBA(x, y, color.RGBA{
			R: uint8((int(c.R) + int(v)) / 2), G: uint8((int(c.G) + int(v)) / 2),
			B: uint8((int(c.B) + int(v)) / 2), A: 255,
		})
	}
	return img
}

// puzzleMask reports whether (dx,dy) inside the piece's local box belongs to
// the classic puzzle-knob shape: a square with one round knob on the right.
func puzzleMask(dx, dy, size int) bool {
	if dx < 0 || dy < 0 || dx >= size || dy >= size {
		return false
	}
	if dx < size*7/10 {
		return true
	}
	// knob: circle centered at (size*7/10, size/2), radius size*3/10
	ccx, ccy, rad := float64(size)*0.7, float64(size)/2, float64(size)*0.3
	return (float64(dx)-ccx)*(float64(dx)-ccx)+(float64(dy)-ccy)*(float64(dy)-ccy) <= rad*rad
}

func cutSliderPiece(img *image.RGBA, x, y int) *image.RGBA {
	piece := image.NewRGBA(image.Rect(0, 0, SliderCaptchaPiece, SliderCaptchaPiece))
	for dy := 0; dy < SliderCaptchaPiece; dy++ {
		for dx := 0; dx < SliderCaptchaPiece; dx++ {
			if puzzleMask(dx, dy, SliderCaptchaPiece) {
				piece.Set(dx, dy, img.At(x+dx, y+dy))
			}
		}
	}
	// Outline for visibility against any background.
	for dy := 0; dy < SliderCaptchaPiece; dy++ {
		for dx := 0; dx < SliderCaptchaPiece; dx++ {
			if !puzzleMask(dx, dy, SliderCaptchaPiece) {
				continue
			}
			edge := !puzzleMask(dx-1, dy, SliderCaptchaPiece) || !puzzleMask(dx+1, dy, SliderCaptchaPiece) ||
				!puzzleMask(dx, dy-1, SliderCaptchaPiece) || !puzzleMask(dx, dy+1, SliderCaptchaPiece)
			if edge {
				piece.Set(dx, dy, color.RGBA{255, 255, 255, 230})
			}
		}
	}
	return piece
}

func punchSliderHole(img *image.RGBA, x, y int) {
	for dy := 0; dy < SliderCaptchaPiece; dy++ {
		for dx := 0; dx < SliderCaptchaPiece; dx++ {
			if puzzleMask(dx, dy, SliderCaptchaPiece) {
				r, g, b, _ := img.At(x+dx, y+dy).RGBA()
				img.Set(x+dx, y+dy, color.RGBA{
					R: uint8(r >> 8 * 45 / 100), G: uint8(g >> 8 * 45 / 100),
					B: uint8(b >> 8 * 45 / 100), A: 255,
				})
			}
		}
	}
}

func fillCircle(img *image.RGBA, cx, cy, rad int, c color.RGBA) {
	for dy := -rad; dy <= rad; dy++ {
		for dx := -rad; dx <= rad; dx++ {
			if dx*dx+dy*dy <= rad*rad {
				blendPixel(img, cx+dx, cy+dy, c)
			}
		}
	}
}

func fillTriangle(img *image.RGBA, cx, cy, size int, c color.RGBA) {
	for dy := 0; dy <= size; dy++ {
		half := dy * size / size
		half = int(float64(dy) * math.Tan(math.Pi/6))
		for dx := -half; dx <= half; dx++ {
			blendPixel(img, cx+dx, cy-size/2+dy, c)
		}
	}
}

func fillRect(img *image.RGBA, x, y, w, h int, c color.RGBA) {
	for dy := 0; dy < h; dy++ {
		for dx := 0; dx < w; dx++ {
			blendPixel(img, x+dx, y+dy, c)
		}
	}
}

func blendPixel(img *image.RGBA, x, y int, c color.RGBA) {
	if x < 0 || y < 0 || x >= img.Bounds().Dx() || y >= img.Bounds().Dy() {
		return
	}
	old := img.RGBAAt(x, y)
	img.SetRGBA(x, y, color.RGBA{
		R: uint8((int(old.R)*2 + int(c.R)) / 3),
		G: uint8((int(old.G)*2 + int(c.G)) / 3),
		B: uint8((int(old.B)*2 + int(c.B)) / 3),
		A: 255,
	})
}

// hslToRGB converts HSL (h in degrees, s/l in [0,1]) to 8-bit RGB.
func hslToRGB(h, s, l float64) (int, int, int) {
	c := (1 - math.Abs(2*l-1)) * s
	hp := math.Mod(h, 360) / 60
	x := c * (1 - math.Abs(math.Mod(hp, 2)-1))
	var r, g, b float64
	switch {
	case hp < 1:
		r, g, b = c, x, 0
	case hp < 2:
		r, g, b = x, c, 0
	case hp < 3:
		r, g, b = 0, c, x
	case hp < 4:
		r, g, b = 0, x, c
	case hp < 5:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	m := l - c/2
	return int((r + m) * 255), int((g + m) * 255), int((b + m) * 255)
}

// Slider pass tokens bridge "passed the puzzle" to the protected route. They
// are HMAC-signed with the shared CryptoSecret, bound to the caller IP, and
// expire in three minutes.
const sliderPassTokenTTL = 3 * time.Minute

func sliderPassPayload(ip string, ts int64) string {
	return "slider-pass|" + ip + "|" + strconvFormatInt64(ts)
}

func strconvFormatInt64(v int64) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func IssueSliderPassToken(ip string) string {
	ts := time.Now().Unix()
	payload := sliderPassPayload(ip, ts)
	mac := GenerateHMAC(payload)
	return strconvFormatInt64(ts) + "." + mac
}

func ValidateSliderPassToken(token, ip string) bool {
	dot := -1
	for i, ch := range token {
		if ch == '.' {
			dot = i
			break
		}
	}
	if dot <= 0 || dot == len(token)-1 {
		return false
	}
	tsStr, mac := token[:dot], token[dot+1:]
	var ts int64
	neg := false
	for i, ch := range tsStr {
		if i == 0 && ch == '-' {
			neg = true
			continue
		}
		if ch < '0' || ch > '9' {
			return false
		}
		ts = ts*10 + int64(ch-'0')
	}
	if neg {
		return false
	}
	if time.Now().Unix()-ts > int64(sliderPassTokenTTL/time.Second) {
		return false
	}
	return GenerateHMAC(sliderPassPayload(ip, ts)) == mac
}
