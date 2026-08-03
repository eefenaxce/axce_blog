package service

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/eefenaxce/axce_blog/internal/models"
	"github.com/eefenaxce/axce_blog/internal/utils"
	"gopkg.in/yaml.v3"
)

const (
	themesDir          = "./themes"
	activeThemeSetting = "active_theme"
	githubSearchURL    = "https://api.github.com/search/repositories"
)

var (
	ErrThemeNotFound    = errors.New("主题不存在")
	ErrInvalidTheme     = errors.New("无效的主题包")
	ErrThemeExtractFail = errors.New("主题解压失败")
	ErrThemeExists      = errors.New("主题已安装，如需覆盖请先删除")
)

type ThemeService struct {
	settingService *SettingService
	redisClient    *utils.RedisClient
}

func NewThemeService(settingService *SettingService, redisClient *utils.RedisClient) *ThemeService {
	return &ThemeService{settingService: settingService, redisClient: redisClient}
}

func (s *ThemeService) Upload(reader io.Reader, filename string) (*models.Theme, error) {
	tmpFile, err := os.CreateTemp("", "theme-*.zip")
	if err != nil {
		return nil, fmt.Errorf("创建临时文件失败: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := io.Copy(tmpFile, reader); err != nil {
		return nil, fmt.Errorf("写入临时文件失败: %w", err)
	}
	tmpFile.Close()

	fallbackID := strings.TrimSuffix(filepath.Base(filename), ".zip")
	return s.installFromZip(tmpFile.Name(), fallbackID, false)
}

// DownloadProgress 单个主题下载进度
type DownloadProgress struct {
	Repo   string        `json:"repo"`
	Pct    int           `json:"pct"`
	Status string        `json:"status"` // "downloading" | "extracting" | "completed" | "error"
	Error  string        `json:"error,omitempty"`
	Theme  *models.Theme `json:"theme,omitempty"`
}

func (s *ThemeService) dlKey(repo, field string) string {
	return fmt.Sprintf("theme_dl:%s:%s", repo, field)
}

// DownloadFromURL 异步从 GitHub 下载主题，立即返回 repo 名，进度写入 Redis
func (s *ThemeService) DownloadFromURL(repoURL string) (string, error) {
	u, err := url.Parse(repoURL)
	if err != nil {
		return "", fmt.Errorf("无效的仓库地址: %w", err)
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 {
		return "", errors.New("无效的仓库地址")
	}
	owner, repo := parts[0], parts[1]

	if s.redisClient != nil {
		// 检查是否已在下载中
		if status, _ := s.redisClient.Get(context.Background(), s.dlKey(repo, "status")); status == "downloading" || status == "extracting" {
			return repo, nil
		}
		// 初始化进度
		ctx := context.Background()
		s.redisClient.Set(ctx, s.dlKey(repo, "pct"), 0, 10*time.Minute)
		s.redisClient.Set(ctx, s.dlKey(repo, "status"), "downloading", 10*time.Minute)
	}

	go func() {
		s.downloadAsync(owner, repo)
	}()
	return repo, nil
}

// githubRelease 表示 GitHub Release API 响应中我们关心的字段
type githubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
		Size               int64  `json:"size"`
	} `json:"assets"`
}

// fetchLatestReleaseAsset 获取仓库最新 Release 的 zip 资源下载地址。
// 返回 downloadURL、资源文件名和大小；如果没有 Release 或没有 zip 资源则返回空。
func fetchLatestReleaseAsset(owner, repo string) (downloadURL string, filename string, size int64) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", 0
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", 0
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", 0
	}

	var release githubRelease
	if err := json.Unmarshal(body, &release); err != nil {
		return "", "", 0
	}

	// 优先选择 .zip 资源
	for _, asset := range release.Assets {
		if strings.HasSuffix(strings.ToLower(asset.Name), ".zip") {
			return asset.BrowserDownloadURL, asset.Name, asset.Size
		}
	}

	// 如果没有 .zip，接受第一个资源
	if len(release.Assets) > 0 {
		return release.Assets[0].BrowserDownloadURL, release.Assets[0].Name, release.Assets[0].Size
	}

	return "", "", 0
}

