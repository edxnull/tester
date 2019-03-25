package main

import (
	"flag"
	"fmt"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
	"io/ioutil"
	"log"
	"math"
	"os"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

// GENERAL
// [ ] fmt.Println(runtime.Caller(0)) use this to get a LINENR when calculating unique ID's for IMGUI
// [ ] maybe it would be possible to use unicode symbols like squares/triangles to indicate clickable objects?
// [ ] add *sdl.Texture to the list structure, to avoid allocating *sdl.Texture pointers in type Line struct?
// [ ] refactor FontSelector
// [ ] changing font size
// [ ] selecting and reloading fonts
// [ ] refactor wrapping text
// [ ] make sure that we don't exceed max sdl.texture width
// [ ] why do we get such a huge GPU commit bump on start/ GPU commit drop after resizing?
// [ ] should we compress strings?? Huffman encoding?
// [ ] should we use hash algorithms?
// [ ] selecting and reloading text
// [ ] proper reloading text on demand
// [ ] searching
// [ ] fuzzy search
// [ ] copy text
// [ ] copy & pasting commands
// [ ] get an N and a list of unique words in a file
// [ ] save words to a trie tree?
// [ ] figure out what to do about languages like left to right and asian languages
// [ ] export/import csv
// [ ] make sure we handle utf8
// [ ] cmd input commands + parsing

// SDL RELATED
// [ ] try sdl.UpdateTexture(texture, null, surface.pixels, surface.pitch) instead of alloc/dealloc all the time.
//     - the pixel format should be in the format of the texture. Use QueryTexture() for that.
//     - apparently this would be slow, but it's worth a shot in order to avoid allocs/deallocs.
//     Alternatively, it's possible to use Lock()/Unlock() functions for updating Textures
// [ ] SDL_ConvertSurface for faster blitting?
// [ ] try using r.SetScale()
// [ ] use r.DrawLines() to draw triangles?
// [ ] use r.SetClipRect r.GetClipRect for rendering
// [ ] USE sdl.WINDOWEVENT_EXPOSED for proper redrawing
// [ ] renderer.SetLogicalSize(WIN_W, WIN_H) -> SetLogicalSize is important for device independant rendering!
// [ ] proper time handling like dt and such
// [ ] how can we not render everything on every frame?

// VISUAL
// [ ] scrollbar actual scrolling functionality
// [ ] http://blog.moagrius.com/actionscript/jsas-understanding-easing/
// [ ] https://github.com/malkia/ufo/tree/master/samples/SDL
// [ ] http://perso.univ-lyon1.fr/thierry.excoffier/ZMW/Welcome.html
// [ ] http://northstar-www.dartmouth.edu/doc/idl/html_6.2/Creating_Widget_Applications.html
// [ ] add equations of motion for nice animation effects https://easings.net/
// [ ] tables [rows x columns]
// [ ] color rgb or rgba [color] [r, g, b] ... [r, g, b, a]
// [ ] checkbox rect within a rect [x] or [[]]
// [ ] tooltip on word hover
// [ ] interactive tooltip
// [ ] progress bar for loading files and other purposes
// [ ] visualising word stats
// [ ] smooth scrolling
// [ ] bezier curve easing functions
// [ ] taskbar / menu bar
// [ ] experiment with imgui style widgets
// [ ] grapical popup error messages like: error => your command is too long, etc...

// AUDIO
// [ ] loading and playing audio files
// [ ] recording audio?

// TESTING
// [ ] automated visual tests
// [ ] create automated tests to scroll through the page from top to bottom checking if we ever fail to allocate/deallocate *Line

// GO RELATED
// [ ] move to a 64-bit version of golang and sdl2 (needed for DELVE debugger)
// [ ] test struct padding?
// [ ] list.go should we set data to nil everytime?
// [ ] get rid of int (because on 64-bit systems it would become 64 bit and waste memory) or not???? maybe use int16 in some cases
// [ ] compare method call vs. function call overhead in golang: asm?

// DEBUGERS
// [ ] try github aarzilli/gdlv
// [ ] try go-delve/delve

const WIN_TITLE string = "GO_TEXT_APPLICATION"

const WIN_W int32 = 800
const WIN_H int32 = 600

const X_OFFSET int = 7
const TTF_FONT_SIZE int = 14
const TTF_FONT_SIZE_FOR_FONT_LIST int = 12
const LINE_LENGTH int = 500

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to 'file'")
var memprofile = flag.String("memprofile", "", "write mem profile to 'file'")

type v2 struct {
	x float32
	y float32
}

type Font struct {
	size          int
	name          string
	data          *ttf.Font
	skipline      int32
	width, height int32
}

type Line struct {
	texture         *sdl.Texture
	bg_rect         sdl.Rect
	words           []string
	word_rects      []sdl.Rect
	mouse_over_word []bool
	slice           []Line
}

type DebugWrapLine struct {
	x1, y1 int32
	x2, y2 int32
}

type Scrollbar struct {
    drag     bool
    selected bool
    rect sdl.Rect
}

type FontSelector struct {
	show              bool
	fonts             []Font
	current_font      *ttf.Font
	current_font_w    int
	current_font_h    int
	current_font_skip int
	alpha_value       uint8
	alpha_f32         float32
	bg_rect           sdl.Rect
	ttf_rects         []sdl.Rect
	highlight_rect    []sdl.Rect
	cursor_rect       sdl.Rect
	textures          []*sdl.Texture
}

func main() {
	// PROFILING SNIPPET
	var debug bool

	flag.BoolVar(&debug, "debug", false, "debug needs a bool value: -debug=true")

	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not *create* CPU profile: ", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not *start* CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}
	// PROFILING SNIPPET

	if debug {
		println("we can put debug if's everywhere!")
	}

	runtime.LockOSThread() // NOTE: not sure I need this here!

	dummy_a := new(int)
	dummy_b := new(string)
	dummy_c := string("H")
	dummy_d := new(int64)
	dummy_e := new(bool)
	dummy_f := true

	println(unsafe.Sizeof(dummy_a))
	println(unsafe.Sizeof(dummy_b))
	println(unsafe.Sizeof(dummy_c))
	println(unsafe.Sizeof(dummy_d))
	println(unsafe.Sizeof(dummy_e))
	println(unsafe.Sizeof(dummy_f))

	if err := sdl.Init(sdl.INIT_TIMER | sdl.INIT_VIDEO | sdl.INIT_AUDIO); err != nil {
		panic(err)
	}

	if err := ttf.Init(); err != nil {
		panic(err)
	}

	window, err := sdl.CreateWindow(WIN_TITLE, sdl.WINDOWPOS_CENTERED, sdl.WINDOWPOS_CENTERED, WIN_W, WIN_H,
		sdl.WINDOW_SHOWN|sdl.WINDOW_RESIZABLE)
	if err != nil {
		panic(err)
	}

	// NOTE: I've heard that PRESENTVSYNC caps FPS
	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED|sdl.RENDERER_PRESENTVSYNC)
	if err != nil {
		panic(err)
	}

	filename := "HP01.txt"
	font_dir := "./fonts/"
	text_dir := "./text/"

	line_tokens := strings.Split(string(get_filedata(text_dir, filename)), "\n")

	ticker := time.NewTicker(time.Second / 40)

	ttf_font_list := get_filenames(font_dir, []string{"ttf", "otf"})
	txt_list := get_filenames(text_dir, []string{".txt"})
	fmt.Println(txt_list)

	var gfonts FontSelector
	allocate_font_space(&gfonts, len(ttf_font_list))
	generate_fonts(&gfonts, ttf_font_list, font_dir)

	font := gfonts.current_font

	generate_rects_for_fonts(renderer, &gfonts)

	// NOTE: should we keep fonts in memory? or free them instead?

	start := time.Now()
	test_tokens := make([]string, determine_nwrap_lines(line_tokens, LINE_LENGTH, gfonts.current_font_w))
	for apos, bpos := 0, 0; apos < len(line_tokens); apos += 1 {
		if len(line_tokens[apos]) > 1 {
			current := do_wrap_lines(line_tokens[apos], LINE_LENGTH, gfonts.current_font_w)
			for pos := range current {
				test_tokens[bpos] = current[pos]
				bpos += 1
			}
		} else {
			test_tokens[bpos] = "\n"
			bpos += 1
		}
	}
	end_start := time.Now().Sub(start)
	fmt.Printf("[[do_wrap_lines loop took %s]]\n", end_start.String())

	now_gen := time.Now()
	all_lines := make([]Line, len(test_tokens))
	generate_and_populate_lines(renderer, font, &all_lines, &test_tokens)
	end_gen := time.Now().Sub(now_gen)
	fmt.Printf("[[generate_and_populate_lines took %s]]\n", end_gen.String())

	cmd := NewCmdConsole(renderer, font)

	dbg_str := make_console_text(0, len(test_tokens))
	dbg_rect := sdl.Rect{X: 0, Y: WIN_H - (cmd.bg_rect.H * 2), W: int32(gfonts.current_font_w * len(dbg_str)), H: int32(gfonts.current_font_h)}
	dbg_ttf := make_ttf_texture(renderer, gfonts.current_font, dbg_str, &sdl.Color{R: 0, G: 0, B: 0, A: 255})

	sdl.SetHint(sdl.HINT_FRAMEBUFFER_ACCELERATION, "1")
	sdl.SetHint(sdl.HINT_RENDER_SCALE_QUALITY, "1")

	renderer.SetDrawBlendMode(sdl.BLENDMODE_BLEND)

	running := true
	print_word := false
	engage_loop := false
	inc_dbg_str := true

	mouseover_word_texture_FONT := make([]bool, len(ttf_font_list))

	wrap_line := false

	move_text_up := false
	move_text_down := false

	wrapline := DebugWrapLine{int32(LINE_LENGTH), 0, int32(LINE_LENGTH), WIN_H}

	curr_char_w := 0

	//viewport_rect := sdl.Rect{0, 0, WIN_W, WIN_H}
	//renderer.SetViewport(&viewport_rect)
    TEST_TOKENS_LEN := len(test_tokens)

	qsize := int(math.RoundToEven(float64(WIN_H)/float64(font.Height()))) + 1
	stack := NewStack(len(all_lines))

	for i := 0; i < qsize; i++ {
		all_lines[i].texture = make_ttf_texture(renderer, font, strings.Join(all_lines[i].words, " "), &sdl.Color{R: 0, G: 0, B: 0, A: 0})
	}

	list := NewList()
	for i := 0; i < qsize; i++ {
		list.Append(&all_lines[i])
	}
	NEXT_ELEMENT := qsize

	re := make([]sdl.Rect, qsize)
	rey := genY(font, qsize)
	for i := 0; i < qsize; i++ {
		re[i] = sdl.Rect{X: 0, Y: int32(rey[i]), W: WIN_W, H: int32(font.Height())}
		all_lines[i].bg_rect.Y = re[i].Y
		for j := 0; j < len(all_lines[i].word_rects); j++ {
			all_lines[i].word_rects[j].Y = re[i].Y
		}
	}

    //foobar := TestMakeTexture(renderer, font, "fooba$", &sdl.Color{0,0,0,255})
    //_, _, fw, fh, _ := foobar.Query()
    //TestUpdateTexture(renderer, foobar, font, "boo", &sdl.Color{0,0,0,255})
    //TestUpdateTexture(renderer, foobar, font, "gooz", &sdl.Color{0,0,0,255})
    //TestUpdateTexture(renderer, foobar, font, "whatever man, this is bullshit", &sdl.Color{0,0,0,255})
    //TestClearTexture(renderer, foobar, font, &sdl.Color{0, 0, 0, 255})
    //TestUpdateTexture(renderer, foobar, font, "A wise man once said: ....", &sdl.Color{0,0,0,255})
    //TestClearTexture(renderer, foobar, font, &sdl.Color{0, 0, 0, 255})
    //TestUpdateTexture(renderer, foobar, font, "....", &sdl.Color{0,0,0,255})
    //foobar_rect := sdl.Rect{int32(X_OFFSET), 0, fw, fh}
    //defer foobar.Destroy()

    scrollbar := &Scrollbar{drag: false, selected: false, rect: sdl.Rect{int32(LINE_LENGTH+X_OFFSET-5), 0, 5, 30}}

	for running {
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch t := event.(type) {
			case *sdl.QuitEvent:
				running = false
			case *sdl.WindowEvent:
				switch t.Event {
				case sdl.WINDOWEVENT_SIZE_CHANGED:
					new_win_w, new_win_h := t.Data1, t.Data2
					if new_win_w <= int32(LINE_LENGTH) {
						wrap_line = true
					} else {
						wrap_line = false
					}

					if new_win_w > WIN_W && new_win_h > WIN_H {
						cmd.Resize(new_win_w, new_win_h)
						wrapline.y2 = new_win_h
					} else {
						cmd.Resize(WIN_W, new_win_h)
						wrapline.y2 = new_win_h
					}
				}
			case *sdl.MouseMotionEvent:
				//fmt.Println(t.X, t.Y)
				current := list.head.next
				for i := 0; i < list.Size(); i++ {
					check_collision_mouse_over_words(t, &current.data.word_rects, &current.data.mouse_over_word)
					current = current.next
				}
				check_collision_mouse_over_words(t, &gfonts.ttf_rects, &mouseover_word_texture_FONT)
                scrollbar.selected = check_collision(t, &scrollbar.rect)
                if scrollbar.drag {
                    scrollbar.rect.Y += t.YRel
                    if scrollbar.rect.Y <= 0 {
                        scrollbar.rect.Y = 0
                    }
                    if (scrollbar.rect.Y+scrollbar.rect.H) >= WIN_H {
                        scrollbar.rect.Y = WIN_H-scrollbar.rect.H
                    }
                }
			case *sdl.MouseWheelEvent:
				switch {
				case t.Y > 0:
					move_text_up = true
				case t.Y < 0:
					move_text_down = true
				}
			case *sdl.MouseButtonEvent:
				switch t.Type {
				case sdl.MOUSEBUTTONDOWN:
				case sdl.MOUSEBUTTONUP:
					print_word = true
				}

                if scrollbar.selected && t.Type == sdl.MOUSEBUTTONDOWN && t.State == sdl.PRESSED {
                    scrollbar.drag = true
                } else {
                    scrollbar.drag = false
                }
			case *sdl.TextInputEvent:
				if cmd.show {
					cmd.WriteChar(renderer, gfonts, t.Text[0])
				}
			case *sdl.KeyboardEvent:
				if cmd.show {
					if t.Keysym.Sym == sdl.K_BACKSPACE {
						if t.Repeat > 0 {
							cmd.Reset(renderer, curr_char_w, gfonts.current_font, gfonts.current_font_w, gfonts.current_font_h)
						}
					}
					switch t.Type {
					case sdl.KEYDOWN:
					case sdl.KEYUP:
						if t.Keysym.Mod == sdl.KMOD_LCTRL && t.Keysym.Sym == sdl.K_v {
							if sdl.HasClipboardText() {
								str, _ := sdl.GetClipboardText()
								cmd.WriteString(renderer, gfonts, str)
							}
						}
					}
				}
				switch t.Type {
				case sdl.KEYDOWN:
				case sdl.KEYUP:
					if t.Keysym.Sym == sdl.K_SPACE {
						if !cmd.show {
							cmd.show = true
						}
					} else {
						switch t.Keysym.Sym {
						case sdl.KEYDOWN:
						case sdl.K_TAB:
							if cmd.show {
								cmd.show = false
							}
						case sdl.K_BACKSPACE:
							cmd.Reset(renderer, curr_char_w, gfonts.current_font, gfonts.current_font_w, gfonts.current_font_h)
						case sdl.K_RETURN:
							if cmd.show {
								if len(cmd.input_buffer.String()) > 0 {
									cmd.MakeNULL()
								}
							}
						case sdl.K_UP:
							move_text_up = true
						case sdl.K_DOWN:
							move_text_down = true
						}
					}
				}
				if t.Keysym.Sym == sdl.K_ESCAPE {
					running = false
				}
				if t.Keysym.Sym == sdl.K_LEFT {
					println("SHOULD SCROLL FONT back")
				}
				if t.Keysym.Sym == sdl.K_RIGHT {
					println("SHOULD SCROLL FONT forward")
				}
			default:
				continue
			}
		}
		renderer.SetDrawColor(255, 255, 255, 0)
		renderer.Clear()

        draw_rect_with_border_filled(renderer, &scrollbar.rect, &sdl.Color{111, 111, 111, 90})
        if scrollbar.drag || scrollbar.selected {
            draw_rect_with_border_filled(renderer, &scrollbar.rect, &sdl.Color{111, 111, 111, 255})
        }

        //draw_rect_with_border_filled(renderer, &foobar_rect, &sdl.Color{212, 111, 222, 30})
        //renderer.Copy(foobar, nil, &foobar_rect)

		current := list.head.next
		for i := 0; i < list.Size(); i++ {
			renderer.Copy(current.data.texture, nil, &current.data.bg_rect)
			for j := 0; j < len(current.data.mouse_over_word); j++ {
				if current.data.mouse_over_word[j] {
					engage_loop = true
				}
			}
			current = current.next
		}

        if print_word && !engage_loop {
            print_word = false
        }

		if engage_loop && !cmd.show {
			current := list.head.next
			for i := 0; i < list.Size(); i++ {
				for j := 0; j < len(current.data.mouse_over_word); j++ {
					if current.data.mouse_over_word[j] && current.data.words[j] != "\n" {
						draw_rect_without_border(renderer, &current.data.word_rects[j], &sdl.Color{R: 255, G: 100, B: 200, A: 100})
						if print_word && current.data.words[j] != "\n" {
							fmt.Printf("%s\n", current.data.words[j])
							print_word = false
						}
					}
				}
				current = current.next
			}
			engage_loop = false
		}

		if move_text_down {
			move_text_down = false
			inc_dbg_str = true
			stack.Push(list.PopFromHead().data)
			stack.GetLast().texture.Destroy()
			stack.GetLast().texture = nil
			all_lines[NEXT_ELEMENT].texture = make_ttf_texture(renderer, font, strings.Join(all_lines[NEXT_ELEMENT].words, " "), &sdl.Color{R: 0, G: 0, B: 0, A: 255})
			list.Append(&all_lines[NEXT_ELEMENT])
			NEXT_ELEMENT += 1
            scrollbar.CalcPos(NEXT_ELEMENT, TEST_TOKENS_LEN)
			current := list.head.next
			for i := 0; i < list.Size(); i++ {
				current.data.bg_rect.Y = re[i].Y
				for j := 0; j < len(current.data.word_rects); j++ {
					current.data.word_rects[j].Y = re[i].Y
				}
				current = current.next
			}
		}

		if move_text_up {
			if stack.IsEmpty() != true {
				move_text_up = false
				inc_dbg_str = true
				list.PopFromTail()
				stack.GetLast().texture = make_ttf_texture(renderer, font, strings.Join(stack.GetLast().words, " "), &sdl.Color{R: 0, G: 0, B: 0, A: 255})
				list.Prepend(stack.Pop())
				NEXT_ELEMENT -= 1
                scrollbar.CalcPos(NEXT_ELEMENT, TEST_TOKENS_LEN)
				current := list.head.next
				for i := 0; i < list.Size(); i++ {
					current.data.bg_rect.Y = re[i].Y
					for j := 0; j < len(current.data.word_rects); j++ {
						current.data.word_rects[j].Y = re[i].Y
					}
					current = current.next
				}
			}
		}

		if wrap_line {
			current := list.head.next
			for i := 0; i < list.Size(); i++ {
				draw_rect_without_border(renderer, &current.data.bg_rect, &sdl.Color{R: 100, G: 255, B: 255, A: 100})
				current = current.next
			}
		}

		if cmd.show {
			current := list.head.next
			for i := 0; i < list.Size(); i++ {
				for j := 0; j < len(current.data.word_rects); j++ {
					draw_rect_without_border(renderer, &current.data.word_rects[j], &sdl.Color{R: 255, G: 100, B: 200, A: 100})
				}
				current = current.next
			}

            for i := range re {
                draw_rect_with_border(renderer, &re[i], &sdl.Color{R: 200, G: 100, B: 0, A: 200})
            }

			draw_rect_with_border_filled(renderer, &cmd.bg_rect, &sdl.Color{R: 255, G: 10, B: 100, A: cmd.alpha_value + 40})
			draw_rect_with_border(renderer, &cmd.ttf_rect, &sdl.Color{R: 255, G: 255, B: 255, A: 0})

			renderer.Copy(cmd.ttf_texture, nil, &cmd.ttf_rect)

			draw_rect_with_border_filled(renderer, &cmd.cursor_rect, &sdl.Color{R: 0, G: 0, B: 0, A: cmd.alpha_value})

			draw_rect_without_border(renderer, &gfonts.bg_rect, &sdl.Color{R: 255, G: 255, B: 255, A: 255})
			//renderer.SetClipRect(&gfonts.bg_rect) we can make our stuff dissapear

			for i := 0; i < len(gfonts.textures); i++ {
				renderer.Copy(gfonts.textures[i], nil, &gfonts.ttf_rects[i])
				if mouseover_word_texture_FONT[i] == true {
					draw_rect_without_border(renderer, &gfonts.highlight_rect[i], &sdl.Color{R: 0, G: 0, B: 0, A: 100})
                    if print_word {
                        println(gfonts.fonts[i].name)
                        print_word = false
                    }
				}
			}

			if inc_dbg_str { // A DIRTY HACK
				inc_dbg_str = false
				dbg_str = make_console_text(NEXT_ELEMENT, len(test_tokens))
				dbg_ttf = reload_ttf_texture(renderer, dbg_ttf, font, dbg_str, &sdl.Color{R: 0, G: 0, B: 0, A: 255})
			}

			draw_rect_with_border_filled(renderer, &dbg_rect, &sdl.Color{R: 180, G: 123, B: 55, A: 255})
			renderer.Copy(dbg_ttf, nil, &dbg_rect)
		}

		renderer.SetDrawColor(255, 100, 0, 100)
		renderer.DrawLine(wrapline.x1+int32(X_OFFSET), wrapline.y1, wrapline.x2+int32(X_OFFSET), wrapline.y2)

		renderer.Present()

		//NOTE: this is not for framerate independance
		//NOTE: it's probably also slower than calling SDL_Timer/SDL_Delay functions
		//NOTE: OR try using sdl2_gfx package functions like: FramerateDelay...
		<-ticker.C
	}

	ticker.Stop()
	renderer.Destroy()
	window.Destroy()

	current := list.head.next
	for i := 0; i < list.Size(); i++ {
		current.data.texture.Destroy()
		current.data.texture = nil
		current = current.next
	}

	if cmd.ttf_texture != nil {
		cmd.ttf_texture.Destroy()
		cmd.ttf_texture = nil
	}

	dbg_ttf.Destroy()

	for index := range ttf_font_list {
		gfonts.fonts[index].data.Close()
		gfonts.current_font.Close()
		gfonts.fonts[index].data = nil
		gfonts.textures[index].Destroy()
	}
	font.Close()

	ttf.Quit()
	sdl.Quit()

	// PROFILING SNIPPET
	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal("could not *create* MEM profile: ", err)
		}
		runtime.GC()
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("could not *start*  MEM profile: ", err)
		}
		f.Close()
	}
	// PROFILING SNIPPET
}

