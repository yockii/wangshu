package skills

import (
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var defaultSkillLoader *Loader

func GetDefaultLoader() *Loader {
	return defaultSkillLoader
}

type Loader struct {
	globalPath  string
	builtinPath string
}

func InitializeSkillLoader(globalPath, builtInPath string) {
	defaultSkillLoader = NewLoader(globalPath, builtInPath)
}

func NewLoader(globalPath, builtInPath string) *Loader {
	return &Loader{
		globalPath:  globalPath,
		builtinPath: builtInPath,
	}
}

func (l *Loader) LoadSkills() ([]*Skill, error) {
	skills := []*Skill{}
	// 从globalPath和builtInPath读取skill元数据
	skills = append(skills, l.loadSkillsFromDir(l.builtinPath)...)
	skills = append(skills, l.loadSkillsFromDir(l.globalPath)...)
	return skills, nil
}

var frontmatterReg = regexp.MustCompile(`^---\n(.*?)\n---\n`)

func (l *Loader) loadSkillsFromDir(dir string) []*Skill {
	skills := []*Skill{}
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		skillMdPath := filepath.Join(path, "SKILL.md")
		if _, err := os.Stat(skillMdPath); err != nil {
			return nil
		}
		// 读取SKILL.md文件
		data, err := os.ReadFile(skillMdPath)
		if err != nil {
			return err
		}
		// 解析frontmatter
		matches := frontmatterReg.FindStringSubmatch(string(data))
		if len(matches) < 2 {
			return nil
		}
		skill := Skill{}
		if err := yaml.NewDecoder(strings.NewReader(matches[1])).Decode(&skill); err != nil {
			slog.Error("Failed to decode skill metadata", "error", err, "file", skillMdPath)
			return nil
		}
		skill.Location, err = filepath.Abs(skillMdPath)
		if err != nil {
			return err
		}
		skills = append(skills, &skill)

		return nil
	})

	return skills
}
