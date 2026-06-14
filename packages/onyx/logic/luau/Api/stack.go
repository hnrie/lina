package Api

/*
#include "lua.h"
#include <string.h>
extern int GoIndex(lua_State* L);
extern int GoNamecall(lua_State* L);
*/
import "C"
import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/sandipmavani/hardwareid"
)

var (
	_index, _namecall LuaCFunction
	_Index, _Namecall func(*LuaState) uintptr
)

func (s *LuaState) GetTop() int {
	return int(C.lua_gettop(s.cptr()))
}

func (s *LuaState) SetTop(idx int) {
	C.lua_settop(s.cptr(), C.int(idx))
}

func (s *LuaState) PushValue(idx int) {
	C.lua_pushvalue(s.cptr(), C.int(idx))
}

func (s *LuaState) Remove(idx int) {
	C.lua_remove(s.cptr(), C.int(idx))
}

func (s *LuaState) Insert(idx int) {
	C.lua_insert(s.cptr(), C.int(idx))
}

func (s *LuaState) Replace(idx int) {
	C.lua_replace(s.cptr(), C.int(idx))
}

func (s *LuaState) CheckStack(sz int) bool {
	return C.lua_checkstack(s.cptr(), C.int(sz)) != 0
}

func (s *LuaState) RawCheckStack(sz int) {
	C.lua_rawcheckstack(s.cptr(), C.int(sz))
}

func (s *LuaState) XMove(to *LuaState, n int) {
	C.lua_xmove(s.cptr(), to.cptr(), C.int(n))
}

func (s *LuaState) NewThread() *LuaState {
	return (*LuaState)(unsafe.Pointer(C.lua_newthread(s.cptr())))
}

func (s *LuaState) MainThread() *LuaState {
	return (*LuaState)(unsafe.Pointer(C.lua_mainthread(s.cptr())))
}

func (s *LuaState) Next(i int) bool {
	return int(C.lua_next(s.cptr(), C.int(i))) == 1
}

func (L *LuaState) LuaHNext(t *LuaTable, key *TValue) int {
	return int(C.luaH_next(
		L.cptr(),
		(*C.LuaTable)(unsafe.Pointer(t)),
		(C.StkId)(unsafe.Pointer(key)),
	))
}

func Hook() {

	time.Sleep(time.Second)

	C.Hook_Calls(Api.LunaState.cptr())

	/*
		state := Api.LunaState
		state.GetGlobal("game")
		state.GetMetaField(-1, "__index")
		cl_i := (*Closure)(state.ToPointer(-1)).AsC()
		state.Pop(2)
		_index = cl_i.F
		purego.RegisterFunc(&_Index, uintptr(cl_i.F))

		state.GetGlobal("game")
		state.GetMetaField(-1, "__namecall")
		cl_n := (*Closure)(state.ToPointer(-1)).AsC()
		state.Pop(2)
		_namecall = cl_n.F
		purego.RegisterFunc(&_Namecall, uintptr(cl_n.F))

		cl_i.F = LuaCFunction(unsafe.Pointer(windows.NewCallbackCDecl(__Index)))
		cl_n.F = LuaCFunction(unsafe.Pointer(windows.NewCallbackCDecl(__Namecall)))
	*/
}

//export GoIndex
func GoIndex(l *C.lua_State) (ret C.int) {
	ls := (*LuaState)(unsafe.Pointer(l))
	if ls.Userdata != nil && ls.Userdata.Source.Expired() {
		if ls.IsString(2) {
			key := strings.ToLower(ls.ToString(2))
			if blockedMethods[key] {
				return 0x10
			}
			if httpMethods[key] {
				ls.GetGlobal("luna_internal_httpget")
				return 1
			}
			if key == "getobjects" {
				ls.GetGlobal("luna_internal_getobjects")
				return 1
			}
		}
	}
	return 0
}

//export GoNamecall
func GoNamecall(l *C.lua_State) (ret C.int) {
	ls := (*LuaState)(unsafe.Pointer(l))
	if ls.Namecall != nil && ls.Userdata.Source.Expired() {
		key := strings.ToLower(C.GoStringN((*C.char)(unsafe.Pointer(&ls.Namecall.Data)), C.int(ls.Namecall.Len)))
		if blockedMethods[key] {
			return 0x10
		}
		if httpMethods[key] {
			return C.int(HttpGet(ls))
		}
		if key == "getobjects" {
			return C.int(GetObjects(ls))
		}
	}
	return 0
}

func getGameIds(ls *LuaState) (string, string) {
	ls.GetGlobal("game")
	if ls.IsNil(-1) {
		ls.Pop(1)
		return "0", ""
	}
	ls.GetField(-1, "PlaceId")
	placeIdNum := ls.ToNumber(-1)
	ls.Pop(1)
	ls.GetField(-1, "JobId")
	defer ls.Pop(2)
	return fmt.Sprintf("%d", int(placeIdNum)), ls.ToString(-1)
}

