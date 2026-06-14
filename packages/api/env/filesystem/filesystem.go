package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	. "main/packages/onyx/logic/luau/Api"
)

const (
	dirPerm  = 0755
	filePerm = 0644
)

var (
	Dir string
)

type Filesystem struct {
	ReadFile   func(*LuaState) int `lua:"readfile"`
	WriteFile  func(*LuaState) int `lua:"writefile"`
	AppendFile func(*LuaState) int `lua:"appendfile"`
	LoadFile   func(*LuaState) int `lua:"loadfile"`

	IsFile   func(*LuaState) int `lua:"isfile"`
	IsFolder func(*LuaState) int `lua:"isfolder"`
	Exists   func(*LuaState) int `lua:"exists" alias:"ispath"`

	MakeFolder   func(*LuaState) int `lua:"makefolder" alias:"make_folder"`
	DeleteFile   func(*LuaState) int `lua:"delfile" alias:"deletefile"`
	DeleteFolder func(*LuaState) int `lua:"delfolder" alias:"deletefolder"`
	ListFiles    func(*LuaState) int `lua:"listfiles"`
}

func Init(L *LuaState) {
	Register(L, Filesystem{
		ReadFile: func(ls *LuaState) int {
			userPath, ok := requireString(ls, 1)
			if !ok {
				return 0
			}

			path, err := Safe(userPath)
			if err != nil {
				ls.Error("%v", err.Error())
				return 0
			}

			info, err := os.Stat(path)
			if err != nil {
				ls.Error("%v", err.Error())
				return 0
			}

			if info.IsDir() {
				ls.Error("cannot read folder as file")
				return 0
			}

			data, err := os.ReadFile(path)
			if err != nil {
				ls.Error("%v", err.Error())
				return 0
			}

			ls.PushString(string(data))
			return 1
		},

		WriteFile: func(ls *LuaState) int {
			userPath, ok := requireString(ls, 1)
			if !ok {
				return 0
			}

			content, ok := requireString(ls, 2)
			if !ok {
				return 0
			}

			path, err := Safe(userPath)
			if err != nil {
				ls.Error("%v", err.Error())
				return 0
			}

			if err := CreateFileWithDir(path, content); err != nil {
				ls.Error("%v", err.Error())
				return 0
			}

			return 0
		},

		AppendFile: func(ls *LuaState) int {
			userPath, ok := requireString(ls, 1)
			if !ok {
				return 0
			}

			content, ok := requireString(ls, 2)
			if !ok {
				return 0
			}

			path, err := Safe(userPath)
			if err != nil {
				ls.Error("%v", err.Error())
				return 0
			}

			if err := AppendToFile(path, content); err != nil {
				ls.Error("%v", err.Error())
				return 0
			}

			return 0
		},

		LoadFile: func(ls *LuaState) int {
			userPath, ok := requireString(ls, 1)
			if !ok {
				return 0
			}

			path, err := Safe(userPath)
			if err != nil {
				ls.PushNil()
				ls.PushString(err.Error())
				return 2
			}

			info, err := os.Stat(path)
			if err != nil {
				ls.PushNil()
				ls.PushString(err.Error())
				return 2
			}

			if info.IsDir() {
				ls.PushNil()
				ls.PushString("cannot load folder as file")
				return 2
			}

			src, err := os.ReadFile(path)
			if err != nil {
				ls.PushNil()
				ls.PushString(err.Error())
				return 2
			}

			data := Compile(string(src), CompileOptions{
				OptimizationLevel: 1,
				DebugLevel:        2,
			})

			if len(data) > 10 && data[0] != 0 {
				chunkName := "@" + filepath.ToSlash(userPath)

				if err := ls.Load(chunkName, data, 0); err != nil {
					ls.PushNil()
					ls.PushString(err.Error())
					return 2
				}

				ls.SetCaps(8)

				return 1
			}

			ls.PushNil()
			ls.PushString(string(data))
			return 2
		},

		IsFile: func(ls *LuaState) int {
			userPath, ok := requireString(ls, 1)
			if !ok {
				return 0
			}

			path, err := Safe(userPath)
			if err != nil {
				ls.PushBoolean(false)
				return 1
			}

			info, err := os.Stat(path)
			ls.PushBoolean(err == nil && !info.IsDir())
			return 1
		},

		IsFolder: func(ls *LuaState) int {
			userPath, ok := requireString(ls, 1)
			if !ok {
				return 0
			}

			path, err := Safe(userPath)
			if err != nil {
				ls.PushBoolean(false)
				return 1
			}

			info, err := os.Stat(path)
			ls.PushBoolean(err == nil && info.IsDir())
			return 1
		},

		Exists: func(ls *LuaState) int {
			userPath, ok := requireString(ls, 1)
			if !ok {
				return 0
			}

			path, err := Safe(userPath)
			if err != nil {
				ls.PushBoolean(false)
				return 1
			}

			_, err = os.Stat(path)
			ls.PushBoolean(err == nil)
			return 1
		},

		MakeFolder: func(ls *LuaState) int {
			userPath, ok := requireString(ls, 1)
			if !ok {
				return 0
			}

			path, err := Safe(userPath)
			if err != nil {
				ls.Error("%v", err.Error())
				return 0
			}

			if err := CreateFolder(path); err != nil {
				ls.Error("%v", err.Error())
				return 0
			}

			return 0
		},

		DeleteFile: func(ls *LuaState) int {
			userPath, ok := requireString(ls, 1)
			if !ok {
				return 0
			}

			path, err := Safe(userPath)
			if err != nil {
				ls.Error("%v", err.Error())
				return 0
			}

			info, err := os.Stat(path)
			if err != nil {
				ls.Error("%v", err.Error())
				return 0
			}

			if info.IsDir() {
				ls.Error("path is a folder, use delfolder instead")
				return 0
			}

			if err := os.Remove(path); err != nil {
				ls.Error("%v", err.Error())
				return 0
			}

			return 0
		},

		DeleteFolder: func(ls *LuaState) int {
			userPath, ok := requireString(ls, 1)
			if !ok {
				return 0
			}

			path, err := Safe(userPath)
			if err != nil {
				ls.Error("%v", err.Error())
				return 0
			}

			if isWorkspaceRoot(path) {
				ls.Error("refusing to delete workspace root")
				return 0
			}

			info, err := os.Stat(path)
			if err != nil {
				ls.Error("%v", err.Error())
				return 0
			}

			if !info.IsDir() {
				ls.Error("path is a file, use delfile instead")
				return 0
			}

			if err := os.RemoveAll(path); err != nil {
				ls.Error("%v", err.Error())
				return 0
			}

			return 0
		},

		ListFiles: func(ls *LuaState) int {
			userPath := ""

			if ls.IsString(1) {
				userPath = ls.ToString(1)
			}

			path, err := Safe(userPath)
			if err != nil {
				ls.Error("%v", err.Error())
				return 0
			}

			info, err := os.Stat(path)
			if err != nil {
				ls.Error("%v", err.Error())
				return 0
			}

			if !info.IsDir() {
				ls.Error("path is not a folder")
				return 0
			}

			entries, err := os.ReadDir(path)
			if err != nil {
				ls.Error("%v", err.Error())
				return 0
			}

			files := make([]string, 0, len(entries))

			for _, entry := range entries {
				full := filepath.Join(path, entry.Name())

				rel, err := filepath.Rel(WorkspaceDir(), full)
				if err != nil {
					continue
				}

				files = append(files, filepath.ToSlash(rel))
			}

			sort.Strings(files)
			pushStringSlice(ls, files)

			return 1
		},
	})
}

