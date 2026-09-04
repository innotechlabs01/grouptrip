package main

import "fmt"

// main is the backend entry point. As layers (HTTP, Turso repository, Polar adapter)
// land, they are wired here. For now it boots the application container.
func main() {
	fmt.Println("grouptrip backend: booting")
}
