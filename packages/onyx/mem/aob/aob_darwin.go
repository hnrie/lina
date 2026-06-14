package aob

import (
	"context"
	"fmt"
	"log"
	. "main/packages/onyx/mem"
	"runtime"
	"sync"
	"unsafe"
)

func getMemoryRegions(l *Luna, debug bool) ([]MemoryReg, error) {
	var regions []MemoryReg
	address := uintptr(0)
	for {
		mbi, err := l.VirtualQuery(address)
		if err != nil {
			if debug {
				log.Printf("VirtualQueryEx failed at 0x%x: %v", address, err)
			}
			break
		}
		if mbi.Protect&uintptr(VM_PROT_READ) != 0 {
			Reg := MemoryReg{
				base: address,
				size: mbi.RegionSize,
				prot: uint32(mbi.Protect),
			}
			regions = append(regions, Reg)
			if debug {
				log.Printf("Region: [0x%x, 0x%x), size: 0x%x, prot: 0x%x", address, address+mbi.RegionSize, mbi.RegionSize, mbi.Protect)
			}
		}
		address += mbi.RegionSize
	}
	if len(regions) == 0 {
		return nil, fmt.Errorf("no readable memory regions found")
	}
	return regions, nil
}

func OptimizedScan(ctx context.Context, l *Luna, pattern *OptimizedCompiledPattern, config ScanConfig) ([]uintptr, error) {
	if config.MaxRegionSize <= 0 {
		config.MaxRegionSize = (1024 * 1024) * 10
	}
	regions, err := getMemoryRegions(l, config.Debug)
	if err != nil {
		return nil, err
	}
	numWorkers := runtime.NumCPU()
	if len(regions) < numWorkers {
		numWorkers = len(regions)
	}
	regionChan := make(chan MemoryReg, len(regions))
	resultsChan := make(chan []uintptr, numWorkers)
	var wg sync.WaitGroup
	scanCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			buf := make([]byte, config.MaxRegionSize)
			for reg := range regionChan {
				select {
				case <-scanCtx.Done():
					return
				default:
				}

				for offset := uintptr(0); offset < reg.size; offset += uintptr(config.MaxRegionSize) {
					chunkSize := uintptr(config.MaxRegionSize)
					if reg.size-offset < chunkSize {
						chunkSize = reg.size - offset
					}

					read, err := l.ReadRaw(reg.base+offset, unsafe.Pointer(&buf[0]), chunkSize)
					if err != nil || read == 0 {
						continue
					}

					matches := pattern.FastScan(buf[:read])
					if len(matches) > 0 {
						found := make([]uintptr, len(matches))
						for idx, m := range matches {
							found[idx] = reg.base + offset + m
						}
						resultsChan <- found
					}
				}
			}
		}()
	}

	for _, r := range regions {
		regionChan <- r
	}
	close(regionChan)

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	var finalResults []uintptr
	for res := range resultsChan {
		finalResults = append(finalResults, res...)
		if (config.Limit > 0 && len(finalResults) >= config.Limit) || config.EndAtOne {
			cancel()
			break
		}
	}

	return finalResults, nil
}
