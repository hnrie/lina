#pragma once

#include <cstdint>
#include <functional>
#include <string_view>
#include <cstdint>
#include <string>

/*
              SET                                 GET
    vmvalue0: data = value                     || value = data
    vmvalue1: data = (value + (data + offset)) || value = (data - (value + offset))
    vmvalue2: data = ((data + offset) - value) || value = ((v + offset) - data)
    vmvalue3: data = (value ^ (data + offset)) || value = ((v + offset) ^ data)
    vmvalue4: data = (value - (data + offset)) || value = ((v + offset) + data)
*/

#define vm_value1 vmvalue1
#define vm_value2 vmvalue2
#define vm_value3 vmvalue3
#define vm_value4 vmvalue4

#define vmvalue1_t vmvalue1
#define vmvalue2_t vmvalue2
#define vmvalue3_t vmvalue3
#define vmvalue4_t vmvalue4

#define addenc_t vmvalue1
#define sub2enc_t vmvalue2
#define xorenc_t vmvalue3
#define sub1enc_t vmvalue4

#define __PAIR64__( high, low ) ( ( ( uint64_t )( high ) << 32 ) | ( uint32_t )( low ) )

template < typename T > struct VMValue1 {
public:
    operator const T() const { return (T)((uintptr_t)storage - (uintptr_t)this); }
    void operator=(const T& value) { storage = (T)((uintptr_t)value + (uintptr_t)this); }
    const T operator->() const { return operator const T(); }
    T get() { return operator const T(); }
    void set(const T& value) { operator=(value); }
private:
    T storage;
};

template < typename T > struct VMValue2 {
public:
    operator const T() const { return (T)((uintptr_t)this - (uintptr_t)storage); }
    void operator=(const T& value) { storage = (T)((uintptr_t)this - (uintptr_t)value); }
    const T operator->() const { return operator const T(); }
    T get() { return operator const T(); }
    void set(const T& value) { operator=(value); }
private:
    T storage;
};

template < typename T > struct VMValue3 {
public:
    operator const T() const { return (T)((uintptr_t)this ^ (uintptr_t)storage); }
    void operator=(const T& value) { storage = (T)((uintptr_t)value ^ (uintptr_t)this); }
    const T operator->() const { return operator const T(); }
    T get() { return operator const T(); }
    void set(const T& value) { operator=(value); }
private:
    T storage;
};

template < typename T > struct VMValue4 {
public:
    operator const T() const { return (T)((uintptr_t)this + (uintptr_t)storage); }
    void operator=(const T& value) { storage = (T)((uintptr_t)value - (uintptr_t)this); }
    const T operator->() const { return operator const T(); }
    T get() { return operator const T(); }
    void set(const T& value) { operator=(value); }
private:
    T storage;
};

template <typename T>
class roblox_offset_t {
	using custom_getter_t = std::function<T(std::uintptr_t)>;
	std::int32_t offset_value = 0;
	custom_getter_t custom_getter;
public:
	roblox_offset_t(std::int32_t offset) {
		offset_value = offset;
	}

	roblox_offset_t(std::int32_t offset, custom_getter_t getter) {
		offset_value = offset;
		custom_getter = getter;
	}

	__forceinline std::int32_t get_offset() {
		return offset_value;
	}

	__forceinline std::uintptr_t raw_ptr(std::uintptr_t address) {
		return address + offset_value;
	}

	__forceinline T get(std::uintptr_t address) {
		if (custom_getter) {
			return custom_getter(raw_ptr(address));
		}
		return *reinterpret_cast<T*>(raw_ptr(address));
	}

	__forceinline void set(std::uintptr_t address, T value) {
		*reinterpret_cast<T*>(raw_ptr(address)) = value;
	}

	__forceinline void set_getter(custom_getter_t getter) {
		this->custom_getter = getter;
	}
};