// downloadFile 下载文件到临时目录，支持进度回调，返回临时文件路径
func downloadFile(url string, pctCallback func(downloaded, total int64)) (tmpPath string, err error) {
	req, _ := http.NewRequest("GET", url, nil)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("下载失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载失败: HTTP %d", resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp("", "theme-download-*.zip")
	if err != nil {
		return "", fmt.Errorf("创建临时文件失败: %w", err)
	}
	tmpPath = tmpFile.Name()
	tmpFile.Close()

	f, err := os.Create(tmpPath)
	if err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("创建临时文件失败: %w", err)
	}

	totalSize := resp.ContentLength
	var downloaded int64
	lastUpdate := time.Now()
	buf := make([]byte, 32*1024)

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, err := f.Write(buf[:n]); err != nil {
				f.Close()
				os.Remove(tmpPath)
				return "", fmt.Errorf("写入失败: %w", err)
			}
			downloaded += int64(n)

			if pctCallback != nil {
				now := time.Now()
				if now.Sub(lastUpdate) > 100*time.Millisecond {
					lastUpdate = now
					pctCallback(downloaded, totalSize)
				}
			}
		}
		if readErr != nil {
			if readErr != io.EOF {
				f.Close()
				os.Remove(tmpPath)
				return "", fmt.Errorf("读取失败: %w", readErr)
			}
			break
		}
	}
	f.Close()

	if pctCallback != nil {
		pctCallback(downloaded, totalSize)
	}

	return tmpPath, nil
}

// downloadAsync 异步下载主题 zip 并安装。
// 优先从 GitHub Releases 下载构建包（与 Halo 行为一致），
// 如果没有 Release 则回退到下载源码归档。
func (s *ThemeService) downloadAsync(owner, repo string) {
	ctx := context.Background()
	statusKey := s.dlKey(repo, "status")
	pctKey := s.dlKey(repo, "pct")
	errKey := s.dlKey(repo, "error")
	themeKey := s.dlKey(repo, "theme")

	update := func(pct int, status string) {
		if s.redisClient == nil {
			return
		}
		s.redisClient.Set(ctx, pctKey, pct, 10*time.Minute)
		s.redisClient.Set(ctx, statusKey, status, 10*time.Minute)
	}
	setError := func(msg string) {
		update(0, "error")
		if s.redisClient != nil {
			s.redisClient.Set(ctx, errKey, msg, 10*time.Minute)
		}
	}

	defer func() {
		if r := recover(); r != nil {
			setError(fmt.Sprintf("panic: %v", r))
		}
	}()

	update(1, "checking")

	var tmpPath string

	releaseURL, releaseFile, releaseSize := fetchLatestReleaseAsset(owner, repo)
	if releaseURL != "" {
		log.Printf("[downloadAsync] 发现 Release 资源: %s (%d bytes)，从 Release 下载", releaseFile, releaseSize)
		update(2, "downloading")

		var dlErr error
		tmpPath, dlErr = downloadFile(releaseURL, func(downloaded, total int64) {
			pct := downloadPct(downloaded, total)
			update(pct, "downloading")
		})
		if dlErr != nil {
			log.Printf("[downloadAsync] Release 下载失败: %v，回退到源码下载", dlErr)
		}
	}

	if tmpPath == "" {
		setError(fmt.Sprintf("无法获取 %s/%s", owner, repo))
		return
	}

	// 解压 — 真实文件计数进度
	update(50, "extracting")
	theme, installErr := s.installFromZipWithProgress(tmpPath, repo, true,
		func(extracted, total int) {
			if total > 0 {
				pct := 50 + extracted*45/total
				update(pct, "extracting")
			}
		})
	os.Remove(tmpPath)

	if installErr != nil {
		setError(installErr.Error())
		return
	}

	update(100, "completed")
	themeJSON, _ := json.Marshal(theme)
	if s.redisClient != nil {
		s.redisClient.Set(ctx, themeKey, themeJSON, 10*time.Minute)
		s.redisClient.Set(ctx, s.dlKey(repo, "folderId"), theme.ID, 24*time.Hour)
		s.redisClient.DeleteByPattern(ctx, "remote_themes:*")
	}
}

func downloadPct(downloaded, total int64) int {
	if total > 0 {
		pct := int(downloaded * 50 / total)
		if pct < 2 {
			pct = 2
		}
		return pct
	}
	// 无 Content-Length：按 100KB/1% 估算，封顶 45%
	pct := int(downloaded / (100 * 1024))
	if pct > 45 {
		pct = 45
	}
	if pct < 2 {
		pct = 2
	}
	return pct
}

// GetDownloadProgress 读取下载进度
func (s *ThemeService) GetDownloadProgress(repo string) *DownloadProgress {
	if s.redisClient == nil {
		return nil
	}
	ctx := context.Background()
	status, err := s.redisClient.Get(ctx, s.dlKey(repo, "status"))
	if err != nil || status == "" {
		return nil
	}
	pctStr, _ := s.redisClient.Get(ctx, s.dlKey(repo, "pct"))
	pct, _ := strconv.Atoi(pctStr)

	dp := &DownloadProgress{Repo: repo, Pct: pct, Status: status}

	if status == "error" {
		dp.Error, _ = s.redisClient.Get(ctx, s.dlKey(repo, "error"))
	}
	if status == "completed" {
		if themeJSON, err := s.redisClient.Get(ctx, s.dlKey(repo, "theme")); err == nil {
			var t models.Theme
			json.Unmarshal([]byte(themeJSON), &t)
			dp.Theme = &t
		}
	}

	return dp
}