func load_font(name string, size int) *ttf.Font {
	var font *ttf.Font
	var err error

	if font, err = ttf.OpenFont(name, size); err != nil {
		panic(err)
	}
	return font
}

func reload_font(font *ttf.Font, name string, size int) *ttf.Font {
	var err error

	if font != nil {
		font.Close()
		if font, err = ttf.OpenFont(name, size); err != nil {
			panic(err)
		}
		return font
	}
	return font
}

func make_ttf_texture(renderer *sdl.Renderer, font *ttf.Font, text string, color *sdl.Color) *sdl.Texture {
	var surface *sdl.Surface
	var texture *sdl.Texture

	assert_if(len(text) <= 0)

	surface, _ = font.RenderUTF8Blended(text, *color)
	texture, _ = renderer.CreateTextureFromSurface(surface)
	surface.Free()
	sdl.ClearError()

	return texture
}

func reload_ttf_texture(r *sdl.Renderer, tex *sdl.Texture, f *ttf.Font, s string, c *sdl.Color) *sdl.Texture {
	if tex != nil {
		tex.Destroy()
		var surface *sdl.Surface

		surface, _ = f.RenderUTF8Blended(s, *c)
		tex, _ = r.CreateTextureFromSurface(surface)
		surface.Free()
		sdl.ClearError()

		return tex
	}
	return tex
}

