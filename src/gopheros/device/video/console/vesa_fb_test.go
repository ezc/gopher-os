package console

import (
	"bytes"
	"fmt"
	"gopheros/device"
	"gopheros/device/video/console/font"
	"gopheros/kernel"
	"gopheros/kernel/cpu"
	"gopheros/kernel/hal/multiboot"
	"gopheros/kernel/mem"
	"gopheros/kernel/mem/pmm"
	"gopheros/kernel/mem/vmm"
	"image/color"
	"reflect"
	"strings"
	"testing"
)

func TestVesaFbTextDimensions(t *testing.T) {
	var cons Device = NewVesaFbConsole(16, 32, 8, 16, nil, 0)

	if w, h := cons.Dimensions(Characters); w != 0 || h != 0 {
		t.Fatalf("expected console dimensions to be 0x0 before setting a font; got %dx%d", w, h)
	}

	specs := []struct {
		offsetY    uint32
		font       *font.Font
		expW, expH uint32
	}{
		{0, mockFont8x10, 2, 3},
		{6, mockFont8x10, 2, 2},
	}

	// Setting a nil font should be a no-op
	cons.(FontSetter).SetFont(nil)
	if w, h := cons.Dimensions(Characters); w != 0 || h != 0 {
		t.Fatalf("expected console character dimensions to be 0x0; got %dx%d", w, h)
	}

	for specIndex, spec := range specs {
		cons.(*VesaFbConsole).offsetY = spec.offsetY
		cons.(FontSetter).SetFont(spec.font)

		if w, h := cons.Dimensions(Characters); w != spec.expW || h != spec.expH {
			t.Fatalf("[spec %d] expected console character dimensions to be %dx%d; got %dx%d", specIndex, spec.expW, spec.expH, w, h)
		}

		if w, h := cons.Dimensions(Pixels); w != 16 || h != 32 {
			t.Fatalf("[spec %d] expected console pixel dimensions to be 16x32; got %dx%d", specIndex, w, h)
		}
	}
}

func TestVesaFbDefaultColors(t *testing.T) {
	var cons Device = NewVesaFbConsole(16, 32, 8, 16, nil, 0)
	if fg, bg := cons.DefaultColors(); fg != 7 || bg != 0 {
		t.Fatalf("expected console default colors to be fg:7, bg:0; got fg:%d, bg: %d", fg, bg)
	}
}

func TestVesaFbWrite8bpp(t *testing.T) {
	specs := []struct {
		consW, consH, offsetY uint32
		font                  *font.Font
		expFb                 []byte
	}{
		{
			16, 16, 6,
			mockFont8x10,
			[]byte("" +
				"0000000000000000" +
				"0000000000000000" +
				"0000000000000000" +
				"0000000000000000" +
				"0000000000000000" +
				"0000000000000000" +
				"0000000000010000" +
				"0000000000111000" +
				"0000000001101100" +
				"0000000011000110" +
				"0000000011000110" +
				"0000000011111110" +
				"0000000011000110" +
				"0000000011000110" +
				"0000000011000110" +
				"0000000011000110",
			),
		},
		{
			20, 20, 3,
			mockFont10x14,
			[]byte("" +
				"00000000000000000000" +
				"00000000000000000000" +
				"00000000000000000000" +
				"00000000000000010000" +
				"00000000000000010000" +
				"00000000000000111000" +
				"00000000000000111000" +
				"00000000000001101100" +
				"00000000000001101100" +
				"00000000000001100110" +
				"00000000000011000110" +
				"00000000000011111110" +
				"00000000000011000110" +
				"00000000000110000110" +
				"00000000000110000011" +
				"00000000000110000011" +
				"00000000001111000111" +
				"00000000000000000000" +
				"00000000000000000000" +
				"00000000000000000000",
			),
		},
	}

	var (
		fg = uint8(1)
		bg = uint8(0)
	)

	for specIndex, spec := range specs {
		fb := make([]uint8, spec.consW*spec.consH)

		cons := NewVesaFbConsole(spec.consW, spec.consH, 8, spec.consW, nil, 0)
		cons.fb = fb
		cons.offsetY = spec.offsetY
		cons.SetFont(spec.font)

		// ASCII 0 maps to the a blank character in the mock font
		// ASCII 1 maps to the letter 'A' in the mock font
		cons.Write(0, fg, bg, 0, 0)
		cons.Write(1, fg, bg, 2, 1)

		// Convert expected contents from ASCII to byte
		for i := 0; i < len(spec.expFb); i++ {
			spec.expFb[i] -= '0'
		}

		if !reflect.DeepEqual(spec.expFb, fb) {
			t.Errorf("[spec %d] unexpected frame buffer contents:\n%s",
				specIndex,
				diffFrameBuffer(spec.consW, spec.consH, spec.consW, spec.expFb, fb),
			)
		}
	}
}

