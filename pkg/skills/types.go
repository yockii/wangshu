package skills

type Skill struct {
	Name        string   `yaml:"name" xml:"name"`
	Description string   `yaml:"description" xml:"description"`
	Triggers    []string `yaml:"triggers" xml:"triggers,omitempty"`
	Location    string   `xml:"location"`
}