func generate_and_populate_lines(r *sdl.Renderer, font *ttf.Font, dest *[]Line, tokens *[]string) {
	for index := 0; index < len(*tokens); index++ {
		new_ttf_texture_line(r, font, &(*dest)[index], (*tokens)[index])
	}
}

func new_ttf_texture_line(rend *sdl.Renderer, font *ttf.Font, line *Line, line_text string) {
	assert_if(len(line_text) == 0)

	//line.texture = make_ttf_texture(rend, font, line_text, &sdl.Color{R: 0, G: 0, B: 0, A: 0})

	text := strings.Split(line_text, " ")
	text_len := len(text)

	assert_if(text_len == 0)

	line.word_rects = make([]sdl.Rect, text_len)
	line.mouse_over_word = make([]bool, text_len)
	line.words = make([]string, text_len)
	copy(line.words, text)

	x, y, _ := font.SizeUTF8(" ")
	tw := x * len(line_text)

	// TODO danger: gobal vars are bad!
	move_x := X_OFFSET
	ix := 0
	for index := 0; index < text_len; index++ {
		ix = x * len(text[index])
		line.word_rects[index] = sdl.Rect{X: int32(move_x), Y: int32(-y), W: int32(ix), H: int32(y)}
		move_x += (ix + x)
	}
	line.bg_rect = sdl.Rect{X: int32(X_OFFSET), Y: int32(-y), W: int32(tw), H: int32(y)}
	text = nil
}