func TestVesaFbWrite24bpp(t *testing.T) {
	specs := []struct {
		consW, consH, offsetY uint32
		font                  *font.Font
		expFb                 []byte
	}{
		{
			16, 16, 6,
			mockFont8x10,
			[]byte("" +
				"000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000321000000000000" +
				"000000000000000000000000000000321321321000000000" +
				"000000000000000000000000000321321000321321000000" +
				"000000000000000000000000321321000000000321321000" +
				"000000000000000000000000321321000000000321321000" +
				"000000000000000000000000321321321321321321321000" +
				"000000000000000000000000321321000000000321321000" +
				"000000000000000000000000321321000000000321321000" +
				"000000000000000000000000321321000000000321321000" +
				"000000000000000000000000321321000000000321321000",
			),
		},
		{
			20, 20, 3,
			mockFont10x14,
			[]byte("" +
				"000000000000000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000321000000000000" +
				"000000000000000000000000000000000000000000000321000000000000" +
				"000000000000000000000000000000000000000000321321321000000000" +
				"000000000000000000000000000000000000000000321321321000000000" +
				"000000000000000000000000000000000000000321321000321321000000" +
				"000000000000000000000000000000000000000321321000321321000000" +
				"000000000000000000000000000000000000000321321000000321321000" +
				"000000000000000000000000000000000000321321000000000321321000" +
				"000000000000000000000000000000000000321321321321321321321000" +
				"000000000000000000000000000000000000321321000000000321321000" +
				"000000000000000000000000000000000321321000000000000321321000" +
				"000000000000000000000000000000000321321000000000000000321321" +
				"000000000000000000000000000000000321321000000000000000321321" +
				"000000000000000000000000000000321321321321000000000321321321" +
				"000000000000000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000000000000000",
			),
		},
	}

	var (
		// BGR
		colorInfo = &multiboot.FramebufferRGBColorInfo{
			RedPosition:   16,
			RedMaskSize:   8,
			GreenPosition: 8,
			GreenMaskSize: 8,
			BluePosition:  0,
			BlueMaskSize:  8,
		}
		fg      = uint8(1)
		fgColor = color.RGBA{R: 1, G: 2, B: 3}
		bg      = uint8(0)
	)

	for specIndex, spec := range specs {
		fb := make([]uint8, spec.consW*spec.consH*3)

		cons := NewVesaFbConsole(spec.consW, spec.consH, 24, spec.consW*3, colorInfo, 0)
		cons.fb = fb
		cons.offsetY = spec.offsetY
		cons.SetFont(spec.font)
		cons.loadDefaultPalette()
		cons.SetPaletteColor(fg, fgColor)

		// ASCII 0 maps to the a blank character in the mock font
		// ASCII 1 maps to the letter 'A' in the mock font
		cons.Write(0, fg, bg, 0, 0)
		cons.Write(1, fg, bg, 2, 1)

		// Convert expected contents from ASCII to byte
		for i := 0; i < len(spec.expFb); i++ {
			spec.expFb[i] -= '0'
		}

		if !reflect.DeepEqual(spec.expFb, fb) {
			t.Errorf("[spec %d] unexpected frame buffer contents:\n%s",
				specIndex,
				diffFrameBuffer(spec.consW, spec.consH, spec.consW*3, spec.expFb, fb),
			)
		}
	}
}