// installFromZip 从 zip 文件安装主题到 themes 目录
// overwrite=false 时，若主题已安装则返回 ErrThemeExists
func (s *ThemeService) installFromZip(zipPath string, fallbackID string, overwrite bool) (*models.Theme, error) {
	zipReader, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, ErrInvalidTheme
	}
	defer zipReader.Close()

	var themeJSON *zip.File
	var rootDir string

	for _, f := range zipReader.File {
		if f.FileInfo().IsDir() {
			if rootDir == "" {
				rootDir = f.Name
			}
			continue
		}
		if filepath.Base(f.Name) == "theme.json" || filepath.Base(f.Name) == "theme.yaml" {
			themeJSON = f
			rootDir = filepath.Dir(f.Name) + "/"
			break
		}
	}

	if themeJSON == nil {
		return nil, ErrInvalidTheme
	}

	rc, err := themeJSON.Open()
	if err != nil {
		return nil, fmt.Errorf("读取 theme.json 失败: %w", err)
	}
	defer rc.Close()

	themeData, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("读取主题元数据失败: %w", err)
	}

	metaFileName := filepath.Base(themeJSON.Name)
	var themeMeta struct {
		ID          string `json:"id" yaml:"id"`
		Name        string `json:"name" yaml:"name"`
		Version     string `json:"version" yaml:"version"`
		Author      string `json:"author" yaml:"author"`
		Description string `json:"description" yaml:"description"`
	}
	if strings.HasSuffix(metaFileName, ".yaml") || strings.HasSuffix(metaFileName, ".yml") {
		yaml.Unmarshal(themeData, &themeMeta)
	} else {
		json.Unmarshal(themeData, &themeMeta)
	}

	if themeMeta.ID == "" {
		themeMeta.ID = fallbackID
	}
	if themeMeta.Name == "" {
		themeMeta.Name = themeMeta.ID
	}

	if err := os.MkdirAll(themesDir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("创建主题目录失败: %w", err)
	}

	destDir := filepath.Join(themesDir, themeMeta.ID)
	if _, err := os.Stat(destDir); !os.IsNotExist(err) {
		if !overwrite {
			return nil, fmt.Errorf("%w: %s", ErrThemeExists, themeMeta.Name)
		}
		os.RemoveAll(destDir)
	}

	fileCount := 0
	for _, f := range zipReader.File {
		relPath := strings.TrimPrefix(f.Name, rootDir)
		if relPath == "" {
			continue
		}
		targetPath := filepath.Join(destDir, relPath)
		if f.FileInfo().IsDir() {
			os.MkdirAll(targetPath, os.ModePerm)
			continue
		}
		os.MkdirAll(filepath.Dir(targetPath), os.ModePerm)
		src, err := f.Open()
		if err != nil {
			continue
		}
		dst, err := os.Create(targetPath)
		if err != nil {
			src.Close()
			continue
		}
		io.Copy(dst, src)
		dst.Close()
		src.Close()
		fileCount++
	}

	log.Printf("[installFromZip] done: %d files extracted to %s", fileCount, destDir)

	return &models.Theme{
		ID:          themeMeta.ID,
		Name:        themeMeta.Name,
		Version:     themeMeta.Version,
		Author:      themeMeta.Author,
		Description: themeMeta.Description,
		Screenshot:  "/api/v1/admin/themes/" + themeMeta.ID + "/screenshot",
		Active:      false,
	}, nil
}

