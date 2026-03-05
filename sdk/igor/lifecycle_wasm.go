// SPDX-License-Identifier: Apache-2.0

//go:build tinygo || wasip1

package igor

import "unsafe"

//export agent_init
func agent_init() {
	if registeredAgent != nil {
		registeredAgent.Init()
	}
}

//export agent_tick
func agent_tick() uint32 {
	if registeredAgent != nil {
		if registeredAgent.Tick() {
			return 1
		}
	}
	return 0
}

//export agent_checkpoint
func agent_checkpoint() uint32 {
	if registeredAgent == nil {
		return 0
	}
	ckptBuf = registeredAgent.Marshal()
	return uint32(len(ckptBuf))
}

//export agent_checkpoint_ptr
func agent_checkpoint_ptr() uint32 {
	if len(ckptBuf) == 0 {
		return 0
	}
	return uint32(uintptr(unsafe.Pointer(&ckptBuf[0])))
}

//export agent_resume
func agent_resume(ptr, size uint32) {
	if registeredAgent == nil || size == 0 {
		return
	}
	data := unsafe.Slice((*byte)(unsafe.Add(unsafe.Pointer(nil), int(ptr))), size)
	registeredAgent.Unmarshal(data)
}