func check_collision_mouse_over_words(event *sdl.MouseMotionEvent, rects *[]sdl.Rect, mouse_over *[]bool) {
	for index := range *rects {
		mx_gt_rx := event.X > (*rects)[index].X
		mx_lt_rx_rw := event.X < (*rects)[index].X+(*rects)[index].W
		my_gt_ry := event.Y > (*rects)[index].Y
		my_lt_ry_rh := event.Y < (*rects)[index].Y+(*rects)[index].H

		if (mx_gt_rx && mx_lt_rx_rw) && (my_gt_ry && my_lt_ry_rh) {
			(*mouse_over)[index] = true
		} else {
			(*mouse_over)[index] = false
		}
	}
}

func check_collision(event *sdl.MouseMotionEvent, rect *sdl.Rect) bool {
	result := false
	mx_gt_rx := event.X > rect.X
	mx_lt_rx_rw := event.X < rect.X+rect.W
	my_gt_ry := event.Y > rect.Y
	my_lt_ry_rh := event.Y < rect.Y+rect.H

	if (mx_gt_rx && mx_lt_rx_rw) && (my_gt_ry && my_lt_ry_rh) {
		result = true
	}
	return result
}

func do_wrap_lines(str string, max_len int, xsize int) []string {
	assert_if(len(str) <= 1)

	result := make([]string, determine_nwrap_lines([]string{str}, max_len, xsize))

	pos := 0
	if (len(str)*xsize)+X_OFFSET <= max_len {
		result[pos] = str
		return result
	} else {
		start := 0
		mmax := int(math.RoundToEven(float64(max_len/xsize))) - 1 // use math.Round instead?
		slice := str[start:mmax]
		end := mmax
		slice_len := 0
		for end < len(str) {
			slice_len = len(slice)
			if !is_space(string(slice[slice_len-1])) {
				for !is_space(string(slice[slice_len-1])) {
					end = end - 1
					slice_len = slice_len - 1
				}
			}
			end = end - 1 // remove space
			slice = str[start:end]
			result[pos] = slice
			pos += 1
			start = end + 1
			end = (end + mmax)
			if end > len(str) {
				slice = str[start : end-(end-len(str))]
				result[pos] = slice
				pos += 1
				break
			}
			slice = str[start:end]
		}
	}
	return result
}