// installFromZipWithProgress 与 installFromZip 相同，但通过回调报告解压进度
func (s *ThemeService) installFromZipWithProgress(zipPath string, fallbackID string, overwrite bool, onProgress func(extracted, total int)) (*models.Theme, error) {
	zipReader, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, ErrInvalidTheme
	}
	defer zipReader.Close()

	var themeJSON *zip.File
	var rootDir string

	for _, f := range zipReader.File {
		if f.FileInfo().IsDir() {
			if rootDir == "" {
				rootDir = f.Name
			}
			continue
		}
		if filepath.Base(f.Name) == "theme.json" || filepath.Base(f.Name) == "theme.yaml" {
			themeJSON = f
			rootDir = filepath.Dir(f.Name) + "/"
			break
		}
	}

	if themeJSON == nil {
		return nil, ErrInvalidTheme
	}

	rc, err := themeJSON.Open()
	if err != nil {
		return nil, fmt.Errorf("读取 theme.json 失败: %w", err)
	}
	defer rc.Close()

	themeData, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("读取主题元数据失败: %w", err)
	}

	metaFileName := filepath.Base(themeJSON.Name)
	var themeMeta struct {
		ID          string `json:"id" yaml:"id"`
		Name        string `json:"name" yaml:"name"`
		Version     string `json:"version" yaml:"version"`
		Author      string `json:"author" yaml:"author"`
		Description string `json:"description" yaml:"description"`
	}
	if strings.HasSuffix(metaFileName, ".yaml") || strings.HasSuffix(metaFileName, ".yml") {
		yaml.Unmarshal(themeData, &themeMeta)
	} else {
		json.Unmarshal(themeData, &themeMeta)
	}

	if themeMeta.ID == "" {
		themeMeta.ID = fallbackID
	}
	if themeMeta.Name == "" {
		themeMeta.Name = themeMeta.ID
	}

	if err := os.MkdirAll(themesDir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("创建主题目录失败: %w", err)
	}

	destDir := filepath.Join(themesDir, themeMeta.ID)
	if _, err := os.Stat(destDir); !os.IsNotExist(err) {
		if !overwrite {
			return nil, fmt.Errorf("%w: %s", ErrThemeExists, themeMeta.Name)
		}
		os.RemoveAll(destDir)
	}

	// 先统计需要解压的文件数
	totalFiles := 0
	for _, f := range zipReader.File {
		relPath := strings.TrimPrefix(f.Name, rootDir)
		if relPath == "" || f.FileInfo().IsDir() {
			continue
		}
		totalFiles++
	}

	extracted := 0
	fileCount := 0
	lastReport := 0
	for _, f := range zipReader.File {
		relPath := strings.TrimPrefix(f.Name, rootDir)
		if relPath == "" {
			continue
		}
		targetPath := filepath.Join(destDir, relPath)
		if f.FileInfo().IsDir() {
			os.MkdirAll(targetPath, os.ModePerm)
			continue
		}
		os.MkdirAll(filepath.Dir(targetPath), os.ModePerm)
		src, err := f.Open()
		if err != nil {
			continue
		}
		dst, err := os.Create(targetPath)
		if err != nil {
			src.Close()
			continue
		}
		io.Copy(dst, src)
		dst.Close()
		src.Close()
		fileCount++
		extracted++

		// 每 5% 或每 20 个文件报告一次
		if onProgress != nil && totalFiles > 0 {
			pct := extracted * 100 / totalFiles
			if pct-lastReport >= 5 || extracted-lastReport*20/totalFiles >= 20 {
				lastReport = pct
				onProgress(extracted, totalFiles)
			}
		}
	}

	if onProgress != nil {
		onProgress(extracted, totalFiles)
	}

	return &models.Theme{
		ID:          themeMeta.ID,
		Name:        themeMeta.Name,
		Version:     themeMeta.Version,
		Author:      themeMeta.Author,
		Description: themeMeta.Description,
		Screenshot:  "/api/v1/admin/themes/" + themeMeta.ID + "/screenshot",
		Active:      false,
	}, nil
}

func (s *ThemeService) List(ctx context.Context) ([]models.Theme, error) {
	var themes []models.Theme
	activeTheme, _ := s.settingService.Get(ctx, activeThemeSetting)

	if err := os.MkdirAll(themesDir, os.ModePerm); err != nil {
		return themes, nil
	}

	entries, err := os.ReadDir(themesDir)
	if err != nil {
		return themes, nil
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		themeDirPath := filepath.Join(themesDir, entry.Name())
		theme, err := s.readTheme(themeDirPath)
		if err != nil {
			continue
		}
		dirName := filepath.Base(themeDirPath)
		theme.Active = theme.ID == activeTheme || dirName == activeTheme
		themes = append(themes, *theme)
	}

	return themes, nil
}

// resolveThemeDir 根据主题 ID 查找实际磁盘目录路径
func (s *ThemeService) resolveThemeDir(id string) (string, error) {
	// 先直接匹配目录名（兼容目录名 == ID 的情况）
	direct := filepath.Join(themesDir, id)
	if _, err := os.Stat(direct); err == nil {
		return direct, nil
	}
	// 遍历主题目录，匹配 theme.yaml/theme.json 中的 id
	entries, err := os.ReadDir(themesDir)
	if err != nil {
		return "", ErrThemeNotFound
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dirPath := filepath.Join(themesDir, entry.Name())
		theme, readErr := s.readTheme(dirPath)
		if readErr != nil {
			continue
		}
		if theme.ID == id {
			return dirPath, nil
		}
	}
	return "", ErrThemeNotFound
}