func TestVesaFbScroll(t *testing.T) {
	var (
		consW, consH uint32 = 16, 16
		offsetY      uint32 = 3
		origFb              = []byte("" +
			"6666666666666666" + // }
			"7777777777777777" + // }- reserved rows
			"8888888888888888" + // }
			"0000000000001000" +
			"0000000000010000" +
			"0000000000100000" +
			"0000000001000000" +
			"0000000010000000" +
			"0000000100000000" +
			"0000001000000000" +
			"0000010000000000" +
			"0000100000000000" +
			"0001000000000000" +
			"0010000000000000" +
			"0100000000000000" +
			"1000000000000000",
		)
	)

	specs := []struct {
		dir   ScrollDir
		lines uint32
		expFb []byte
	}{
		{
			ScrollDirUp,
			0,
			[]byte("" +
				"6666666666666666" + // }
				"7777777777777777" + // }- reserved rows
				"8888888888888888" + // }
				"0000000000001000" +
				"0000000000010000" +
				"0000000000100000" +
				"0000000001000000" +
				"0000000010000000" +
				"0000000100000000" +
				"0000001000000000" +
				"0000010000000000" +
				"0000100000000000" +
				"0001000000000000" +
				"0010000000000000" +
				"0100000000000000" +
				"1000000000000000",
			),
		},
		{
			ScrollDirUp,
			10000,
			[]byte("" +
				"6666666666666666" + // }
				"7777777777777777" + // }- reserved rows
				"8888888888888888" + // }
				"0000000000001000" +
				"0000000000010000" +
				"0000000000100000" +
				"0000000001000000" +
				"0000000010000000" +
				"0000000100000000" +
				"0000001000000000" +
				"0000010000000000" +
				"0000100000000000" +
				"0001000000000000" +
				"0010000000000000" +
				"0100000000000000" +
				"1000000000000000",
			),
		},
		{
			ScrollDirUp,
			1,
			[]byte("" +
				"6666666666666666" + // }
				"7777777777777777" + // }- reserved rows
				"8888888888888888" + // }
				"0000000000010000" +
				"0000000000100000" +
				"0000000001000000" +
				"0000000010000000" +
				"0000000100000000" +
				"0000001000000000" +
				"0000010000000000" +
				"0000100000000000" +
				"0001000000000000" +
				"0010000000000000" +
				"0100000000000000" +
				"1000000000000000" +
				"1000000000000000",
			),
		},
		{
			ScrollDirUp,
			2,
			[]byte("" +
				"6666666666666666" + // }
				"7777777777777777" + // }- reserved rows
				"8888888888888888" + // }
				"0000000000100000" +
				"0000000001000000" +
				"0000000010000000" +
				"0000000100000000" +
				"0000001000000000" +
				"0000010000000000" +
				"0000100000000000" +
				"0001000000000000" +
				"0010000000000000" +
				"0100000000000000" +
				"1000000000000000" +
				"0100000000000000" +
				"1000000000000000",
			),
		},
		{
			ScrollDirDown,
			1,
			[]byte("" +
				"6666666666666666" + // }
				"7777777777777777" + // }- reserved rows
				"8888888888888888" + // }
				"0000000000001000" +
				"0000000000001000" +
				"0000000000010000" +
				"0000000000100000" +
				"0000000001000000" +
				"0000000010000000" +
				"0000000100000000" +
				"0000001000000000" +
				"0000010000000000" +
				"0000100000000000" +
				"0001000000000000" +
				"0010000000000000" +
				"0100000000000000",
			),
		},
		{
			ScrollDirDown,
			2,
			[]byte("" +
				"6666666666666666" + // }
				"7777777777777777" + // }- reserved rows
				"8888888888888888" + // }
				"0000000000001000" +
				"0000000000010000" +
				"0000000000001000" +
				"0000000000010000" +
				"0000000000100000" +
				"0000000001000000" +
				"0000000010000000" +
				"0000000100000000" +
				"0000001000000000" +
				"0000010000000000" +
				"0000100000000000" +
				"0001000000000000" +
				"0010000000000000",
			),
		},
	}

	// Convert original fb contents from ASCII to byte
	for i := 0; i < len(origFb); i++ {
		origFb[i] -= '0'
	}

	for specIndex, spec := range specs {
		// Convert expected contents from ASCII to byte
		for i := 0; i < len(spec.expFb); i++ {
			spec.expFb[i] -= '0'
		}

		fb := make([]uint8, consW*consH)
		copy(fb, origFb)

		cons := NewVesaFbConsole(consW, consH, 8, consW, nil, 0)
		cons.fb = fb
		cons.offsetY = offsetY

		// calling scroll before setting the font should be a no-op
		cons.Scroll(spec.dir, spec.lines)
		if !reflect.DeepEqual(origFb, fb) {
			t.Errorf("[spec %d] unexpected frame buffer contents:\n%s",
				specIndex,
				diffFrameBuffer(consW, consH, consW, origFb, fb),
			)
		}

		cons.SetFont(&font.Font{
			GlyphWidth:  8,
			GlyphHeight: 1,
			BytesPerRow: 1,
		})

		cons.Scroll(spec.dir, spec.lines)

		if !reflect.DeepEqual(spec.expFb, fb) {
			t.Errorf("[spec %d] unexpected frame buffer contents:\n%s",
				specIndex,
				diffFrameBuffer(consW, consH, consW, spec.expFb, fb),
			)
		}
	}
}