// TODO
// This function will fail if MAX_LEN
// is small enough to trigger is_space ifinite loop!
func determine_nwrap_lines(str []string, max_len int, xsize int) int32 {
	var result int32
	for index := 0; index < len(str); index++ {
		if (len(str[index])*xsize)+X_OFFSET <= max_len {
			result += 1
		} else {
			start := 0
			mmax := int(math.RoundToEven(float64(max_len/xsize))) - 1 // use math.Round instead?
			//println(mmax > len(str[index]), "index", index, "strlen", len(str[index]), "mmax", mmax)
			//assert_if(mmax > len(str[index]))
			slice := str[index][start:mmax]
			end := mmax
			slice_len := 0
			for end < len(str[index]) {
				slice_len = len(slice)
				if !is_space(string(slice[slice_len-1])) {
					for !is_space(string(slice[slice_len-1])) {
						end = end - 1
						slice_len = slice_len - 1
					}
				}
				end = end - 1 // remove space
				slice = str[index][start:end]
				result += 1
				start = end + 1
				end = (end + mmax)
				if end > len(str[index]) {
					slice = str[index][start : end-(end-len(str[index]))]
					result += 1
					break
				}
				slice = str[index][start:end]
			}
		}
	}
	return result
}

func assert_if(cond bool) {
	if cond {
		panic("assertion failed")
	}
}

