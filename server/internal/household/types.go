package household

const (
	KindFamily = "family"
	KindTeam   = "team"
)

type Household struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	CreatedAt string `json:"created_at"`
}

type Member struct {
	UserID      string `json:"user_id"`
	DisplayName string `json:"display_name"`
	Login       string `json:"login"`
	Role        string `json:"role"`
	JoinedAt    string `json:"joined_at"`
}

type MeResponse struct {
	Household *Household `json:"household"`
	Role      string     `json:"role,omitempty"`
	Members   []Member   `json:"members,omitempty"`
}

type Invite struct {
	ID        string `json:"id"`
	Code      string `json:"code"`
	URL       string `json:"url,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	ExpiresAt string `json:"expires_at"`
}

type InvitePreview struct {
	HouseholdName    string `json:"household_name"`
	OwnerDisplayName string `json:"owner_display_name"`
	ExpiresAt        string `json:"expires_at"`
}