func TestVesaFbFill8(t *testing.T) {
	var (
		consW, consH uint32 = 16, 26
		bg           uint8  = 1
		origFb              = []byte("" +
			"6666666666666666" + // }
			"7777777777777777" + // }- reserved rows
			"8888888888888888" + // }
			"0000000000000000" +
			"0000000000000000" +
			"0000000000000000" +
			"0000000000000000" +
			"0000000000000000" +
			"0000000000000000" +
			"0000000000000000" +
			"0000000000000000" +
			"0000000000000000" +
			"0000000000000000" +
			"0000000000000000" +
			"0000000000000000" +
			"0000000000000000" +
			"0000000000000000" +
			"0000000000000000" +
			"0000000000000000" +
			"0000000000000000" +
			"0000000000000000" +
			"0000000000000000" +
			"0000000000000000" +
			"0000000000000000" +
			"0000000000000000" +
			"0000000000000000",
		)
	)
	specs := []struct {
		// Input rect in characters
		x, y, w, h uint32
		offsetY    uint32
		expFb      []byte
	}{
		{
			0, 0, 1, 1,
			0,
			[]byte("" +
				"1111111166666666" + // }
				"1111111177777777" + // }- reserved rows
				"1111111188888888" + // }
				"1111111100000000" +
				"1111111100000000" +
				"1111111100000000" +
				"1111111100000000" +
				"1111111100000000" +
				"1111111100000000" +
				"1111111100000000" +
				"0000000000000000" +
				"0000000000000000" +
				"0000000000000000" +
				"0000000000000000" +
				"0000000000000000" +
				"0000000000000000" +
				"0000000000000000" +
				"0000000000000000" +
				"0000000000000000" +
				"0000000000000000" +
				"0000000000000000" +
				"0000000000000000" +
				"0000000000000000" +
				"0000000000000000" +
				"0000000000000000" +
				"0000000000000000",
			),
		},
		{
			2, 0, 10, 1,
			3,
			[]byte("" +
				"6666666666666666" + // }
				"7777777777777777" + // }- reserved rows
				"8888888888888888" + // }
				"0000000011111111" +
				"0000000011111111" +
				"0000000011111111" +
				"0000000011111111" +
				"0000000011111111" +
				"0000000011111111" +
				"0000000011111111" +
				"0000000011111111" +
				"0000000011111111" +
				"0000000011111111" +
				"0000000000000000" +
				"0000000000000000" +
				"0000000000000000" +
				"0000000000000000" +
				"0000000000000000" +
				"0000000000000000" +
				"0000000000000000" +
				"0000000000000000" +
				"0000000000000000" +
				"0000000000000000" +
				"0000000000000000" +
				"0000000000000000" +
				"0000000000000000",
			),
		},
		{
			0, 0, 100, 100,
			3,
			[]byte("" +
				"6666666666666666" + // }
				"7777777777777777" + // }- reserved rows
				"8888888888888888" + // }
				"1111111111111111" +
				"1111111111111111" +
				"1111111111111111" +
				"1111111111111111" +
				"1111111111111111" +
				"1111111111111111" +
				"1111111111111111" +
				"1111111111111111" +
				"1111111111111111" +
				"1111111111111111" +
				"1111111111111111" +
				"1111111111111111" +
				"1111111111111111" +
				"1111111111111111" +
				"1111111111111111" +
				"1111111111111111" +
				"1111111111111111" +
				"1111111111111111" +
				"1111111111111111" +
				"1111111111111111" +
				"0000000000000000" +
				"0000000000000000" +
				"0000000000000000",
			),
		},
		{
			100, 100, 1, 1,
			6,
			[]byte("" +
				"6666666666666666" + // }
				"7777777777777777" + // }- reserved rows
				"8888888888888888" + // }
				"0000000000000000" +
				"0000000000000000" +
				"0000000000000000" +
				"0000000000000000" +
				"0000000000000000" +
				"0000000000000000" +
				"0000000000000000" +
				"0000000000000000" +
				"0000000000000000" +
				"0000000000000000" +
				"0000000000000000" +
				"0000000000000000" +
				"0000000000000000" +
				"0000000011111111" +
				"0000000011111111" +
				"0000000011111111" +
				"0000000011111111" +
				"0000000011111111" +
				"0000000011111111" +
				"0000000011111111" +
				"0000000011111111" +
				"0000000011111111" +
				"0000000011111111",
			),
		},
	}

	// Convert original fb contents from ASCII to byte
	for i := 0; i < len(origFb); i++ {
		origFb[i] -= '0'
	}

	for specIndex, spec := range specs {
		// Convert expected contents from ASCII to byte
		for i := 0; i < len(spec.expFb); i++ {
			spec.expFb[i] -= '0'
		}

		fb := make([]uint8, consW*consH)
		copy(fb, origFb)

		cons := NewVesaFbConsole(consW, consH, 8, consW, nil, 0)
		cons.fb = fb
		cons.offsetY = spec.offsetY

		// Calling fill before selecting a font should be a no-op
		cons.Fill(spec.x, spec.y, spec.w, spec.h, 0, bg)
		if !reflect.DeepEqual(origFb, fb) {
			t.Errorf("[spec %d] unexpected frame buffer contents:\n%s",
				specIndex,
				diffFrameBuffer(consW, consH, consW, origFb, fb),
			)
		}

		cons.SetFont(mockFont8x10)

		cons.Fill(spec.x, spec.y, spec.w, spec.h, 0, bg)

		if !reflect.DeepEqual(spec.expFb, fb) {
			t.Errorf("[spec %d] unexpected frame buffer contents:\n%s",
				specIndex,
				diffFrameBuffer(consW, consH, consW, spec.expFb, fb),
			)
		}
	}
}

