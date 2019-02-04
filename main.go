package main

import (
    "os"
    "log"
    "fmt"
    "math"
    "time"
    "flag"
    "bytes"
    "strconv"
    "strings"
    "runtime"
    "io/ioutil"
    "math/rand"
    "runtime/pprof"
    "github.com/veandco/go-sdl2/sdl"
    "github.com/veandco/go-sdl2/ttf"
)

// TODO
// https://gist.github.com/tetsuok/3025333
// we have to turn off compiler optimizations in order to debug properly
// TODO  try to use: go tool vet 
// TODO: https://appliedgo.net/big-o/ 

// TODO: USE sdl.WINDOWEVENT_EXPOSED for proper redrawing

// TODO: add notification icon (please use WINDOWS docs for that, as SDL doesn't support it for now)
// https://stackoverflow.com/questions/41441807/minimize-window-to-system-tray
// https://gamedev.stackexchange.com/questions/136473/sdl2-taskbar-icon-notification-blinking-flashing-orange

const WIN_TITLE string = "GO_TEXT_APPLICATION"

const WIN_W int32 = 800
const WIN_H int32 = 600

const X_OFFSET int = 7
const TTF_FONT_SIZE int = 18
const TTF_FONT_SIZE_FOR_FONT_LIST int = 14
const TEXT_SCROLL_SPEED int32 = 14
const LINE_LENGTH int = 730

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to 'file'")
var memprofile = flag.String("memprofile", "", "write mem profile to 'file'")

// @GLOBAL MUT VARS 
var global_win_w int32
var global_win_h int32
var GLOBAL_WASTE_VAR int
var MAX_INDEX int = 40  // TODO: make MAX AND START INDEX scrollable
var START_INDEX int = 0 // TODO: make MAX AND START INDEX scrollable

type Font struct {
    size int
    name string
    data *ttf.Font
    skipline int32
    width, height int32
}

type Line struct {
    texture *sdl.Texture
    bg_rect sdl.Rect
    word_rects []sdl.Rect  //DELETE
}

type DebugWrapLine struct {
    x1, y1 int32
    x2, y2 int32
}

type CmdConsole struct {
    show bool
    move_left bool
    alpha_value uint8 //anim_alpha
    bg_rect sdl.Rect
    ttf_rect sdl.Rect
    cursor_rect sdl.Rect
    ttf_texture *sdl.Texture
    input_buffer bytes.Buffer
}

// should we have current_font *ttf.Font?
// struct FontInfo: W? H? SKIP? Flags?
type FontSelector struct {
    fonts []Font
    show bool
    move_up bool
    move_down bool
    current_font *ttf.Font
    current_font_w int
    current_font_h int
    current_font_skip int
    alpha_value uint8
    bg_rect sdl.Rect
    ttf_rects []sdl.Rect
    cursor_rect sdl.Rect
    textures []*sdl.Texture
}

var gfonts FontSelector = FontSelector{}

//NOTE
//I would like to benchmark sdl.Rect vs MyRect. The reason for it is that
//I have a suspicion that sdl.Rect might have a calling overhead. If that's the case
//this means that I could perhaps store a single pointer to an sdl.Rect
//and just pass arbitrary X, Y, W, H values to that pointer on demand.