func requireString(ls *LuaState, idx int) (string, bool) {
	if !ls.IsString(idx) {
		ls.TypeError(idx, "string")
		return "", false
	}

	return ls.ToString(idx), true
}

func SetDir(dir string) {
	Dir = filepath.Clean(filepath.FromSlash(dir))
}

func RootDir() string {
	if strings.TrimSpace(Dir) != "" {
		return filepath.Clean(filepath.FromSlash(Dir))
	}

	wd, _ := os.Getwd()
	return wd
}

func Path(subPath string) string {
	return filepath.Join(RootDir(), filepath.FromSlash(subPath))
}

func Safe(userPath string) (string, error) {
	base, err := workspaceAbs()
	if err != nil {
		return "", err
	}

	if strings.ContainsRune(userPath, '\x00') {
		return "", fmt.Errorf("path contains null byte")
	}

	cleaned := filepath.Clean(filepath.FromSlash(userPath))

	if cleaned == "." {
		return base, nil
	}

	if filepath.IsAbs(cleaned) || filepath.VolumeName(cleaned) != "" {
		return "", fmt.Errorf("absolute paths are not allowed")
	}

	absolutePath, err := filepath.Abs(filepath.Join(base, cleaned))
	if err != nil {
		return "", fmt.Errorf("failed to resolve user path: %w", err)
	}

	if err := ensureInside(base, absolutePath); err != nil {
		return "", err
	}

	return absolutePath, nil
}

