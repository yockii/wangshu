package constant

import "regexp"

var MdFrontmatterReg = regexp.MustCompile(`---\s*\n([\s\S]*?)\n---\s*|\n`)
