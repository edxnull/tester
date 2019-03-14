package main

import (
	"bytes"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

type CmdConsole struct {
	show         bool
	move_left    bool
	alpha_value  uint8
	bg_rect      sdl.Rect
	ttf_rect     sdl.Rect
	cursor_rect  sdl.Rect
	ttf_texture  *sdl.Texture
	input_buffer bytes.Buffer
}

func NewCmdConsole(renderer *sdl.Renderer, font *ttf.Font) CmdConsole {
	cmd := CmdConsole{}
	cmd.alpha_value = 100
	fw, fh, _ := font.SizeUTF8(" ")
	cmd.ttf_texture = make_ttf_texture(renderer, font, " ", &sdl.Color{R: 0, G: 0, B: 0, A: 255})
	cmd.ttf_rect = sdl.Rect{
		X: 0,
		Y: WIN_H - int32(fh),
		W: int32(fw * len(" ")),
		H: int32(fh),
	}
	cmd.bg_rect = sdl.Rect{
		X: 0,
		Y: WIN_H - int32(fh),
		W: WIN_W,
		H: int32(fh),
	}
	cmd.cursor_rect = sdl.Rect{
		X: 0,
		Y: WIN_H - int32(fh),
		W: int32(fw),
		H: int32(fh),
	}
	cmd.input_buffer.Grow(64)
	return cmd
}

func (cmd *CmdConsole) Resize(new_win_w int32, new_win_h int32) {
	cmd.bg_rect.W = new_win_w
	cmd.bg_rect.Y = new_win_h - cmd.cursor_rect.H
	cmd.ttf_rect.Y = new_win_h - cmd.cursor_rect.H
	cmd.cursor_rect.Y = new_win_h - cmd.cursor_rect.H
}

func (cmd *CmdConsole) MakeTexture(renderer *sdl.Renderer, font *ttf.Font, text string, color *sdl.Color) {
	var surface *sdl.Surface
	surface, _ = font.RenderUTF8Blended(text, *color)
	cmd.ttf_texture, _ = renderer.CreateTextureFromSurface(surface)
	surface.Free()
}

func (cmd *CmdConsole) WriteChar(renderer *sdl.Renderer, font FontSelector, t uint8) {
	if cmd.input_buffer.Len() <= (cmd.input_buffer.Cap() - 1) {
		input_char := string(t)
		cmd.input_buffer.WriteString(input_char)
		cmd.ttf_texture.Destroy()
		cmd.MakeTexture(renderer, font.current_font, cmd.input_buffer.String(), &sdl.Color{R: 0, G: 0, B: 0, A: 255})
		curr_char_w := font.current_font_w * len(input_char)
		cmd.ttf_rect.W = int32(font.current_font_w * len(cmd.input_buffer.String()))
		cmd.ttf_rect.H = int32(font.current_font_h)
		cmd.cursor_rect.X += int32(curr_char_w)
	}
}

func (cmd *CmdConsole) Reset(renderer *sdl.Renderer, curr_char_w int, font *ttf.Font, fontw int, fonth int) {
	if cmd.cursor_rect.X <= 0 {
		cmd.cursor_rect.X = 0
	} else {
		temp_string := cmd.input_buffer.String()[0 : len(cmd.input_buffer.String())-1]
		cmd.input_buffer.Reset()
		cmd.input_buffer.WriteString(temp_string)

		cmd.ttf_texture.Destroy()

		if len(cmd.input_buffer.String()) > 0 {
			cmd.ttf_texture = make_ttf_texture(renderer, font, temp_string, &sdl.Color{R: 0, G: 0, B: 0, A: 255})
		}

		if len(temp_string) != 0 {
			curr_char_w = fontw * len(string(temp_string[len(temp_string)-1]))

			cmd.cursor_rect.X -= int32(curr_char_w)

			cmd.ttf_rect.W = int32(fontw * len(cmd.input_buffer.String()))
			cmd.ttf_rect.H = int32(fonth)
		} else {
			cmd.cursor_rect.X = 0
		}
	}
}

func (cmd *CmdConsole) MakeNULL() {
	cmd.input_buffer.Reset()
	cmd.ttf_texture.Destroy()
    cmd.ttf_texture = nil
	cmd.cursor_rect.X = 0
}
