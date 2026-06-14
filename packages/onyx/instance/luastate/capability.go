package luastate

type Capability struct {
	value uint64
	shift bool
}

var allCapabilities = map[string]Capability{
	"Plugin":             {0x1, false},
	"LocalUser":          {0x2, false},
	"WritePlayer":        {0x4, false},
	"RobloxScript":       {0x8, false},
	"RobloxEngine":       {0x10, false},
	"NotAccessible":      {0x20, false},
	"RunClientScript":    {0x8, true},
	"RunServerScript":    {0x9, true},
	"AccessOutsideWrite": {0xb, true},
	"Unassigned":         {0xf, true},
	"AssetRequire":       {0x10, true},
	"LoadString":         {0x11, true},
	"ScriptGlobals":      {0x12, true},
	"CreateInstances":    {0x13, true},
	"Basic":              {0x14, true},
	"Audio":              {0x15, true},
	"DataStore":          {0x16, true},
	"Network":            {0x17, true},
	"Physics":            {0x18, true},
	"UI":                 {0x19, true},
	"CSG":                {0x1a, true},
	"Chat":               {0x1b, true},
	"Animation":          {0x1c, true},
	"Avatar":             {0x1d, true},
	"Input":              {0x1e, true},
	"Environment":        {0x1f, true},
	"RemoteEvent":        {0x20, true},
	"LegacySound":        {0x21, true},
	"PluginOrOpenCloud":  {0x3d, true},
	"Assistant":          {0x3e, true},
}

var identityCapabilities = map[int][]string{
	0: {},
	1: {"RunClientScript", "UI", "Input", "Animation", "Avatar", "RemoteEvent", "LegacySound"},
	2: {"CSG", "Chat", "Animation", "RemoteEvent", "Avatar", "LegacySound"},
	3: {"RunServerScript", "Plugin", "LocalUser", "RobloxScript", "RunClientScript", "AccessOutsideWrite", "Avatar", "RemoteEvent", "Environment", "Input", "LegacySound"},
	4: {"Plugin", "LocalUser", "RemoteEvent", "Avatar", "LegacySound"},
	5: {"Plugin", "LocalUser", "RunClientScript", "RemoteEvent", "Avatar", "LegacySound", "UI", "Input"},
	6: {"RunServerScript", "Plugin", "LocalUser", "Avatar", "RobloxScript", "RunClientScript", "AccessOutsideWrite", "Input", "Environment", "RemoteEvent", "PluginOrOpenCloud", "LegacySound"},
	7: {"Plugin", "LocalUser", "WritePlayer", "RobloxScript", "RobloxEngine", "NotAccessible", "RunClientScript", "RunServerScript", "AccessOutsideWrite", "Unassigned", "AssetRequire", "LoadString", "ScriptGlobals", "CreateInstances", "Basic", "Audio", "DataStore", "Network", "Physics", "UI", "CSG", "Chat", "Animation", "Avatar", "Input", "Environment", "RemoteEvent", "PluginOrOpenCloud", "Assistant", "LegacySound"},
	8: {"Plugin", "LocalUser", "WritePlayer", "RobloxScript", "RobloxEngine", "NotAccessible", "RunClientScript", "RunServerScript", "AccessOutsideWrite", "Unassigned", "AssetRequire", "LoadString", "ScriptGlobals", "CreateInstances", "Basic", "Audio", "DataStore", "Network", "Physics", "UI", "CSG", "Chat", "Animation", "Avatar", "Input", "Environment", "RemoteEvent", "PluginOrOpenCloud", "Assistant", "LegacySound"},
}

var BaseMask uint64 = 0xFFFFFFFFF

func identityTag(identity uint32) uint64 {
	switch identity {
	case 6, 10:
		return 0x6000000000000000
	case 12:
		return 0x1000000000000000
	default:
		return 0x2000000000000000
	}
}

func IdentityToCapabilities(identity uint32) uint64 {
	caps := BaseMask | identityTag(identity)
	if names, ok := identityCapabilities[int(identity)]; ok {
		for _, name := range names {
			if c, ok := allCapabilities[name]; ok {
				if c.shift {
					caps |= (uint64(1) << c.value)
				} else {
					caps |= c.value
				}
			}
		}
	}
	return caps
}
