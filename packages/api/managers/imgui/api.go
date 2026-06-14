package imgui

/*
#include <stdint.h>
#include <stdbool.h>
#include <stdlib.h>

void Im_Begin(char* name, bool* open, int flags);
void Im_End();
void Im_Text(char* text);
bool Im_Button(char* label);
void Im_SetNextWindowSize(float w, float h);
bool Im_Checkbox(char* label, bool* v);
bool Im_SliderFloat(char* label, float* v, float v_min, float v_max, char* format);
bool Im_SliderInt(char* label, int* v, int v_min, int v_max, char* format);
bool Im_ColorEdit4(char* label, float* col, int flags);
void Im_SameLine();
void Im_Separator();
void Im_Spacing();
bool Im_InputTextMultiline(char* label, char* buf, size_t buf_size, float size_x, float size_y);
bool Im_InputText(char* label, char* buf, size_t buf_size);
bool Im_BeginTabBar(char* str_id);
void Im_EndTabBar();
bool Im_BeginTabItem(char* label);
void Im_EndTabItem();
void Im_Notify(char* title, char* msg, int type, int duration);
void Im_PushStyleColor(int idx, float r, float g, float b, float a);
void Im_PopStyleColor(int count);
void Im_Dummy(float w, float h);
void Editor_Init();
void Editor_Render(char* title, float w, float h);
void Editor_SetText(char* text);
const char* Editor_GetText();

bool Im_BeginChild(char* str_id, float w, float h, bool border, int flags);
void Im_EndChild();
void Im_PushStyleVarFloat(int idx, float val);
void Im_PushStyleVarVec2(int idx, float val_x, float val_y);
void Im_PopStyleVar(int count);
float Im_GetWindowWidth();
float Im_GetWindowHeight();
float Im_GetContentRegionAvailWidth();
float Im_GetContentRegionAvailHeight();
*/
import "C"
import (
	"fmt"
	"sync"
	"time"
	"unsafe"
)

const (
	TypeSuccess = 0
	TypeWarning = 1
	TypeError   = 2
	TypeInfo    = 3
)

var (
	notifyOnce sync.Once
	notifyMu   sync.Mutex
)

type notifyRequest struct {
	Title    *C.char
	Msg      *C.char
	TypeID   int
	Duration int
}

var notifyQueue = make(chan notifyRequest, 20)

func sendCNotify(title, msg string, typeID int, duration int) {
	if duration <= 0 {
		duration = 3000
	}

	cTitle := C.CString(title)
	cMsg := C.CString(msg)

	select {
	case notifyQueue <- notifyRequest{
		Title:    cTitle,
		Msg:      cMsg,
		TypeID:   typeID,
		Duration: duration,
	}:
	default:
		C.free(unsafe.Pointer(cTitle))
		C.free(unsafe.Pointer(cMsg))
	}
}

func ProcessNotifications() {
	for {
		select {
		case req := <-notifyQueue:
			C.Im_Notify(req.Title, req.Msg, C.int(req.TypeID), C.int(req.Duration))
			go func(title, msg *C.char, dur int) {
				time.Sleep(time.Duration(dur+1500) * time.Millisecond)
				C.free(unsafe.Pointer(title))
				C.free(unsafe.Pointer(msg))
			}(req.Title, req.Msg, req.Duration)
		default:
			return
		}
	}
}

var Message messageSystem

type messageSystem struct{}

func (messageSystem) Success(title, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	sendCNotify(title, msg, TypeSuccess, 2600)
}

func (messageSystem) Info(title, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	sendCNotify(title, msg, TypeInfo, 2400)
}

func (messageSystem) Warn(title, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	sendCNotify(title, msg, TypeWarning, 3800)
}

func (messageSystem) Error(title, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	sendCNotify(title, msg, TypeError, 5000)
}

func SetNextWindowSize(w, h float32) {
	C.Im_SetNextWindowSize(C.float(w), C.float(h))
}

func Begin(name string) bool {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))
	C.Im_Begin(cName, nil, 0)
	return true
}

func End() {
	C.Im_End()
}

func SameLine() {
	C.Im_SameLine()
}

func Separator() {
	C.Im_Separator()
}

func Spacing() {
	C.Im_Spacing()
}
func Dummy(width, height float32) {
	C.Im_Dummy(C.float(width), C.float(height))
}

func Text(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	cMsg := C.CString(msg)
	defer C.free(unsafe.Pointer(cMsg))
	C.Im_Text(cMsg)
}