func Headers(ls *LuaState, req *http.Request, customHeaders map[string]string) {
	configName := "luna"
	configVersion := "v1.0.0"

	hwid, _ := hardwareid.ProtectedID("luna")

	placeId, jobId := getGameIds(ls)

	req.Header.Set("User-Agent", fmt.Sprintf("%s/%s", configName, configVersion))
	req.Header.Set("Exploit-Guid", hwid)
	req.Header.Set(fmt.Sprintf("%s-Fingerprint", configName), hwid)
	req.Header.Set("Roblox-Place-Id", placeId)
	req.Header.Set("Roblox-Game-Id", jobId)
	sessionJson := fmt.Sprintf(`{"GameId":"%s","PlaceId":"%s"}`, jobId, placeId)
	req.Header.Set("Roblox-Session-Id", sessionJson)

	for k, v := range customHeaders {
		req.Header.Set(k, v)
	}
}

func GetObjects(ls *LuaState) uintptr {
	ls.CheckType(1, LUA_TUSERDATA)
	ls.CheckType(2, LUA_TSTRING)

	ls.GetGlobal("game")
	ls.GetField(-1, "GetService")

	ls.PushValue(-2)

	ls.PushString("InsertService")
	ls.PCall(2, 1)

	ls.GetField(-1, "LoadLocalAsset")

	ls.PushValue(-2)
	ls.PushValue(2)
	ls.PCall(2, 1)

	if ls.Type(-2) == LUA_TSTRING {
		ls.Error(ls.ToString(-1))
	}

	ls.CreateTable(0, 0)
	ls.PushValue(-2)
	ls.RawSetI(-2, 1)

	return 1
}

func HttpGet(ls *LuaState) uintptr {
	var url string
	if t := ls.Type(2); t == LUA_TSTRING || t == LUA_TNUMBER {
		url = ls.ToString(2)
	} else if t := ls.Type(1); t == LUA_TSTRING || t == LUA_TNUMBER {
		url = ls.ToString(1)
	}

	if url == "" {
		ls.PushNil()
		ls.PushString("[luna:error] url cannot be null")
		return ^uintptr(0xF)
	}
	lower := strings.ToLower(url)
	if !strings.HasPrefix(lower, "http://") &&
		!strings.HasPrefix(lower, "https://") {
		ls.PushNil()
		ls.PushString("[luna:error] invalid protocol, only http/https allowed")
		return ^uintptr(0xF)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		ls.PushNil()
		ls.PushString(err.Error())
		return ^uintptr(0xF)
	}

	Headers(ls, req, map[string]string{})

	req.Header.Set("User-Agent", "Roblox/WinInet")

	return uintptr(YieldFunc(ls, func(ls *LuaState) func(ls *LuaState) int {
		var responseBody []byte
		var requestError string
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			requestError = err.Error()
		} else {
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				requestError = err.Error()
			} else {
				responseBody = body
			}
		}
		return func(ls *LuaState) int {
			if requestError != "" {
				ls.Error("%v", requestError)
				return 0
			}
			ls.PushString(string(responseBody))
			return 1
		}
	}))
}