func is_alpha(schr string) bool {
	return (schr >= "A") && (schr <= "z")
}

func is_space(s string) bool {
	return s == " "
}

func get_word_lengths(s *string) []int {
	var result []int
	curr := 0
	for index := 0; index < len(*s); index++ {
		//if (string((*s)[index]) == "\n") {
		//    break
		//}
		//if (string((*s)[index]) == "\r") {
		//    break
		//}
		if !is_space(string((*s)[index])) {
			curr += 1
		} else {
			result = append(result, curr)
			curr = 0
		}
	}
	if curr > 0 {
		result = append(result, curr)
	}
	return result
}

func sum_word_lengths(n []int) int {
	sum := 0
	for i := 0; i < len(n); i++ {
		sum += n[i]
	}
	return sum
}

func draw_rect_with_border(renderer *sdl.Renderer, rect *sdl.Rect, c *sdl.Color) {
	renderer.SetDrawColor((*c).R, (*c).G, (*c).B, (*c).A)
	renderer.DrawRect(rect)
}

func draw_rect_with_border_filled(renderer *sdl.Renderer, rect *sdl.Rect, c *sdl.Color) {
	renderer.SetDrawColor((*c).R, (*c).G, (*c).B, (*c).A)
	renderer.FillRect(rect)
	renderer.DrawRect(rect)
}

