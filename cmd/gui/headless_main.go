//go:build !gui
// +build !gui

package main

import "fmt"

func main() {
    fmt.Println("GUI build is disabled (no 'gui' build tag). Use CLI: go run ./cmd/sider2api --token <SIDER_TOKEN>. For GUI, install OpenGL/GLFW deps and run: go run -tags gui ./cmd/gui")
}
