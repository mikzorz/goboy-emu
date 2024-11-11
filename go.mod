module github.com/mikzorz/gameboy-emulator

go 1.22.5

replace github.com/gen2brain/raylib-go/raylib => /megalodon/Code/go/pkg/mod/github.com/gen2brain/raylib-go/raylib@v0.0.0-20240930075631-c66f9e2942fe/

require github.com/gen2brain/raylib-go/raylib v0.0.0-00010101000000-000000000000

require (
	github.com/ebitengine/purego v0.7.1 // indirect
	golang.org/x/exp v0.0.0-20240506185415-9bf2ced13842 // indirect
	golang.org/x/sys v0.20.0 // indirect
)

replace github.com/ebitengine/purego => /megalodon/Code/go/pkg/mod/github.com/ebitengine/purego@v0.7.1/

replace golang.org/x/exp => /megalodon/Code/go/pkg/mod/golang.org/x/exp@v0.0.0-20240506185415-9bf2ced13842/

replace golang.org/x/sys => /megalodon/Code/go/pkg/mod/golang.org/x/sys@v0.20.0/
