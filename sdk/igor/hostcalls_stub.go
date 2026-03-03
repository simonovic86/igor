//go:build !tinygo && !wasip1

package igor

func clockNow() int64 {
	panic("igor: ClockNow requires WASM runtime (build with tinygo)")
}

func randBytes(_ uint32, _ uint32) int32 {
	panic("igor: RandBytes requires WASM runtime (build with tinygo)")
}

func logEmit(_ uint32, _ uint32) {
	panic("igor: Log requires WASM runtime (build with tinygo)")
}
