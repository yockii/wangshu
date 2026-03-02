package types

import (
	"encoding/xml"
)

type SkillsParent struct {
	XMLName   xml.Name `xml:"skills"`
	SkillList []*Skill `xml:"skill"`
}
