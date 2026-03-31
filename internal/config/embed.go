package config

import "embed"

//go:embed workspace
var embeddedFiles embed.FS

//go:embed skills
var embeddedSkills embed.FS

//go:embed all:live2d_models
var embeddedLive2DModels embed.FS
