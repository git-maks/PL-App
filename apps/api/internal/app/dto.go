package app

import (
	"time"

	"github.com/git-maks/pl-app/apps/api/internal/game"
)

// DTOs – JSON shapes from packages/contracts/openapi.yaml (components.schemas).

type Consents struct {
	Terms         bool `json:"terms"`
	Notifications bool `json:"notifications"`
}

type Me struct {
	ID          string    `json:"id"`
	Kind        string    `json:"kind"`
	Nickname    string    `json:"nickname"`
	Email       *string   `json:"email"`
	Role        string    `json:"role"`
	AvatarURL   *string   `json:"avatarUrl"`
	AvatarFrame *string   `json:"avatarFrame"`
	Consents    Consents  `json:"consents"`
	CreatedAt   time.Time `json:"createdAt"`
}

type MePatch struct {
	Nickname    *string   `json:"nickname"`
	Consents    *Consents `json:"consents"`
	AvatarFrame *string   `json:"avatarFrame"`
}

type Event struct {
	ID          string      `json:"id"`
	Slug        string      `json:"slug"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	StartsAt    time.Time   `json:"startsAt"`
	EndsAt      time.Time   `json:"endsAt"`
	Timezone    string      `json:"timezone"`
	Status      string      `json:"status"`
	Config      game.Config `json:"config"`
	TeamMin     int         `json:"teamMin"`
	TeamMax     int         `json:"teamMax"`

	queuedGrace time.Duration
}

type Session struct {
	ID           string    `json:"id"`
	StationID    string    `json:"stationId"`
	Room         string    `json:"room"`
	StartsAt     time.Time `json:"startsAt"`
	EndsAt       time.Time `json:"endsAt"`
	ScanOpensAt  time.Time `json:"scanOpensAt"`
	ScanClosesAt time.Time `json:"scanClosesAt"`
}

type TeamRule struct {
	MinMembers int `json:"minMembers"`
}

type Station struct {
	ID          string      `json:"id"`
	Slug        string      `json:"slug"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Kind        string      `json:"kind"`
	Faculty     *string     `json:"faculty"`
	Poi         string      `json:"poi"`
	BasePoints  int         `json:"basePoints"`
	Weights     game.Vector `json:"weights"`
	Team        *TeamRule   `json:"team"`
	Sessions    []Session   `json:"sessions"`

	eventID     string
	secret      []byte
	manualState *string
	manualUntil *time.Time
}

type TeamProgress struct {
	Scanned  int `json:"scanned"`
	Required int `json:"required"`
}

type Scan struct {
	ID            string        `json:"id"`
	StationID     string        `json:"stationId"`
	StationName   string        `json:"stationName"`
	ScannedAt     time.Time     `json:"scannedAt"`
	Rating        *int          `json:"rating"`
	Team          bool          `json:"team"`
	TeamStatus    string        `json:"teamStatus"`
	TeamProgress  *TeamProgress `json:"teamProgress"`
	Deltas        game.Stats    `json:"deltas"`
	RateableUntil time.Time     `json:"rateableUntil"`
}

type ScanResult struct {
	Scan           Scan `json:"scan"`
	AlreadyScanned bool `json:"alreadyScanned"`
}

type Badge struct {
	ID        string    `json:"id"`
	AwardedAt time.Time `json:"awardedAt"`
}

type Progress struct {
	EventID         string             `json:"eventId"`
	Stats           game.Stats         `json:"stats"`
	XP              float64            `json:"xp"`
	Level           int                `json:"level"`
	LevelProgress   game.LevelProgress `json:"levelProgress"`
	RatedCount      int                `json:"ratedCount"`
	Archetypes      []game.Share       `json:"archetypes"`
	TitleKey        *string            `json:"titleKey"`
	CardUnlocked    bool               `json:"cardUnlocked"`
	MinScansForCard int                `json:"minScansForCard"`
	Scans           []Scan             `json:"scans"`
	Badges          []Badge            `json:"badges"`
}

type RatingResult struct {
	Scan     Scan     `json:"scan"`
	Progress Progress `json:"progress"`
}

type Character struct {
	Unlocked bool                  `json:"unlocked"`
	Progress Progress              `json:"progress"`
	Programs []game.Recommendation `json:"programs"`
	Clubs    []game.Recommendation `json:"clubs"`
}

type TeamMember struct {
	UserID    string    `json:"userId"`
	Nickname  string    `json:"nickname"`
	AvatarURL *string   `json:"avatarUrl"`
	Role      string    `json:"role"`
	XP        float64   `json:"xp"`
	JoinedAt  time.Time `json:"joinedAt"`
}

type Team struct {
	ID         string       `json:"id"`
	Name       string       `json:"name"`
	Emblem     string       `json:"emblem"`
	CaptainID  string       `json:"captainId"`
	Members    []TeamMember `json:"members"`
	Score      float64      `json:"score"`
	MaxMembers int          `json:"maxMembers"`
	CreatedAt  time.Time    `json:"createdAt"`
}

type TeamInvite struct {
	Code      string    `json:"code"`
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expiresAt"`
	UsesLeft  int       `json:"usesLeft"`
}

type LeaderboardEntry struct {
	Rank    int     `json:"rank"`
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Score   float64 `json:"score"`
	Members *int    `json:"members,omitempty"`
	IsMe    bool    `json:"isMe"`
}

type Leaderboard struct {
	Type    string             `json:"type"`
	Entries []LeaderboardEntry `json:"entries"`
}

type Notification struct {
	ID        string     `json:"id"`
	Kind      string     `json:"kind"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	URL       *string    `json:"url"`
	CreatedAt time.Time  `json:"createdAt"`
	ReadAt    *time.Time `json:"readAt"`
}

type StaffStation struct {
	Station       Station    `json:"station"`
	ScanOpen      bool       `json:"scanOpen"`
	ScanOpenUntil *time.Time `json:"scanOpenUntil"`
	ScansCount    int        `json:"scansCount"`
	RatedCount    int        `json:"ratedCount"`
	AvgRating     *float64   `json:"avgRating"`
}

type StationQR struct {
	Payload       string    `json:"payload"`
	ManualCode    string    `json:"manualCode"`
	ValidUntil    time.Time `json:"validUntil"`
	PeriodSeconds int       `json:"periodSeconds"`
	ScanOpen      bool      `json:"scanOpen"`
	ScansCount    int       `json:"scansCount"`
}

type StationPatch struct {
	BasePoints     *int        `json:"basePoints"`
	Weights        game.Vector `json:"weights"`
	TeamMinMembers *int        `json:"teamMinMembers"`
	StaffEmails    []string    `json:"staffEmails"`
}

type AnalyticsStation struct {
	StationID string   `json:"stationId"`
	Name      string   `json:"name"`
	Visits    int      `json:"visits"`
	Rated     int      `json:"rated"`
	AvgRating *float64 `json:"avgRating"`
}

type AnalyticsArchetype struct {
	Code  string `json:"code"`
	Users int    `json:"users"`
}

type Analytics struct {
	Participants int                  `json:"participants"`
	Scans        int                  `json:"scans"`
	Rated        int                  `json:"rated"`
	Stations     []AnalyticsStation   `json:"stations"`
	Archetypes   []AnalyticsArchetype `json:"archetypes"`
}

type PushSubscription struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}
