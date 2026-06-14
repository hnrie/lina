package aob

import (
	"context"
	. "main/packages/onyx/mem"
	"runtime"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

func getMemoryRegions(l *Luna) []MemoryReg {
	var regions []MemoryReg
	var mbi windows.MemoryBasicInformation
	address := uintptr(0)
	for {
		err := windows.VirtualQueryEx(windows.Handle(l.Handle), address, &mbi, unsafe.Sizeof(mbi))
		if err != nil {
			break
		}
		if mbi.State == 0x1000 && mbi.Protect == 0x04 && mbi.AllocationProtect == 0x04 {
			regions = append(regions, MemoryReg{
				base:  address,
				size:  mbi.RegionSize,
				state: mbi.State,
				prot:  mbi.Protect,
				alloc: mbi.AllocationProtect,
			})
		}
		address += mbi.RegionSize
	}
	return regions
}

func OptimizedScan(ctx context.Context, l *Luna, pattern *OptimizedCompiledPattern, config ScanConfig) ([]uintptr, error) {
	if config.MaxRegionSize <= 0 {
		config.MaxRegionSize = (1024 * 1024) * 10
	}
	regions := getMemoryRegions(l)
	numWorkers := min(len(regions), runtime.NumCPU())
	regionChan := make(chan MemoryReg, len(regions))
	resultsChan := make(chan []uintptr, numWorkers)
	var wg sync.WaitGroup
	scanCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	for range numWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			buf := make([]byte, config.MaxRegionSize)
			for reg := range regionChan {
				select {
				case <-scanCtx.Done():
					return
				default:
					if reg.state != windows.MEM_COMMIT {
						continue
					}
					for offset := uintptr(0); offset < reg.size; offset += uintptr(config.MaxRegionSize) {
						chunkSize := min(reg.size-offset, uintptr(config.MaxRegionSize))
						read, err := l.ReadRaw(reg.base+offset, unsafe.Pointer(&buf[0]), chunkSize)
						if err != nil || read == 0 {
							continue
						}
						matches := pattern.FastScan(buf[:read])
						if len(matches) > 0 {
							found := make([]uintptr, len(matches))
							for i, m := range matches {
								found[i] = reg.base + offset + m
							}
							resultsChan <- found
						}
					}
				}
			}
		}()
	}

	for _, r := range regions {
		regionChan <- r
	}
	close(regionChan)

	var finalResults []uintptr
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	for res := range resultsChan {
		finalResults = append(finalResults, res...)
		if (config.Limit > 0 && len(finalResults) >= config.Limit) || config.EndAtOne {
			cancel()
			break
		}
	}

	return finalResults, nil
}
