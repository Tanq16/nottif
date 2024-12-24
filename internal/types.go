package internal

const (
	MaxFieldLength = 1024 // Discord's max field length
	AvatarURL      = "https://raw.githubusercontent.com/tanq16/nottif/main/.github/assets/logo.png"
)

type Notifier struct {
	webhookURLs []string
}

type DiscordWebhook struct {
	Content   string  `json:"content,omitempty"`
	Username  string  `json:"username,omitempty"`
	AvatarURL string  `json:"avatar_url,omitempty"`
	Embeds    []Embed `json:"embeds,omitempty"`
}

type Embed struct {
	Description string `json:"description"`
	Color       int    `json:"color"`
	Footer      Footer `json:"footer"`
	Timestamp   string `json:"timestamp"`
}

type Footer struct {
	Text string `json:"text"`
}
