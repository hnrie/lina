package instance

import (
	. "main/packages/onyx/mem"
)

var OldAccessibleFlags = make(map[uintptr]int32)

type PropertyDescriptor struct {
	Luna      *Luna
	Reference *Instance
	Address   uintptr
}

func (i *Instance) PropertyDescriptor(address uintptr) *PropertyDescriptor {
	return &PropertyDescriptor{Reference: i, Luna: i.Luna, Address: address}
}

func (pd *PropertyDescriptor) Name() string {
	var (
		name               = ReadProcessMemory[uintptr](pd.Luna, pd.Address+0x8)
		buffer [0x100]byte = ReadProcessMemory[[0x100]byte](pd.Luna, name)
	)
	size := ReadProcessMemory[uint16](pd.Luna, name+0x10)
	if size > 0x100 {
		size = 0x100
	}
	return string(buffer[:size])
}

func (pd *PropertyDescriptor) Capabilities() int32 {
	return ReadProcessMemory[int32](pd.Luna, pd.Address+0x38)
}

func (pd *PropertyDescriptor) AccessibleFlags() int32 {
	return ReadProcessMemory[int32](pd.Luna, pd.Address+0x40)
}

func (pd *PropertyDescriptor) IsHiddenValue() bool {
	return pd.AccessibleFlags() < 32
}

func (pd *PropertyDescriptor) SetScriptable(scriptable bool) bool {
	if pd.Address == 0 {
		return false
	}
	flagsAddr := pd.Address + 0x88
	if scriptable {
		if _, exists := OldAccessibleFlags[pd.Address]; !exists {
			old := pd.AccessibleFlags()
			OldAccessibleFlags[pd.Address] = old
			newFlags := old | 0x10
			WriteProcessMemory(pd.Luna, flagsAddr, newFlags)
		}
		return true
	}
	if oldFlag, exists := OldAccessibleFlags[pd.Address]; exists {
		WriteProcessMemory(pd.Luna, flagsAddr, oldFlag)
		delete(OldAccessibleFlags, pd.Address)
		return true
	}
	return false
}

func (pd *PropertyDescriptor) IsScriptable() bool {
	return ReadProcessMemory[uintptr](pd.Luna, pd.Address+0x88)&0x10 != 0
}

type PropertyDescriptorContainer struct {
	Address   uintptr
	Luna      *Luna
	Reference *Instance
}

func (i *Instance) NewPropertyDescriptorContainer() *PropertyDescriptorContainer {
	return &PropertyDescriptorContainer{
		Address:   ReadProcessMemory[uintptr](i.Luna, uintptr(i.Self)+0x18),
		Luna:      i.Luna,
		Reference: i,
	}
}

func (pdc *PropertyDescriptorContainer) GetAllYield() []*PropertyDescriptor {
	var descriptors []*PropertyDescriptor
	var (
		start uint64 = ReadProcessMemory[uint64](pdc.Luna, pdc.Address+0x40)
	)

	for i := 0; i < 0x1000; i += 0x10 {
		if descriptorAddr := ReadProcessMemory[uintptr](pdc.Luna, uintptr(start)+uintptr(i)); descriptorAddr > 1000 {
			descriptors = append(descriptors, pdc.Reference.PropertyDescriptor(descriptorAddr))
		}
	}

	return descriptors
}

func (pdc *PropertyDescriptorContainer) Get(name string) *PropertyDescriptor {
	for _, descriptor := range pdc.GetAllYield() {
		if descriptor.Name() == name {
			return descriptor
		}
	}
	return pdc.Reference.PropertyDescriptor(0)
}

const (
	ReflectionType_Bool          = 0x1
	ReflectionType_Int           = 0x2
	ReflectionType_Float         = 0x4
	ReflectionType_String        = 0x6
	ReflectionType_Instance      = 0x8
	ReflectionType_SystemAddress = 0x1d
)

const (
	Update_get_set      = 0x98
	Update_getter       = 0x18
	Update_setter       = 0x20
	Update_ttype        = 0x68
	Update_ttype_number = 0x30
)

func (pd *PropertyDescriptor) TypeNumber() int {
	tType := ReadProcessMemory[uintptr](pd.Luna, pd.Address+Update_ttype)
	if tType == 0 {
		return -1
	}
	rawVal := ReadProcessMemory[uintptr](pd.Luna, tType+Update_ttype_number)
	return int(rawVal & 0xFFFFFFFF)
}

func (pd *PropertyDescriptor) GetSetImpl() uintptr {
	return ReadProcessMemory[uintptr](pd.Luna, pd.Address+Update_get_set)
}

func (pd *PropertyDescriptor) Getter() uintptr {
	impl := pd.GetSetImpl()
	if impl == 0 {
		return 0
	}

	vft := ReadProcessMemory[uintptr](pd.Luna, impl)
	if vft == 0 {
		return 0
	}

	return ReadProcessMemory[uintptr](pd.Luna, vft+Update_getter)
}

func (pd *PropertyDescriptor) Setter() uintptr {
	impl := pd.GetSetImpl()
	if impl == 0 {
		return 0
	}
	vft := ReadProcessMemory[uintptr](pd.Luna, impl)
	if vft == 0 {
		return 0
	}
	return ReadProcessMemory[uintptr](pd.Luna, vft+Update_setter)
}
