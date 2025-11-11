package types

import (
	"html/template"
	"log/slog"
	"time"
)

type Conf struct {
	Dev                 bool
	DisableAuth         bool
	Media               string
	Cache               string
	Data                string
	Quality             int
	TranscodeResolution int
	PreGenerateThumb    bool
	ResizeService       string
	LocationService     string
	LocationDataset     string
	Logger              *slog.Logger
	TileServer          string
	SessionLength       int
	IncludeOriginals    bool
	Aliases             struct {
		Lenses map[string]string `yaml:"lenses"`
	} `yaml:"aliases"`
	CustomHTML template.HTML `yaml:"custom_html"`
	Meta       Meta
	OnThisDay  bool
}

type MediaItems []Media

type Media struct {
	Path          string          `json:"path"`
	Subject       Subjects        `json:"subjects"`
	Hash          uint32          `json:"hash"`
	Width         int             `json:"width"`
	Height        int             `json:"height"`
	Ratio         float32         `json:"ratio"`
	Padding       float32         `json:"padding"`
	Date          time.Time       `json:"date"`
	Modified      time.Time       `json:"modified"`
	Folder        string          `json:"folder"`
	Srcset        template.Srcset `json:"srcset"`
	Rating        float64         `json:"rating"`
	ShutterSpeed  string          `json:"shutterspeed"`
	Aperture      float64         `json:"aperture"`
	Iso           float64         `json:"iso"`
	Lens          string          `json:"lens"`
	Camera        string          `json:"camera"`
	Focallength   float64         `json:"focallength"`
	Altitude      float64         `json:"altitude"`
	Latitude      float64         `json:"latitude"`
	Longitude     float64         `json:"longitude"`
	Type          string          `json:"type"`
	FocusDistance float64         `json:"focusDistance"`
	FocalLength35 float64         `json:"focalLength35"`
	Color         string          `json:"color"`
	Location      string          `json:"location"`
	Description   string          `json:"description"`
	Title         string          `json:"title"`
	Software      string          `json:"software"`
	Offset        float64         `json:"offset"`
	Rotation      float64         `json:"-"` // only used for HEIC thumbnail creation
}

type DatabaseMedia struct {
	Hash          uint32
	Path          string
	Subject       string
	Width         int
	Height        int
	Ratio         float32
	Padding       float32
	Date          string
	Modified      string
	Folder        string
	Rating        float64
	ShutterSpeed  string
	Aperture      float64
	Iso           float64
	Lens          string
	Camera        string
	Focallength   float64
	Altitude      float64
	Latitude      float64
	Longitude     float64
	Mediatype     string
	Focusdistance float64
	Focallength35 float64
	Color         string
	Location      string
	Description   string
	Title         string
	Software      string
	Offset        float64
	Rotation      float64 // only used for HEIC thumbnail creation
}

type Subjects []Subject

type Subject struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Days []Day

type Day struct {
	Key   string  `json:"key"`
	Value string  `json:"value"`
	Media []Media `json:"media"`
	Total int     `json:"total"`
}

type Meta struct {
	Commit     string
	Tag        string
	CustomHTML template.HTML
}

type ResponseFilter struct {
	ResponseSegment ResponseSegment `json:"segment"`
	Total           int             `json:"total"`
	PageSize        int             `json:"pagesize"`
	Page            int             `json:"page"`
	OrderBy         string          `json:"orderby"`
	Direction       string          `json:"direction"`
	Section         string          `json:"-"`
	Filter          Filter          `json:"filter"`
	HideNavFooter   bool            `json:"-"`
	// Folders          []string        `json:"folders"`
	// Cameras []string `json:"cameras"`
	// Lenses  []string `json:"lenses"`
	// Ratings []string `json:"ratings"`
	Meta Meta `json:"-"`
}

type Filter struct {
	Camera        string  `json:"camera"`
	Lens          string  `json:"lens"`
	Term          string  `json:"term"`
	Mediatype     string  `json:"type"`
	Rating        int     `json:"rating"`
	Subject       string  `json:"subject"`
	Folder        string  `json:"folder"`
	Software      string  `json:"software"`
	FocalLength35 float64 `json:"focalLength35"`
}

type ResponseMedia struct {
	Media         Media      `json:"media"`
	Previous      []PrevNext `json:"previous"`
	Next          []PrevNext `json:"next"`
	Collection    string     `json:"collection"`
	Slug          string     `json:"slug"`
	Section       string     `json:"-"`
	HideNavFooter bool       `json:"-"`
	TileServer    string     `json:"-"`
	Meta          Meta       `json:"-"`
}

type PrevNext struct {
	Hash      uint32          `json:"hash"`
	Color     string          `json:"color"`
	Mediatype string          `json:"type"`
	Path      string          `json:"path"`
	Width     int             `json:"width"`
	Height    int             `json:"height"`
	Srcset    template.Srcset `json:"srcset"`
	Date      time.Time       `json:"-"`
}

type ResponseMediaItems struct {
	MediaItems    []Media `json:"mediaItems"`
	Title         string  `json:"title"`
	Slug          string  `json:"slug"`
	Total         int     `json:"total"`
	PageSize      int     `json:"pagesize"`
	Page          int     `json:"page"`
	OrderBy       string  `json:"orderby"`
	Direction     string  `json:"direction"`
	Collection    string  `json:"collection"`
	Section       string  `json:"-"`
	HideNavFooter bool    `json:"-"`
	Meta          Meta    `json:"-"`
}

