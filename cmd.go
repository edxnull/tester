package main

import (
	"bytes"
	_ "fmt"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

// TODO: create one texture and just draw into it instead of create/delete calls

type CmdConsole struct {
	show         bool
	move_left    bool
	alpha_value  uint8
	bg_rect      sdl.Rect
	ttf_rect     sdl.Rect
	cursor_rect  sdl.Rect
	font         *ttf.Font
	font_w       int
	font_h       int
	ttf_texture  *sdl.Texture
	input_buffer bytes.Buffer
}

func NewCmdConsole(renderer *sdl.Renderer) CmdConsole {
	cmd := CmdConsole{}
	cmd.alpha_value = 100

	font_dir := "./fonts/"
	font_name := "Inconsolata-Regular.ttf"
	font_size := 14

	var err error
	if cmd.font, err = ttf.OpenFont(font_dir+font_name, font_size); err != nil {
		panic(err)
	}

	cmd.font_w, cmd.font_h, _ = cmd.font.SizeUTF8(" ")
	cmd.ttf_texture = MakeTTF_Texture(renderer, cmd.font, " ", &sdl.Color{R: 0, G: 0, B: 0, A: 255})
	cmd.ttf_rect = sdl.Rect{
		X: 0,
		Y: WIN_H - int32(cmd.font_h),
		W: int32(cmd.font_w * len(" ")),
		H: int32(cmd.font_h),
	}
	cmd.bg_rect = sdl.Rect{
		X: 0,
		Y: WIN_H - int32(cmd.font_h),
		W: WIN_W,
		H: int32(cmd.font_h),
	}
	cmd.cursor_rect = sdl.Rect{
		X: 0,
		Y: WIN_H - int32(cmd.font_h),
		W: int32(cmd.font_w),
		H: int32(cmd.font_h),
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

func (cmd *CmdConsole) MakeTexture(renderer *sdl.Renderer, text string, color *sdl.Color) {
	var surface *sdl.Surface
	surface, _ = cmd.font.RenderUTF8Blended(text, *color)
	cmd.ttf_texture, _ = renderer.CreateTextureFromSurface(surface)
	surface.Free()
}

func (cmd *CmdConsole) WriteChar(renderer *sdl.Renderer, t uint8) {
	if cmd.input_buffer.Len() <= (cmd.input_buffer.Cap() - 1) {
		input_char := string(t)
		cmd.input_buffer.WriteString(input_char)
		cmd.ttf_texture.Destroy()
		cmd.MakeTexture(renderer, cmd.input_buffer.String(), &sdl.Color{R: 0, G: 0, B: 0, A: 255})
		curr_char_w := cmd.font_w * len(input_char)
		cmd.ttf_rect.W = int32(cmd.font_w * len(cmd.input_buffer.String()))
		cmd.ttf_rect.H = int32(cmd.font_h)
		cmd.cursor_rect.X += int32(curr_char_w)
	}
}

func (cmd *CmdConsole) WriteString(renderer *sdl.Renderer, str string) {
	for i := range str {
		if cmd.input_buffer.Len() >= (cmd.input_buffer.Cap() - 1) {
			break
		}
		cmd.input_buffer.WriteString(string(str[i]))
		curr_char_w := cmd.font_w * len(str)
		cmd.ttf_rect.W = int32(cmd.font_w * len(cmd.input_buffer.String()))
		cmd.ttf_rect.H = int32(cmd.font_h)
		cmd.cursor_rect.X += int32(curr_char_w)
	}
	cmd.ttf_texture.Destroy()
	cmd.MakeTexture(renderer, cmd.input_buffer.String(), &sdl.Color{R: 0, G: 0, B: 0, A: 255})
}

func (cmd *CmdConsole) Reset(renderer *sdl.Renderer) {
	if cmd.cursor_rect.X <= 0 {
		cmd.cursor_rect.X = 0
	} else {
		temp_string := cmd.input_buffer.String()[0 : len(cmd.input_buffer.String())-1]
		cmd.input_buffer.Reset()
		cmd.input_buffer.WriteString(temp_string)

		cmd.ttf_texture.Destroy()

		if len(cmd.input_buffer.String()) > 0 {
			cmd.ttf_texture = MakeTTF_Texture(renderer, cmd.font, temp_string, &sdl.Color{R: 0, G: 0, B: 0, A: 255})
		}

		if len(temp_string) != 0 {
			curr_char_w := cmd.font_w * len(string(temp_string[len(temp_string)-1]))
			cmd.cursor_rect.X -= int32(curr_char_w)
			cmd.ttf_rect.W = int32(cmd.font_w * len(cmd.input_buffer.String()))
			cmd.ttf_rect.H = int32(cmd.font_h)
		} else {
			cmd.cursor_rect.X = 0
		}
	}
}

func (cmd *CmdConsole) MakeNULL() {
	cmd.input_buffer.Reset()
	if cmd.ttf_texture != nil {
		cmd.ttf_texture.Destroy()
		cmd.ttf_texture = nil
	}
	cmd.cursor_rect.X = 0
}
