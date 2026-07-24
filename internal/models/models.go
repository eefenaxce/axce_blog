package models

import (
	"time"
)

type User struct {
	ID              int        `json:"id"`
	Username        string     `json:"username"`
	Email           string     `json:"email"`
	PasswordHash    string     `json:"-"`
	Nickname        string     `json:"nickname"`
	Avatar          string     `json:"avatar"`
	Bio             string     `json:"bio"`
	Group           string     `json:"group"`
	Status          int        `json:"status"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

type Article struct {
	ID             int         `json:"id"`
	Title          string      `json:"title"`
	Slug           string      `json:"slug"`
	Summary        string      `json:"summary"`
	Content        string      `json:"content"`
	CoverURL       string      `json:"coverUrl"`
	UserID         int         `json:"authorId"`
	Status         string      `json:"status"`
	CommentEnabled bool        `json:"commentEnabled"`
	ViewCount      int         `json:"viewCount"`
	UpvoteCount    int         `json:"upvoteCount"`
	Tags           []*Tag      `json:"tags,omitempty"`
	Categories     []*Category `json:"categories,omitempty"`
	CreatedAt      time.Time   `json:"createdAt"`
	UpdatedAt      time.Time   `json:"updatedAt"`
	DeletedAt      *time.Time  `json:"deletedAt,omitempty"`
}

type Category struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	Icon        string    `json:"icon"`
	CreatedAt   time.Time `json:"createdAt"`
}

type Tag struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
	Icon string `json:"icon"`
}

type ArticleTag struct {
	ArticleID int `json:"articleId"`
	TagID     int `json:"tagId"`
}

type Comment struct {
	ID           int        `json:"id"`
	ArticleID    int        `json:"articleId"`
	ParentID     *int       `json:"parentId,omitempty"`
	UserID       *int       `json:"userId,omitempty"`
	AuthorName   string     `json:"authorName"`
	AuthorEmail  string     `json:"authorEmail"`
	AuthorURL    string     `json:"authorUrl"`
	AuthorAvatar string     `json:"authorAvatar"`
	Content      string     `json:"content"`
	Status       string     `json:"status"`
	IPAddress    string     `json:"ipAddress,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	DeletedAt    *time.Time `json:"deletedAt,omitempty"`
}

type Menu struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type MenuItem struct {
	ID       int    `json:"id"`
	MenuID   int    `json:"menuId"`
	Name     string `json:"name"`
	URL      string `json:"url"`
	ParentID *int   `json:"parentId,omitempty"`
	Priority int    `json:"priority"`
}

type Setting struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description"`
}

type Theme struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Author      string `json:"author"`
	Description string `json:"description"`
	Screenshot  string `json:"screenshot"`
	Active      bool   `json:"active"`
	// Halo metadata
	HasSettings   bool   `json:"hasSettings"`
	SettingName   string `json:"settingName,omitempty"`
	ConfigMapName string `json:"configMapName,omitempty"`
	Repo          string `json:"repo,omitempty"`
	Homepage      string `json:"homepage,omitempty"`
}

type ThemeListItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Author      string `json:"author"`
	Description string `json:"description"`
	Screenshot  string `json:"screenshot"`
	Active      bool   `json:"active"`
	Installed   bool   `json:"installed"`
	RepoURL     string `json:"repoUrl"`
	Homepage    string `json:"homepage"`
}

// ─── Halo settings.yaml 解析结构 ───

// HaloSettingDef 表示 Halo 主题 settings.yaml 的完整定义
type HaloSettingDef struct {
	APIVersion string            `yaml:"apiVersion"`
	Kind       string            `yaml:"kind"`
	Metadata   HaloSettingMeta   `yaml:"metadata"`
	Spec       HaloSettingSpec   `yaml:"spec"`
}

type HaloSettingMeta struct {
	Name string `yaml:"name"`
}

type HaloSettingSpec struct {
	Forms []HaloSettingForm `yaml:"forms"`
}

// HaloSettingForm 一个设置分组（如"基本设置"、"评论设置"）
type HaloSettingForm struct {
	Group       string               `yaml:"group" json:"group"`
	Label       string               `yaml:"label" json:"label"`
	FormSchema  []HaloSettingField   `yaml:"formSchema" json:"formSchema"`
}

// HaloSettingField 单个设置字段，使用 map 灵活存储 FormKit 格式
type HaloSettingField struct {
	Formkit   string             `yaml:"$formkit" json:"$formkit"`
	Name      string             `yaml:"name" json:"name"`
	ID        string             `yaml:"id" json:"id,omitempty"`
	Key       string             `yaml:"key" json:"key,omitempty"`
	Label     string             `yaml:"label" json:"label"`
	Value     any                `yaml:"value" json:"value"`
	Help      string             `yaml:"help" json:"help,omitempty"`
	If        string             `yaml:"if" json:"if,omitempty"`
	Options   []HaloSettingOption `yaml:"options" json:"options,omitempty"`
	Children  []HaloSettingField  `yaml:"children" json:"children,omitempty"`
	Min       *float64           `yaml:"min" json:"min,omitempty"`
	Max       *float64           `yaml:"max" json:"max,omitempty"`
	Step      *float64           `yaml:"step" json:"step,omitempty"`
	Rows      *int               `yaml:"rows" json:"rows,omitempty"`
	Placeholder string           `yaml:"placeholder" json:"placeholder,omitempty"`
	Validation string            `yaml:"validation" json:"validation,omitempty"`
	Accept    string             `yaml:"accept" json:"accept,omitempty"`
}

type HaloSettingOption struct {
	Value any    `yaml:"value" json:"value"`
	Label string `yaml:"label" json:"label"`
}

// ThemeSettingsResponse 返回给前端的主题设置（表单定义 + 当前值）
type ThemeSettingsResponse struct {
	SettingName string             `json:"settingName"`
	Forms       []HaloSettingForm  `json:"forms"`
	Values      map[string]any     `json:"values"`
}

