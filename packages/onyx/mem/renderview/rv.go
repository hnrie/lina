//go:build windows

package renderview

import "C"
import (
	"context"
	. "main/packages/onyx/instance"
	. "main/packages/onyx/mem"
	. "main/packages/onyx/mem/aob"
)

type (
	Render struct {
		Rv   uintptr
		Luna *Luna
	}
	Container struct {
		C    uintptr
		Luna *Luna
	}
	Jobs struct {
		Address uintptr
		Name    string
	}
)

func RenderView(L *Luna) Render {
	results, _ := OptimizedScan(context.Background(), L, NewPatternFromBytes(ExactMatch("RenderJob")), ScanConfig{
		Limit: 10,
	})
	for _, results := range results {
		if rv := PointerWalk(L, results, 1, 0x300, 0x8, RTTIScanMatcher("RBX::Graphics::RenderView")); rv != nil {
			return Render{
				Rv:   rv.FoundAddress,
				Luna: L,
			}
		}
	}
	return Render{
		Rv:   0,
		Luna: L,
	}
}

func (R *Render) Container() *Container {
	return &Container{
		C:    ReadProcessMemory[uintptr](R.Luna, PointerWalk(R.Luna, R.Rv, 1, 0x300, 0x8, RTTIScanMatcher("RBX::DataModel")).FoundAddress+0x30),
		Luna: R.Luna,
	}
}

func (Con *Container) Jobs() (Data []*Jobs) {
	var (
		getJob = func(job_container uintptr) *Jobs {
			job := ReadProcessMemory[uintptr](Con.Luna, job_container)
			ptr := job + 0x18
			size := ReadProcessMemory[uint16](Con.Luna, job+0x28)
			if size > 0x100 {
				return &Jobs{}
			}
			if size > 15 {
				ptr = ReadProcessMemory[uintptr](Con.Luna, ptr)
			}
			name := ReadProcessMemory[[0x100]byte](Con.Luna, ptr)
			return &Jobs{
				Address: job,
				Name:    string(name[:size]),
			}
		}
	)
	for i := 0x8; i < 0x1000; i += 0x18 {
		job := getJob(Con.C + uintptr(i))
		if job.Name == "" {
			break
		}
		Data = append(Data, job)
	}
	return
}

func (Con *Container) Job(name string) *Jobs {
	for _, job := range Con.Jobs() {
		if job.Name == name {
			return job
		}
	}
	return &Jobs{}
}

func (R *Render) DataModel() *Instance {
	defer func() {
		recover()
	}()
	return NewInstance(R.Luna,
		PointerWalk(R.Luna,
			PointerWalk(R.Luna,
				R.Rv,
				1,
				0x300,
				0x8,
				RTTIScanMatcher("RBX::DataModel")).FoundAddress,
			1,
			0x300,
			0x8,
			RTTIScanMatcher("RBX::DataModel")).FoundAddress)
}