func TestVesaFbFill24(t *testing.T) {
	var (
		consW, consH uint32 = 16, 26
		// BGR
		colorInfo = &multiboot.FramebufferRGBColorInfo{
			RedPosition:   16,
			RedMaskSize:   8,
			GreenPosition: 8,
			GreenMaskSize: 8,
			BluePosition:  0,
			BlueMaskSize:  8,
		}
		bg      uint8 = 1
		bgColor       = color.RGBA{R: 1, G: 2, B: 3}
		origFb        = []byte("" +
			"666666666666666666666666666666666666666666666666" + // }
			"777777777777777777777777777777777777777777777777" + // }- reserved rows
			"888888888888888888888888888888888888888888888888" + // }
			"000000000000000000000000000000000000000000000000" +
			"000000000000000000000000000000000000000000000000" +
			"000000000000000000000000000000000000000000000000" +
			"000000000000000000000000000000000000000000000000" +
			"000000000000000000000000000000000000000000000000" +
			"000000000000000000000000000000000000000000000000" +
			"000000000000000000000000000000000000000000000000" +
			"000000000000000000000000000000000000000000000000" +
			"000000000000000000000000000000000000000000000000" +
			"000000000000000000000000000000000000000000000000" +
			"000000000000000000000000000000000000000000000000" +
			"000000000000000000000000000000000000000000000000" +
			"000000000000000000000000000000000000000000000000" +
			"000000000000000000000000000000000000000000000000" +
			"000000000000000000000000000000000000000000000000" +
			"000000000000000000000000000000000000000000000000" +
			"000000000000000000000000000000000000000000000000" +
			"000000000000000000000000000000000000000000000000" +
			"000000000000000000000000000000000000000000000000" +
			"000000000000000000000000000000000000000000000000" +
			"000000000000000000000000000000000000000000000000" +
			"000000000000000000000000000000000000000000000000" +
			"000000000000000000000000000000000000000000000000",
		)
	)
	specs := []struct {
		// Input rect in characters
		x, y, w, h uint32
		offsetY    uint32
		expFb      []byte
	}{
		{
			0, 0, 1, 1,
			0,
			[]byte("" +
				"321321321321321321321321666666666666666666666666" + // }
				"321321321321321321321321777777777777777777777777" + // }- reserved rows
				"321321321321321321321321888888888888888888888888" + // }
				"321321321321321321321321000000000000000000000000" +
				"321321321321321321321321000000000000000000000000" +
				"321321321321321321321321000000000000000000000000" +
				"321321321321321321321321000000000000000000000000" +
				"321321321321321321321321000000000000000000000000" +
				"321321321321321321321321000000000000000000000000" +
				"321321321321321321321321000000000000000000000000" +
				"000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000",
			),
		},
		{
			2, 0, 10, 1,
			3,
			[]byte("" +
				"666666666666666666666666666666666666666666666666" + // }
				"777777777777777777777777777777777777777777777777" + // }- reserved rows
				"888888888888888888888888888888888888888888888888" + // }
				"000000000000000000000000321321321321321321321321" +
				"000000000000000000000000321321321321321321321321" +
				"000000000000000000000000321321321321321321321321" +
				"000000000000000000000000321321321321321321321321" +
				"000000000000000000000000321321321321321321321321" +
				"000000000000000000000000321321321321321321321321" +
				"000000000000000000000000321321321321321321321321" +
				"000000000000000000000000321321321321321321321321" +
				"000000000000000000000000321321321321321321321321" +
				"000000000000000000000000321321321321321321321321" +
				"000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000",
			),
		},
		{
			0, 0, 100, 100,
			3,
			[]byte("" +
				"666666666666666666666666666666666666666666666666" + // }
				"777777777777777777777777777777777777777777777777" + // }- reserved rows
				"888888888888888888888888888888888888888888888888" + // }
				"321321321321321321321321321321321321321321321321" +
				"321321321321321321321321321321321321321321321321" +
				"321321321321321321321321321321321321321321321321" +
				"321321321321321321321321321321321321321321321321" +
				"321321321321321321321321321321321321321321321321" +
				"321321321321321321321321321321321321321321321321" +
				"321321321321321321321321321321321321321321321321" +
				"321321321321321321321321321321321321321321321321" +
				"321321321321321321321321321321321321321321321321" +
				"321321321321321321321321321321321321321321321321" +
				"321321321321321321321321321321321321321321321321" +
				"321321321321321321321321321321321321321321321321" +
				"321321321321321321321321321321321321321321321321" +
				"321321321321321321321321321321321321321321321321" +
				"321321321321321321321321321321321321321321321321" +
				"321321321321321321321321321321321321321321321321" +
				"321321321321321321321321321321321321321321321321" +
				"321321321321321321321321321321321321321321321321" +
				"321321321321321321321321321321321321321321321321" +
				"321321321321321321321321321321321321321321321321" +
				"000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000",
			),
		},
		{
			100, 100, 1, 1,
			6,
			[]byte("" +
				"666666666666666666666666666666666666666666666666" + // }
				"777777777777777777777777777777777777777777777777" + // }- reserved rows
				"888888888888888888888888888888888888888888888888" + // }
				"000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000" +
				"000000000000000000000000000000000000000000000000" +
				"000000000000000000000000321321321321321321321321" +
				"000000000000000000000000321321321321321321321321" +
				"000000000000000000000000321321321321321321321321" +
				"000000000000000000000000321321321321321321321321" +
				"000000000000000000000000321321321321321321321321" +
				"000000000000000000000000321321321321321321321321" +
				"000000000000000000000000321321321321321321321321" +
				"000000000000000000000000321321321321321321321321" +
				"000000000000000000000000321321321321321321321321" +
				"000000000000000000000000321321321321321321321321",
			),
		},
	}

	// Convert original fb contents from ASCII to byte
	for i := 0; i < len(origFb); i++ {
		origFb[i] -= '0'
	}

	for specIndex, spec := range specs {
		// Convert expected contents from ASCII to byte
		for i := 0; i < len(spec.expFb); i++ {
			spec.expFb[i] -= '0'
		}

		fb := make([]uint8, consW*consH*3)
		copy(fb, origFb)

		cons := NewVesaFbConsole(consW, consH, 24, consW*3, colorInfo, 0)
		cons.fb = fb
		cons.offsetY = spec.offsetY
		cons.loadDefaultPalette()
		cons.SetPaletteColor(bg, bgColor)

		// Calling fill before selecting a font should be a no-op
		cons.Fill(spec.x, spec.y, spec.w, spec.h, 0, bg)
		if !reflect.DeepEqual(origFb, fb) {
			t.Errorf("[spec %d] unexpected frame buffer contents:\n%s",
				specIndex,
				diffFrameBuffer(consW, consH, consW*3, origFb, fb),
			)
		}

		cons.SetFont(mockFont8x10)

		cons.Fill(spec.x, spec.y, spec.w, spec.h, 0, bg)

		if !reflect.DeepEqual(spec.expFb, fb) {
			t.Errorf("[spec %d] unexpected frame buffer contents:\n%s",
				specIndex,
				diffFrameBuffer(consW, consH, consW*3, spec.expFb, fb),
			)
		}
	}
}

