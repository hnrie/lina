package misc

import (
	"io"
	. "main/packages/onyx/logic/luau/Api"
	"net/http"
	"strings"

	block "github.com/pierrec/lz4/v4"
	"github.com/sandipmavani/hardwareid"
	"golang.org/x/sys/windows"

	"github.com/atotto/clipboard"
)

type Misc struct {
	Loadstring       func(*LuaState) int `lua:"loadstring"`
	IdentifyExecutor func(*LuaState) int `lua:"identifyexecutor" alias:"getexecutorname"`
	Lz4Compress      func(*LuaState) int `lua:"lz4compress"`
	Lz4Decompress    func(*LuaState) int `lua:"lz4decompress"`
	MessageBox       func(*LuaState) int `lua:"messagebox"`
	QueueOnTeleport  func(*LuaState) int `lua:"queue_on_teleport" alias:"queueonteleport"`
	Request          func(*LuaState) int `lua:"request" alias:"http_request" alias:"http.request"`
	SetClipboard     func(*LuaState) int `lua:"setclipboard" alias:"toclipboard"`
	SetFpsCap        func(*LuaState) int `lua:"setfpscap"`
	GetHWID          func(*LuaState) int `lua:"gethwid"`
}

func Init(L *LuaState) {
	var (
		Id, Version string = "luna", "v1.0.0"
	)

	Register(L, Misc{
		Loadstring: func(ls *LuaState) int {
			/*
				if !ls.IsString(1) {
					ls.SetTop(0)
					ls.PushNil()
					ls.PushString("index 1 needs to be a string.")
					return 2
				}
			*/

			if err := ls.Load(ls.OptString(2, "luna"), Compile(ls.ToString(1), CompileOptions{
				OptimizationLevel: 1,
				DebugLevel:        2,
			}), 0); err != nil {
				ls.SetTop(0)
				ls.PushNil()
				ls.PushString(err.Error())
				return 2
			}
			ls.SetCaps(8)
			ls.SetSafeEnv(LUA_GLOBALSINDEX, false)
			return 1
		},
		IdentifyExecutor: func(ls *LuaState) int {
			ls.PushString(Id)
			ls.PushString(Version)
			return 2
		},
		Lz4Compress: func(ls *LuaState) int {
			data := ls.ToString(1)
			if data == "" {
				ls.PushString("")
				return 1
			}

			src := []byte(data)

			maxSize := block.CompressBlockBound(len(src))
			dst := make([]byte, maxSize)

			compSize, err := block.CompressBlock(src, dst, []int{})
			if err != nil {
				ls.Error("lz4compress failed: " + err.Error())
				return 0
			}

			ls.PushString(string(dst[:compSize]))
			return 1
		},
		Lz4Decompress: func(ls *LuaState) int {
			data := ls.ToString(1)
			uncompressedSize := ls.ToInteger(2)

			if data == "" {
				ls.PushString("")
				return 1
			}

			if uncompressedSize <= 0 {
				ls.Error("lz4decompress requires the exact uncompressed size (integer) as argument #2")
				return 0
			}

			src := []byte(data)
			dst := make([]byte, uncompressedSize)

			uncompSize, err := block.UncompressBlock(src, dst)
			if err != nil {
				ls.Error("lz4decompress failed: " + err.Error())
				return 0
			}

			ls.PushString(string(dst[:uncompSize]))
			return 1
		},
		QueueOnTeleport: func(ls *LuaState) int {
			if ls.IsString(1) {
				Api.QueuedList = append(Api.QueuedList,
					Yieldable{
						Type: Execute,
						Source: Compile(ls.ToString(1), CompileOptions{
							OptimizationLevel: 1,
							DebugLevel:        2,
						}),
					},
				)
			}
			return 0
		},
		Request: func(ls *LuaState) int {
			type RequestOptions struct {
				Url     string            `lua:"Url"`
				Method  string            `lua:"Method"`
				Body    *string           `lua:"Body"`
				Headers map[string]string `lua:"Headers"`
				Cookies map[string]string `lua:"Cookies"`
			}

			type Response struct {
				Success       bool              `lua:"Success"`
				Body          string            `lua:"Body"`
				StatusCode    int               `lua:"StatusCode"`
				StatusMessage string            `lua:"StatusMessage"`
				Headers       map[string]string `lua:"Headers"`
			}

			opts, err := TableToStruct[RequestOptions](ls, 1)
			if err != nil {
				ls.Error("%v", err.Error())
				return 0
			}

			if opts.Url == "" {
				ls.Error("missing url")
				return 0
			}

			if opts.Method == "" {
				opts.Method = "GET"
			}

			var bodyReader io.Reader
			if opts.Body != nil && *opts.Body != "" {
				bodyReader = strings.NewReader(*opts.Body)
			}

			req, err := http.NewRequest(opts.Method, opts.Url, bodyReader)
			if err != nil {
				ls.Error("%v", err.Error())
				return 0
			}

			Headers(ls, req, opts.Headers)

			if len(opts.Cookies) > 0 {
				parts := make([]string, 0, len(opts.Cookies))
				for k, v := range opts.Cookies {
					parts = append(parts, k+"="+v)
				}
				req.Header.Set("Cookie", strings.Join(parts, "; "))
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				ls.Error("%v", err.Error())
				return 0
			}
			defer resp.Body.Close()

			data, err := io.ReadAll(resp.Body)
			if err != nil {
				ls.Error("%v", err.Error())
				return 0
			}

			respHeaders := map[string]string{}
			for k, v := range resp.Header {
				respHeaders[k] = strings.Join(v, ", ")
			}

			StructToTable(ls, Response{
				Success:       resp.StatusCode >= 200 && resp.StatusCode < 300,
				Body:          string(data),
				StatusCode:    resp.StatusCode,
				StatusMessage: resp.Status,
				Headers:       respHeaders,
			})

			return 1
		},
		SetClipboard: func(ls *LuaState) int {
			if ls.IsString(1) {
				clipboard.WriteAll(ls.ToString(1))
			}
			return 0
		},
		MessageBox: func(ls *LuaState) int {
			text := ls.OptString(1, "")
			caption := ls.OptString(2, "")
			flags := ls.OptInteger(3, 0)

			return YieldFunc(ls, func(ls *LuaState) func(ls *LuaState) int {
				return func(ls *LuaState) int {
					ret, err := windows.MessageBox(
						0,
						windows.StringToUTF16Ptr(text),
						windows.StringToUTF16Ptr(caption),
						uint32(flags),
					)
					if err != nil {
						ls.Error("%v", err.Error())
					}
					ls.PushInteger(int(ret))
					return 1
				}
			})
		},
		GetHWID: func(ls *LuaState) int {
			hwid, err := hardwareid.ProtectedID("luna")

			if err != nil {
				ls.Error("%v", err.Error())
				return 0
			}

			ls.PushString(hwid)
			return 1
		},
		SetFpsCap: func(ls *LuaState) int { return 0 },
	})
}