func (s *ThemeService) Activate(ctx context.Context, id string) error {
	themeDir, err := s.resolveThemeDir(id)
	if err != nil {
		return err
	}

	// 解析主题元数据获取 settingName
	theme, err := s.readTheme(themeDir)
	if err != nil {
		return err
	}

	// 如果主题有 settings.yaml，初始化默认配置到数据库
	if theme.SettingName != "" {
		settingDef, err := s.ParseSettingsYAML(themeDir)
		if err == nil && settingDef != nil {
			// 检查是否已有配置（避免覆盖用户已修改的配置）
			existing, _ := s.getThemeConfig(ctx, theme.SettingName)
			if len(existing) == 0 {
				defaults := s.ExtractDefaults(settingDef)
				s.saveThemeConfig(ctx, theme.SettingName, defaults)
				log.Printf("[Activate] 初始化主题 %s 的默认配置 (%d 项)", theme.Name, len(defaults))
			}
		}
	}

	return s.settingService.Set(ctx, activeThemeSetting, id, "当前激活的主题")
}

func (s *ThemeService) Delete(ctx context.Context, id string) error {
	themeDir, err := s.resolveThemeDir(id)
	if err != nil {
		return err
	}

	activeTheme, _ := s.settingService.Get(ctx, activeThemeSetting)
	if activeTheme == id {
		s.settingService.Set(ctx, activeThemeSetting, "", "")
	}

	return os.RemoveAll(themeDir)
}

const themeConfigPrefix = "theme_config:"

// ParseSettingsYAML 解析主题目录下的 settings.yaml
func (s *ThemeService) ParseSettingsYAML(themeDir string) (*models.HaloSettingDef, error) {
	settingPath := filepath.Join(themeDir, "settings.yaml")
	data, err := os.ReadFile(settingPath)
	if err != nil {
		return nil, err
	}

	var def models.HaloSettingDef
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("解析 settings.yaml 失败: %w", err)
	}

	if def.Kind != "Setting" {
		return nil, fmt.Errorf("settings.yaml 格式不正确，期望 kind=Setting，得到 %s", def.Kind)
	}

	return &def, nil
}

// getThemeConfig 从数据库读取主题配置
func (s *ThemeService) getThemeConfig(ctx context.Context, settingName string) (map[string]any, error) {
	key := themeConfigPrefix + settingName
	val, err := s.settingService.Get(ctx, key)
	if err != nil || val == "" {
		return nil, err
	}

	var config map[string]any
	if err := json.Unmarshal([]byte(val), &config); err != nil {
		return nil, err
	}
	return config, nil
}

// saveThemeConfig 保存主题配置到数据库
func (s *ThemeService) saveThemeConfig(ctx context.Context, settingName string, values map[string]any) error {
	key := themeConfigPrefix + settingName
	data, err := json.Marshal(values)
	if err != nil {
		return fmt.Errorf("序列化主题配置失败: %w", err)
	}
	return s.settingService.Set(ctx, key, string(data), "主题配置: "+settingName)
}

// ExtractDefaults 从 HaloSettingDef 中提取所有字段的默认值
// 字段名以分组名为前缀，例如 group=style, name=color_scheme → "style.color_scheme"
// 这样模板可以通过 ${theme.config.style.color_scheme} 访问。
func (s *ThemeService) ExtractDefaults(def *models.HaloSettingDef) map[string]any {
	values := make(map[string]any)
	for _, form := range def.Spec.Forms {
		s.collectDefaults(form.Group, form.FormSchema, values)
	}
	return values
}

// collectDefaults 递归收集字段默认值，键格式为 "group.fieldName"
func (s *ThemeService) collectDefaults(group string, fields []models.HaloSettingField, values map[string]any) {
	for _, field := range fields {
		prefix := group
		// group 类型的字段会引入一层嵌套命名空间
		if field.Formkit == "group" && field.Name != "" {
			if prefix == "" {
				prefix = field.Name
			} else {
				prefix = prefix + "." + field.Name
			}
		}

		if field.Name != "" && field.Value != nil && field.Formkit != "group" {
			key := field.Name
			if prefix != "" {
				key = prefix + "." + field.Name
			}
			values[key] = field.Value
		}
		if len(field.Children) > 0 {
			s.collectDefaults(prefix, field.Children, values)
		}
	}
}

// GetThemeSettings 获取主题的完整设置（表单定义 + 当前值）
func (s *ThemeService) GetThemeSettings(ctx context.Context, themeID string) (*models.ThemeSettingsResponse, error) {
	themeDir, err := s.resolveThemeDir(themeID)
	if err != nil {
		return nil, err
	}

	theme, err := s.readTheme(themeDir)
	if err != nil {
		return nil, err
	}

	if theme.SettingName == "" {
		return nil, fmt.Errorf("该主题没有设置项")
	}

	settingDef, err := s.ParseSettingsYAML(themeDir)
	if err != nil {
		return nil, err
	}

	// 读取当前配置值
	values, _ := s.getThemeConfig(ctx, theme.SettingName)
	if values == nil {
		values = make(map[string]any)
	}

	return &models.ThemeSettingsResponse{
		SettingName: theme.SettingName,
		Forms:       settingDef.Spec.Forms,
		Values:      values,
	}, nil
}