func WorkspaceDir() string {
	return Path("luna/workspace")
}

func workspaceAbs() (string, error) {
	if err := os.MkdirAll(WorkspaceDir(), dirPerm); err != nil {
		return "", err
	}

	absoluteBase, err := filepath.Abs(WorkspaceDir())
	if err != nil {
		return "", fmt.Errorf("failed to resolve workspace: %w", err)
	}

	return filepath.Clean(absoluteBase), nil
}

func ensureInside(base, target string) error {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return fmt.Errorf("failed to validate path: %w", err)
	}

	if rel == ".." ||
		strings.HasPrefix(rel, ".."+string(os.PathSeparator)) ||
		filepath.IsAbs(rel) {
		return fmt.Errorf("path escapes the workspace directory")
	}

	return nil
}

func isWorkspaceRoot(path string) bool {
	base, err := workspaceAbs()
	if err != nil {
		return false
	}

	a, _ := filepath.Abs(path)
	b, _ := filepath.Abs(base)

	return filepath.Clean(a) == filepath.Clean(b)
}

func CreateFileWithDir(filePath string, content string) error {
	dir := filepath.Dir(filePath)

	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return fmt.Errorf("failed to create parent directories: %w", err)
	}

	info, err := os.Stat(filePath)
	if err == nil && info.IsDir() {
		if err := os.RemoveAll(filePath); err != nil {
			return fmt.Errorf("path exists as folder and couldn't be removed: %w", err)
		}
	}

	return os.WriteFile(filePath, []byte(content), filePerm)
}

func CreateFolder(folderPath string) error {
	info, err := os.Stat(folderPath)
	if err == nil {
		if info.IsDir() {
			return nil
		}

		if err := os.Remove(folderPath); err != nil {
			return fmt.Errorf("path exists as file and couldn't be removed: %w", err)
		}
	}

	if err := os.MkdirAll(folderPath, dirPerm); err != nil {
		return fmt.Errorf("mkdir failed: %w", err)
	}

	return nil
}

func AppendToFile(filePath string, content string) error {
	dir := filepath.Dir(filePath)

	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return fmt.Errorf("failed to create parent directories: %w", err)
	}

	info, err := os.Stat(filePath)
	if err == nil && info.IsDir() {
		if err := os.RemoveAll(filePath); err != nil {
			return fmt.Errorf("path exists as folder and couldn't be removed: %w", err)
		}
	}

	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, filePerm)
	if err != nil {
		return fmt.Errorf("error opening file: %w", err)
	}
	defer f.Close()

	_, err = f.WriteString(content)
	return err
}

func pushStringSlice(ls *LuaState, values []string) {
	ls.CreateTable(len(values), 0)

	for i, value := range values {
		ls.PushString(value)
		ls.RawSetI(-2, i+1)
	}
}