func TestVesaFbPalette(t *testing.T) {
	defer func() {
		portWriteByteFn = cpu.PortWriteByte
	}()

	expPal := make(color.Palette, 0)
	expPal = append(expPal,
		color.RGBA{R: 0, G: 0, B: 0},       /* black */
		color.RGBA{R: 0, G: 0, B: 128},     /* blue */
		color.RGBA{R: 0, G: 128, B: 1},     /* green */
		color.RGBA{R: 0, G: 128, B: 128},   /* cyan */
		color.RGBA{R: 128, G: 0, B: 1},     /* red */
		color.RGBA{R: 128, G: 0, B: 128},   /* magenta */
		color.RGBA{R: 64, G: 64, B: 1},     /* brown */
		color.RGBA{R: 128, G: 128, B: 128}, /* light gray */
		color.RGBA{R: 64, G: 64, B: 64},    /* dark gray */
		color.RGBA{R: 0, G: 0, B: 255},     /* light blue */
		color.RGBA{R: 0, G: 255, B: 1},     /* light green */
		color.RGBA{R: 0, G: 255, B: 255},   /* light cyan */
		color.RGBA{R: 255, G: 0, B: 1},     /* light red */
		color.RGBA{R: 255, G: 0, B: 255},   /* light magenta */
		color.RGBA{R: 255, G: 255, B: 1},   /* yellow */
		color.RGBA{R: 255, G: 255, B: 255}, /* white */
	)

	for i := len(expPal); i < 256; i++ {
		expPal = append(expPal, expPal[0])
	}

	var (
		dacIndex       uint8
		compIndex      uint8
		portWriteCount int
	)
	portWriteByteFn = func(port uint16, val uint8) {
		portWriteCount++
		switch port {
		case 0x3c8:
			dacIndex = val
			compIndex = 0
		case 0x3c9:
			r, g, b, _ := expPal[dacIndex].RGBA()

			var expVal uint8
			switch compIndex {
			case 0:
				expVal = uint8(r) >> 2
			case 1:
				expVal = uint8(g) >> 2
			case 2:
				expVal = uint8(b) >> 2
			}

			if val != expVal {
				t.Errorf("expected component %d for DAC entry %d to be %d; got %d", compIndex, dacIndex, expVal, val)
			}

			compIndex++
		}
	}

	cons := NewVesaFbConsole(0, 0, 8, 0, nil, 0)
	cons.loadDefaultPalette()

	customColor := color.RGBA{R: 251, G: 252, B: 253}
	expPal[255] = customColor
	cons.SetPaletteColor(255, customColor)

	// Setting the same RGB value should be a no-op
	cons.SetPaletteColor(255, customColor)

	if exp := 257 * 4; portWriteCount != exp {
		t.Errorf("expected %d calls to cpu.portWriteByte; got %d", exp, portWriteCount)
	}

	got := cons.Palette()
	for index, exp := range expPal {
		if got[index] != exp {
			t.Errorf("palette entry %d: want %v; got %v", index, exp, got[index])
		}
	}
}