// UpdateThemeSetting 更新主题的单个或多个设置
func (s *ThemeService) UpdateThemeSetting(ctx context.Context, themeID string, updates map[string]any) error {
	themeDir, err := s.resolveThemeDir(themeID)
	if err != nil {
		return err
	}

	theme, err := s.readTheme(themeDir)
	if err != nil {
		return err
	}

	if theme.SettingName == "" {
		return fmt.Errorf("该主题没有设置项")
	}

	// 读取现有配置
	values, _ := s.getThemeConfig(ctx, theme.SettingName)
	if values == nil {
		values = make(map[string]any)
	}

	// 合并更新
	// 清理旧带点键（如风格项更新 "primary_color" 时删除已有的 "style.primary_color" 等），
	// 避免 DB 中同时存在带点和无点键导致 buildThemeObject 渲染时出现非确定性覆盖。
	for k, v := range updates {
		suffix := "." + k
		for existingKey := range values {
			if strings.HasSuffix(existingKey, suffix) {
				delete(values, existingKey)
			}
		}
		values[k] = v
	}

	return s.saveThemeConfig(ctx, theme.SettingName, values)
}

// GetActiveTheme 获取当前激活主题的信息（公开接口）
func (s *ThemeService) GetActiveTheme(ctx context.Context) (*models.Theme, error) {
	activeID, err := s.settingService.Get(ctx, activeThemeSetting)
	if err != nil || activeID == "" {
		return nil, ErrThemeNotFound
	}

	themeDir, err := s.resolveThemeDir(activeID)
	if err != nil {
		return nil, err
	}

	theme, err := s.readTheme(themeDir)
	if err != nil {
		return nil, err
	}

	theme.Active = true
	return theme, nil
}

// GetThemeDir 根据主题 ID 返回主题磁盘目录路径（公开方法）
func (s *ThemeService) GetThemeDir(id string) (string, error) {
	return s.resolveThemeDir(id)
}

func (s *ThemeService) ServeThemeAsset(id, assetPath string) (string, string, error) {
	themeDir, err := s.resolveThemeDir(id)
	if err != nil {
		return "", "", err
	}
	// Try theme root first, then templates/ subdirectory (Halo convention)
	fullPath := filepath.Join(themeDir, assetPath)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		// Try templates/ subdirectory
		fullPath = filepath.Join(themeDir, "templates", assetPath)
	}
	absThemeDir, _ := filepath.Abs(themeDir)
	absAsset, _ := filepath.Abs(fullPath)
	if !strings.HasPrefix(absAsset, absThemeDir) {
		return "", "", errors.New("invalid path")
	}
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return "", "", ErrThemeNotFound
	}

	ext := strings.ToLower(filepath.Ext(fullPath))
	contentType := "application/octet-stream"
	switch ext {
	case ".png":
		contentType = "image/png"
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".svg":
		contentType = "image/svg+xml"
	case ".css":
		contentType = "text/css"
	case ".js":
		contentType = "application/javascript"
	}
	return fullPath, contentType, nil
}