func draw_rect_without_border(renderer *sdl.Renderer, rect *sdl.Rect, c *sdl.Color) {
	renderer.SetDrawColor((*c).R, (*c).G, (*c).B, (*c).A)
	renderer.FillRect(rect)
}

func number_as_string(n int) string {
	return strconv.Itoa(n)
}

func make_console_text(current int, total int) string {
	return strings.Join([]string{"LINE: ", strconv.Itoa(current), "/", strconv.Itoa(total), " [",
		strconv.FormatFloat(float64((float64(current)/float64(total))*100), 'f', 1, 32), "%]"}, "")
}

func v2_to_int32(v *v2) (int32, int32) {
	return int32((*v).x), int32((*v).y)
}

func v2_add(a *v2, b *v2) v2 {
	return v2{(*a).x + (*b).x, (*a).y + (*b).y}
}

func v2_sub(a *v2, b *v2) v2 {
	return v2{(*a).x - (*b).x, (*a).y - (*b).y}
}

func v2_mult(a *v2, scalar float32) v2 {
	return v2{(*a).x * scalar, (*a).y * scalar}
}

func v2_div(a *v2, scalar float32) v2 {
	return v2{(*a).x / scalar, (*a).y / scalar}
}

func v2_mag(v *v2) float32 {
	return float32(math.Sqrt(float64((*v).x*(*v).x) + float64((*v).y*(*v).y)))
}

func lerp(a float32, b float32, t float32) float32 {
	if t > 1 || t < 0 {
		return 0.0
	}
	return (1-t)*a + t*b
}

func normalize(n float32, max float32) float32 {
	return n / max
}

func get_filenames(path string, format []string) []string {
	var result []string

	list, err := ioutil.ReadDir(path)
	if err != nil {
		panic(err)
	}

	for index := 0; index < len(list); index++ {
		for i := 0; i < len(format); i++ {
			if strings.Contains(list[index].Name(), format[i]) {
				result = append(result, list[index].Name())
				break
			}
		}
	}
	list = nil
	return result
}

func get_filedata(path string, filename string) []byte {
	file_stat, err := os.Stat(path + filename)
	if err != nil {
		panic(err)
	}

	result := make([]byte, file_stat.Size())

	file, err := os.Open(path + filename)
	if err != nil {
		panic(err)
	}

	file.Read(result)
	file.Close()

	return result
}

