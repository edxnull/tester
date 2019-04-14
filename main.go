// +build windows, 386

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
	"runtime/trace"
	"strconv"
	"strings"
	"time"
)

// GENERAL
// [ ] https://bell0bytes.eu/the-game-loop/
// [ ] https://dewitters.com/dewitters-gameloop/
// [ ] http://gameprogrammingpatterns.com/game-loop.html
// [ ] http://svanimpe.be/blog/game-loops-fx
// [ ] https://gafferongames.com/post/fix_your_timestep/
// [ ] http://blog.moagrius.com/actionscript/jsas-understanding-easing/
// [ ] use https://godoc.org/github.com/fsnotify/fsnotify for checking if our settings file has been changed?

// [ ] try in main_loop: t := time.Now() [...] time.Sleep(time.Second/time.Duration(fps) - time.Since(t)) where fps = any num from 10..60

// [ ] separate updating and rendering?
// [ ] fmt.Println(runtime.Caller(0)) use this to get a LINENR when calculating unique ID's for IMGUI
// [ ] maybe it would be possible to use unicode symbols like squares/triangles to indicate clickable objects?
// [ ] refactor FontSelector
// [ ] make sure that we don't exceed max sdl.texture width
// [ ] should we compress strings?? Huffman encoding?
// [ ] should we use hash algorithms?
// [ ] searching
// [ ] justify text
// [ ] fuzzy search
// [ ] copy text
// [ ] copy & pasting commands
// [ ] get an N and a list of unique words in a file
// [ ] save words to a trie tree?
// [ ] figure out what to do about languages like left to right and asian languages
// [ ] export/import csv
// [ ] make sure we handle utf8
// [ ] cmd input commands + parsing
// [ ] [bug_icon] in-app file a bug button & menu
// [ ] should we keep fonts in memory? or free them instead?
// [ ] https://en.wikipedia.org/wiki/Newline
//     use sdl.GetPlatform() || [runtime.GOOS == ""] || [foo_unix.go; foo_windows.go style]
// [ ] try to implement imgui style widgets: https://sol.gfxile.net/imgui/index.html
// [ ] add proper error handling
// [ ] add logs???

// DB RELATED
// [ ] use bbolt key/value store as a database?

// SDL RELATED
// [ ] optimize TextBox Update and Clear (somehow)
// [ ] try using r.SetScale() => sdl.SetLogicalSize + sdl.SetHint(sdl.HINT_RENDER_SCALE_QUALITY, "linear")
// [ ] use r.DrawLines() to draw triangles?
// [ ] use r.DrawRects() r.FillRects() for speed?
// [ ] use (t *sdl.Texture) GLBind/GLUnbind for faster rendering?
// [ ] use r.SetClipRect r.GetClipRect for rendering
// [ ] USE sdl.WINDOWEVENT_EXPOSED for proper redrawing
// [ ] renderer.SetLogicalSize(WIN_W, WIN_H) -> SetLogicalSize is important for device independant rendering!
// [ ] proper time handling like dt and such
// [ ] how can we not render everything on every frame?
// [ ] add error handling code like println(sdl.GetError())?

// VISUAL
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
// [ ] needs to support tags/breakpoints for situations where you can't hear clearly or don't understand

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

var (
	COLOR_WHITE = sdl.Color{R: 255, G: 255, B: 255, A: 255}
	COLOR_BLACK = sdl.Color{R: 0, G: 0, B: 0, A: 255}
	COLOR_RED   = sdl.Color{R: 255, G: 0, B: 0, A: 255}
	COLOR_GREEN = sdl.Color{R: 0, G: 255, B: 0, A: 255}
	COLOR_BLUE  = sdl.Color{R: 0, G: 0, B: 255, A: 255}
)

type LineMetaData struct {
	words           []string
	word_rects      []sdl.Rect
	mouse_over_word []bool
}

type TextBox struct {
	data       []*sdl.Texture
	texture_w  int32
	texture_h  int32
	data_rects []sdl.Rect
	metadata   []*LineMetaData // store [START:END] instead?
	fmt        *sdl.PixelFormat
}

type DebugWrapLine struct {
	x1, y1 int32
	x2, y2 int32
}

type Scrollbar struct {
	drag     bool
	selected bool
	rect     sdl.Rect
}

type FontSelector struct {
	show              bool
	fonts             []Font
	current_font      *ttf.Font
	current_font_w    int
	current_font_h    int
	current_font_skip int
	current_name      string
	alpha_value       uint8
	alpha_f32         float32
	bg_rect           sdl.Rect
	ttf_rects         []sdl.Rect
	highlight_rect    []sdl.Rect
	cursor_rect       sdl.Rect
	textures          []*sdl.Texture
}

// [      [o][x]]
const NB = 2

type Toolbar struct {
	bg_rect      sdl.Rect
	bg_color     sdl.Color
	texture      [NB]*sdl.Texture
	texture_rect [NB]sdl.Rect
}

const CPN = 5

type ColorPicker struct {
	bg_rect       sdl.Rect
	bg_color      sdl.Color
	show          bool
	clicked       bool
	font          *ttf.Font
	texture       *sdl.Texture
	texture_rect  sdl.Rect
	color         [CPN]sdl.Color
	rects         [CPN]sdl.Rect
	rect_textures [CPN]*sdl.Texture
	rect_bgs      [CPN]sdl.Rect
	toolbar       Toolbar
}

const (
	CURSOR_TYPE_ARROW = iota
	CURSOR_TYPE_HAND
	CURSOR_TYPE_SIZEWE
)

