package main

import "strings"

// PathSeparator string
const PathSeparator = "/"

// Path will contain the parsed url params
type Path struct {
	Path string
	ID   string
}

// NewPath returns a pointer to a new Path
func NewPath(p string) *Path {
	var id string
	p = strings.Trim(p, PathSeparator)
	s := strings.Split(p, PathSeparator)
	if len(s) > 1 {
		id = s[len(s)-1]
		p = strings.Join(s[:len(s)-1], PathSeparator)
	}
	return &Path{Path: p, ID: id}
}

// HasID determines if a Path has an ID
func (p *Path) HasID() bool {
	return len(p.ID) > 0
}
