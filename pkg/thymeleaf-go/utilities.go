package thymeleaf

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// ─── Utility Objects ───
// Halo themes use #strings, #lists, #dates, #locale, #theme, #authentication
// All methods are exported (PascalCase) and the expression engine maps
// camelCase calls (e.g. isEmpty) to PascalCase (e.g. IsEmpty) via findMethod.

func getUtility(name string, ctx *Context) (any, error) {
	switch name {
	case "#strings":
		return &StringsUtility{}, nil
	case "#lists":
		return &ListsUtility{}, nil
	case "#dates":
		return &DatesUtility{}, nil
	case "#arrays":
		return &ArraysUtility{}, nil
	case "#locale":
		return &LocaleUtility{}, nil
	case "#theme":
		// Read the theme base path from context (set by theme_render.go)
		base := ""
		if v, ok := ctx.Get("__theme_base_path"); ok {
			base = toStr(v)
		}
		return &ThemeUtility{assetsBase: base}, nil
	case "#authentication":
		return &AuthenticationUtility{}, nil
	case "#numbers":
		return &NumbersUtility{}, nil
	case "#annotations":
		return &AnnotationsUtility{}, nil
	default:
		return nil, nil
	}
}

// ─── #numbers ───

type NumbersUtility struct{}

// Sequence returns a slice of integers from start to end (inclusive).
// Mirrors Thymeleaf's #numbers.sequence(start, end).
func (n *NumbersUtility) Sequence(start, end any) []int {
	s, ok1 := toInt(start)
	e, ok2 := toInt(end)
	if !ok1 || !ok2 {
		return []int{}
	}
	if e < s {
		return []int{}
	}
	result := make([]int, 0, e-s+1)
	for i := s; i <= e; i++ {
		result = append(result, i)
	}
	return result
}

// ─── #strings ───

type StringsUtility struct{}

func (s *StringsUtility) ToString(val any) string {
	return toStr(val)
}

func (s *StringsUtility) IsEmpty(val any) bool {
	if val == nil {
		return true
	}
	str := toStr(val)
	return str == ""
}

func (s *StringsUtility) Trim(val any) string {
	return strings.TrimSpace(toStr(val))
}

func (s *StringsUtility) Length(val any) int {
	return len(toStr(val))
}

func (s *StringsUtility) StartsWith(str, prefix any) bool {
	return strings.HasPrefix(toStr(str), toStr(prefix))
}

func (s *StringsUtility) EndsWith(str, suffix any) bool {
	return strings.HasSuffix(toStr(str), toStr(suffix))
}

func (s *StringsUtility) Substring(str any, args ...any) string {
	s2 := toStr(str)
	if len(args) == 0 {
		return s2
	}
	start, ok := toInt(args[0])
	if !ok {
		return s2
	}
	if start < 0 {
		start = 0
	}
	if start >= len(s2) {
		return ""
	}
	if len(args) >= 2 {
		end, ok := toInt(args[1])
		if !ok || end > len(s2) {
			end = len(s2)
		}
		if end < start {
			return ""
		}
		return s2[start:end]
	}
	return s2[start:]
}

func (s *StringsUtility) ToLowerCase(str any) string {
	return strings.ToLower(toStr(str))
}

func (s *StringsUtility) ToUpperCase(str any) string {
	return strings.ToUpper(toStr(str))
}

func (s *StringsUtility) Equals(a, b any) bool {
	return toStr(a) == toStr(b)
}

func (s *StringsUtility) EqualsIgnoreCase(a, b any) bool {
	return strings.EqualFold(toStr(a), toStr(b))
}

func (s *StringsUtility) Contains(str, substr any) bool {
	return strings.Contains(toStr(str), toStr(substr))
}

func (s *StringsUtility) IndexOf(str, substr any) int {
	return strings.Index(toStr(str), toStr(substr))
}

func (s *StringsUtility) Replace(str, old, new any) string {
	return strings.ReplaceAll(toStr(str), toStr(old), toStr(new))
}

func (s *StringsUtility) Split(str, sep any) []string {
	return strings.Split(toStr(str), toStr(sep))
}

func (s *StringsUtility) Concat(args ...any) string {
	var sb strings.Builder
	for _, arg := range args {
		sb.WriteString(toStr(arg))
	}
	return sb.String()
}

// ─── #lists ───

type ListsUtility struct{}

func (l *ListsUtility) IsEmpty(val any) bool {
	if val == nil {
		return true
	}
	rv := reflect.ValueOf(val)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		return rv.Len() == 0
	case reflect.Map:
		return rv.Len() == 0
	case reflect.String:
		return rv.Len() == 0
	}
	return true
}

func (l *ListsUtility) Size(val any) int {
	if val == nil {
		return 0
	}
	rv := reflect.ValueOf(val)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		return rv.Len()
	case reflect.Map:
		return rv.Len()
	}
	return 0
}

func (l *ListsUtility) Contains(list, item any) bool {
	if list == nil {
		return false
	}
	rv := reflect.ValueOf(list)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return false
	}
	itemStr := toStr(item)
	for i := 0; i < rv.Len(); i++ {
		if toStr(rv.Index(i).Interface()) == itemStr {
			return true
		}
	}
	return false
}

// ─── #arrays ───

type ArraysUtility struct{}

func (a *ArraysUtility) IsEmpty(val any) bool {
	if val == nil {
		return true
	}
	rv := reflect.ValueOf(val)
	return rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array || rv.Len() == 0
}

func (a *ArraysUtility) Length(val any) int {
	if val == nil {
		return 0
	}
	rv := reflect.ValueOf(val)
	if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
		return rv.Len()
	}
	return 0
}

// ─── #dates ───

type DatesUtility struct{}

