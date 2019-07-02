// +build windows, 386

package main

import (
	"flag"
	"fmt"
	"github.com/veandco/go-sdl2/img"
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
	"unicode"
	"unicode/utf16"
	"unicode/utf8"
)

// TODO
// Added a custom GlyphIsProvided to C:\Users\Edgaras\go\src\github.com\veandco\go-sdl2\ttf
// Should we submit that to the actual repo?
// Q: what happens when we pass in a value bigger than uint16?

// GENERAL

// [ ] what if instead of splitting strings all over our code, we load a string once and then just have pointers or position int32's?
//     when we reload fonts we inevitably have to reload text, which is not great (it also creates a lot of garbage).

// [ ] https://4gophers.ru/articles/smid-optimizaciya-v-go/
// [ ] https://habr.com/ru/company/badoo/blog/301990/
// [ ] http://m0sth8.github.io/runtime-1/#1
// [ ] optimizing go: https://www.youtube.com/watch?v=0i1nO9gwACY
// [ ] optimizing go binaries: https://www.youtube.com/watch?v=HpriPuIfrGE

// [ ] https://pavelfatin.com/scrolling-with-pleasure/
// [ ] https://github.com/dlion/modularLocalization
// [ ] https://arslan.io/2017/09/14/the-ultimate-guide-to-writing-a-go-tool/
// [ ] https://dave.cheney.net/high-performance-go-workshop/dotgo-paris.html
// [ ] https://www.youtube.com/user/dotconferences/videos
// [ ] add "runtime" for if runtime.GOOS == "windows" { println("blah") } else { println("blah") }
// [ ] use sdl.GetPlatform() || [runtime.GOOS == ""] || [foo_unix.go; foo_windows.go style]
// [ ] https://github.com/golang-standarts/project-layout
// [ ] instead of having images saved in a application folder, maybe we could generate img and then just load it up into a texture?
//     - use fogleman/gg or golang/image for that
// [ ] use C:\Windows\fonts for fonts?
// [ ] I'm sure that the app needs to have a modal way of execution, otherwise it's a nightmare to maintain.
// [ ] create a telegram bot for this app?
// [ ] use telegram for saving messages/audio and stuff?
// [ ] maybe try using github.com/golang/freetype/truetype package instead of sdl2 ttf one!
// [ ] https://stackoverflow.com/questions/29105540/aligning-text-in-golang-with-truetype
// [ ] checkout github.com/fatih/structs
// [ ] use asciinema.org for inspiration!
// [ ] use https://godoc.org/github.com/fsnotify/fsnotify for checking if our settings file has been changed?
// [ ] separate updating and rendering?
// [ ] maybe it would be possible to use unicode symbols like squares/triangles to indicate clickable objects?
// [ ] predefined colors in a .settings file?
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
// [ ] add proper error handling
// [ ] add logs???
// [ ] try proper font resizing -> resize the rect first and then reload ? or it's just enough to resize the rect by using font query?

// SDL RELATED
// [ ] !batch optimize Cgo calls
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
// [ ] grapical popup error messages like: error => your command is too long, etc...

// AUDIO
// [ ] loading and playing audio files
// [ ] recording audio?
// [ ] needs to support tags/breakpoints for situations where you can't hear clearly or don't understand

// TESTING
// [ ] automated visual tests
// [ ] create automated tests to scroll through the page from top to bottom checking if we ever fail to allocate/deallocate *Line
// [ ] add a way to submit github tickets within the app for alpha/beta testing?

// GO RELATED
// [ ] move to a 64-bit version of golang and sdl2 (needed for DELVE debugger)
// [ ] test struct padding?
// [ ] list.go should we set data to nil everytime?
// [ ] get rid of int (because on 64-bit systems it would become 64 bit and waste memory) or not???? maybe use int16 in some cases

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

// template
// sdl.Color{R:, G:, B:, A: }
// http://www.flatuicolorpicker.com/

var (
	COLOR_WHITE            = sdl.Color{R: 255, G: 255, B: 255, A: 255}
	COLOR_BLACK            = sdl.Color{R: 0, G: 0, B: 0, A: 255}
	COLOR_RED              = sdl.Color{R: 255, G: 0, B: 0, A: 255}
	COLOR_GREEN            = sdl.Color{R: 0, G: 255, B: 0, A: 255}
	COLOR_GREEN_MADANG     = sdl.Color{R: 200, G: 247, B: 197, A: 255}
	COLOR_BLUE             = sdl.Color{R: 0, G: 0, B: 255, A: 255}
	COLOR_WISTERIA         = sdl.Color{R: 155, G: 89, B: 182, A: 255}
	COLOR_WISTFUL          = sdl.Color{R: 174, G: 168, B: 211, A: 255}
	COLOR_LIGHT_GREEN      = sdl.Color{R: 123, G: 239, B: 178, A: 255}
	COLOR_IRON             = sdl.Color{R: 218, G: 223, B: 225, A: 255}
	COLOR_SAN_MARINER      = sdl.Color{R: 44, G: 130, B: 201, A: 255}
	COLOR_ELECTRIC_PURPLE  = sdl.Color{R: 165, G: 55, B: 253, A: 255}
	COLOR_PICKLED_BLUEWOOD = sdl.Color{R: 52, G: 73, B: 94, A: 255}
	COLOR_SUPERNOVA        = sdl.Color{R: 255, G: 203, B: 5, A: 255}
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

type GlobalMousePosition struct {
	X int32
	Y int32
}

//var globmousepos = GlobalMousePosition{0, 0}

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
	updated       bool
	font          *ttf.Font
	texture       *sdl.Texture
	texture_rect  sdl.Rect
	color         [CPN]sdl.Color
	rects         [CPN]sdl.Rect
	rect_textures [CPN]*sdl.Texture
	rect_bgs      [CPN]sdl.Rect
	toolbar       Toolbar
}