func main() {
	// PROFILING SNIPPET

	var debug bool
	var do_trace bool

	flag.BoolVar(&debug, "debug", false, "debug needs a bool value: -debug=true")
	flag.BoolVar(&do_trace, "trace", false, "trace needs a bool value: -trace=true")

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

	if debug {
		println("we can put debug if's everywhere!")
	}

	if do_trace {
		tr, err := os.Create("trace.out")
		if err != nil {
			panic(err)
		}
		defer tr.Close()

		err = trace.Start(tr)
		if err != nil {
			panic(err)
		}
		defer trace.Stop()
	}

	runtime.LockOSThread() // NOTE: not sure I need this here!

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

	cursors := []*sdl.Cursor{
		sdl.CreateSystemCursor(sdl.SYSTEM_CURSOR_ARROW),
		sdl.CreateSystemCursor(sdl.SYSTEM_CURSOR_HAND),
		sdl.CreateSystemCursor(sdl.SYSTEM_CURSOR_SIZEWE),
	}

	defer sdl.FreeCursor(cursors[CURSOR_TYPE_ARROW])
	defer sdl.FreeCursor(cursors[CURSOR_TYPE_HAND])
	defer sdl.FreeCursor(cursors[CURSOR_TYPE_SIZEWE])

	sdl.SetCursor(cursors[CURSOR_TYPE_ARROW])
	cursor_state := CURSOR_TYPE_ARROW

	filename := "HP01.txt"
	font_dir := "./fonts/"
	text_dir := "./text/"

	line_tokens := strings.Split(string(get_filedata(text_dir, filename)), "\r\n") // "\r\n" instead of "\n"

	ticker := time.NewTicker(time.Second / 40)

	ttf_font_list := get_filenames(font_dir, []string{"ttf", "otf"})
	txt_list := get_filenames(text_dir, []string{".txt"})
	fmt.Println(txt_list)

	var gfonts FontSelector
	allocate_font_space(&gfonts, len(ttf_font_list))
	generate_fonts(&gfonts, ttf_font_list, font_dir)

	font := gfonts.current_font

	generate_rects_for_fonts(renderer, &gfonts)

	test_tokens := WrapLines(line_tokens, LINE_LENGTH, gfonts.current_font_w)

	TEST_TOKENS_LEN := len(test_tokens)

	linemeta := make([]LineMetaData, TEST_TOKENS_LEN)
	generate_line_metadata(font, &linemeta, &test_tokens)

	cmd := NewCmdConsole(renderer, font)

	dbg_str := make_console_text(0, TEST_TOKENS_LEN)
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
	page_up := false
	page_down := false

	wrapline := DebugWrapLine{int32(LINE_LENGTH), 0, int32(LINE_LENGTH), WIN_H}

	curr_char_w := 0

	// TODO: this ain't working properly oon zoom out's
	qsize := int(math.RoundToEven(float64(WIN_H)/float64(font.Height()))) + 1

	NEXT_ELEMENT := qsize
	START_ELEMENT := 0

	textbox := TextBox{
		data:       make([]*sdl.Texture, qsize),
		texture_w:  0,
		texture_h:  0,
		data_rects: make([]sdl.Rect, qsize),
		metadata:   make([]*LineMetaData, qsize),
		fmt:        nil,
	}

	for i := 0; i < len(textbox.data); i++ {
		textbox.metadata[i] = &linemeta[i]
	}

	textbox.CreateEmpty(renderer, font, sdl.Color{R: 0, G: 0, B: 0, A: 255})
	textbox.Update(renderer, font, test_tokens[0:qsize], sdl.Color{R: 0, G: 0, B: 0, A: 255})

	re := make([]sdl.Rect, qsize)
	rey := genY(font, qsize)
	for i := 0; i < qsize; i++ {
		re[i] = sdl.Rect{X: int32(X_OFFSET), Y: int32(rey[i]), W: int32(LINE_LENGTH), H: int32(font.Height())}
		for j := 0; j < len(textbox.metadata[i].word_rects); j++ {
			textbox.metadata[i].word_rects[j].Y = re[i].Y
		}
	}

	scrollbar := &Scrollbar{drag: false, selected: false, rect: sdl.Rect{X: int32(LINE_LENGTH + X_OFFSET - 5), Y: 0, W: 5, H: 30}}

	test_font_name := gfonts.current_name
	test_font_size := TTF_FONT_SIZE

	easerout := struct {
		rect           sdl.Rect
		animate        bool
		animation_time float32
	}{sdl.Rect{0, 50, 100, 100}, true, 0.0}

	easerin := struct {
		rect           sdl.Rect
		animate        bool
		animation_time float32
	}{sdl.Rect{0, 150, 100, 100}, true, 0.0}

	easerinout := struct {
		rect           sdl.Rect
		animate        bool
		animation_time float32
	}{sdl.Rect{0, 250, 100, 100}, true, 0.0}

	color_picker := ColorPicker{
		bg_rect:      sdl.Rect{X: 0, Y: 0, W: 80, H: 40},
		bg_color:     sdl.Color{R: 100, G: 100, B: 255, A: 255},
		font:         load_font(font_dir+"Inconsolata-Regular.ttf", 9),
		texture_rect: sdl.Rect{X: 0, Y: 0, W: 80, H: 40},
		color: [5]sdl.Color{
			sdl.Color{R: 100, G: 160, B: 50, A: 160},
			sdl.Color{R: 100, G: 180, B: 50, A: 180},
			sdl.Color{R: 100, G: 200, B: 50, A: 200},
			sdl.Color{R: 100, G: 220, B: 50, A: 220},
			sdl.Color{R: 100, G: 240, B: 50, A: 240},
		},
	}
	color_picker.texture = make_ttf_texture(renderer, color_picker.font, "this is our demo popup", &sdl.Color{R: 0, G: 0, B: 0, A: 0})

	cp := color_picker.bg_color // ! only used here
	color_picker.toolbar = Toolbar{
		bg_rect:  sdl.Rect{color_picker.bg_rect.X, color_picker.bg_rect.Y, color_picker.bg_rect.W, 10},
		bg_color: sdl.Color{cp.R, cp.G - 22, cp.B - 50, cp.A - 10},
	}

	color_picker.toolbar.texture[0] = make_ttf_texture(renderer, color_picker.font, "o", &COLOR_WHITE)
	color_picker.toolbar.texture[1] = make_ttf_texture(renderer, color_picker.font, "x", &COLOR_WHITE)

	_, _, cptw_0, cpth_0, _ := color_picker.toolbar.texture[0].Query()
	_, _, cptw_1, cpth_1, _ := color_picker.toolbar.texture[1].Query()

	for i := 0; i < len(color_picker.rects); i++ {
		color_picker.rect_textures[i] = make_ttf_texture(renderer, color_picker.font, strconv.Itoa(i), &sdl.Color{R: 0, G: 0, B: 0, A: 0})
	}

	_, _, qw, qh, _ := color_picker.texture.Query()
	color_picker.bg_rect.W = qw
	color_picker.texture_rect.W = qw
	color_picker.texture_rect.H = qh

	color_picker.toolbar.bg_rect.W = qw
	color_picker.toolbar.texture_rect[0] = sdl.Rect{X: color_picker.bg_rect.W - (cptw_0 * 2) - 1, Y: 0, W: cptw_0, H: cpth_0}
	color_picker.toolbar.texture_rect[1] = sdl.Rect{X: color_picker.bg_rect.W - (cptw_1), Y: 0, W: cptw_1, H: cpth_1}

	_, _, clrqw, clrqh, _ := color_picker.rect_textures[0].Query()
	acc := int32(0)
	MAGIC_PICKER_W := int32(clrqw)
	MAGIC_PICKER_SKIP := int32(clrqw + 7)
	for i := 0; i < len(color_picker.rects); i++ {
		color_picker.rects[i] = sdl.Rect{X: acc, Y: clrqh + 10, W: MAGIC_PICKER_W, H: clrqh}
		color_picker.rect_bgs[i] = sdl.Rect{X: acc, Y: clrqh + 10, W: MAGIC_PICKER_W + 6, H: clrqh}
		acc += MAGIC_PICKER_SKIP
	}

	color_picker.CenterRectAB() // TODO: REMOVE THIS TEMP HACK
	color_picker.CenterRects()  // TODO: REMOVE THIS TEMP HACK

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
				for i := 0; i < len(textbox.data); i++ {
					check_collision_mouse_over_words(t, &textbox.metadata[i].word_rects, &textbox.metadata[i].mouse_over_word)
				}
				check_collision_mouse_over_words(t, &gfonts.ttf_rects, &mouseover_word_texture_FONT)

				scrollbar.selected = check_collision(t, &scrollbar.rect)

				wrapline_selected := t.X == (wrapline.x1+int32(X_OFFSET)) && (t.Y >= wrapline.y1 && t.Y <= wrapline.y2)
				if wrapline_selected && !scrollbar.selected && !scrollbar.drag {
					println("SIZEWE")
					sdl.SetCursor(cursors[CURSOR_TYPE_SIZEWE])
					cursor_state = CURSOR_TYPE_SIZEWE
				}

				if scrollbar.selected && cursor_state != CURSOR_TYPE_HAND {
					println("HAND")
					sdl.SetCursor(cursors[CURSOR_TYPE_HAND])
					cursor_state = CURSOR_TYPE_HAND
				}

				if !scrollbar.selected && cursor_state != CURSOR_TYPE_ARROW && !scrollbar.drag && !wrapline_selected {
					println("ARROW")
					sdl.SetCursor(cursors[CURSOR_TYPE_ARROW])
					cursor_state = CURSOR_TYPE_ARROW
				}
				if scrollbar.drag {
					scrollbar.rect.Y += t.YRel
					if scrollbar.rect.Y <= 0 {
						scrollbar.rect.Y = 0
					}
					if (scrollbar.rect.Y + scrollbar.rect.H) >= WIN_H {
						scrollbar.rect.Y = WIN_H - scrollbar.rect.H
					}
					scrollbar.CalcPosDuringAction(int(scrollbar.rect.Y), TEST_TOKENS_LEN)
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

				if scrollbar.drag {
					scrollbar.drag = false
				}

				if scrollbar.selected && t.Type == sdl.MOUSEBUTTONDOWN && t.State == sdl.PRESSED {
					scrollbar.drag = true
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
					switch t.Keysym.Sym {
					case sdl.KEYDOWN:
					case sdl.K_TAB:
						cmd.show = !cmd.show
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
					case sdl.K_RIGHT:
						page_down = true
					case sdl.K_LEFT:
						page_up = true
					case sdl.K_d: // TESTING RESIZING FONTS
						test_font_size -= 1
						font = reload_font(font, font_dir+test_font_name, test_font_size)
						qw, _, _ := font.SizeUTF8(" ")
						test_tokens = nil
						test_tokens = WrapLines(line_tokens, LINE_LENGTH, qw)
						textbox.MakeNULL() // could this be a problem later?

						ClearMetadata(&linemeta)
						linemeta = nil
						TEST_TOKENS_LEN = len(test_tokens)
						linemeta = make([]LineMetaData, TEST_TOKENS_LEN)
						generate_line_metadata(font, &linemeta, &test_tokens)

						prev_qsize := qsize
						qsize = int(math.RoundToEven(float64(WIN_H)/float64(font.Height()))) + 1
						//println("prev_qsize", prev_qsize,"start:", START_ELEMENT, "next:", NEXT_ELEMENT, "qsize-prev_qsize:", qsize-prev_qsize)
						if START_ELEMENT >= prev_qsize {
							START_ELEMENT -= (qsize - prev_qsize)
						}
						NEXT_ELEMENT += (qsize - prev_qsize)
						println(qsize)
						println(qsize - prev_qsize)

						textbox.data = nil
						textbox.data_rects = nil
						textbox.metadata = nil
						textbox.fmt.Free()
						textbox = TextBox{
							data:       make([]*sdl.Texture, qsize),
							texture_w:  0,
							texture_h:  0,
							data_rects: make([]sdl.Rect, qsize),
							metadata:   make([]*LineMetaData, qsize),
							fmt:        nil,
						}

						for i := 0; i < len(textbox.data); i++ {
							textbox.metadata[i] = &linemeta[START_ELEMENT+i]
						}

						textbox.CreateEmpty(renderer, font, sdl.Color{R: 0, G: 0, B: 0, A: 255})
						textbox.Update(renderer, font, test_tokens[START_ELEMENT:NEXT_ELEMENT], sdl.Color{R: 0, G: 0, B: 0, A: 255})

						re = nil
						re = make([]sdl.Rect, qsize)
						rey = nil
						rey = genY(font, qsize)
						for i := 0; i < qsize; i++ {
							re[i] = sdl.Rect{X: int32(X_OFFSET), Y: int32(rey[i]), W: int32(LINE_LENGTH), H: int32(font.Height())}
							for j := 0; j < len(textbox.metadata[i].word_rects); j++ {
								textbox.metadata[i].word_rects[j].Y = re[i].Y
							}
						}
					case sdl.K_f: // TESTING RESIZING FONTS
						test_font_size += 1
						font = reload_font(font, font_dir+test_font_name, test_font_size)
						qw, _, _ := font.SizeUTF8(" ")
						test_tokens = nil
						test_tokens = WrapLines(line_tokens, LINE_LENGTH, qw)
						textbox.MakeNULL() // could this be a problem later?

						ClearMetadata(&linemeta)
						linemeta = nil
						TEST_TOKENS_LEN = len(test_tokens)
						linemeta = make([]LineMetaData, TEST_TOKENS_LEN)
						generate_line_metadata(font, &linemeta, &test_tokens)

						prev_qsize := qsize
						qsize = int(math.RoundToEven(float64(WIN_H)/float64(font.Height()))) + 1
						//println("start:", START_ELEMENT, "next:", NEXT_ELEMENT, "qsize-prev_qsize:", qsize-prev_qsize)
						NEXT_ELEMENT += (qsize - prev_qsize)
						println(qsize)

						textbox.data = nil
						textbox.data_rects = nil
						textbox.metadata = nil
						textbox.fmt.Free()
						textbox = TextBox{
							data:       make([]*sdl.Texture, qsize),
							texture_w:  0,
							texture_h:  0,
							data_rects: make([]sdl.Rect, qsize),
							metadata:   make([]*LineMetaData, qsize),
							fmt:        nil,
						}

						for i := 0; i < len(textbox.data); i++ {
							textbox.metadata[i] = &linemeta[START_ELEMENT+i]
						}

						textbox.CreateEmpty(renderer, font, sdl.Color{R: 0, G: 0, B: 0, A: 255})
						textbox.Update(renderer, font, test_tokens[START_ELEMENT:NEXT_ELEMENT], sdl.Color{R: 0, G: 0, B: 0, A: 255})

						re = nil
						re = make([]sdl.Rect, qsize)
						rey = nil
						rey = genY(font, qsize)
						for i := 0; i < qsize; i++ {
							re[i] = sdl.Rect{X: int32(X_OFFSET), Y: int32(rey[i]), W: int32(LINE_LENGTH), H: int32(font.Height())}
							for j := 0; j < len(textbox.metadata[i].word_rects); j++ {
								textbox.metadata[i].word_rects[j].Y = re[i].Y
							}
						}
					}
				}
				if t.Keysym.Sym == sdl.K_ESCAPE {
					running = false
				}
			default:
				continue
			}
		}
		renderer.SetDrawColor(255, 255, 255, 0)
		renderer.Clear()

		if easerout.animate {
			easerout.rect.X = int32(EaseOutQuad(float32(easerout.rect.X), float32(400), float32(400-easerout.rect.X), easerout.animation_time))
			easerout.animation_time += 2
			if easerout.rect.X >= 400-10 {
				easerout.animate = false
				easerout.animation_time = 0.0
			}
			draw_rect_without_border(renderer, &easerout.rect, &sdl.Color{R: 100, G: 200, B: 50, A: 100})
		}

		if easerin.animate {
			easerin.rect.X = int32(EaseInQuad(float32(easerin.rect.X), float32(400), float32(400-easerin.rect.X), easerin.animation_time))
			easerin.animation_time += 2
			if easerin.rect.X >= 400-10 {
				easerin.animate = false
				easerin.animation_time = 0.0
			}
			draw_rect_without_border(renderer, &easerin.rect, &sdl.Color{R: 200, G: 20, B: 50, A: 100})
		}

		if easerinout.animate {
			easerinout.rect.X = int32(EaseInOutQuad(float32(easerinout.rect.X), float32(400), float32(400-easerinout.rect.X), easerinout.animation_time))
			easerinout.animation_time += 2
			if easerinout.rect.X >= 400-10 {
				easerinout.animate = false
				easerinout.animation_time = 0.0
			}
			draw_rect_without_border(renderer, &easerinout.rect, &sdl.Color{R: 20, G: 20, B: 240, A: 100})
		}

		for i := 0; i < len(textbox.data); i++ {
			renderer.Copy(textbox.data[i], nil, &textbox.data_rects[i])
			for j := 0; j < len(textbox.metadata[i].mouse_over_word); j++ {
				if textbox.metadata[i].mouse_over_word[j] {
					engage_loop = true
				}
			}
		}

		// TODO: REMOVE THIS TEMP HACK
		if print_word {
			color_picker.show = true
		}
		if !engage_loop {
			color_picker.show = false
		}
		// TODO: REMOVE THIS TEMP HACK

		draw_rect_with_border_filled(renderer, &scrollbar.rect, &sdl.Color{R: 111, G: 111, B: 111, A: 90})

		// TODO: test what happens on &&?
		if scrollbar.drag || scrollbar.selected {
			draw_rect_with_border_filled(renderer, &scrollbar.rect, &sdl.Color{R: 111, G: 111, B: 111, A: 255})
		}

		if print_word && !engage_loop {
			print_word = false
		}

		if engage_loop && !cmd.show {
			for i := 0; i < len(textbox.data); i++ {
				for j := 0; j < len(textbox.metadata[i].mouse_over_word); j++ {
					if textbox.metadata[i].mouse_over_word[j] && textbox.metadata[i].words[j] != "\n" {
						if color_picker.show {
							// TOOLBAR
							color_picker.toolbar.bg_rect.X = textbox.metadata[i].word_rects[j].X
							color_picker.toolbar.bg_rect.Y = textbox.metadata[i].word_rects[j].Y + textbox.metadata[i].word_rects[j].H
							for r := 0; r < len(color_picker.toolbar.texture_rect); r++ {
								color_picker.toolbar.texture_rect[r].X = color_picker.toolbar.bg_rect.X + color_picker.toolbar.bg_rect.W - (color_picker.toolbar.texture_rect[r].W * int32((r + 1))) - (int32(r))
								color_picker.toolbar.texture_rect[r].Y = color_picker.toolbar.bg_rect.Y
							}
							// WINDOW
							color_picker.bg_rect.X = textbox.metadata[i].word_rects[j].X
							color_picker.bg_rect.Y = textbox.metadata[i].word_rects[j].Y + textbox.metadata[i].word_rects[j].H + color_picker.toolbar.bg_rect.H
							color_picker.texture_rect.X = textbox.metadata[i].word_rects[j].X
							color_picker.texture_rect.Y = textbox.metadata[i].word_rects[j].Y + textbox.metadata[i].word_rects[j].H + color_picker.toolbar.bg_rect.H
							acc = 0
							for r := 0; r < len(color_picker.rects); r++ {
								color_picker.rects[r].X = (textbox.metadata[i].word_rects[j].X) + acc
								color_picker.rects[r].Y = (textbox.metadata[i].word_rects[j].Y) + 10 + textbox.metadata[i].word_rects[j].H + color_picker.toolbar.bg_rect.H
								color_picker.rect_bgs[r].X = (textbox.metadata[i].word_rects[j].X) + acc
								color_picker.rect_bgs[r].Y = (textbox.metadata[i].word_rects[j].Y) + 10 + textbox.metadata[i].word_rects[j].H + color_picker.toolbar.bg_rect.H
								acc += MAGIC_PICKER_SKIP
							}

							color_picker.CenterRectAB() // TODO: REMOVE THIS TEMP HACK
							color_picker.CenterRects()  // TODO: REMOVE THIS TEMP HACK
						}
						draw_rect_without_border(renderer, &textbox.metadata[i].word_rects[j], &sdl.Color{R: 255, G: 100, B: 200, A: 100})
						if print_word && textbox.metadata[i].words[j] != "\n" {
							println(textbox.metadata[i].words[j])
							print_word = false
						}
					}
				}
			}
			engage_loop = false
		}

		if color_picker.show {
			draw_rect_with_border_filled(renderer, &color_picker.bg_rect, &color_picker.bg_color)
			renderer.Copy(color_picker.texture, nil, &color_picker.texture_rect)
			for i := 0; i < len(color_picker.rects); i++ {
				//draw_rect_without_border(renderer, &color_picker.rect_bgs[i], &sdl.Color{255, 255, 255, 255})
				draw_rect_without_border(renderer, &color_picker.rect_bgs[i], &color_picker.color[i])
				draw_rect_without_border(renderer, &color_picker.rects[i], &color_picker.color[i])
				renderer.Copy(color_picker.rect_textures[i], nil, &color_picker.rects[i])
			}
			draw_rect_with_border_filled(renderer, &color_picker.toolbar.bg_rect, &color_picker.toolbar.bg_color)
			for i := 0; i < len(color_picker.toolbar.texture); i++ {
				//draw_rect_without_border(renderer, &color_picker.toolbar.texture_rect[i], &color_picker.bg_color)
				renderer.Copy(color_picker.toolbar.texture[i], nil, &color_picker.toolbar.texture_rect[i])
			}
		}

		if move_text_down {
			move_text_down = false
			if NEXT_ELEMENT <= TEST_TOKENS_LEN {
				NEXT_ELEMENT += 1
				START_ELEMENT += 1
				textbox.Clear(renderer, font)
				textbox.Update(renderer, font, test_tokens[START_ELEMENT:NEXT_ELEMENT], sdl.Color{R: 0, G: 0, B: 0, A: 255})
				scrollbar.CalcPos(NEXT_ELEMENT, TEST_TOKENS_LEN)
				inc_dbg_str = true
				for i := 0; i < len(textbox.data); i++ {
					textbox.metadata[i] = &linemeta[START_ELEMENT+i]
					for j := 0; j < len(textbox.metadata[i].word_rects); j++ {
						textbox.metadata[i].word_rects[j].Y = re[i].Y
					}
				}
			}
		}

		if move_text_up {
			move_text_up = false
			if START_ELEMENT > 0 {
				NEXT_ELEMENT -= 1
				START_ELEMENT -= 1
				textbox.Clear(renderer, font)
				textbox.Update(renderer, font, test_tokens[START_ELEMENT:NEXT_ELEMENT], sdl.Color{R: 0, G: 0, B: 0, A: 255})
				scrollbar.CalcPos(NEXT_ELEMENT, TEST_TOKENS_LEN)
				inc_dbg_str = true
				for i := 0; i < len(textbox.data); i++ {
					textbox.metadata[i] = &linemeta[START_ELEMENT+i]
					for j := 0; j < len(textbox.metadata[i].word_rects); j++ {
						textbox.metadata[i].word_rects[j].Y = re[i].Y
					}
				}
			}
		}

		if page_down {
			page_down = false
			inc_dbg_str = true
			START_ELEMENT = NEXT_ELEMENT
			NEXT_ELEMENT += qsize
			if NEXT_ELEMENT >= TEST_TOKENS_LEN {
				START_ELEMENT = TEST_TOKENS_LEN - qsize
				NEXT_ELEMENT = TEST_TOKENS_LEN
			}
			textbox.Clear(renderer, font)
			textbox.Update(renderer, font, test_tokens[START_ELEMENT:NEXT_ELEMENT], sdl.Color{R: 0, G: 0, B: 0, A: 255})
			for i := 0; i < len(textbox.data); i++ {
				textbox.metadata[i] = &linemeta[START_ELEMENT+i]
				for j := 0; j < len(textbox.metadata[i].word_rects); j++ {
					textbox.metadata[i].word_rects[j].Y = re[i].Y
				}
			}
		}

		if page_up {
			page_up = false
			inc_dbg_str = true
			START_ELEMENT = NEXT_ELEMENT - (qsize * 2)
			NEXT_ELEMENT -= qsize
			if START_ELEMENT < 0 {
				START_ELEMENT = 0
				NEXT_ELEMENT = qsize
			}
			textbox.Clear(renderer, font)
			textbox.Update(renderer, font, test_tokens[START_ELEMENT:NEXT_ELEMENT], sdl.Color{R: 0, G: 0, B: 0, A: 255})
			for i := 0; i < len(textbox.data); i++ {
				textbox.metadata[i] = &linemeta[START_ELEMENT+i]
				for j := 0; j < len(textbox.metadata[i].word_rects); j++ {
					textbox.metadata[i].word_rects[j].Y = re[i].Y
				}
			}
		}

		if wrap_line {
			for i := 0; i < len(textbox.data); i++ {
				draw_rect_without_border(renderer, &textbox.data_rects[i], &sdl.Color{R: 100, G: 255, B: 255, A: 100})
			}
		}

		if cmd.show {
			for i := 0; i < len(textbox.metadata); i++ {
				draw_rect_with_border(renderer, &textbox.data_rects[i], &sdl.Color{R: 200, G: 100, B: 0, A: 200})
			}

			draw_rect_with_border_filled(renderer, &cmd.bg_rect, &sdl.Color{R: 255, G: 10, B: 100, A: cmd.alpha_value + 40})
			draw_rect_with_border(renderer, &cmd.ttf_rect, &sdl.Color{R: 255, G: 255, B: 255, A: 0})

			renderer.Copy(cmd.ttf_texture, nil, &cmd.ttf_rect)

			draw_rect_with_border_filled(renderer, &cmd.cursor_rect, &sdl.Color{R: 0, G: 0, B: 0, A: cmd.alpha_value})

			draw_rect_without_border(renderer, &gfonts.bg_rect, &sdl.Color{R: 255, G: 255, B: 255, A: 255})

			for i := 0; i < len(gfonts.textures); i++ {
				renderer.Copy(gfonts.textures[i], nil, &gfonts.ttf_rects[i])
				if mouseover_word_texture_FONT[i] == true {
					draw_rect_without_border(renderer, &gfonts.highlight_rect[i], &sdl.Color{R: 0, G: 0, B: 0, A: 100})
					if print_word { // this is bad, we shouldn't mix vars for states in multiple places
						if int32(gfonts.current_font_w) >= gfonts.fonts[i].width && int32(gfonts.current_font_h) >= gfonts.fonts[i].height {
							font = reload_font(font, font_dir+gfonts.fonts[i].name, test_font_size)
							test_font_name = gfonts.fonts[i].name
							textbox.MakeNULL()
							textbox.CreateEmpty(renderer, font, sdl.Color{R: 0, G: 0, B: 0, A: 255})
							textbox.Update(renderer, font, test_tokens[START_ELEMENT:NEXT_ELEMENT], sdl.Color{R: 0, G: 0, B: 0, A: 255})

							ClearMetadata(&linemeta)
							generate_line_metadata(font, &linemeta, &test_tokens)

							for i := 0; i < len(textbox.data); i++ {
								textbox.metadata[i] = &linemeta[START_ELEMENT+i]
							}

							rey = nil
							rey = genY(font, qsize)
							for i := 0; i < qsize; i++ {
								re[i] = sdl.Rect{X: int32(X_OFFSET), Y: int32(rey[i]), W: int32(LINE_LENGTH), H: int32(font.Height())}
								for j := 0; j < len(textbox.metadata[i].word_rects); j++ {
									textbox.metadata[i].word_rects[j].Y = re[i].Y
								}
							}
						}
						print_word = false
					}
				}
			}

			if inc_dbg_str { // A DIRTY HACK
				inc_dbg_str = false
				dbg_str = make_console_text(NEXT_ELEMENT, TEST_TOKENS_LEN)
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
		// fmt.Println(time.Now().Second())
	}

	ticker.Stop()
	renderer.Destroy()
	window.Destroy()

	textbox.MakeNULL()
	textbox.fmt.Free()

	for i := 0; i < len(color_picker.rects); i++ {
		color_picker.rect_textures[i].Destroy()
	}
	for i := 0; i < len(color_picker.toolbar.texture); i++ {
		color_picker.toolbar.texture[i].Destroy()
	}
	color_picker.texture.Destroy()
	color_picker.font.Close()

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

func generate_line_metadata(font *ttf.Font, dest *[]LineMetaData, tokens *[]string) {
	x, y, _ := font.SizeUTF8(" ")
	cap_x, cap_y, _ := font.SizeUTF8("A")
	low_x, low_y, _ := font.SizeUTF8("a")
	println("space:", x, y)
	println("cap:", cap_x, cap_y)
	println("low:", low_x, low_y)
	println("font face is fixed width:", font.FaceIsFixedWidth())
	println("----------------")
	for index := 0; index < len(*tokens); index++ {
		populate_line_metadata(&(*dest)[index], (*tokens)[index], x, y)
	}
}

func populate_line_metadata(line *LineMetaData, line_text string, x int, y int) {
	assert_if(len(line_text) == 0)

	text := strings.Split(line_text, " ")
	text_len := len(text)

	if text[text_len-1] == "" { // guard agains an empty ""
		text_len -= 1
	}

	line.word_rects = make([]sdl.Rect, text_len)
	line.mouse_over_word = make([]bool, text_len)
	line.words = make([]string, text_len)
	copy(line.words, text)

	move_x := X_OFFSET
	ix := 0
	for index := 0; index < text_len; index++ {
		ix = x * len(text[index])
		line.word_rects[index] = sdl.Rect{X: int32(move_x), Y: int32(-y), W: int32(ix), H: int32(y)}
		move_x += (ix + x)
	}
	text = nil
}

// TODO: refactor later
func ClearMetadata(line *[]LineMetaData) {
	for i := 0; i < len((*line)); i++ {
		(*line)[i].words = nil
		(*line)[i].word_rects = nil
		(*line)[i].mouse_over_word = nil
	}
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

func WrapLines(tokens []string, length int, font_w int) []string {
	// TODO: do we need current here? can't we just append to it instead of creating result?
	// TODO: both of determine_nwrap_lines and do_wrap_lines might be failing when input size is i < n && n > i
	result := make([]string, determine_nwrap_lines(tokens, length, font_w))
	for i, j := 0, 0; i < len(tokens); i += 1 {
		if len(tokens[i]) > 1 {
			current := do_wrap_lines(tokens[i], length, font_w)
			for k := range current {
				result[j] = current[k]
				j += 1
			}
			// should we do current = nil here?
		} else {
			result[j] = "\n"
			j += 1
		}
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
	}
	start := 0
	mmax := int(math.RoundToEven(float64(max_len/xsize))) - 1 // use math.Round instead?
	slice := str[start:mmax]
	end := mmax
	slice_len := 0
	for end < len(str) {
		slice_len = len(slice)
		if !is_space(slice[slice_len-1]) {
			for !is_space(slice[slice_len-1]) {
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
	// set slices to nil?
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
				if !is_space(slice[slice_len-1]) {
					for !is_space(slice[slice_len-1]) {
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
	// set slices to nil?
	return result
}

func assert_if(cond bool) {
	if cond {
		panic("assertion failed")
	}
}

// pass byte instead of string here in the future
func is_alpha(schr string) bool {
	return (schr >= "A") && (schr <= "z")
}

func is_space(s byte) bool {
	return s == byte(' ')
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
		if !is_space((*s)[index]) {
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

func lerp(a, b, t float32) float32 {
	if t > 1 || t < 0 {
		return 0.0
	}
	return (1-t)*a + t*b
}

func EaseInQuad(b, d, c, t float32) float32 {
	return c*(t/d)*(t/d) + b
}

func EaseOutQuad(b, d, c, t float32) float32 {
	return -c*(t/d)*((t/d)-2) + b
}

func EaseInOutQuad(b, d, c, t float32) float32 {
	if ((t / d) / 2) < 1 {
		return c/2*(t/d)*(t/d) + b
	}
	return -c/2*((t/d)*((t/d)-2)-1) + b
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
	CURRENT := "Inconsolata-Regular.ttf"
	//CURRENT := "DejaVuSansMono.ttf"
	for index, element := range ttf_font_list {
		if CURRENT == element {
			font.current_font = load_font(font_dir+element, TTF_FONT_SIZE)
			w, h, _ := font.current_font.SizeUTF8(" ")
			skp := font.current_font.LineSkip()
			font.current_font_w = w
			font.current_font_h = h
			font.current_font_skip = skp
			font.current_name = element
		}
		font.fonts[index].data = load_font(font_dir+element, TTF_FONT_SIZE_FOR_FONT_LIST)
		font.fonts[index].name = element
	}
}

func generate_rects_for_fonts(renderer *sdl.Renderer, font *FontSelector) {
	adder_y := 0
	for index, element := range font.fonts {
		gx, gy, _ := (*font).fonts[index].data.SizeUTF8(" ")
		font.fonts[index].size = gx * len(element.name)
		font.fonts[index].width = int32(gx)
		font.fonts[index].height = int32(gy)

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

func (fs *FontSelector) get_font(want string) *ttf.Font {
	for index := range fs.fonts {
		if fs.fonts[index].name == want {
			return fs.fonts[index].data
		}
	}
	return nil
}

func genY(font *ttf.Font, size int) []int {
	result := make([]int, size)

	for i := 0; i < size; i++ {
		result[i] = i * font.LineSkip()
	}
	return result
}

func (sc *Scrollbar) CalcPos(current int, total int) {
	sc.rect.Y = int32(float64(current)/float64(total)*float64(WIN_H)) - sc.rect.H
	if sc.rect.Y < 0 {
		sc.rect.Y = 0
	}
}

func (sc *Scrollbar) CalcPosDuringAction(current int, total int) {
	println(int((float64(current+int(sc.rect.H)) / float64(WIN_H)) * float64(total)))
}

func (tbox *TextBox) CreateEmpty(renderer *sdl.Renderer, font *ttf.Font, color sdl.Color) {
	surface, _ := font.RenderUTF8Blended(" ", color)
	if tbox.fmt == nil {
		tbox.fmt, _ = sdl.AllocFormat(sdl.PIXELFORMAT_RGBA8888)
	}
	converted, _ := surface.Convert(tbox.fmt, 0)

	var err error
	for i := 0; i < len(tbox.data); i++ {
		tbox.data[i], err = renderer.CreateTexture(sdl.PIXELFORMAT_RGBA8888, sdl.TEXTUREACCESS_STREAMING, int32(LINE_LENGTH), surface.H)
		if err != nil {
			fmt.Println(err)
		}
		err = tbox.data[i].Update(&sdl.Rect{X: 0, Y: 0, W: surface.W, H: surface.H}, converted.Pixels(), int(converted.Pitch))
		if err != nil {
			fmt.Println(err)
		}
		tbox.data[i].SetBlendMode(sdl.BLENDMODE_BLEND)
	}

	_, _, qw, qh, _ := tbox.data[0].Query()
	tbox.texture_w = qw
	tbox.texture_h = qh
	accy := int32(0)
	skip := int32(font.LineSkip())
	for i := 0; i < len(tbox.data); i++ {
		tbox.data_rects[i] = sdl.Rect{X: int32(X_OFFSET), Y: accy, W: qw, H: qh}
		accy += skip
	}
	surface.Free()
	converted.Free()
}

func (tbox *TextBox) Update(renderer *sdl.Renderer, font *ttf.Font, text []string, color sdl.Color) {
	var err error
	for i := 0; i < len(tbox.data); i++ {
		if text[i] != "\n" {
			surface, _ := font.RenderUTF8Blended(text[i], color)
			converted, _ := surface.Convert(tbox.fmt, 0)
			if surface.W <= int32(LINE_LENGTH) {
				// make sure that texture H >= surface.H
				err = tbox.data[i].Update(&sdl.Rect{X: 0, Y: 0, W: surface.W, H: tbox.texture_h}, converted.Pixels(), int(converted.Pitch))
				if err != nil {
					fmt.Println(err)
				}
				// TODO: check if we are wes till using this else clause?
			} else {
				err = tbox.data[i].Update(&sdl.Rect{X: 0, Y: 0, W: int32(LINE_LENGTH), H: surface.H}, converted.Pixels(), int(converted.Pitch))
				if err != nil {
					fmt.Println(err)
				}
			}
			surface.Free()
			converted.Free()
		}
	}
}

func (tbox *TextBox) Clear(renderer *sdl.Renderer, font *ttf.Font) {
	surface, _ := font.RenderUTF8Blended(" ", sdl.Color{R: 0, G: 0, B: 0, A: 0})
	converted, _ := surface.Convert(tbox.fmt, 0)
	for i := 0; i < len(tbox.data); i++ {
		bytes, _, _ := tbox.data[i].Lock(nil)
		copy(bytes, converted.Pixels())
		tbox.data[i].Unlock()
	}
	surface.Free()
	converted.Free()
}

func (tbox *TextBox) MakeNULL() {
	for i := 0; i < len(tbox.data); i++ {
		tbox.data[i].Destroy()
		tbox.data[i] = nil
	}
}

func (CP *ColorPicker) CenterRectAB() {
	for i := 0; i < len(CP.rects); i++ {
		CP.rects[i].X = CP.rects[i].X + (CP.rect_bgs[i].W / 2) - (CP.rects[i].W / 2)
		CP.rects[i].Y = CP.rects[i].Y + (CP.rect_bgs[i].H / 2) - (CP.rects[i].H / 2)
	}
}

func (CP *ColorPicker) CenterRects() {
	for i := 0; i < len(CP.rects); i++ {
		CP.rects[i].X = (CP.rects[i].X + (CP.bg_rect.W / 2)) - CP.rects[i].W*int32(len(CP.rects)+1)
		CP.rects[i].Y = (CP.rects[i].Y + (CP.bg_rect.H / 2)) - (CP.rects[i].H + (10 / 2)) // TODO: remove magic numbers
		CP.rect_bgs[i].X = (CP.rects[i].X - (CP.rects[i].W / 2)) - 1
		CP.rect_bgs[i].Y = CP.rects[i].Y
	}
}