func TestVesaFbReplace24(t *testing.T) {
	var (
		consW, consH uint32 = 4, 4
		// BGR
		colorInfo = &multiboot.FramebufferRGBColorInfo{
			RedPosition:   16,
			RedMaskSize:   8,
			GreenPosition: 8,
			GreenMaskSize: 8,
			BluePosition:  0,
			BlueMaskSize:  8,
		}
	)

	specs := []struct {
		bpp   uint8
		inpFb []byte
		expFb []byte
	}{
		{
			24,
			[]byte("" +
				"000100010002" +
				"000100010002" +
				"000100010002" +
				"000100010002",
			),
			[]byte("" +
				"765100010002" +
				"765100010002" +
				"765100010002" +
				"765100010002",
			),
		},
		{
			32,
			[]byte("" +
				"0000100001000020" +
				"0000100001000020" +
				"0000100001000020" +
				"0000100001000020",
			),
			[]byte("" +
				"7650100001000020" +
				"7650100001000020" +
				"7650100001000020" +
				"7650100001000020",
			),
		},
	}

	for specIndex, spec := range specs {
		// Convert spec fb contents from ASCII to byte
		for i := 0; i < len(spec.expFb); i++ {
			spec.inpFb[i] -= '0'
			spec.expFb[i] -= '0'
		}
		fb := make([]uint8, consW*consH*uint32(spec.bpp)>>3)
		copy(fb, spec.inpFb)

		cons := NewVesaFbConsole(consW, consH, spec.bpp, consW*uint32(spec.bpp)>>3, colorInfo, 0)
		cons.fb = fb
		cons.palette = make(color.Palette, 1)

		// First color update should not trigger a replace as the color is not used yet
		cons.SetPaletteColor(0, color.RGBA{})
		if !reflect.DeepEqual(spec.inpFb, fb) {
			t.Errorf("[spec %d] unexpected frame buffer contents:\n%s",
				specIndex,
				diffFrameBuffer(consW, consH, cons.pitch, spec.expFb, fb),
			)
		}

		// Second color update should replace existing pixels with the new RGB value
		cons.SetPaletteColor(0, color.RGBA{R: 5, G: 6, B: 7})
		if !reflect.DeepEqual(spec.expFb, fb) {
			t.Errorf("[spec %d] unexpected frame buffer contents:\n%s",
				specIndex,
				diffFrameBuffer(consW, consH, cons.pitch, spec.expFb, fb),
			)
		}
	}
}

