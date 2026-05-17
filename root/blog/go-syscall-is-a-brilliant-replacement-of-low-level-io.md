---
id: c90263519492728e7cc2d0ce840057b6
author: Yunjin Lee
title: Go syscall is a brilliant replacement of low-level I/O
description: Let's learn how to syscall
language: en
date: 2025-10-26T12:31:21.655228078Z
path: /blog/posts/go-syscall-is-a-brilliant-replacement-of-low-level-i-o-za50951e1
---

## Summary

We will learn about direct system call on Go.
Since Go is offering strict compiler errors and rigid GC, it is much better to replace low-level calls in Pure Go.
Luckily, most of the C function calls are fully reimplemented in Go, in a good and contemporary manner.
Let's take a look at it.

## System Call
System call is a direct request to the operating system.
Since system is usually written in rigid, old-fashioned style since it is running right on a hardware, we need to consider that its call must deliver strict, and correct form of a request.
So, even if we don't need some variables, we still need to fill out the size regardless of use.
Let's check with fully-working example.

## Full Example

```go
package main
import (
	"fmt"
	"syscall"
	"unsafe"
)

type sysinfo_t struct {
	Uptime    int64
	Loads     [3]uint64
	Totalram  uint64
	Freeram   uint64
	Sharedram uint64
	Bufferram uint64
	Totalswap uint64
	Freeswap  uint64
	Procs     uint16
	Pad       uint16
	_         [4]byte
	Totalhigh uint64
	Freehigh  uint64
	MemUnit   uint32
	_         [4]byte
}

func main() {
	var info sysinfo_t
	_, _, errno := syscall.Syscall(syscall.SYS_SYSINFO, uintptr(unsafe.Pointer(&info)), 0, 0)
	if errno != 0 {
		fmt.Println("sysinfo syscall failed:", errno)
		return
	}

	scale := float64(1 << 16)
	fmt.Printf("Uptime: %d seconds\n", info.Uptime)
	fmt.Printf("Load Average: %.2f %.2f %.2f\n",
		float64(info.Loads[0])/scale,
		float64(info.Loads[1])/scale,
		float64(info.Loads[2])/scale)
	fmt.Printf("Memory: total=%d MB free=%d MB buffer=%d MB\n",
		info.Totalram*uint64(info.MemUnit)/1024/1024,
		info.Freeram*uint64(info.MemUnit)/1024/1024,
		info.Bufferram*uint64(info.MemUnit)/1024/1024)
	fmt.Printf("Swap: total=%d MB free=%d MB\n",
		info.Totalswap*uint64(info.MemUnit)/1024/1024,
		info.Freeswap*uint64(info.MemUnit)/1024/1024)
	fmt.Printf("Processes: %d\n", info.Procs)
}
```

This example includes all variables, and print extensive information of current system information.
We can compare this code, as a locker and a key.
`syscall.SYS_SYSINFO` is a key that unlocks a locker that is inside of a kernel.
Therefore, using correct key for a locker is important.
What will happen when we use `syscall.SYS_GETPID` for this call?
This is a key for a locker that contains Process ID.
This will attempt to get a PID from a space for system information.
As a result, none of the information can be read correctly; the call must be retured as failed state.

Now, we need to know which items are contained, and how items are ordered.
In a first slot of a locker, we have Uptime, with a size of 2^64.
If we try to read this with 2^32, the bit sequence is not fully read.
We cannot use these kinds of partial binaries unless we are going to write low-level tricks.

After reading 64 bits of binary data, finally we are on a second slot.
It can only be read accurately when we have read previous 64-bit sized integer.

With repeating those strict, and logical flows to obtain a proper information from a system, we can properly handle read data.

## How to skip 'variable names'

Even though we cannot 'skip' variables themselves, it is important to distinguish used variables and discarded ones.
If use of the program is clear enough, it is better to use nameless variables as placeholders than labeling each values even if they are not used forever.
Let's check this with an example, "Free Memory Checker"

## Example - Free Memory Checker
When checking Free Memory/Swaps, we don't need other information that is indicating different resources.
To achieve a better visibility, you can make anonymous variables to hold specific spaces.

```go
package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

type sysinfo_t struct {
	_          int64
	_         [3]uint64
	Totalram  uint64
	Freeram   uint64
	Sharedram uint64
	Bufferram uint64
	Totalswap uint64
	Freeswap  uint64
	_         uint16  // anonymous, and unused ones are marked as _
	_         uint16  
	_         [4]byte 
	_         uint64  
	_         uint64  
	MemUnit   uint32
	_         [4]byte
}

func main() {
	var info sysinfo_t
	_, _, errno := syscall.Syscall(syscall.SYS_SYSINFO, uintptr(unsafe.Pointer(&info)), 0, 0)
	if errno != 0 {
		fmt.Println("sysinfo syscall failed:", errno)
		return
	}

	fmt.Printf("Memory: total=%d MB free=%d MB buffer=%d MB\n",
		info.Totalram*uint64(info.MemUnit)/1024/1024,
		info.Freeram*uint64(info.MemUnit)/1024/1024,
		info.Bufferram*uint64(info.MemUnit)/1024/1024)
	fmt.Printf("Swap: total=%d MB free=%d MB\n",
		info.Totalswap*uint64(info.MemUnit)/1024/1024,
		info.Freeswap*uint64(info.MemUnit)/1024/1024)
}
```

Consequently, variables are read without labels.
Although anonymous values are actually stored into a structure, there is no labels/legible marks on a code.

## Conclusion
- Using Go's `syscall` and `unsafe` is still safer than C/CGo
- If you are writing a huge project that can be expaneded easily:
  - Don't make anonymous variables; make each names for the members.
- If you are writing a project that has limited use:
  - You can use anonymous variables to hold spaces those are actually unused.
- Go's `syscall` is powerful and modern to handle low-level calls

## Read Further

[syscall](https://pkg.go.dev/syscall)
[unsafe](https://pkg.go.dev/unsafe)
[x/sys/unix](https://pkg.go.dev/golang.org/x/sys/unix)