func (s *ThemeService) FetchRemote(keyword string, page, size int) ([]models.ThemeListItem, int, error) {
	q := "topic:halo-theme"
	if keyword != "" {
		q += " " + keyword
	}

	// 尝试从 Redis 缓存读取
	cacheKey := fmt.Sprintf("remote_themes:%s:%d:%d", keyword, page, size)
	if s.redisClient != nil {
		if cached, err := s.redisClient.Get(context.Background(), cacheKey); err == nil && cached != "" {
			var cacheData struct {
				Themes []models.ThemeListItem `json:"themes"`
				Total  int                    `json:"total"`
			}
			if json.Unmarshal([]byte(cached), &cacheData) == nil {
				// 缓存命中后仍需实时扫描本地目录，覆盖 Installed 字段
				installedMap := s.buildInstalledMap()
				for i := range cacheData.Themes {
					cacheData.Themes[i].Installed = s.isThemeInstalled(cacheData.Themes[i].ID, installedMap)
				}
				return cacheData.Themes, cacheData.Total, nil
			}
		}
	}

	params := url.Values{}
	params.Set("q", q)
	params.Set("sort", "stars")
	params.Set("order", "desc")
	params.Set("page", strconv.Itoa(page+1))
	params.Set("per_page", strconv.Itoa(size))

	apiURL := githubSearchURL + "?" + params.Encode()

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("获取远程主题列表失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("GitHub API 返回异常状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("读取远程数据失败: %w", err)
	}

	var result struct {
		TotalCount int `json:"total_count"`
		Items      []struct {
			Name        string `json:"name"`
			FullName    string `json:"full_name"`
			Description string `json:"description"`
			HTMLURL     string `json:"html_url"`
			Homepage    string `json:"homepage"`
			Owner       struct {
				Login string `json:"login"`
			} `json:"owner"`
			DefaultBranch string `json:"default_branch"`
		} `json:"items"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, 0, fmt.Errorf("解析远程数据失败: %w", err)
	}

	// 读取本地主题目录
	installedMap := s.buildInstalledMap()

	var themes []models.ThemeListItem
	for _, item := range result.Items {
		// 使用主题仓库根目录的 screenshot.png 作为封面
		branch := item.DefaultBranch
		if branch == "" {
			branch = "main"
		}
		screenshot := fmt.Sprintf("https://gh.jasonzeng.dev/https://raw.githubusercontent.com/%s/%s/screenshot.png", item.FullName, branch)
		homepage := item.Homepage
		if homepage == "" {
			homepage = item.HTMLURL
		}

		themes = append(themes, models.ThemeListItem{
			ID:          item.Name,
			Name:        item.Name,
			Version:     "",
			Author:      item.Owner.Login,
			Description: item.Description,
			Screenshot:  screenshot,
			Installed:   s.isThemeInstalled(item.Name, installedMap),
			Active:      false,
			RepoURL:     item.HTMLURL,
			Homepage:    homepage,
		})
	}

	if themes == nil {
		themes = make([]models.ThemeListItem, 0)
	}

	// 写入 Redis 缓存（5 分钟过期）
	if s.redisClient != nil {
		cacheData, _ := json.Marshal(map[string]interface{}{
			"themes": themes,
			"total":  result.TotalCount,
		})
		_ = s.redisClient.Set(context.Background(), cacheKey, cacheData, 5*time.Minute)
	}

	return themes, result.TotalCount, nil
}

// buildInstalledMap 扫描本地主题目录，返回所有本地文件夹名的集合（仅包含 theme.json 存在的目录）
func (s *ThemeService) buildInstalledMap() map[string]bool {
	m := make(map[string]bool)
	entries, err := os.ReadDir(themesDir)
	if err != nil {
		return m
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		// 只有存在 theme.json 或 theme.yaml 才算有效安装
		if _, err := os.Stat(filepath.Join(themesDir, entry.Name(), "theme.json")); err != nil {
			if _, err := os.Stat(filepath.Join(themesDir, entry.Name(), "theme.yaml")); err != nil {
				continue
			}
		}
		m[entry.Name()] = true
	}
	return m
}

// isThemeInstalled 判断远程主题是否已安装到本地
// 1. 文件夹名直接匹配 repo 名
// 2. 通过 Redis 映射 repo → 文件夹名，并验证文件夹确实存在
func (s *ThemeService) isThemeInstalled(repo string, localDirs map[string]bool) bool {
	if localDirs[repo] {
		return true
	}
	if s.redisClient != nil {
		folderID, err := s.redisClient.Get(context.Background(), s.dlKey(repo, "folderId"))
		if err == nil && folderID != "" && localDirs[folderID] {
			return true
		}
	}
	return false
}

func (s *ThemeService) readTheme(themeDir string) (*models.Theme, error) {
	// 优先读 theme.json，找不到再试 theme.yaml / theme.yml
	metaFile, data, err := readThemeMetaFile(themeDir)
	if err != nil {
		return nil, err
	}

	var meta struct {
		ID            string
		Name          string
		Version       string
		Author        string
		Description   string
		Logo          string // Halo format spec.logo, fallback for screenshot
		SettingName   string // Halo format spec.settingName
		ConfigMapName string // Halo format spec.configMapName
		Repo          string // Halo format spec.repo
		Homepage      string // Halo format spec.homepage
	}

	isYAML := strings.HasSuffix(metaFile, ".yaml") || strings.HasSuffix(metaFile, ".yml")

	if isYAML {
		// 优先按 Halo 格式解析（apiVersion: theme.halo.run/...），失败再回退扁平格式
		if !parseHaloTheme(data, &meta) {
			parseFlatTheme(data, &meta, true)
		}
	} else {
		parseFlatTheme(data, &meta, false)
	}

	if meta.ID == "" {
		meta.ID = filepath.Base(themeDir)
	}
	if meta.Name == "" {
		meta.Name = meta.ID
	}

	// 检查截图文件，没有则回退到主题 logo
	screenshot := ""
	for _, name := range []string{"screenshot.png", "screenshot.jpg", "screenshot.webp"} {
		if _, err := os.Stat(filepath.Join(themeDir, name)); err == nil {
			screenshot = "/api/v1/admin/themes/" + meta.ID + "/screenshot"
			break
		}
	}
	if screenshot == "" && meta.Logo != "" {
		screenshot = meta.Logo
	}

	// 检查是否存在 settings.yaml（用于创建主题设置表单）
	hasSettings := false
	if _, err := os.Stat(filepath.Join(themeDir, "settings.yaml")); err == nil {
		hasSettings = true
	}

	return &models.Theme{
		ID:            meta.ID,
		Name:          meta.Name,
		Version:       meta.Version,
		Author:        meta.Author,
		Description:   meta.Description,
		Screenshot:    screenshot,
		HasSettings:   hasSettings,
		SettingName:   meta.SettingName,
		ConfigMapName: meta.ConfigMapName,
		Repo:          meta.Repo,
		Homepage:      meta.Homepage,
	}, nil
}

// readThemeMetaFile 依次尝试读 theme.json / theme.yaml / theme.yml
func readThemeMetaFile(themeDir string) (filePath string, data []byte, err error) {
	for _, name := range []string{"theme.json", "theme.yaml", "theme.yml"} {
		p := filepath.Join(themeDir, name)
		if d, e := os.ReadFile(p); e == nil {
			return p, d, nil
		}
	}
	return "", nil, fmt.Errorf("theme.json / theme.yaml / theme.yml not found in %s", themeDir)
}

// parseHaloTheme 解析 Halo 格式的 theme.yaml，成功返回 true
func parseHaloTheme(data []byte, meta *struct {
	ID            string
	Name          string
	Version       string
	Author        string
	Description   string
	Logo          string
	SettingName   string
	ConfigMapName string
	Repo          string
	Homepage      string
}) bool {
	var halo struct {
		APIVersion string `yaml:"apiVersion"`
		Metadata   struct {
			Name string `yaml:"name"`
		} `yaml:"metadata"`
		Spec struct {
			DisplayName string `yaml:"displayName"`
			Version     string `yaml:"version"`
			Author      struct {
				Name string `yaml:"name"`
			} `yaml:"author"`
			Description   string `yaml:"description"`
			Logo          string `yaml:"logo"`
			SettingName   string `yaml:"settingName"`
			ConfigMapName string `yaml:"configMapName"`
			Repo          string `yaml:"repo"`
			Homepage      string `yaml:"homepage"`
		} `yaml:"spec"`
	}
	if err := yaml.Unmarshal(data, &halo); err != nil {
		return false
	}
	if !strings.HasPrefix(halo.APIVersion, "theme.halo.run/") {
		return false
	}
	meta.ID = halo.Metadata.Name
	meta.Name = halo.Spec.DisplayName
	meta.Version = halo.Spec.Version
	meta.Author = halo.Spec.Author.Name
	meta.Description = halo.Spec.Description
	meta.Logo = halo.Spec.Logo
	meta.SettingName = halo.Spec.SettingName
	meta.ConfigMapName = halo.Spec.ConfigMapName
	meta.Repo = halo.Spec.Repo
	meta.Homepage = halo.Spec.Homepage
	return true
}

// parseFlatTheme 解析扁平格式（theme.json 或简单 theme.yaml）
func parseFlatTheme(data []byte, meta *struct {
	ID            string
	Name          string
	Version       string
	Author        string
	Description   string
	Logo          string
	SettingName   string
	ConfigMapName string
	Repo          string
	Homepage      string
}, isYAML bool) {
	var flat struct {
		ID            string `json:"id" yaml:"id"`
		Name          string `json:"name" yaml:"name"`
		Version       string `json:"version" yaml:"version"`
		Author        any    `json:"author" yaml:"author"`
		Description   string `json:"description" yaml:"description"`
		Logo          any    `json:"logo" yaml:"logo"`
		SettingName   string `json:"settingName" yaml:"settingName"`
		ConfigMapName string `json:"configMapName" yaml:"configMapName"`
		Repo          string `json:"repo" yaml:"repo"`
		Homepage      string `json:"homepage" yaml:"homepage"`
	}
	if isYAML {
		yaml.Unmarshal(data, &flat)
	} else {
		json.Unmarshal(data, &flat)
	}
	meta.ID = flat.ID
	meta.Name = flat.Name
	meta.Version = flat.Version
	meta.Author = extractStringOrName(flat.Author)
	meta.Description = flat.Description
	meta.Logo = extractStringOrName(flat.Logo)
	meta.SettingName = flat.SettingName
	meta.ConfigMapName = flat.ConfigMapName
	meta.Repo = flat.Repo
	meta.Homepage = flat.Homepage
}

// extractStringOrName 从 string 或 {name: "..."} 对象中提取字符串值
func extractStringOrName(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	if m, ok := v.(map[string]any); ok {
		if name, ok := m["name"].(string); ok {
			return name
		}
	}
	return ""
}
