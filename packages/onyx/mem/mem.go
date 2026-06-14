package mem

func (p *Luna) ValidPtr(address uintptr) bool {
	if address > 0x10000 && address < 0x7FFFFFFFFFFF {
		return true
	}
	return false
}
