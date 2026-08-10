package browserplaywrightsvc

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
)

type persistedSnapshot struct {
	CreatedAt time.Time     `json:"createdAt"`
	Items     []SnapshotRef `json:"items"`
	Path      string        `json:"path"`
	PageURL   string        `json:"pageUrl"`
	PageTitle string        `json:"pageTitle"`
	Target    string        `json:"target,omitempty"`
	Depth     int           `json:"depth,omitempty"`
}

func captureSnapshot(page playwright.Page, path string, target string, depth int) (*SnapshotInfo, error) {
	title, _ := page.Title()
	info := &SnapshotInfo{
		Path:      path,
		PageURL:   page.URL(),
		PageTitle: title,
		CreatedAt: time.Now(),
		Depth:     depth,
		Target:    target,
	}

	raw, err := page.Evaluate(`() => {
		const elements = Array.from(document.querySelectorAll('*'));
		const out = [];
		for (const el of elements) {
			const tag = (el.tagName || '').toLowerCase();
			if (!tag || tag === 'html' || tag === 'head' || tag === 'body') continue;
			const text = ((el.innerText || el.textContent || '').trim().replace(/\s+/g, ' ')).slice(0, 120);
			const role = el.getAttribute('role') || '';
			const id = el.id || '';
			const name = el.getAttribute('name') || '';
			const type = el.getAttribute('type') || '';
			const href = el.getAttribute('href') || '';
			const value = ('value' in el ? String(el.value || '') : '').trim().replace(/\s+/g, ' ').slice(0, 120);
			if (!(text || role || id || name || type || href)) continue;
			const rect = typeof el.getBoundingClientRect === 'function' ? el.getBoundingClientRect() : null;
			out.push({
				text,
				role,
				tag,
				id,
				name,
				type,
				href,
				value,
				x: rect ? rect.x : 0,
				y: rect ? rect.y : 0,
				w: rect ? rect.width : 0,
				h: rect ? rect.height : 0
			});
			if (out.length >= 400) break;
		}
		return out;
	}`)
	if err != nil {
		return nil, err
	}

	payloadBytes, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var items []SnapshotRef
	if err := json.Unmarshal(payloadBytes, &items); err != nil {
		return nil, err
	}
	for i := range items {
		items[i].Ref = fmt.Sprintf("e%d", i+1)
		items[i].Text = trimText(items[i].Text)
		items[i].Role = stringify(items[i].Role)
		items[i].Tag = stringify(items[i].Tag)
		items[i].ID = stringify(items[i].ID)
		items[i].Name = stringify(items[i].Name)
		items[i].Type = stringify(items[i].Type)
		items[i].Href = stringify(items[i].Href)
		items[i].Value = trimText(stringify(items[i].Value))
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Y == items[j].Y {
			return items[i].X < items[j].X
		}
		return items[i].Y < items[j].Y
	})
	info.Items = items
	info.Description = renderSnapshot(items)

	payload := persistedSnapshot{
		CreatedAt: info.CreatedAt,
		Items:     info.Items,
		Path:      info.Path,
		PageURL:   info.PageURL,
		PageTitle: info.PageTitle,
		Target:    info.Target,
		Depth:     info.Depth,
	}
	if err := writeJSONFile(path, payload); err != nil {
		return nil, err
	}
	return info, nil
}

func loadSnapshot(path string) (*SnapshotInfo, error) {
	var payload persistedSnapshot
	if err := readJSONFile(path, &payload); err != nil {
		return nil, err
	}
	return &SnapshotInfo{
		Path:        payload.Path,
		PageURL:     payload.PageURL,
		PageTitle:   payload.PageTitle,
		CreatedAt:   payload.CreatedAt,
		Depth:       payload.Depth,
		Target:      payload.Target,
		Items:       payload.Items,
		Description: renderSnapshot(payload.Items),
	}, nil
}

func resolveSnapshotRef(path, ref string) (SnapshotRef, error) {
	snapshot, err := loadSnapshot(path)
	if err != nil {
		return SnapshotRef{}, err
	}
	for _, item := range snapshot.Items {
		if item.Ref == ref {
			return item, nil
		}
	}
	return SnapshotRef{}, fmt.Errorf("unknown snapshot ref: %s", ref)
}

func meaningfulRef(ref SnapshotRef) bool {
	if ref.Tag == "" {
		return false
	}
	if ref.Tag == "html" || ref.Tag == "head" || ref.Tag == "body" {
		return false
	}
	return ref.Text != "" || ref.Role != "" || ref.ID != "" || ref.Name != "" || ref.Type != "" || ref.Href != ""
}

func trimText(value string) string {
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if len(value) > 120 {
		return value[:117] + "..."
	}
	return value
}

func stringify(value any) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func renderSnapshot(items []SnapshotRef) string {
	lines := make([]string, 0, len(items))
	for _, item := range items {
		parts := []string{item.Ref}
		if item.Tag != "" {
			parts = append(parts, "<"+item.Tag+">")
		}
		if item.Role != "" {
			parts = append(parts, "role="+item.Role)
		}
		if item.Text != "" {
			parts = append(parts, "text="+quoteIfNeeded(item.Text))
		}
		if item.ID != "" {
			parts = append(parts, "id="+item.ID)
		}
		if item.Name != "" {
			parts = append(parts, "name="+item.Name)
		}
		if item.Type != "" {
			parts = append(parts, "type="+item.Type)
		}
		lines = append(lines, "- "+strings.Join(parts, " "))
	}
	return strings.Join(lines, "\n")
}

func quoteIfNeeded(value string) string {
	if strings.ContainsAny(value, " \t") {
		return strconvQuote(value)
	}
	return value
}

func strconvQuote(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `\"`) + `"`
}

func snapshotExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
