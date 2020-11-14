package xorg

/*
#cgo linux CFLAGS: -I/usr/src -I/usr/local/include/
#cgo linux LDFLAGS: -L/usr/src -L/usr/local/lib -lX11 -lXtst -lXrandr -lxcb

#include "xorg.h"
*/
import "C"

import (
	"fmt"
	"sync"
	"time"
	"unsafe"
	"regexp"

	"demodesk/neko/internal/types"
)

var ScreenConfigurations = make(map[int]types.ScreenConfiguration)

var debounce_button = make(map[int]time.Time)
var debounce_key = make(map[uint64]time.Time)
var mu = sync.Mutex{}

func GetScreenConfigurations() {
	C.XGetScreenConfigurations()
}

func DisplayOpen(display string) error {
	mu.Lock()
	defer mu.Unlock()

	displayUnsafe := C.CString(display)
	defer C.free(unsafe.Pointer(displayUnsafe))

	err := C.XDisplayOpen(displayUnsafe)
	if int(err) == 1 {
		return fmt.Errorf("Could not open display %s.", display)
	}

	return nil
}

func DisplayClose() {
	mu.Lock()
	defer mu.Unlock()
	
	C.XDisplayClose()
}

func Move(x, y int) {
	mu.Lock()
	defer mu.Unlock()

	C.XMove(C.int(x), C.int(y))
}

func Scroll(x, y int) {
	mu.Lock()
	defer mu.Unlock()

	C.XScroll(C.int(x), C.int(y))
}

func ButtonDown(code int) error {
	mu.Lock()
	defer mu.Unlock()

	if _, ok := debounce_button[code]; ok {
		return fmt.Errorf("debounced button %v", code)
	}

	debounce_button[code] = time.Now()

	C.XButton(C.uint(code), C.int(1))
	return nil
}

func KeyDown(code uint64) error {
	mu.Lock()
	defer mu.Unlock()

	if _, ok := debounce_key[code]; ok {
		return fmt.Errorf("debounced key %v", code)
	}

	debounce_key[code] = time.Now()

	C.XKey(C.ulong(code), C.int(1))
	return nil
}

func ButtonUp(code int) error {
	mu.Lock()
	defer mu.Unlock()

	if _, ok := debounce_button[code]; !ok {
		return fmt.Errorf("debounced button %v", code)
	}

	delete(debounce_button, code)

	C.XButton(C.uint(code), C.int(0))
	return nil
}

func KeyUp(code uint64) error {
	mu.Lock()
	defer mu.Unlock()

	if _, ok := debounce_key[code]; !ok {
		return fmt.Errorf("debounced key %v", code)
	}

	delete(debounce_key, code)

	C.XKey(C.ulong(code), C.int(0))
	return nil
}

func ResetKeys() {
	for code := range debounce_button {
		//nolint
		ButtonUp(code)

		delete(debounce_button, code)
	}
	for code := range debounce_key {
		//nolint
		KeyUp(code)

		delete(debounce_key, code)
	}
}

func CheckKeys(duration time.Duration) {
	t := time.Now()
	for code, start := range debounce_button {
		if t.Sub(start) < duration {
			continue
		}
		//nolint
		ButtonUp(code)

		delete(debounce_button, code)
	}
	for code, start := range debounce_key {
		if t.Sub(start) < duration {
			continue
		}
		//nolint
		KeyUp(code)

		delete(debounce_key, code)
	}
}

func ChangeScreenSize(width int, height int, rate int) error {
	mu.Lock()
	defer mu.Unlock()

	for index, size := range ScreenConfigurations {
		if size.Width == width && size.Height == height {
			for _, fps := range size.Rates {
				if int16(rate) == fps {
					C.XSetScreenConfiguration(C.int(index), C.short(fps))
					return nil
				}
			}
		}
	}

	return fmt.Errorf("Unknown screen configuration %dx%d@%d.", width, height, rate)
}

func GetScreenSize() *types.ScreenSize {
	mu.Lock()
	defer mu.Unlock()

	index := int(C.XGetScreenSize())
	rate := int16(C.XGetScreenRate())

	if conf, ok := ScreenConfigurations[index]; ok {
		return &types.ScreenSize{
			Width:  conf.Width,
			Height: conf.Height,
			Rate:   rate,
		}
	}

	return nil
}

func SetKeyboardLayout(layout string) {
	mu.Lock()
	defer mu.Unlock()

	if !regexp.MustCompile(`^[a-zA-Z]+$`).MatchString(layout) {
		return
	}

	layoutUnsafe := C.CString(layout)
	defer C.free(unsafe.Pointer(layoutUnsafe))

	C.SetKeyboardLayout(layoutUnsafe)
}

func SetKeyboardModifiers(num_lock int, caps_lock int, scroll_lock int) {
	mu.Lock()
	defer mu.Unlock()

	C.SetKeyboardModifiers(C.int(num_lock), C.int(caps_lock), C.int(scroll_lock))
}

//export goCreateScreenSize
func goCreateScreenSize(index C.int, width C.int, height C.int, mwidth C.int, mheight C.int) {
	ScreenConfigurations[int(index)] = types.ScreenConfiguration{
		Width:  int(width),
		Height: int(height),
		Rates:  make(map[int]int16),
	}
}

//export goSetScreenRates
func goSetScreenRates(index C.int, rate_index C.int, rate C.short) {
	ScreenConfigurations[int(index)].Rates[int(rate_index)] = int16(rate)
}