package logic

type Config struct {
	TopMost     bool   `json:"topMost"`
	AutoAttach  bool   `json:"autoAttach"`
	Theme       string `json:"theme"`
	AutoExecute bool   `json:"autoExecute"`
	DiscordRPC  bool   `json:"discordRPC"`
	Error       string `json:"error,omitempty"`
}