func TestVesaFbDriverInterface(t *testing.T) {
	defer func() {
		mapRegionFn = vmm.MapRegion
		portWriteByteFn = cpu.PortWriteByte
	}()
	var dev device.Driver = NewVesaFbConsole(320, 200, 8, 320, nil, uintptr(0xa0000))

	if dev.DriverName() == "" {
		t.Fatal("DriverName() returned an empty string")
	}

	if major, minor, patch := dev.DriverVersion(); major+minor+patch == 0 {
		t.Fatal("DriverVersion() returned an invalid version number")
	}

	t.Run("init success", func(t *testing.T) {
		mapRegionFn = func(_ pmm.Frame, _ mem.Size, _ vmm.PageTableEntryFlag) (vmm.Page, *kernel.Error) {
			return 0xa0000, nil
		}

		portWriteByteFn = func(_ uint16, _ uint8) {}

		if err := dev.DriverInit(nil); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("init fail", func(t *testing.T) {
		expErr := &kernel.Error{Module: "test", Message: "something went wrong"}
		mapRegionFn = func(_ pmm.Frame, _ mem.Size, _ vmm.PageTableEntryFlag) (vmm.Page, *kernel.Error) {
			return 0, expErr
		}

		if err := dev.DriverInit(nil); err != expErr {
			t.Fatalf("expected error: %v; got %v", expErr, err)
		}
	})
}

func TestVesaFbProbe(t *testing.T) {
	defer func() {
		getFramebufferInfoFn = multiboot.GetFramebufferInfo
	}()

	getFramebufferInfoFn = func() *multiboot.FramebufferInfo {
		return &multiboot.FramebufferInfo{
			Width:    320,
			Height:   20,
			Pitch:    320,
			Bpp:      8,
			PhysAddr: 0xa0000,
			Type:     multiboot.FramebufferTypeIndexed,
		}
	}

	if drv := probeForVesaFbConsole(); drv == nil {
		t.Fatal("expected probeForVesaFbConsole to return a driver")
	}
}

func TestVesaFbPackColor24(t *testing.T) {
	specs := []struct {
		colorInfo *multiboot.FramebufferRGBColorInfo
		input     color.RGBA
		exp       [3]uint8
	}{
		{
			// RGB
			&multiboot.FramebufferRGBColorInfo{
				RedPosition:   0,
				RedMaskSize:   8,
				GreenPosition: 8,
				GreenMaskSize: 8,
				BluePosition:  16,
				BlueMaskSize:  8,
			},
			color.RGBA{R: 100, G: 200, B: 255},
			[3]uint8{100, 200, 255},
		},
		{
			// BGR
			&multiboot.FramebufferRGBColorInfo{
				RedPosition:   16,
				RedMaskSize:   8,
				GreenPosition: 8,
				GreenMaskSize: 8,
				BluePosition:  0,
				BlueMaskSize:  8,
			},
			color.RGBA{R: 250, G: 200, B: 120},
			[3]uint8{120, 200, 250},
		},
	}

	cons := NewVesaFbConsole(0, 0, 32, 0, nil, 0)
	cons.palette = make(color.Palette, 256)

	for specIndex, spec := range specs {
		cons.colorInfo = spec.colorInfo
		cons.palette[0] = spec.input

		if got := cons.packColor24(0); got != spec.exp {
			t.Errorf("[spec %d] expected: %v; got %v", specIndex, spec.exp, got)
		}
	}
}

func dumpFramebuffer(consW, consH, consPitch uint32, fb []byte) string {
	var buf bytes.Buffer

	for y := uint32(0); y < consH; y++ {
		fmt.Fprintf(&buf, "%04d |", y)
		index := (y * consPitch)
		for x := uint32(0); x < consPitch; x++ {
			fmt.Fprintf(&buf, "%d", fb[index+x])
		}
		fmt.Fprintln(&buf, "|")
	}

	return strings.TrimSpace(buf.String())
}

func diffFrameBuffer(consW, consH, consPitch uint32, exp, got []byte) string {
	expDump := strings.Split(dumpFramebuffer(consW, consH, consPitch, exp), "\n")
	gotDump := strings.Split(dumpFramebuffer(consW, consH, consPitch, got), "\n")

	maxLines := len(expDump)
	if l := len(gotDump); l > maxLines {
		maxLines = l
	}

	var buf bytes.Buffer
	var left, right string

	buf.WriteString("exp:")
	buf.WriteString(strings.Repeat(" ", len(expDump[0])-4))
	buf.WriteString(" | got:\n")

	for line := 0; line < maxLines; line++ {
		if line < len(expDump) {
			left = expDump[line]
		} else {
			left = ""
		}

		if line < len(gotDump) {
			right = gotDump[line]
		} else {
			right = ""
		}

		fmt.Fprintf(&buf, "%s | %s\n", left, right)
	}

	return buf.String()
}

var mockFont8x10 = &font.Font{
	GlyphWidth:  8,
	GlyphHeight: 10,
	BytesPerRow: 1,
	Data: []byte{
		0x00, /* 00000000 */
		0x00, /* 00000000 */
		0x00, /* 00000000 */
		0x00, /* 00000000 */
		0x00, /* 00000000 */
		0x00, /* 00000000 */
		0x00, /* 00000000 */
		0x00, /* 00000000 */
		0x00, /* 00000000 */
		0x00, /* 00000000 */
		// glyph 1
		0x10, /* 00010000 */
		0x38, /* 00111000 */
		0x6c, /* 01101100 */
		0xc6, /* 11000110 */
		0xc6, /* 11000110 */
		0xfe, /* 11111110 */
		0xc6, /* 11000110 */
		0xc6, /* 11000110 */
		0xc6, /* 11000110 */
		0xc6, /* 11000110 */
	},
}

var mockFont10x14 = &font.Font{
	GlyphWidth:  10,
	GlyphHeight: 14,
	BytesPerRow: 2,
	Data: []byte{
		0x00, 0x00, /* 0000000000 */
		0x00, 0x00, /* 0000000000 */
		0x00, 0x00, /* 0000000000 */
		0x00, 0x00, /* 0000000000 */
		0x00, 0x00, /* 0000000000 */
		0x00, 0x00, /* 0000000000 */
		0x00, 0x00, /* 0000000000 */
		0x00, 0x00, /* 0000000000 */
		0x00, 0x00, /* 0000000000 */
		0x00, 0x00, /* 0000000000 */
		0x00, 0x00, /* 0000000000 */
		0x00, 0x00, /* 0000000000 */
		0x00, 0x00, /* 0000000000 */
		0x00, 0x00, /* 0000000000 */
		// glyph 1
		0x04, 0x00, /* 0000010000 */
		0x04, 0x00, /* 0000010000 */
		0x0e, 0x00, /* 0000111000 */
		0x0e, 0x00, /* 0000111000 */
		0x1b, 0x00, /* 0001101100 */
		0x1b, 0x00, /* 0001101100 */
		0x19, 0x80, /* 0001100110 */
		0x31, 0x80, /* 0011000110 */
		0x3f, 0x80, /* 0011111110 */
		0x31, 0x80, /* 0011000110 */
		0x61, 0x80, /* 0110000110 */
		0x60, 0xc0, /* 0110000011 */
		0x60, 0xc0, /* 0110000011 */
		0xf1, 0xc0, /* 1111000111 */
	},
}