func Button(label string) bool {
	cLabel := C.CString(label)
	defer C.free(unsafe.Pointer(cLabel))
	return bool(C.Im_Button(cLabel))
}

func Checkbox(label string, value *bool) bool {
	cLabel := C.CString(label)
	defer C.free(unsafe.Pointer(cLabel))
	cVal := C.bool(*value)
	result := C.Im_Checkbox(cLabel, &cVal)
	*value = bool(cVal)
	return bool(result)
}

func SliderFloat(label string, value *float32, min, max float32) bool {
	cLabel := C.CString(label)
	defer C.free(unsafe.Pointer(cLabel))
	cFmt := C.CString("%.3f")
	defer C.free(unsafe.Pointer(cFmt))
	cVal := C.float(*value)
	ret := C.Im_SliderFloat(cLabel, &cVal, C.float(min), C.float(max), cFmt)
	*value = float32(cVal)
	return bool(ret)
}

func SliderInt(label string, value *int, min, max int) bool {
	cLabel := C.CString(label)
	defer C.free(unsafe.Pointer(cLabel))
	cFmt := C.CString("%d")
	defer C.free(unsafe.Pointer(cFmt))

	cVal := C.int(*value)
	ret := C.Im_SliderInt(cLabel, &cVal, C.int(min), C.int(max), cFmt)
	*value = int(cVal)
	return bool(ret)
}

func InputText(label string, buffer []byte) bool {
	cLabel := C.CString(label)
	defer C.free(unsafe.Pointer(cLabel))
	return bool(C.Im_InputText(cLabel, (*C.char)(unsafe.Pointer(&buffer[0])), C.size_t(len(buffer))))
}

func InputTextMultiline(label string, buffer []byte, width, height float32) bool {
	cLabel := C.CString(label)
	defer C.free(unsafe.Pointer(cLabel))
	return bool(C.Im_InputTextMultiline(
		cLabel,
		(*C.char)(unsafe.Pointer(&buffer[0])),
		C.size_t(len(buffer)),
		C.float(width),
		C.float(height),
	))
}

func BeginTabBar(id string) bool {
	cId := C.CString(id)
	defer C.free(unsafe.Pointer(cId))
	return bool(C.Im_BeginTabBar(cId))
}

func EndTabBar() {
	C.Im_EndTabBar()
}

func BeginTabItem(label string) bool {
	cLabel := C.CString(label)
	defer C.free(unsafe.Pointer(cLabel))
	return bool(C.Im_BeginTabItem(cLabel))
}

func EndTabItem() {
	C.Im_EndTabItem()
}

func EditorInit() {
	C.Editor_Init()
}

func EditorRender(title string, w, h float32) {
	cTitle := C.CString(title)
	defer C.free(unsafe.Pointer(cTitle))
	C.Editor_Render(cTitle, C.float(w), C.float(h))
}

func EditorSetText(text string) {
	cText := C.CString(text)
	defer C.free(unsafe.Pointer(cText))
	C.Editor_SetText(cText)
}

func EditorGetText() string {
	cStr := C.Editor_GetText()
	return C.GoString(cStr)
}

func PushStyleColor(idx int, r, g, b, a float32) {
	C.Im_PushStyleColor(C.int(idx), C.float(r), C.float(g), C.float(b), C.float(a))
}

func PopStyleColor(count int) {
	C.Im_PopStyleColor(C.int(count))
}

func BeginChild(id string, width, height float32, border bool) bool {
	cId := C.CString(id)
	defer C.free(unsafe.Pointer(cId))
	cBorder := C.bool(border)
	return bool(C.Im_BeginChild(cId, C.float(width), C.float(height), cBorder, 0))
}

func EndChild() {
	C.Im_EndChild()
}

func PushStyleVarFloat(idx int, val float32) {
	C.Im_PushStyleVarFloat(C.int(idx), C.float(val))
}

func PushStyleVarVec2(idx int, valX, valY float32) {
	C.Im_PushStyleVarVec2(C.int(idx), C.float(valX), C.float(valY))
}

func PopStyleVar(count int) {
	C.Im_PopStyleVar(C.int(count))
}

func GetWindowWidth() float32 {
	return float32(C.Im_GetWindowWidth())
}

func GetWindowHeight() float32 {
	return float32(C.Im_GetWindowHeight())
}

func GetContentRegionAvailWidth() float32 {
	return float32(C.Im_GetContentRegionAvailWidth())
}

func GetContentRegionAvailHeight() float32 {
	return float32(C.Im_GetContentRegionAvailHeight())
}