func (d *DatesUtility) Format(date any, pattern string) string {
	var t time.Time
	switch v := date.(type) {
	case time.Time:
		t = v
	case int64:
		// Millisecond timestamp (Java Date.getTime())
		t = time.UnixMilli(v)
	case int:
		t = time.UnixMilli(int64(v))
	case float64:
		t = time.UnixMilli(int64(v))
	case string:
		// Try parsing common formats
		formats := []string{
			time.RFC3339,
			"2006-01-02T15:04:05Z",
			"2006-01-02 15:04:05",
			"2006-01-02",
		}
		for _, f := range formats {
			if parsed, err := time.Parse(f, v); err == nil {
				t = parsed
				break
			}
		}
		if t.IsZero() {
			return toStr(date)
		}
	default:
		return toStr(date)
	}

	// Convert Java SimpleDateFormat pattern to Go format
	goFormat := javaToGoFormat(pattern)
	return t.Format(goFormat)
}

func javaToGoFormat(pattern string) string {
	// Common conversions from Java date format to Go
	replacer := strings.NewReplacer(
		"yyyy", "2006",
		"yy", "06",
		"MM", "01",
		"dd", "02",
		"HH", "15",
		"mm", "04",
		"ss", "05",
		"SSS", "000",
		"a", "PM",
		"EEE", "Mon",
		"EEEE", "Monday",
		"MMM", "Jan",
		"MMMM", "January",
	)
	return replacer.Replace(pattern)
}

// ─── #locale ───

type LocaleUtility struct{}

func (l *LocaleUtility) ToLanguageTag() string {
	return "zh-CN"
}

func (l *LocaleUtility) String() string {
	return "zh-CN"
}

// ─── #theme ───

type ThemeUtility struct {
	assetsBase string
}

func (t *ThemeUtility) Assets(path string) string {
	// Halo's #theme.assets(path) returns /themes/<theme-id>/assets/<path>.
	// assetsBase is "/themes/<theme-id>", so we insert "/assets" before the path.
	if t.assetsBase != "" {
		return t.assetsBase + "/assets" + path
	}
	return "/assets" + path
}

// ─── #authentication ───

type AuthenticationUtility struct{}

func (a *AuthenticationUtility) GetName() string {
	return "anonymousUser"
}

func (a *AuthenticationUtility) String() string {
	return "anonymousUser"
}

// ─── #annotations ───

// AnnotationsUtility reads Halo-style metadata.annotations from objects.
// Halo themes use #annotations.getOrDefault(object, key, default) to read
// per-post/page configuration flags such as enable_comment.
type AnnotationsUtility struct{}

// Get returns the annotation value for key, or nil if absent.
func (a *AnnotationsUtility) Get(obj, key any) any {
	return getAnnotation(obj, toStr(key))
}

// GetOrDefault returns the annotation value for key, or defaultValue if absent.
func (a *AnnotationsUtility) GetOrDefault(obj, key, defaultValue any) any {
	if v := getAnnotation(obj, toStr(key)); v != nil {
		return v
	}
	return defaultValue
}

// getAnnotation navigates obj.metadata.annotations[key] for map/struct values.
func getAnnotation(obj any, key string) any {
	if obj == nil || key == "" {
		return nil
	}
	rv := reflect.ValueOf(obj)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	var metadata any
	switch rv.Kind() {
	case reflect.Map:
		for _, k := range rv.MapKeys() {
			if strings.EqualFold(k.String(), "metadata") {
				metadata = rv.MapIndex(k).Interface()
				break
			}
		}
	case reflect.Struct:
		if f := findField(rv, "Metadata"); f.IsValid() {
			metadata = f.Interface()
		}
	}
	if metadata == nil {
		return nil
	}

	mrv := reflect.ValueOf(metadata)
	if mrv.Kind() == reflect.Ptr {
		mrv = mrv.Elem()
	}
	var annotations any
	switch mrv.Kind() {
	case reflect.Map:
		for _, k := range mrv.MapKeys() {
			if strings.EqualFold(k.String(), "annotations") {
				annotations = mrv.MapIndex(k).Interface()
				break
			}
		}
	case reflect.Struct:
		if f := findField(mrv, "Annotations"); f.IsValid() {
			annotations = f.Interface()
		}
	}
	if annotations == nil {
		return nil
	}

	arv := reflect.ValueOf(annotations)
	if arv.Kind() == reflect.Ptr {
		arv = arv.Elem()
	}
	switch arv.Kind() {
	case reflect.Map:
		for _, k := range arv.MapKeys() {
			if strings.EqualFold(k.String(), key) {
				return arv.MapIndex(k).Interface()
			}
		}
	case reflect.Struct:
		if f := findField(arv, key); f.IsValid() {
			return f.Interface()
		}
	}
	return nil
}

// ─── Helpers ───

func toInt(val any) (int, bool) {
	switch v := val.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case string:
		i, err := strconv.Atoi(v)
		if err != nil {
			f, err2 := strconv.ParseFloat(v, 64)
			if err2 != nil {
				return 0, false
			}
			return int(f), true
		}
		return i, true
	case bool:
		if v {
			return 1, true
		}
		return 0, true
	}
	return 0, false
}

// flattenToNested converts flat dotted keys ("a.b.c": value) into nested maps
func flattenToNested(flat map[string]any) map[string]any {
	result := make(map[string]any)
	for k, v := range flat {
		parts := strings.Split(k, ".")
		current := result
		for i, part := range parts {
			if i == len(parts)-1 {
				current[part] = v
			} else {
				if next, ok := current[part].(map[string]any); ok {
					current = next
				} else {
					next = make(map[string]any)
					current[part] = next
					current = next
				}
			}
		}
	}
	return result
}

var _ = fmt.Sprintf // keep fmt import