func main() {
    // PROFILING SNIPPET
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

    runtime.LockOSThread()

    if err := sdl.Init(sdl.INIT_TIMER|sdl.INIT_VIDEO|sdl.INIT_AUDIO); err != nil {
        panic(err)
    }

    if err := ttf.Init(); err != nil {
        panic(err)
    }

    window, err := sdl.CreateWindow(WIN_TITLE, sdl.WINDOWPOS_CENTERED,
                                               sdl.WINDOWPOS_CENTERED,
                                               WIN_W, WIN_H,
                                               sdl.WINDOW_SHOWN | sdl.WINDOW_RESIZABLE | sdl.WINDOW_OPENGL)
    if err != nil {
        panic(err)
    }

    // NOTE: I've heard that PRESENTVSYNC caps FPS
    renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED | sdl.RENDERER_PRESENTVSYNC)
    if err != nil {
        panic(err)
    }

    // NOTE: important!
    // SetLogicalSize is important for device independant rendering!
    // renderer.SetLogicalSize(WIN_W, WIN_H)

    filename := "text/HP01.txt"

    file_stat, err := os.Stat(filename)
    if err != nil {
        panic(err)
    }

    file_size := file_stat.Size()

    file, err := os.Open(filename)
    if err != nil {
        panic(err)
    }

    file_data := make([]byte, file_size)

    file.Read(file_data)
    file.Close()

    line_tokens := strings.Split(string(file_data), "\n")

    file_data = nil

    ticker := time.NewTicker(time.Second / 60)

    //////////////////////////
    // ------ CREATE_FONTS
    //////////////////////////

    var font *ttf.Font

    file_names, err := ioutil.ReadDir("./fonts/")
    if err != nil {
        panic(err)
    }

    var ttf_font_list []string
    for _, f := range file_names {
        if strings.Contains(f.Name(), ".ttf") {
            ttf_font_list = append(ttf_font_list, f.Name())
        }
        if strings.Contains(f.Name(), ".otf") {
            ttf_font_list = append(ttf_font_list, f.Name())
        }
    }

    file_names = nil

    allfonts := make([]Font, len(ttf_font_list))

    gfonts.fonts = make([]Font, len(ttf_font_list))
    gfonts.textures = make([]*sdl.Texture, len(ttf_font_list))
    gfonts.ttf_rects = make([]sdl.Rect, len(ttf_font_list))

	// NOTE: maybe I should font = all_fonts[...]
	// and just interate over font = all_fonts[...]
	// so that I don't have to do extra allocations
	// basically we would keep them all in memory at all times

    args := os.Args
    DEBUG_INDEX := 6
    if len(args) > 1 {
        DEBUG_INDEX, _ = strconv.Atoi(args[1])
    }

	for index, element := range ttf_font_list {
		allfonts[index].data = load_font("./fonts/" + element, TTF_FONT_SIZE)
		allfonts[index].name = element
		allfonts[index].size = TTF_FONT_SIZE

        if DEBUG_INDEX == index {
            gfonts.current_font = load_font("./fonts/" + element, TTF_FONT_SIZE)
            w, h, _ := gfonts.current_font.SizeUTF8(" ")
            skp := gfonts.current_font.LineSkip()
            gfonts.current_font_w = w
            gfonts.current_font_h = h
            gfonts.current_font_skip = skp
        }
        gfonts.fonts[index].data = load_font("./fonts/" + element, TTF_FONT_SIZE_FOR_FONT_LIST)
		gfonts.fonts[index].name = element
	}

    font = gfonts.current_font

    gfonts.bg_rect = sdl.Rect{}
    adder_y := 0
	for index, element := range gfonts.fonts {
        gx, gy, _ := gfonts.fonts[index].data.SizeUTF8(" ")
		gfonts.fonts[index].size = gx * len(element.name)

        gfonts.textures[index] = make_ttf_texture(renderer, gfonts.fonts[index].data,
                                                                          gfonts.fonts[index].name,
                                                                          &sdl.Color{0, 0, 0, 0})

        gfonts.ttf_rects[index] = sdl.Rect{0, int32(adder_y), int32(gx*len(element.name)), int32(gy)}

        if gfonts.bg_rect.W < gfonts.ttf_rects[index].W {
            gfonts.bg_rect.W = gfonts.ttf_rects[index].W
        }

        gfonts.bg_rect.H += gfonts.ttf_rects[index].H
        adder_y += gy
    }

    // TODO: should we keep fonts in memory? or free them instead?

    start := time.Now()
    test_tokens := make([]string, determine_nwrap_lines(line_tokens, LINE_LENGTH, gfonts.current_font_w))
    for apos, bpos := 0, 0; apos < len(line_tokens); apos += 1 {
        if (len(line_tokens[apos]) > 1) {
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

	//@PERFORMANCE SLOW
    now_gen := time.Now()

    all_lines := make([]Line, len(test_tokens))
    generate_and_populate_lines(renderer, font, &all_lines, &test_tokens)

    end_gen := time.Now().Sub(now_gen)
    fmt.Printf("[[generate_and_populate_lines took %s]]\n", end_gen.String())

    cmd_console_test_str := strings.Join([]string{"LINE COUNT: ", strconv.Itoa(len(test_tokens))}, "")

    cmd_win_h := int32(18)
    cmd := CmdConsole{}
    cmd.alpha_value = 100
    cmd.ttf_texture = make_ttf_texture(renderer, font, cmd_console_test_str, &sdl.Color{0, 0, 0, 255})
    cmd.ttf_rect    = sdl.Rect{0, WIN_H-cmd_win_h, int32(gfonts.current_font_w * len(cmd_console_test_str)), int32(gfonts.current_font_h)}
    cmd.bg_rect     = sdl.Rect{0, WIN_H-cmd_win_h, WIN_W, int32(gfonts.current_font_h)}
    cmd.cursor_rect = sdl.Rect{0, WIN_H-cmd_win_h, int32(gfonts.current_font_w), int32(gfonts.current_font_h)}
    cmd.input_buffer.Grow(128) // we need to make sure we never write past this value?

    sdl.SetHint(sdl.HINT_FRAMEBUFFER_ACCELERATION, "1")
    sdl.SetHint(sdl.HINT_RENDER_SCALE_QUALITY, "1")

    renderer.SetDrawBlendMode(sdl.BLENDMODE_BLEND)

    running := true
    print_word := false
    engage_loop := false
    add_new_line := false
    del_new_line := false

    num_word_textures := 0
    for index := 0; index <= MAX_INDEX; index++ {
        num_word_textures += len(all_lines[index].word_rects)
    }

    mouseover_word_texture := make([]bool, num_word_textures)
    mouseover_word_texture_FONT := make([]bool, len(ttf_font_list))

    _RECTS_ := make([]sdl.Rect, num_word_textures)
    for index, apos := 0, 0; index <= MAX_INDEX; index++ {
        for pos := 0; pos < len(all_lines[index].word_rects); pos++ {
            _RECTS_[apos] = all_lines[index].word_rects[pos]
            apos += 1
        }
    }

    _WORDS_ := make([]string, num_word_textures)
    for index, apos := 0, 0; index <= MAX_INDEX; index++ {
        for _, rct := range strings.Split(test_tokens[index], " ") {
            _WORDS_[apos] = rct
            apos += 1
        }
    }

    wrap_line := false

    move_text_up := false
    move_text_down := false

    test_rand_color := sdl.Color{uint8(rand.Intn(255)),uint8(rand.Intn(255)),uint8(rand.Intn(255)),uint8(rand.Intn(255))}

    wrapline := DebugWrapLine{int32(LINE_LENGTH), 0, int32(LINE_LENGTH), WIN_H}

    curr_char_w := 0

	//viewport_rect := sdl.Rect{0, 0, WIN_W, WIN_H}
	//renderer.SetViewport(&viewport_rect)

    for running {
        for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
            switch t := event.(type) {
                case *sdl.QuitEvent:
                    running = false
                    break
                case *sdl.WindowEvent:
                    switch t.Event {
                        case sdl.WINDOWEVENT_SIZE_CHANGED:
                            global_win_w, global_win_h = t.Data1, t.Data2
                            if global_win_w <= int32(LINE_LENGTH) {
                                wrap_line = true
                            } else {
                                wrap_line = false
                            }

                            if global_win_w > WIN_W && global_win_h > WIN_H {
                                cmd.bg_rect.W = global_win_w
                                cmd.bg_rect.Y = global_win_h-cmd_win_h
                                cmd.ttf_rect.Y = global_win_h-cmd_win_h
                                cmd.cursor_rect.Y = global_win_h-cmd_win_h

                                wrapline.y2 = global_win_h
								//renderer.SetViewport(&viewport_rect)
                            } else {
                                cmd.bg_rect.W = global_win_w
                                cmd.bg_rect.Y = global_win_h-cmd_win_h
                                cmd.ttf_rect.Y = global_win_h-cmd_win_h
                                cmd.cursor_rect.Y = global_win_h-cmd_win_h

                                wrapline.y2 = global_win_h
								//renderer.SetViewport(&viewport_rect)
                            }
                            break
                        default:
                            break
                    }
                    break
                case *sdl.MouseMotionEvent:
                    //fmt.Printf("~> %d %d\n", t.X, t.Y)
                    check_collision_mouse_over_words(t, &_RECTS_, &mouseover_word_texture)
                    check_collision_mouse_over_words(t, &gfonts.ttf_rects, &mouseover_word_texture_FONT)
                    //check_collision_mouse_over_words(t, &line.word_rects, &test_mouse_over)
                    break
                case *sdl.MouseWheelEvent:
                    //fmt.Printf("%#v\n", t)
					if t.Y == 1 {
						move_text_up = true
					}
					if t.Y == -1 {
						move_text_down = true
					}
					break
                case *sdl.MouseButtonEvent:
                    switch t.Type {
                        case sdl.MOUSEBUTTONDOWN:
                        case sdl.MOUSEBUTTONUP:
                            print_word = true
                            break
                        default:
                            break
                    }
                    break
                case *sdl.TextInputEvent:
                    if cmd.show {
                        fmt.Printf("[debug] keyinput: %c\n", t.Text[0])
                        input_char := string(t.Text[0])
                        cmd.input_buffer.WriteString(input_char)
                        cmd.ttf_texture.Destroy()
                        cmd.ttf_texture = make_ttf_texture(renderer, font, cmd.input_buffer.String(), &test_rand_color)
                        curr_char_w = gfonts.current_font_w * len(input_char)
                        cmd.ttf_rect.W = int32(gfonts.current_font_w * len(cmd.input_buffer.String()))
                        cmd.ttf_rect.H = int32(gfonts.current_font_h)
                        cmd.cursor_rect.X += int32(curr_char_w)
                    }
                    break
                case *sdl.KeyboardEvent:
                    if cmd.show { // TODO: @REFACTOR into a func
                        if t.Keysym.Sym == sdl.K_BACKSPACE {
                            if t.Repeat > 0 {
                                if cmd.cursor_rect.X <= 0 {
                                    cmd.cursor_rect.X = 0
                                } else {
                                    temp_string := cmd.input_buffer.String()[0:len(cmd.input_buffer.String())-1]
                                    cmd.input_buffer.Reset()
                                    cmd.input_buffer.WriteString(temp_string)

                                    cmd.ttf_texture.Destroy()

                                    if len(cmd.input_buffer.String()) > 0 {
                                        cmd.ttf_texture = make_ttf_texture(renderer, font, temp_string, &sdl.Color{0, 0, 0, 255})
                                    }

                                    if len(temp_string) != 0 {
                                        curr_char_w = gfonts.current_font_w * len(string(temp_string[len(temp_string)-1]))

                                        cmd.cursor_rect.X -= int32(curr_char_w)

                                        cmd.ttf_rect.W = int32(gfonts.current_font_w * len(cmd.input_buffer.String()))
                                        cmd.ttf_rect.H = int32(gfonts.current_font_h)
                                        println(temp_string)
                                    } else {
                                        cmd.cursor_rect.X = 0
                                    }
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
                                    case sdl.K_TAB: // TEMPORARY
                                            if cmd.show {
                                                cmd.show = false
                                            }
                                            break
                                    case sdl.K_BACKSPACE: // TODO: @REFACTOR into a func
                                        if cmd.cursor_rect.X <= 0 {
                                            cmd.cursor_rect.X = 0
                                        } else {
                                            temp_string := cmd.input_buffer.String()[0:len(cmd.input_buffer.String())-1]
                                            cmd.input_buffer.Reset()
                                            cmd.input_buffer.WriteString(temp_string)

                                            cmd.ttf_texture.Destroy()

                                            if len(cmd.input_buffer.String()) > 0 {
                                                cmd.ttf_texture = make_ttf_texture(renderer, font, temp_string, &sdl.Color{0, 0, 0, 255})
                                            }

                                            if len(temp_string) != 0 {
                                                curr_char_w = gfonts.current_font_w * len(string(temp_string[len(temp_string)-1]))

                                                cmd.cursor_rect.X -= int32(curr_char_w)

                                                cmd.ttf_rect.W = int32(gfonts.current_font_w * len(cmd.input_buffer.String()))
                                                cmd.ttf_rect.H = int32(gfonts.current_font_h)
                                                println(temp_string)
                                            } else {
                                                cmd.cursor_rect.X = 0
                                            }
                                        }
                                        break
                                    case sdl.K_RETURN: // TODO: @REFACTOR into a func
                                        // TODO: I need to add a command_history and a command_buffer here!
                                        //       I'm just not sure which data structure to use, at the moment.
                                        if cmd.show {
                                            if len(cmd.input_buffer.String()) > 0 {
                                                fmt.Printf("[debug] PRE-Reset Buffer len %d \n", len(cmd.input_buffer.String()))
                                                cmd.input_buffer.Reset()
                                                cmd.ttf_texture.Destroy()
                                                cmd.cursor_rect.X = 0
                                                fmt.Printf("[debug] Reset Buffer len %d \n", len(cmd.input_buffer.String()))
                                                fmt.Printf("[debug] cmd_text_buffer (cap): %d\n", cmd.input_buffer.Cap())
                                            }
                                        }
                                        break
                                    default:
                                        break
                                }
                            }
                            break
                        default:
                            break
                    }
                    if t.Keysym.Sym == sdl.K_ESCAPE {
                        running = false
                        break
                    }
                    if t.Keysym.Sym == sdl.K_UP {
                        move_text_up = true
                    }
                    if t.Keysym.Sym == sdl.K_DOWN {
                        move_text_down = true
                    }
                    if t.Keysym.Sym == sdl.K_LEFT {
                        println("SHOULD SCROLL FONT back")
                    }
                    if t.Keysym.Sym == sdl.K_RIGHT {
                        println("SHOULD SCROLL FONT forward")
                    }
                    break
                default:
                    continue
            }
        }
        renderer.SetDrawColor(255, 255, 255, 0)
        renderer.Clear()

        // RENDERING TTF LINES
        for i := range all_lines[START_INDEX:MAX_INDEX] {
            renderer.Copy(all_lines[i].texture, nil, &all_lines[i].bg_rect)
        }

        for i := range mouseover_word_texture {
            if mouseover_word_texture[i] == true {
                engage_loop = true
            }
        }

        if engage_loop  && !cmd.show {
            for index := range _RECTS_ {
                if mouseover_word_texture[index] {
                    if _WORDS_[index] != "\n" {
                        draw_rect_without_border(renderer, &_RECTS_[index], &sdl.Color{255, 100, 200, 100})
                        if print_word {
                            if _WORDS_[index] != "\n" {
                                fmt.Printf("%s\n", _WORDS_[index])
                                print_word = false
                            }
                        }
                    }
                } else {
                    renderer.SetDrawColor(0, 0, 0, 0)
                }
            }
            engage_loop = false
        }

        if move_text_down {
            move_text_down = false
            for index := range all_lines[START_INDEX:MAX_INDEX] {
                all_lines[index].bg_rect.Y -= TEXT_SCROLL_SPEED
            }
            for index := range _RECTS_ {
                _RECTS_[index].Y -= TEXT_SCROLL_SPEED
            }
            add_new_line = true
        }
        if move_text_up {
            move_text_up = false
            for index := range all_lines[START_INDEX:MAX_INDEX] {
                all_lines[index].bg_rect.Y += TEXT_SCROLL_SPEED
            }
            for index := range _RECTS_ {
                _RECTS_[index].Y += TEXT_SCROLL_SPEED
            }
            del_new_line = true
        }

        if add_new_line {
            MAX_INDEX = MAX_INDEX + 1
            all_lines[MAX_INDEX].bg_rect.Y = all_lines[MAX_INDEX-1].bg_rect.Y + (all_lines[MAX_INDEX].bg_rect.H - TEXT_SCROLL_SPEED)
            all_lines[MAX_INDEX-1].bg_rect.Y -= TEXT_SCROLL_SPEED
            add_new_line = false
        }

        if del_new_line {
            MAX_INDEX = MAX_INDEX - 1
            del_new_line = false
        }

        if wrap_line {
            for index := range all_lines[START_INDEX:MAX_INDEX] {
                draw_rect_without_border(renderer, &all_lines[index].bg_rect, &sdl.Color{100, 255, 255, 100})
            }
        }

        if cmd.show {
            draw_rect_with_border_filled(renderer, &cmd.bg_rect, &sdl.Color{255, 10, 100, cmd.alpha_value})
            draw_rect_with_border(renderer, &cmd.ttf_rect, &sdl.Color{255, 255, 255, 0})

            renderer.Copy(cmd.ttf_texture, nil, &cmd.ttf_rect)

            draw_rect_with_border_filled(renderer, &cmd.cursor_rect, &sdl.Color{0, 0, 0, cmd.alpha_value})

            draw_rect_without_border(renderer, &gfonts.bg_rect, &sdl.Color{255, 0, 255, 255})

            for i := 0; i < len(gfonts.textures); i++ {
                renderer.Copy(gfonts.textures[i], nil, &gfonts.ttf_rects[i]) // why nil?
            }

            clr := sdl.Color{255, 0, 255, 100}
            for index := 0; index < len(gfonts.ttf_rects); index++ {
                if (mouseover_word_texture_FONT[index] == true) {
                    draw_rect_without_border(renderer, &gfonts.ttf_rects[index], &clr)
                } else {
                    draw_rect_without_border(renderer, &gfonts.ttf_rects[index], &sdl.Color{255, 255, 255, 200}) // debug
                }
            }
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

    destroy_lines(&all_lines) // @WIP

    if cmd.ttf_texture != nil {
        cmd.ttf_texture.Destroy()
        cmd.ttf_texture = nil
    }

	for index := range ttf_font_list {
		allfonts[index].data.Close() // @TEMPORARY HACK @SLOW
        allfonts[index].data = nil

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

func load_font(name string, size int) (*ttf.Font) {
    var font *ttf.Font
    var err error

    if font, err = ttf.OpenFont(name, size); err != nil {
        panic(err)
    }
    return font
}

func reload_font(font *ttf.Font, name string, size int) (*ttf.Font) {
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

func make_ttf_texture(renderer *sdl.Renderer, font *ttf.Font, text string, color *sdl.Color) (*sdl.Texture) {
    var surface *sdl.Surface
    var texture *sdl.Texture

	assert_if(len(text) <= 0)

    surface , _= font.RenderUTF8Blended(text, *color)
    texture , _= renderer.CreateTextureFromSurface(surface)
    surface.Free()

    return texture
}

func reload_ttf_texture(r *sdl.Renderer, tex *sdl.Texture, f *ttf.Font, s string, c sdl.Color) (*sdl.Texture) {
    if tex != nil {
        tex.Destroy()
        var surface *sdl.Surface
        surface, _ = f.RenderUTF8Blended(s, c)
        tex, _ = r.CreateTextureFromSurface(surface)
        surface.Free()
        return tex
    }
    return tex
}

func generate_and_populate_lines(r *sdl.Renderer, font *ttf.Font, dest *[]Line, tokens *[]string) {
    for index := 0; index < len(*tokens); index++ {
        new_ttf_texture_line(r, font, &(*dest)[index], (*tokens)[index], int32(index))
    }
}

func new_ttf_texture_line(rend *sdl.Renderer, font *ttf.Font, line *Line, line_text string, skip_nr int32) {
	assert_if(len(line_text) == 0)

    line.texture = make_ttf_texture(rend, font, line_text, &sdl.Color{0, 0, 0, 0})

    text := strings.Split(line_text, " ")
    text_len := len(text)
    line.word_rects = make([]sdl.Rect, text_len)

    x, y, _ := font.SizeUTF8(" ")
    lineskip := font.LineSkip()

    tw := x * len(line_text)

    skipline := int32(lineskip)
    if (skip_nr > 0) {
        skipline *= skip_nr
    } else {
        skipline = 0
    }
    move_x  := X_OFFSET
    move_y  := skip_nr
    ix := 0
    for index := 0; index < text_len; index++ {
        ix = x * len(text[index])
        if index == 0 {
            move_y *= int32(lineskip)
        }
        line.word_rects[index] = sdl.Rect{int32(move_x), int32(move_y), int32(ix), int32(y)}
        move_x += (ix + x)
    }
    line.bg_rect = sdl.Rect{int32(X_OFFSET), skipline, int32(tw), int32(y)}
    text = nil
}

func check_collision_mouse_over_words(event *sdl.MouseMotionEvent, rects *[]sdl.Rect, mouse_over *[]bool) {
    for index := range *rects {
        mx_gt_rx :=    event.X > (*rects)[index].X
        mx_lt_rx_rw := event.X < (*rects)[index].X + (*rects)[index].W
        my_gt_ry :=    event.Y > (*rects)[index].Y
        my_lt_ry_rh := event.Y < (*rects)[index].Y + (*rects)[index].H

        if ((mx_gt_rx && mx_lt_rx_rw) && (my_gt_ry && my_lt_ry_rh)) {
            (*mouse_over)[index] = true
        } else {
            (*mouse_over)[index] = false
        }
    }
}

func do_wrap_lines(str string, max_len int, xsize int) []string {
    assert_if(len(str) <= 1)

    result := make([]string, determine_nwrap_lines([]string{str}, max_len, xsize))

    pos := 0
    if (len(str) * xsize) + X_OFFSET <= max_len {
        result[pos] = str
        return result
    } else {
        start := 0
        mmax := int(math.RoundToEven(float64(max_len / xsize)))-1 // use math.Round instead?
        slice := str[start:mmax]
        end := mmax
        slice_len := 0
        for end < len(str) {
            slice_len = len(slice)
            if !is_space(string(slice[slice_len-1])) {
                for !is_space(string(slice[slice_len-1])) {
                    end = end-1
                    slice_len = slice_len - 1
                }
            }
            end = end - 1 // remove space
            slice = str[start:end]
            result[pos] = slice
            pos += 1
            start = end+1
            end = (end + mmax)
            if (end > len(str)) {
                slice = str[start:end-(end-len(str))]
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
        if (len(str[index]) * xsize) + X_OFFSET <= max_len {
            result += 1
        } else {
            start := 0
            mmax := int(math.RoundToEven(float64(max_len / xsize)))-1 // use math.Round instead?
            //println(mmax > len(str[index]), "index", index, "strlen", len(str[index]), "mmax", mmax)
            //assert_if(mmax > len(str[index]))
            slice := str[index][start:mmax]
            end := mmax
            slice_len := 0
            for end < len(str[index]) {
                slice_len = len(slice)
                if !is_space(string(slice[slice_len-1])) {
                    for !is_space(string(slice[slice_len-1])) {
                        end = end-1
                        slice_len = slice_len - 1
                    }
                }
                end = end - 1 // remove space
                slice = str[index][start:end]
                result += 1
                start = end+1
                end = (end + mmax)
                if (end > len(str[index])) {
                    slice = str[index][start:end-(end-len(str[index]))]
                    result += 1
                    break
                }
                slice = str[index][start:end]
            }
        }
    }
    return result
}

func destroy_lines(lines *[]Line) {
    for _, line := range *lines {
        line.texture.Destroy()
        line.texture = nil
    }
}

func assert_if(cond bool) {
	if (cond) {
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
        if (string((*s)[index]) == "\n") {
            break
        }
        if (string((*s)[index]) == "\r") {
            break
        }
        if (!is_space(string((*s)[index]))) {
            curr += 1
        } else {
            curr *= 7
            result = append(result, curr)
            curr = 0
        }
    }
    if (curr > 0) {
        curr *= 7
        result = append(result, curr)
    }
    return result
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
