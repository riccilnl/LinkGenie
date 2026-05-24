package db

import "testing"

func TestTagListHandlesLegacyNullDates(t *testing.T) {
	setupTestDB(t)

	if _, err := DB.Exec(`
		INSERT INTO tags (name, category, usage_count, last_used, date_added)
		VALUES (?, ?, ?, NULL, NULL)
	`, "legacy-null-tag", "candidate", 0); err != nil {
		t.Fatalf("准备 legacy tag 数据失败: %v", err)
	}

	repo := NewTagRepository()
	tags, err := repo.List()
	if err != nil {
		t.Fatalf("查询标签失败: %v", err)
	}

	for _, tag := range tags {
		if tag.Name != "legacy-null-tag" {
			continue
		}
		if tag.LastUsed != "" {
			t.Fatalf("预期 legacy tag 的 last_used 为空字符串，实际为 %q", tag.LastUsed)
		}
		if tag.DateAdded != "" {
			t.Fatalf("预期 legacy tag 的 date_added 为空字符串，实际为 %q", tag.DateAdded)
		}
		return
	}

	t.Fatalf("未找到 legacy-null-tag，说明 List 仍然跳过了 NULL 时间字段记录")
}