func allocate_font_space(font *FontSelector, size int) {
	font.fonts = make([]Font, size)
	font.textures = make([]*sdl.Texture, size)
	font.ttf_rects = make([]sdl.Rect, size)
	font.highlight_rect = make([]sdl.Rect, size)
}

func generate_fonts(font *FontSelector, ttf_font_list []string, font_dir string) {
	CURRENT := 6 // magic number
	for index, element := range ttf_font_list {
		if CURRENT == index {
			font.current_font = load_font(font_dir+element, TTF_FONT_SIZE)
			w, h, _ := font.current_font.SizeUTF8(" ")
			skp := font.current_font.LineSkip()
			font.current_font_w = w
			font.current_font_h = h
			font.current_font_skip = skp
		}
		font.fonts[index].data = load_font(font_dir+element, TTF_FONT_SIZE_FOR_FONT_LIST)
		font.fonts[index].name = element
	}
}

func generate_rects_for_fonts(renderer *sdl.Renderer, font *FontSelector) {
	font.bg_rect = sdl.Rect{}
	adder_y := 0
	for index, element := range font.fonts {
		gx, gy, _ := (*font).fonts[index].data.SizeUTF8(" ")
		font.fonts[index].size = gx * len(element.name)

		font.textures[index] = make_ttf_texture(renderer, font.fonts[index].data,
			font.fonts[index].name,
			&sdl.Color{R: 0, G: 0, B: 0, A: 0})

		font.ttf_rects[index] = sdl.Rect{X: 0, Y: int32(adder_y), W: int32(gx * len(element.name)), H: int32(gy)}

		if font.bg_rect.W < font.ttf_rects[index].W {
			font.bg_rect.W = font.ttf_rects[index].W
		}

		font.highlight_rect[index] = font.ttf_rects[index]

		font.bg_rect.H += font.ttf_rects[index].H
		adder_y += gy

		if index == len(font.fonts)-1 {
			for i := 0; i < len(font.ttf_rects); i++ {
				font.highlight_rect[i].W = font.bg_rect.W
			}
		}
	}
}

func genY(font *ttf.Font, size int) []int {
	result := make([]int, size)

	for i := 0; i < size; i++ {
		result[i] = i * font.LineSkip()
	}
	return result
}

// https://github.com/zielmicha/SDL2/blob/master/src/render/SDL_render.c 
// refactor this function!
func TestMakeTexture(renderer *sdl.Renderer, font *ttf.Font, text string, color *sdl.Color) *sdl.Texture{
    var surface *sdl.Surface
    var ttf_texture *sdl.Texture
    surface, _ = font.RenderUTF8Blended(text, *color)
    ttf_texture, _ = renderer.CreateTexture(sdl.PIXELFORMAT_RGBA8888, sdl.TEXTUREACCESS_STREAMING, int32(LINE_LENGTH), surface.H)
    fmt, _ := sdl.AllocFormat(sdl.PIXELFORMAT_RGBA8888)
    converted, _ := surface.Convert(fmt, 0)
    ttf_texture.Update(&sdl.Rect{0, 0, surface.W, surface.H}, converted.Pixels(), int(converted.Pitch))
    ttf_texture.SetBlendMode(sdl.BLENDMODE_BLEND)
    fmt.Free()
    surface.Free()
    converted.Free()
    return ttf_texture
}

func TestUpdateTexture(renderer *sdl.Renderer, texture *sdl.Texture, font *ttf.Font, text string, color *sdl.Color) {
    var surface *sdl.Surface
    surface, _ = font.RenderUTF8Blended(text, *color)
    fmt, _ := sdl.AllocFormat(sdl.PIXELFORMAT_RGBA8888)
    converted, _ := surface.Convert(fmt, 0)
    texture.Update(&sdl.Rect{0, 0, surface.W, surface.H}, converted.Pixels(), int(converted.Pitch))
    fmt.Free()
    surface.Free()
    converted.Free()
}

func TestClearTexture(renderer *sdl.Renderer, texture *sdl.Texture, font *ttf.Font, color *sdl.Color) {
    var surface *sdl.Surface
    surface, _ = font.RenderUTF8Blended(" ", *color)
    fmt, _ := sdl.AllocFormat(sdl.PIXELFORMAT_RGBA8888)
    converted, _ := surface.Convert(fmt, 0)
    bytes, _, _ := texture.Lock(nil)
    copy(bytes, converted.Pixels())
    texture.Unlock()
    fmt.Free()
    surface.Free()
    converted.Free()
}

func (sc *Scrollbar) CalcPos(current int, total int) {
	sc.rect.Y = int32(float64(current)/float64(total)*float64(WIN_H)) - sc.rect.H
    if sc.rect.Y < 0 {
        sc.rect.Y = 0
    }
}