var blockedMethods = map[string]bool{
	"load": true, "openscreenshotsfolder": true, "openvideosfolder": true,
	"reportingoogleanalytics": true,
	"performmanagedupdate":    true, "openwechatauthwindow": true,
	"setallowinventoryreadaccess": true, "promptallowinventoryreadaccess": true, "signalcreateoutfitfailed": true, "signalcreateoutfitpermissiondenied": true, "signaldeleteoutfitfailed": true,
	"signaldeleteoutfitpermissiondenied": true, "signalrenameoutfitfailed": true, "signalrenameoutfitpermissiondenied": true, "signalsaveavatarpermissiondenied": true, "signalsetfavoritefailed": true,
	"signalsetfavoritepermissiondenied": true, "signalupdateoutfitfailed": true, "signalupdateoutfitpermissiondenied": true,
	"httprequestasync":              true,
	"getapiv1":                      true,
	"invokeasync":                   true,
	"registeropencloud":             true,
	"fetchassetwithformat":          true,
	"registerugcvalidationfunction": true,
	"performcancelsubscription":     true, "performpurchasev2": true,
	"preparecollectiblespurchase": true, "promptcancelsubscription": true,
	"reportassetsale": true, "getusersubscriptiondetailsinternalasync": true,
	"performpurchase": true, "promptbundlepurchase": true,
	"promptgamepasspurchase": true, "promptproductpurchase": true,
	"promptpurchase": true, "promptrobloxpurchase": true,
	"promptthirdpartypurchase": true, "getrobuxbalance": true,
	"promptbulkpurchase": true, "performbulkpurchase": true,
	"performsubscriptionpurchase": true, "performsubscriptionpurchasev2": true,
	"promptcollectiblespurchase": true, "promptnativepurchase": true,
	"promptnativepurchasewithlocalplayer": true, "promptpremiumpurchase": true,
	"promptsubscriptionpurchase":             true,
	"getusersubscriptionpaymenthistoryasync": true,
	"getusersubscriptionstatusasync":         true,
	"requestinternal":                        true, "getasync": true, "requestasync": true,
	"postasync": true, "sethttpenabled": true, "postasyncfullurl": true,
	"getasyncfullurl": true, "requestlimitedasync": true,
	"emithybridevent": true, "openwechtauthwindow": true,
	"executejavascript": true, "openbrowserwindow": true,
	"opennativeoverlay": true, "returntojavascript": true,
	"copyauthcookiefrombrowsertoengine": true, "sendcommand": true,
	"call": true, "getlast": true, "getmessageid": true,
	"getprotocolmethodrequestmessageid":  true,
	"getprotocolmethodresponsemessageid": true, "makerequest": true,
	"publish": true, "publishprotocolmethodrequest": true,
	"publishprotocolmethodresponse": true, "subscribe": true,
	"subscribetoprotocolmethodrequest":  true,
	"subscribetoprotocolmethodresponse": true, "setrequesthandler": true,

	"broadcastnotification": true, "setpurchasepromptisshown": true,
	"addcorescriptlocal": true, "savescriptprofilingdata": true,
	"deletecapture": true, "deletecapturesasync": true,
	"getcapturefilepathasync": true, "createpostasync": true,
	"savecapturetoexternalstorage":       true,
	"savecapturestoexternalstorageasync": true,
	"getcapturesizeasync":                true, "getcapturestoragesizeasync": true,
	"getcaptureuploaddataasync":   true,
	"promptsavecapturestogallery": true, "promptsharecapture": true,
	"capturescreenshot": true, "retrievecaptures": true,
	"savescreenshotcapture": true,
	"takescreenshot":        true, "togglerecording": true,
	"reportabuse": true, "reportabusev3": true, "reportchatabuse": true,
	"nopromptsetfavorite": true, "nopromptupdateoutfit": true,
	"performcreateoutfitwithdescription": true, "performdeleteoutfit": true,
	"performrenameoutfit": true, "performsaveavatarwithdescription": true,
	"performsetfavorite": true, "performupdateoutfit": true,
	"promptallowingventoryreadaccess": true, "promptcreateoutfit": true,
	"promptdeleteoutfit": true, "promptrenameoutfit": true,
	"promptsaveavatar": true, "promptsetfavorite": true,
	"promptupdateoutfit": true, "nopromptsaveavatarthumbnailcustomization": true,
	"nopromptsaveavatar": true, "nopromptrenameoutfit": true,
	"nopromptdeleteoutfit": true, "nopromptcreateoutfit": true,
	"openurl": true, "detecturl": true, "getandclearlastpendingurl": true,
	"getlastluaurl": true, "isurlregistered": true, "registerluaurl": true,
	"startluaurldelivery": true, "stopluaurldelivery": true,
	"supportsswitchtosettingsapp": true, "switchtosettingsapp": true,
	"getcredentialsheaders": true, "getdeviceaccesstoken": true,
	"getdeviceintegritytoken": true, "getdeviceintegritytokenyield": true,
	"publishasync": true, "subscribeasync": true,
	"run": true, "runasync": true, "require": true,
	"getlocalfilecontents": true,
	"setbaseurl":           true,
	"acquirecontextfocus":  true, "generatesessioninfostring": true,
	"getcreatedtimestamputcms": true, "getmetadata": true,
	"getrootsid": true, "getsessiontag": true, "iscontextfocused": true,
	"releasecontextfocus": true, "removemetadata": true,
	"removesession": true, "removesessionswithmetadatakey": true,
	"replacesession": true, "sessionexists": true, "setmetadata": true,
	"setsession": true, "getsessionid": true,
	"promptcommerceproductpurchase":         true,
	"promptrealworldcommercebrowser":        true,
	"usereligibleforrealworldcommerceasync": true,
	"listfilesinfoldersasync":               true, "openfileinwebbrowser": true,
	"openfolder": true, "revealfileinfolder": true,
	"createclient": true,
}

var httpMethods = map[string]bool{
	"httpget": true, "httpgetasync": true,
}

var (
	newCCacheMut sync.RWMutex
	newCCache    = make(map[*Closure]*Closure)
)

func ClosureTt(Ca *Closure) ClosureType {
	if Ca.IsC == 0 {
		return 0
	} else {
		return 1
	}
}

var Cclosure = func(ls *LuaState) int {
	ls.CheckType(1, LUA_TFUNCTION)
	function := ls.ToObject(1).ClValue()
	if ClosureTt(function) == 0 {
		ls.PushValue(1)
		C.bridge_push_newcclosure(unsafe.Pointer(ls.cptr()))
		newCC := ls.ToObject(-1).ClValue()
		ls.Ref(-1)
		newCCacheMut.Lock()
		newCCache[newCC] = function
		newCCacheMut.Unlock()

		return 1
	}
	ls.PushValue(1)
	return 1
}