type ResponseFolders struct {
	Folders       []*TreeNode `json:"folders"`
	Title         string      `json:"title"`
	Total         int         `json:"total"`
	PageSize      int         `json:"pagesize"`
	Page          int         `json:"page"`
	OrderBy       string      `json:"orderby"`
	Direction     string      `json:"direction"`
	Collection    string      `json:"collection"`
	Section       string      `json:"-"`
	HideNavFooter bool        `json:"-"`
	Meta          Meta        `json:"-"`
}

type ResponseTags struct {
	Tags          Subjects `json:"tags"`
	Title         string   `json:"title"`
	Total         int      `json:"total"`
	PageSize      int      `json:"pagesize"`
	Page          int      `json:"page"`
	OrderBy       string   `json:"orderby"`
	Direction     string   `json:"direction"`
	Section       string   `json:"-"`
	HideNavFooter bool     `json:"-"`
	Meta          Meta     `json:"-"`
}
type ResponseDates struct {
	Dates         Dates  `json:"dates"`
	Section       string `json:"-"`
	HideNavFooter bool   `json:"-"`
}

type Years []Year

type Year struct {
	Key    string  `json:"key"`
	Months []Month `json:"month"`
}

type Month struct {
	Key int `json:"key"`
}

type Dates map[int][]string

type Date struct {
	Key string `json:"key"`
}

type RawMinimalMedia struct {
	Width     *int     `json:"width"`
	Height    *int     `json:"height"`
	Date      *string  `json:"date"`
	Hash      *uint32  `json:"hash"`
	Color     *string  `json:"color"`
	MediaType *string  `json:"mediatype"`
	Offset    *float64 `json:"-"`
	Modified  *string  `json:"modified"`
}

type ResponseSegment []*SegmentGroup

type SegmentGroup struct { // rename to DayGroup
	SectionId *string    `json:"sectionId"`
	Total     *int       `json:"totalItems"`
	Segments  *[]Segment `json:"segments"`
}

type Segment struct { // rename to MonthGroup
	SegmentId string         `json:"s"`
	Media     []SegmentMedia `json:"i"`
}

type SegmentMedia []interface{}

type GearItem struct {
	Name  string `json:"name"`
	Total int    `json:"total"`
}

type GearItems []GearItem

type ResponseGear struct {
	Cameras       []GearItem `json:"camera"`
	Lenses        []GearItem `json:"lens"`
	FocalLength35 []GearItem `json:"focalLength35"`
	Section       string     `json:"-"`
	Software      []GearItem `json:"software"`
	HideNavFooter bool       `json:"-"`
	Title         string     `json:"-"`
	Meta          Meta       `json:"-"`
}

type ResponsAuth struct {
	HideNavFooter bool
	Section       string
}

type ResponseAdmin struct {
	HideNavFooter bool
	HideAuth      bool
	Section       string
	Key           ApiCredentials
	Keys          []ApiCredentials
	Users         []User
	UserName      string
	UserRole      string
	Meta          Meta `json:"-"`
}

type ResponseNotFound struct {
	Title         string `json:"-"`
	Message       string `json:"-"`
	Section       string `json:"-"`
	HideNavFooter bool   `json:"-"`
}

type FilterParams struct {
	PageSize      int
	Json          bool
	Page          int
	Rating        int
	Direction     string
	From          string
	To            string
	Camera        string
	Lens          string
	MediaType     string
	Term          string
	OrderBy       string
	Folder        string
	Subject       string
	Software      string
	FocalLength35 float64
}

type Folder struct {
	Key    string      `json:"key"`
	Parent string      `json:"parent"`
	Media  FolderMedia `json:"media"`
}

type TreeNode struct {
	Id         int            `json:"id,omitempty"`
	Name       string         `json:"name"`
	Path       string         `json:"path"`
	Media      *[]FolderMedia `json:"media,omitempty"`
	Children   []*TreeNode    `json:"children,omitempty"`
	ImageCount int            `json:"imageCount"`
}

type FolderMedia struct {
	Hash   uint32          `json:"hash"`
	Path   string          `json:"path"`
	Width  int             `json:"width"`
	Height int             `json:"height"`
	Color  string          `json:"color"`
	Srcset template.Srcset `json:"srcset"`
}

type Directory struct {
	Id         int
	Key        string
	Media      *[]FolderMedia // Stores up to 5 most recent items
	ImageCount int            `json:"imageCount"`
}

type UserCredentials struct {
	Password string `json:"-"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

type User struct {
	Username string `json:"username"`
	Role     string `json:"role"`
}

type ApiCredentials struct {
	Name    string    `json:"name"`
	Key     string    `json:"key"`
	Created time.Time `json:"created"`
}

type CacheKey struct{}
type ConfigKey struct{}
type ParamsKey struct{}
type UserKey struct {
	UserName string
	UserRole string
}

type ResponseMap struct {
	Section       string    `json:"-"`
	MapItems      []MapItem `json:"mapItems"`
	HideNavFooter bool      `json:"-"`
	TileServer    string    `json:"-"`
	Meta          Meta      `json:"-"`
}

// short hand json properties to limit response size on large responses.
type MapItem []float64
