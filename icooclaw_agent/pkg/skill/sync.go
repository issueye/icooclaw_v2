package skill

import (
	"fmt"
	"os"
	"path/filepath"

	"icooclaw/pkg/storage"
)

func SyncWorkspaceSkills(workspace string, store *storage.SkillStorage) error {
	if store == nil {
		return fmt.Errorf("skill storage is nil")
	}

	skillsRoot := filepath.Join(workspace, "skills")
	entries, err := os.ReadDir(skillsRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read workspace skills: %w", err)
	}

	parser := NewParser()
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillFile := filepath.Join(skillsRoot, entry.Name(), "SKILL.md")
		parsed, err := parser.ParseFile(skillFile)
		if err != nil {
			continue
		}

		skillRecord := &storage.Skill{
			Name:        parsed.Name,
			Title:       parsed.Title,
			Description: parsed.Description,
			Version:     parsed.Version,
			Enabled:     true,
			Type:        storage.SkillTypeSkill,
			Path:        filepath.ToSlash(filepath.Join("skills", entry.Name())),
		}
		if err := store.SaveSkill(skillRecord); err != nil {
			return fmt.Errorf("failed to sync skill %s: %w", parsed.Name, err)
		}
	}

	return nil
}