type MultiLine struct {
	texture  *sdl.Texture
	bg_rect  sdl.Rect
	fmt      *sdl.PixelFormat
	lineskip int32
}

type MenuWithButtons struct {
	rect    sdl.Rect
	buttons []sdl.Rect
}

type Sidebar struct {
	rect         sdl.Rect
	bg_rect      sdl.Rect
	buttons      []sdl.Rect
	highlight    bool
	buttonindex  int
	font         *ttf.Font
	font_rect    []sdl.Rect
	font_texture []*sdl.Texture
	text         []string
	callbacks    map[string]func()
}

const (
	CURSOR_TYPE_ARROW = iota
	CURSOR_TYPE_HAND
	CURSOR_TYPE_SIZEWE
)

const (
	GUI_ID_NONE = iota
	GUI_ID_CLRSL
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

	// TODO: investigate how to create software that could respond/work with available cores
	//       what happens when only one core is available, as opposed to multiple cores?

	if err := sdl.Init(sdl.INIT_TIMER | sdl.INIT_VIDEO | sdl.INIT_AUDIO); err != nil {
		panic(err)
	}

	if err := ttf.Init(); err != nil {
		panic(err)
	}

	if img.Init(img.INIT_PNG) == 0 {
		panic("img.Init failed!")
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

	img_surf, err := img.Load("./img/cube2.png")
	if err != nil {
		fmt.Errorf("Something went wrong with img.Load(): %v", err)
	}
	//key, err := img_surf.GetColorKey()
	//println("COLOR KEY:", key)
	img_surf.SetColorKey(true, 0x0)
	img_tx, _ := renderer.CreateTextureFromSurface(img_surf)

	img_surf.Free()
	defer img_tx.Destroy()
	//img_tx.SetBlendMode(sdl.BLENDMODE_BLEND)
	img_tx_rect := sdl.Rect{int32(LINE_LENGTH) - 100, 0, 40, 40}

	db := DBOpen()
	defer db.Close()

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

	//filename := "rus_bal_hiwnikov.txt"
	//filename := "Russian.txt"
	//filename := "French.txt"
	//filename := "Mandarin.txt" // //TODO: we are crashing!
	//filename := "Hanyu.txt"
	// TODO: proper Wrapping for non ASCII texts!
	filename := "HP01.txt"
	//filename := "hobbit_rus.txt" //TODO: we are crashing!
	// TODO: proper Wrapping for non ASCII texts!
	font_dir := "./fonts/"
	text_dir := "./text/"

	line_tokens := strings.Split(string(get_filedata(text_dir, filename)), "\r\n") // "\r\n" instead of "\n"

	ticker := time.NewTicker(time.Second / 60)

	ttf_font_list := get_filenames(font_dir, []string{"ttf", "otf"})
	txt_list := get_filenames(text_dir, []string{".txt"})
	fmt.Println(txt_list)

	var gfonts FontSelector
	allocate_font_space(&gfonts, len(ttf_font_list))
	generate_fonts(&gfonts, ttf_font_list, font_dir)

	font := gfonts.current_font

	generate_rects_for_fonts(renderer, &gfonts)

	//FontHasGlyphsFromRangeTable(font, unicode.Latin)

	test_tokens := WrapLines(line_tokens, LINE_LENGTH, gfonts.current_font_w)

	TEST_TOKENS_LEN := len(test_tokens)

	linemeta := make([]LineMetaData, TEST_TOKENS_LEN)
	generate_line_metadata(font, &linemeta, &test_tokens)

	cmd := NewCmdConsole(renderer)

	dbg_str := make_console_text(0, TEST_TOKENS_LEN)
	dbg_rect := sdl.Rect{X: 0, Y: WIN_H - (cmd.bg_rect.H * 2), W: int32(gfonts.current_font_w * len(dbg_str)), H: int32(gfonts.current_font_h)}
	dbg_ttf := make_ttf_texture(renderer, gfonts.current_font, dbg_str, &sdl.Color{R: 0, G: 0, B: 0, A: 255})

	sdl.SetHint(sdl.HINT_FRAMEBUFFER_ACCELERATION, "1")
	sdl.SetHint(sdl.HINT_RENDER_SCALE_QUALITY, "1")

	renderer.SetDrawBlendMode(sdl.BLENDMODE_BLEND)

	known_word_data := GetUniqueWords(line_tokens)

	// DB stuff
	if err = DBInit(db, known_word_data); err != nil {
		fmt.Errorf("Something went wrong %v", err)
	}

	if err = DBInsert(db, "hobbit"); err != nil {
		fmt.Errorf("Something went wrong %v", err)
	}
	found, _ := DBView(db, "hobbit")
	println("the word 'hobbit' was found: ", found)
	// DB stuff

	running := true
	print_word := false
	engage_loop := false
	inc_dbg_str := true

	current_GuiID := GUI_ID_NONE

	mouseover_word_texture_FONT := make([]bool, len(ttf_font_list))

	wrap_line := false

	move_text_up := false
	move_text_down := false
	page_up := false
	page_down := false

	wrapline := DebugWrapLine{int32(LINE_LENGTH), 0, int32(LINE_LENGTH), WIN_H}

	// TODO: this ain't working properly oon zoom out's
	qsize := int(math.RoundToEven(float64(WIN_H)/float64(font.Height()))) + 1

	println("qsize :: ", qsize, len(linemeta))
	// TODO: This is a temporary hack!
	// We need it in order not to break/panic every time we have less lines than qsize.
	// This is not a solution, however. The real solution lies in Line Update function, where we should not lines
	// with no data in them. Or something like that. It's been a while since I've looked at this codebase.
	if len(linemeta) < qsize {
		qsize = len(linemeta)
	}

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

	if len(textbox.metadata) > len(linemeta) {
		for i := 0; i < len(linemeta); i++ {
			textbox.metadata[i] = &linemeta[i]
		}
	} else {
		for i := 0; i < len(textbox.data); i++ {
			textbox.metadata[i] = &linemeta[i]
		}
	}

	textbox.CreateEmpty(renderer, font, sdl.Color{R: 0, G: 0, B: 0, A: 255})
	//textbox.Update(renderer, font, test_tokens[0:qsize], sdl.Color{R: 0, G: 0, B: 0, A: 255})
	textbox.Update(renderer, font, test_tokens[0:textbox.MetadataSize()], sdl.Color{R: 0, G: 0, B: 0, A: 255})

	re := make([]sdl.Rect, qsize)
	rey := genY(font, qsize)
	//for i := 0; i < qsize; i++ {
	for i := 0; i < textbox.MetadataSize(); i++ {
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

	easerin_reverse := struct {
		rect           sdl.Rect
		animate        bool
		animation_time float32
	}{sdl.Rect{410, 150, 100, 100}, true, 0.0}

	easerinout := struct {
		rect           sdl.Rect
		animate        bool
		animation_time float32
	}{sdl.Rect{0, 250, 100, 100}, true, 0.0}

	smooth := struct {
		rect           sdl.Rect
		animate        bool
		animation_time float32
		new_max_dest   int32
		reverse        bool
		skip           int32
	}{sdl.Rect{int32(X_OFFSET), 0, int32(LINE_LENGTH), 15}, false, 0.0, 0, false, int32(gfonts.current_font.LineSkip())}

	foobar_animation := FoobarEaserOut(renderer, sdl.Rect{0, 350, 100, 100}, EaseInQuad)

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

	// test shit
	var multiline_texture MultiLine
	multiline_texture.New(renderer, color_picker.font)
	defer multiline_texture.texture.Destroy()

	multiline_texture.Write(color_picker.font, "foobar", COLOR_BLACK, 0, 0)
	multiline_texture.Write(color_picker.font, "cooonoobar", COLOR_BLACK, 0, multiline_texture.lineskip)
	MLSkip := multiline_texture.lineskip
	rm_px := int32(0)
	multiline_texture.ClearAndWrite(
		renderer,
		color_picker.font,
		test_tokens[0:24],
		//[]string{"the road goes ever ever one", "under cloud and under start", "yet feet that wondering have gone"},
		MLSkip,
		rm_px,
		//[]int32{0, 1*MLSkip - rm_px, 2*MLSkip - rm_px},
	)
	// test shit

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

	menuwbtn := MenuWithButtons{
		rect: sdl.Rect{0, 50, int32(LINE_LENGTH), 20},
	}

	button_a := sdl.Rect{0, 50, 20, 20}
	button_b := sdl.Rect{22, 50, 20, 20}
	button_c := sdl.Rect{44, 50, 20, 20}

	menuwbtn.AddButtons(button_a, button_b, button_c)

	sidebar := Sidebar{
		bg_rect: sdl.Rect{0, 0, 150, WIN_H},
		rect:    sdl.Rect{0, 0, 150, WIN_H},
		font:    load_font(font_dir+"Inconsolata-Regular.ttf", 14),
		text: []string{
			"Open File",
			"Load Font",
			"Debug Menu",
			"Properties",
			"...More",
		},
		callbacks: make(map[string]func()),
	}
	sidebar.font_texture = make([]*sdl.Texture, len(sidebar.text))
	sidebar.font_rect = make([]sdl.Rect, len(sidebar.text))

	for index := range sidebar.font_texture {
		sidebar.font_texture[index] = make_ttf_texture(
			renderer,
			sidebar.font,
			sidebar.text[index],
			&COLOR_WHITE,
		)
	}

	sidebar.AddButtons(5, 20)

	for index := range sidebar.font_rect {
		_, _, tw, th, _ := sidebar.font_texture[index].Query()
		sidebar.font_rect[index].X = sidebar.buttons[index].X // NOTE: no need to do this before CenterTextRectX
		sidebar.font_rect[index].Y = sidebar.buttons[index].Y
		sidebar.font_rect[index].W = tw
		sidebar.font_rect[index].H = th
	}
	sidebar.CenterTextRectX()

	for index := range sidebar.text {
		txt := sidebar.text[index]
		sidebar.callbacks[txt] = func() {
			fmt.Println(txt)
		}
	}

	rendererInfo, err := renderer.GetInfo()
	if err != nil {
		panic(err)
	}

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
				//globmousepos.X = t.X
				//globmousepos.Y = t.Y
				//println("[DEBUG] ", globmousepos.X, globmousepos.Y)

				for i := 0; i < textbox.MetadataSize(); i++ {
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

				// NOTE: We can skip sidebar.highlight = false here
				//       in this block, but it's totally fine (I think).
				//       Also, I think that MouseOver should call Update
				//       behind the scenes, so that we don't have to have
				//       all of this spaghetty crap.
				if buttonIndex, ok := sidebar.MouseOver(t); ok {
					if sidebar.buttonindex != buttonIndex {
						sidebar.buttonindex = buttonIndex
						sidebar.highlight = true
						txt := sidebar.text[sidebar.buttonindex]
						sidebar.callbacks[txt]()
					}
				} else {
					if sidebar.highlight {
						sidebar.buttonindex = -1 // reset buttonindex
						sidebar.highlight = false
					}
				}
			case *sdl.MouseWheelEvent:
				println("(debug) mouse motion event in Y: ", t.Y)
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
					cmd.WriteChar(renderer, t.Text[0])
				}
			case *sdl.KeyboardEvent:
				if cmd.show {
					if t.Keysym.Sym == sdl.K_BACKSPACE {
						if t.Repeat > 0 {
							cmd.Reset(renderer)
						}
					}
					switch t.Type {
					case sdl.KEYDOWN:
					case sdl.KEYUP:
						if t.Keysym.Mod == sdl.KMOD_LCTRL && t.Keysym.Sym == sdl.K_v {
							if sdl.HasClipboardText() {
								str, _ := sdl.GetClipboardText()
								cmd.WriteString(renderer, str)
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
						cmd.Reset(renderer)
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
						//TODO: we should only MAKENULL textbox when prev qsize has changed
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
						//TODO: we should only MAKENULL textbox when prev qsize has changed
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

		// TODO: What if instead of rendering everything to the screen 60 fps, we would only redraw
		//       needed elements through first saving to an external backbuffer texture?
		//       Should this be achieved by:
		//       - set the backbuffer texture as the current render target
		//       - render stuff to the texture
		//       - switch the current render target back to the default one
		//       - render the default render target
		//       We need to check if we can have a texture render target through RendererInfo

		//menuwbtn.Draw(renderer)
		if easerout.animate {
			easerout.rect.X = int32(EaseOutQuad(float32(easerout.rect.X), float32(400), float32(400-easerout.rect.X), easerout.animation_time))
			easerout.animation_time += 2
			if easerout.rect.X >= 400-10 {
				easerout.animate = false
				easerout.animation_time = 0.0
			}
			draw_rect_without_border(renderer, &easerout.rect, &sdl.Color{R: 100, G: 200, B: 50, A: 100})
		}

		if foobar_animation != nil {
			if !foobar_animation() {
				foobar_animation = nil
				println("END of animation")
			}
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

		if easerin_reverse.animate {
			easerin_reverse.rect.X -= int32(EaseInQuad(float32(0), float32(400), float32(easerin_reverse.rect.X+10), easerin_reverse.animation_time))
			easerin_reverse.animation_time += 2
			if easerin_reverse.rect.X <= 0 {
				easerin_reverse.animate = false
				easerin_reverse.animation_time = 0.0
			}
			draw_rect_without_border(renderer, &easerin_reverse.rect, &sdl.Color{R: 200, G: 20, B: 50, A: 100})
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

		// TESTING
		if smooth.animate {
			if smooth.reverse == false {
				if smooth.new_max_dest <= 0 {
					//println("[debug] smooth.step < 0")
					smooth.new_max_dest = smooth.skip * 2
					//smooth.rect.Y = int32(EaseInQuad(float32(smooth.rect.Y),
					//    float32(smooth.new_max_dest), float32(smooth.new_max_dest), smooth.animation_time))
				}
				smooth.rect.Y = int32(EaseInQuad(float32(smooth.rect.Y),
					float32(smooth.new_max_dest), float32(smooth.new_max_dest), smooth.animation_time))
				//println("down ", smooth.rect.Y, smooth.new_max_dest)
				smooth.animation_time += 1
				if smooth.rect.Y >= smooth.new_max_dest {
					smooth.animate = false
					smooth.animation_time = 0.0
					smooth.rect.Y = smooth.new_max_dest // error correction, cuz sometimes it's wrong yo!
				}
			} else {
				if smooth.new_max_dest <= 0 { // another error correction
					smooth.rect.Y -= int32(EaseInQuad(float32(0),
						float32(smooth.skip), float32(smooth.skip), smooth.animation_time))
				} else {
					smooth.rect.Y -= int32(EaseInQuad(float32(0),
						float32(smooth.new_max_dest), float32(smooth.new_max_dest), smooth.animation_time))
				}
				println("up ", smooth.rect.Y, smooth.new_max_dest, smooth.skip)
				smooth.animation_time += 1
				if smooth.rect.Y <= smooth.new_max_dest {
					smooth.animate = false
					smooth.animation_time = 0.0
					smooth.rect.Y = smooth.new_max_dest // error correction, cuz sometimes it's wrong yo!
				}
			}

			// TODO: instead of doing it this way where we have test_tokens[0:24] at all times
			//       we should have start:end variables that control how much data we have on the screen
			//       just like we did in our main textbox window. Otherwise we have to keep the track of <= 0 numbers
			//       which would reach crash at some point. It's late at night and hot, so I might change my mind
			//       about this sometime later.

			// temp
			multiline_texture.ClearAndWrite(
				renderer,
				color_picker.font,
				test_tokens[0:24],
				//[]string{"the road goes ever ever one", "under cloud and under start", "yet feet that wondering have gone"},
				MLSkip,
				rm_px+smooth.rect.Y,
			)
			// temp
		}
		draw_rounded_rect_with_border_filled(renderer, &smooth.rect, &COLOR_WISTERIA)

		draw_rounded_rect_with_border_filled(renderer, &multiline_texture.bg_rect, &COLOR_IRON)
		renderer.Copy(multiline_texture.texture, nil, &multiline_texture.bg_rect)

		for i := 0; i < textbox.MetadataSize(); i++ {
			renderer.Copy(textbox.data[i], nil, &textbox.data_rects[i])
			for j := 0; j < len(textbox.metadata[i].mouse_over_word); j++ {
				if textbox.metadata[i].mouse_over_word[j] {
					engage_loop = true
				}
			}
		}

		draw_rect_with_border_filled(renderer, &scrollbar.rect, &sdl.Color{R: 111, G: 111, B: 111, A: 90})

		// TODO: test what happens on &&?
		if scrollbar.drag || scrollbar.selected {
			draw_rect_with_border_filled(renderer, &scrollbar.rect, &sdl.Color{R: 111, G: 111, B: 111, A: 255})
		}

		if print_word && !engage_loop {
			print_word = false
		}

		// TODO: this won't work on selecting color_picker elements
		if print_word {
			color_picker.show = !color_picker.show
			color_picker.updated = false // this should probably be: color_picker.redrawn
		}

		//draw_rect_with_border_filled(renderer, &img_tx_rect, &sdl.Color{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF})
		renderer.Copy(img_tx, nil, &img_tx_rect)

		if engage_loop && !cmd.show {
			for i := 0; i < textbox.MetadataSize(); i++ {
				for j := 0; j < len(textbox.metadata[i].mouse_over_word); j++ {
					if textbox.metadata[i].mouse_over_word[j] && textbox.metadata[i].words[j] != "\n" {
						if color_picker.show && color_picker.updated == false {
							current_rect := textbox.metadata[i].word_rects[j]
							color_picker.UpdateToolbarPos(current_rect)                   // TOOLBAR
							color_picker.UpdateWindowPos(current_rect, MAGIC_PICKER_SKIP) // WINDOW
							color_picker.CenterRectAB()                                   // TODO: REMOVE THIS TEMP HACK
							color_picker.CenterRects()                                    // TODO: REMOVE THIS TEMP HACK
						}
						color_picker.updated = true
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

		if color_picker.show && current_GuiID == GUI_ID_CLRSL {
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

			// TODO: use this to fade in and out text
			//draw_rect_without_border(renderer, &sdl.Rect{0, 0, WIN_W, WIN_W}, &sdl.Color{255, 255, 255, 100})
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
			// test
			smooth.animate = true
			smooth.reverse = false
			smooth.new_max_dest += smooth.skip * 2

			//rm_px += 1

			//multiline_texture.ClearAndWrite(
			//	renderer,
			//	color_picker.font,
			//	test_tokens[0:24],
			//	//[]string{"the road goes ever ever one", "under cloud and under start", "yet feet that wondering have gone"},
			//	MLSkip,
			//	rm_px,
			//	//[]int32{rm_px, 1*MLSkip + rm_px, 2*MLSkip + rm_px},
			//)
			// test
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

			smooth.animate = true
			smooth.reverse = true
			smooth.new_max_dest -= smooth.skip * 2
			// test
			//rm_px -= 1
			//multiline_texture.ClearAndWrite(
			//	renderer,
			//	color_picker.font,
			//	test_tokens[0:24],
			//	//[]string{"the road goes ever ever one", "under cloud and under start", "yet feet that wondering have gone"},
			//	MLSkip,
			//	//rm_px,
			//	rm_px,
			//	//[]int32{rm_px, 1*MLSkip + rm_px, 2*MLSkip + rm_px},
			//)
			// test
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
			draw_multiple_rects_with_border(renderer, textbox.data_rects, &sdl.Color{R: 200, G: 100, B: 0, A: 200})

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

		sidebar.Draw(renderer)

		renderer.Present()

		//NOTE: this is not for framerate independance
		//NOTE: it's probably also slower than calling SDL_Timer/SDL_Delay functions
		//NOTE: OR try using sdl2_gfx package functions like: FramerateDelay...
		<-ticker.C
		// fmt.Println(time.Now().Second())
	}

	println("[INFO] RENDERER TEXTURE MAX_W:", rendererInfo.MaxTextureWidth, "MAX_H:", rendererInfo.MaxTextureHeight)

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
	cmd.font.Close()

	dbg_ttf.Destroy()

	for index := range ttf_font_list {
		gfonts.fonts[index].data.Close()
		gfonts.current_font.Close()
		gfonts.fonts[index].data = nil
		gfonts.textures[index].Destroy()
	}

	for index := range sidebar.font_texture {
		if sidebar.font_texture[index] != nil {
			sidebar.font_texture[index].Destroy()
		}
	}

	if sidebar.font != nil {
		sidebar.font.Close()
	}

	font.Close()

	ttf.Quit()
	sdl.Quit()
	img.Quit()

	println("[INFO] NumCPU on this system: ", runtime.NumCPU())
	println("[INFO] NumCgoCall during this application run: ", runtime.NumCgoCall())

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
	cap_x, cap_y, _ := font.SizeUTF8("A") // BAD
	low_x, low_y, _ := font.SizeUTF8("a") // BAD
	println("space:", x, y)
	println("cap:", cap_x, cap_y)
	println("low:", low_x, low_y)
	println("font face is fixed width:", font.FaceIsFixedWidth())
	println("----------------")
	for index := 0; index < len(*tokens); index++ {
		populate_line_metadata(&(*dest)[index], (*tokens)[index], font, x, y)
	}
}

func populate_line_metadata(line *LineMetaData, line_text string, font *ttf.Font, x int, y int) {
	assert_if(len(line_text) == 0)

	text := strings.Split(line_text, " ")
	text_len := len(text)

	if text[text_len-1] == "" { // guard against an empty ""
		text_len -= 1
	}

	line.word_rects = make([]sdl.Rect, text_len)
	line.mouse_over_word = make([]bool, text_len)
	line.words = make([]string, text_len)
	copy(line.words, text)

	//f := func(r rune) bool { //   return r < 'A' || r > 'z' //}
	//f := func(r rune) bool { //	return r > 0x7F // 127 //}

	//switch ret := strings.IndexFunc(foo, bar) {
	//    case -1: //    // ASCII
	//    case >= 0: //    // NonASCII?
	//    default: //    // NonASCII?
	//}

	move_x := X_OFFSET
	ix := 0
	for index := 0; index < text_len; index++ {
		// we should probably get the position of the element here
		// if pos := strings.IndexFunc(text[index], f); pos != -1 { //-1 is none is found
		//str := text[index]
		//for len(str) > 0 {
		//	r, size := utf8.DecodeRuneInString(str)
		//	_ = r
		//	//font_has_glyph := font.GlyphIsProvided(uint16(r)) // 0 equals to NOT_FOUND
		//	//fmt.Printf("%c %v %d \n", r, size, font_has_glyph)
		//	str = str[size:]
		//}
		if strings.IndexFunc(text[index], func(r rune) bool { return r > 0x7f }) != -1 { //-1 is none is found
			//r, size := utf8.DecodeRune(text[index][pos]) //println(r, size)
			//fmt.Println("non-Ascii found: ", text[index])
			//fmt.Println(text[index], UTF8_CharCount(text[index]))
			ix = x * UTF8_CharCount(text[index])
			line.word_rects[index] = sdl.Rect{X: int32(move_x), Y: int32(-y), W: int32(ix), H: int32(y)}
			move_x += (ix + x)
		} else {
			//fmt.Println("This should be ASCII then", text[index])
			ix = x * len(text[index])
			line.word_rects[index] = sdl.Rect{X: int32(move_x), Y: int32(-y), W: int32(ix), H: int32(y)}
			move_x += (ix + x)
		}
		//ix = x * len(text[index])
		//line.word_rects[index] = sdl.Rect{X: int32(move_x), Y: int32(-y), W: int32(ix), H: int32(y)}
		//move_x += (ix + x)
	}
	text = nil
}

func UTF8_CharCount(text string) int {
	result := 0
	for len(text) > 0 {
		_, size := utf8.DecodeRuneInString(text)
		text = text[size:]
		result += 1
	}
	return result
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
	// TODO: both of NumWrappedLines and DoWrapLines might be failing when input size is i < n && n > i
	// TODO: handle UTF8
	result := make([]string, NumWrappedLines(tokens, length, font_w))
	for i, j := 0, 0; i < len(tokens); i += 1 {
		if len(tokens[i]) > 1 {
			current := DoWrapLines(tokens[i], length, font_w)
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

func DoWrapLines(str string, max_len int, xsize int) []string {
	assert_if(len(str) <= 1)

	result := make([]string, NumWrappedLines([]string{str}, max_len, xsize))

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
func NumWrappedLines(str []string, max_len int, xsize int) int32 {
	var result int32
	for index := 0; index < len(str); index++ {
		//if strings.IndexFunc(str[index], func(r rune) bool { return r > 0x7f }) != -1 { //-1 is none is found
		//    println("we found some unicode")
		//}
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

func draw_rounded_rect_with_border_filled(renderer *sdl.Renderer, rect *sdl.Rect, c *sdl.Color) {
	renderer.SetDrawColor((*c).R, (*c).G, (*c).B, (*c).A)
	renderer.FillRect(rect)
	renderer.DrawRect(rect)
	renderer.SetDrawColor(255, 255, 255, 255) // temporary
	renderer.DrawPoints([]sdl.Point{
		sdl.Point{rect.X, rect.Y},                           // top
		sdl.Point{rect.X, rect.Y + rect.H - 1},              // bottom
		sdl.Point{rect.X + rect.W - 1, rect.Y},              // top
		sdl.Point{rect.X + rect.W - 1, rect.Y + rect.H - 1}, // bottom
	})
}

func draw_multiple_rects_with_border(renderer *sdl.Renderer, rects []sdl.Rect, c *sdl.Color) {
	renderer.SetDrawColor((*c).R, (*c).G, (*c).B, (*c).A)
	renderer.DrawRects(rects)
}

func draw_multiple_rects_with_border_filled(renderer *sdl.Renderer, rects []sdl.Rect, c *sdl.Color) {
	renderer.SetDrawColor((*c).R, (*c).G, (*c).B, (*c).A)
	renderer.FillRects(rects)
	renderer.DrawRects(rects)
}

func draw_multiple_rects_without_border_filled(renderer *sdl.Renderer, rects []sdl.Rect, c *sdl.Color) {
	renderer.SetDrawColor((*c).R, (*c).G, (*c).B, (*c).A)
	renderer.FillRects(rects)
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
	//CURRENT := "Inconsolata-Regular.ttf"
	CURRENT := "AnonymousPro-Regular.ttf"
	//CURRENT := "DejaVuSansMono.ttf"
	//CURRENT := "Miroslav.ttf"
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
		result[i] = i * font.LineSkip() // NOTE(Edgar) no need to font.LineSkip() on every iteration
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

func (tbox *TextBox) MetadataSize() int {
	result := 0
	for i := 0; i < len(tbox.metadata); i++ {
		if tbox.metadata[i] == nil {
			break
		}
		result += 1
	}
	return result
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

// TODO: why do we pass renderer here?
func (tbox *TextBox) Update(renderer *sdl.Renderer, font *ttf.Font, text []string, color sdl.Color) {
	var err error
	for i := 0; i < tbox.MetadataSize(); i++ {
		if text[i] != "\n" {
			surface, _ := font.RenderUTF8Blended(text[i], color)
			converted, _ := surface.Convert(tbox.fmt, 0)
			if surface.W <= int32(LINE_LENGTH) { // TODO: make sure that texture H >= surface.H ??
				err = tbox.data[i].Update(&sdl.Rect{X: 0, Y: 0, W: surface.W, H: tbox.texture_h}, converted.Pixels(), int(converted.Pitch))
				if err != nil {
					fmt.Println(err)
				}
			} else { // TODO: check if we are wes till using this else clause?
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

// TODO: why do we pass renderer here?
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

func (CP *ColorPicker) UpdateToolbarPos(r sdl.Rect) {
	CP.toolbar.bg_rect.X = r.X
	CP.toolbar.bg_rect.Y = r.Y + r.H
	for i := 0; i < len(CP.toolbar.texture_rect); i++ {
		CP.toolbar.texture_rect[i].X = CP.toolbar.bg_rect.X + CP.toolbar.bg_rect.W - (CP.toolbar.texture_rect[i].W * int32((i + 1))) - (int32(i))
		CP.toolbar.texture_rect[i].Y = CP.toolbar.bg_rect.Y
	}
}

func (CP *ColorPicker) UpdateWindowPos(r sdl.Rect, skip int32) {
	CP.bg_rect.X = r.X
	CP.bg_rect.Y = r.Y + r.H + CP.toolbar.bg_rect.H
	CP.texture_rect.X = r.X
	CP.texture_rect.Y = r.Y + r.H + CP.toolbar.bg_rect.H
	acc := int32(0)
	for i := 0; i < len(CP.rects); i++ {
		CP.rects[i].X = (r.X) + acc
		CP.rects[i].Y = (r.Y) + 10 + r.H + CP.toolbar.bg_rect.H
		CP.rect_bgs[i].X = (r.X) + acc
		CP.rect_bgs[i].Y = (r.Y) + 10 + r.H + CP.toolbar.bg_rect.H
		acc += skip
	}
}

//TODO: add .R32
func FontHasGlyphsFromRangeTable(font *ttf.Font, rtable *unicode.RangeTable) {
	for current := 0; current < len(rtable.R16); current += 1 {
		delta := rtable.R16[current].Hi - rtable.R16[current].Lo

		if delta > 0 {
			//print(current) //print(" ")
			//print(delta) //print(" ")
			mk := make([]uint16, delta)

			i := 0
			for rng := rtable.R16[current].Lo; rng < rtable.R16[current].Hi; rng += rtable.R16[current].Stride {
				mk[i] = rng
				i++ // move this i++
			}

			r := utf16.Decode(mk)
			for _, x := range r {
				//if unicode.IsLetter(x) && font.GlyphIsProvided(uint16(x)) > 0 {
				if font.GlyphIsProvided(uint16(x)) > 0 {
					print(string(x))
				} else {
					println("not provided: ", string(x), x)
				}
			}
			println("")
			mk = nil
		}
	}
}

// TODO: refactor this later to take an int instead of sdl.Rect
//       that way i'll be able to use it for X, Y, W, H and R, G, B, A
func FoobarEaserOut(renderer *sdl.Renderer, r sdl.Rect, f func(b, d, c, t float32) float32) func() bool {
	easerout := struct {
		rect           sdl.Rect
		animate        bool
		animation_time float32
	}{r, true, 0.0}
	return func() bool {
		if easerout.animate {
			easerout.rect.X = int32(f(float32(easerout.rect.X), float32(400), float32(400-easerout.rect.X), easerout.animation_time))
			easerout.animation_time += 2
			if easerout.rect.X >= 400-10 {
				easerout.animate = false
				easerout.animation_time = 0.0
			}
			draw_rect_without_border(renderer, &easerout.rect, &sdl.Color{R: 100, G: 200, B: 50, A: 100})
			return true
		} else {
			return false
		}
	}
}

func (ML *MultiLine) New(renderer *sdl.Renderer, font *ttf.Font) {
	surface, _ := font.RenderUTF8Blended(" ", COLOR_BLACK)
	ML.fmt, _ = sdl.AllocFormat(sdl.PIXELFORMAT_RGBA8888)
	converted, _ := surface.Convert(ML.fmt, 0)
	ML.lineskip = int32(font.LineSkip())
	ML.texture, _ = renderer.CreateTexture(sdl.PIXELFORMAT_RGBA8888, sdl.TEXTUREACCESS_STREAMING, 300, 300)
	ML.texture.Update(&sdl.Rect{X: 0, Y: 0, W: surface.W, H: surface.H}, converted.Pixels(), int(converted.Pitch))
	ML.texture.SetBlendMode(sdl.BLENDMODE_BLEND)
	ML.bg_rect = sdl.Rect{int32(LINE_LENGTH), 0, 300, 300}
	surface.Free()
	converted.Free()
}

func (ML *MultiLine) Write(font *ttf.Font, text string, color sdl.Color, x, y int32) {
	surface, _ := font.RenderUTF8Blended(text, color)
	converted, _ := surface.Convert(ML.fmt, 0)

	ML.texture.Update(&sdl.Rect{X: x, Y: y, W: surface.W, H: surface.H}, converted.Pixels(), int(converted.Pitch))

	//if y < ML.bg_rect.H {
	//	ML.texture.Update(&sdl.Rect{X: x, Y: y, W: surface.W, H: surface.H}, converted.Pixels(), int(converted.Pitch))
	//} else if y >= ML.bg_rect.H { // test
	//	ML.texture.Update(&sdl.Rect{X: x, Y: surface.H, W: surface.W, H: surface.H}, converted.Pixels(), int(converted.Pitch))
	//}
	//if y > 0 {
	//    ML.texture.Update(&sdl.Rect{X: x, Y: y, W: surface.W, H: surface.H}, converted.Pixels(), int(converted.Pitch))
	//} else if y <= -10 { // test
	//    ML.texture.Update(&sdl.Rect{X: x, Y: -10, W: surface.W, H: surface.H}, converted.Pixels(), int(converted.Pitch))
	//}

	surface.Free()
	converted.Free()
}

func (ML *MultiLine) Clear(renderer *sdl.Renderer, font *ttf.Font) {
	surface, _ := font.RenderUTF8Blended(" ", sdl.Color{R: 0, G: 0, B: 0, A: 0})
	converted, _ := surface.Convert(ML.fmt, 0)

	// TODO: do we need to do converted pixels here??
	bytes, _, _ := ML.texture.Lock(nil)
	copy(bytes, converted.Pixels())
	ML.texture.Unlock()

	surface.Free()
	converted.Free()
}

func (ML *MultiLine) ClearAndWrite(renderer *sdl.Renderer, font *ttf.Font, text []string, lineskip, adder int32) {
	mult := int32(0)
	can_clear := true
	for i, t := range text {
		// NOTE(Edgar): Not sure why we are crashing here
		mult = int32(i)*lineskip + adder
		if mult <= ML.bg_rect.H-10 && mult > -10 { // TODO: should have a ML.surface_H/ML.surface_W
			if can_clear { // NOTE(Edgar): This is a hack that stops us from crashing
				ML.Clear(renderer, font)
				can_clear = false
			}
			ML.Write(font, t, COLOR_BLACK, 0, mult)
		}
	}
}

func (mbtn *MenuWithButtons) AddButtons(buttons ...sdl.Rect) {
	mbtn.buttons = make([]sdl.Rect, len(buttons))
	for i, button := range buttons {
		mbtn.buttons[i] = button
	}
}

func (mbtn *MenuWithButtons) Draw(renderer *sdl.Renderer) {
	draw_rounded_rect_with_border_filled(renderer, &mbtn.rect, &COLOR_IRON)
	draw_multiple_rects_with_border_filled(renderer, mbtn.buttons, &COLOR_LIGHT_GREEN)
}

func (sbar *Sidebar) Draw(renderer *sdl.Renderer) {
	draw_rect_without_border(renderer, &sbar.rect, &COLOR_PICKLED_BLUEWOOD)

	if len(sbar.buttons) > 0 {
		draw_multiple_rects_without_border_filled(renderer, sbar.buttons, &COLOR_WISTERIA)
	}

	draw_multiple_rects_with_border_filled(renderer, sbar.font_rect, &COLOR_WISTERIA)
	for index := range sbar.font_texture {
		renderer.Copy(sbar.font_texture[index], nil, &sbar.font_rect[index])
	}

	// NOTE: here we are drawing on top of draw_multiple_rects...
	if sbar.highlight {
		draw_rect_without_border(renderer, &sbar.buttons[sbar.buttonindex], &sdl.Color{100, 100, 155, 60})
	}

}

func (sbar *Sidebar) AddButtons(numbtn, height int32) {
	rectW := sbar.rect.W
	sbar.buttons = make([]sdl.Rect, numbtn)
	space := int32(3)
	for i := range sbar.buttons {
		if i == 0 {
			sbar.buttons[i] = sdl.Rect{X: 0, Y: height*int32(i) + space, W: rectW, H: height}
		} else {
			sbar.buttons[i] = sdl.Rect{X: 0, Y: height*int32(i) + space, W: rectW, H: height}
		}
		space += 3
	}
}

// returning int here is the position index
func (sbar *Sidebar) MouseOver(event *sdl.MouseMotionEvent) (int, bool) {
	var x, y, xw, xy bool
	for index := range sbar.buttons {
		x = event.X > sbar.buttons[index].X
		y = event.Y > sbar.buttons[index].Y
		xw = event.X < sbar.buttons[index].X+sbar.buttons[index].W
		xy = event.Y < sbar.buttons[index].Y+sbar.buttons[index].H

		if (x && y) && (xw && xy) {
			return index, true
		}
	}
	return 0, false
}

func (sbar *Sidebar) CenterTextRectX() {
	for index := range sbar.font_rect {
		sbar.font_rect[index].X = (sbar.bg_rect.W / 2) - (sbar.font_rect[index].W / 2)
	}
}
