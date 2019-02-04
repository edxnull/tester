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
    alpha_value int //anim_alpha
    bg_rect sdl.Rect
    ttf_rect sdl.Rect
    cursor_block sdl.Rect
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
    alpha_value int
    bg_rect sdl.Rect
    ttf_rects []sdl.Rect
    cursor_rect sdl.Rect
    textures []*sdl.Texture
}

var global_font_selector FontSelector = FontSelector{}

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

    global_font_selector.fonts = make([]Font, len(ttf_font_list))
    global_font_selector.textures = make([]*sdl.Texture, len(ttf_font_list))
    global_font_selector.ttf_rects = make([]sdl.Rect, len(ttf_font_list))

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
            global_font_selector.current_font = load_font("./fonts/" + element, TTF_FONT_SIZE)
        }
        global_font_selector.fonts[index].data = load_font("./fonts/" + element, TTF_FONT_SIZE_FOR_FONT_LIST)
		global_font_selector.fonts[index].name = element
	}

    font = global_font_selector.current_font

    CHAR_W, CHAR_H, _ := font.SizeUTF8(" ")
    SKIP_LINE := font.LineSkip()

    global_font_selector.bg_rect = sdl.Rect{}
    adder_y := 0
	for index, element := range global_font_selector.fonts {
        gx, gy, _ := global_font_selector.fonts[index].data.SizeUTF8(" ")
		global_font_selector.fonts[index].size = gx * len(element.name)

        global_font_selector.textures[index] = make_ttf_texture(renderer, global_font_selector.fonts[index].data,
                                                                          global_font_selector.fonts[index].name,
                                                                          &sdl.Color{0, 0, 0, 0})

        global_font_selector.ttf_rects[index] = sdl.Rect{0, int32(adder_y), int32(gx*len(element.name)), int32(gy)}

        if global_font_selector.bg_rect.W < global_font_selector.ttf_rects[index].W {
            global_font_selector.bg_rect.W = global_font_selector.ttf_rects[index].W
        }

        global_font_selector.bg_rect.H += global_font_selector.ttf_rects[index].H
        adder_y += gy
    }

    // TODO: should we keep fonts in memory? or free them instead?

    start := time.Now()
    test_tokens := make([]string, determine_nwrap_lines(line_tokens, LINE_LENGTH, CHAR_W))
    for apos, bpos := 0, 0; apos < len(line_tokens); apos += 1 {
        if (len(line_tokens[apos]) > 1) {
            current := do_wrap_lines(line_tokens[apos], LINE_LENGTH, CHAR_W)
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
    slice := test_tokens[0:100] // NOTE: TEST
    // nlines_to_render := math.RoundToEven(float32(SCREEN_H/SKIP_LINE))
    //nlines := 100
    //func get_lines_to_render(renderer, font, all_lines, curr_nlines, max_lines) []Line {
    //    // ...
    //}
    //println(SKIP_LINE, CHAR_H)
    all_lines := generate_and_populate_lines(renderer, font, &slice, CHAR_W, CHAR_H, SKIP_LINE)
    end_gen := time.Now().Sub(now_gen)
    fmt.Printf("[[generate_and_populate_lines took %s]]\n", end_gen.String())

    //////////////////////////
    // CMD_CONSOLE_STUFF
    //////////////////////////

    cmd_win_h := int32(18)
    show_cmd_console_rect := false
    cmd_console_test_str := strings.Join([]string{"LINE COUNT: ", strconv.Itoa(len(test_tokens))}, "")
    cmd_console_anim_alpha := 0
    cmd_move_left := false

    //@SPEED this is slow. Use strings.Builder{} instead, or something else.
    var cmd_text_buffer bytes.Buffer
    // TODO: we need to save our commands in a bytes.Buffer. We also need a command_list.
    var cmd_console_ttf_texture *sdl.Texture

    cmd_rand_color := sdl.Color{0, 0, 0, 255}

    cmd_console_ttf_texture = make_ttf_texture(renderer, font, cmd_console_test_str, &cmd_rand_color)

    cmd_console_ttf_rect     := sdl.Rect{0, WIN_H-cmd_win_h, int32(CHAR_W * len(cmd_console_test_str)), int32(CHAR_H)}
    cmd_console_rect         := sdl.Rect{0, WIN_H-cmd_win_h, WIN_W, int32(CHAR_H)}
    cmd_console_cursor_block := sdl.Rect{0, WIN_H-cmd_win_h, int32(CHAR_W), int32(CHAR_H)}

    //////////////////////////
    // END_CMD_CONSOLE_STUFF
    //////////////////////////

    sdl.SetHint(sdl.HINT_FRAMEBUFFER_ACCELERATION, "1")
    sdl.SetHint(sdl.HINT_RENDER_SCALE_QUALITY, "1")

    renderer.SetDrawBlendMode(sdl.BLENDMODE_BLEND)

    running := true
    first_pass := true
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
                                cmd_console_rect.W = global_win_w
                                cmd_console_rect.Y = global_win_h-cmd_win_h
                                cmd_console_ttf_rect.Y = global_win_h-cmd_win_h
                                cmd_console_cursor_block.Y = global_win_h-cmd_win_h

                                wrapline.y2 = global_win_h
								//renderer.SetViewport(&viewport_rect)
                            } else {
                                cmd_console_rect.W = global_win_w
                                cmd_console_rect.Y = global_win_h-cmd_win_h
                                cmd_console_ttf_rect.Y = global_win_h-cmd_win_h
                                cmd_console_cursor_block.Y = global_win_h-cmd_win_h

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
                    check_collision_mouse_over_words(t, &global_font_selector.ttf_rects, &mouseover_word_texture_FONT)
                    //check_collision_mouse_over_words(t, &line.word_rects, &test_mouse_over)
                    break
                case *sdl.MouseWheelEvent:
                    //fmt.Printf("%#v\n", t)
					if t.Y == -1 {
						move_text_up = true
					}
					if t.Y == 1 {
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
                    if show_cmd_console_rect {
                        fmt.Printf("keyinput: %c\n", t.Text[0])
                        input_char := string(t.Text[0])
                        cmd_text_buffer.WriteString(input_char)

                        cmd_console_ttf_texture.Destroy()
                        cmd_console_ttf_texture = make_ttf_texture(renderer, font, cmd_text_buffer.String(), &test_rand_color)

                        curr_char_w = CHAR_W * len(input_char)

                        cmd_console_ttf_rect.W = int32(CHAR_W * len(cmd_text_buffer.String()))
                        cmd_console_ttf_rect.H = int32(CHAR_H)

                        cmd_console_cursor_block.X += int32(curr_char_w)
                    }
                    break
                case *sdl.KeyboardEvent:
                    if show_cmd_console_rect { // TODO: @REFACTOR into a func
                        if t.Keysym.Sym == sdl.K_BACKSPACE {
                            if t.Repeat > 0 {
                                if cmd_console_cursor_block.X <= 0 {
                                    cmd_console_cursor_block.X = 0
                                } else {
                                    temp_string := cmd_text_buffer.String()[0:len(cmd_text_buffer.String())-1]
                                    cmd_text_buffer.Reset()
                                    cmd_text_buffer.WriteString(temp_string)

                                    cmd_console_ttf_texture.Destroy()

                                    if len(cmd_text_buffer.String()) > 0 {
                                        cmd_console_ttf_texture = make_ttf_texture(renderer, font, temp_string, &cmd_rand_color)
                                    }

                                    if len(temp_string) != 0 {
                                        curr_char_w = CHAR_W * len(string(temp_string[len(temp_string)-1]))

                                        cmd_console_cursor_block.X -= int32(curr_char_w)

                                        cmd_console_ttf_rect.W = int32(CHAR_W * len(cmd_text_buffer.String()))
                                        cmd_console_ttf_rect.H = int32(CHAR_H)

                                        println(temp_string)
                                    } else {
                                        cmd_console_cursor_block.X = 0
                                    }
                                }
                            }
                        }
                    }
                    switch t.Type {
                        case sdl.KEYDOWN:
                        case sdl.KEYUP:
                            if t.Keysym.Sym == sdl.K_SPACE {
                                if !show_cmd_console_rect {
                                    show_cmd_console_rect = true
                                }
                            } else {
                                switch t.Keysym.Sym {
                                    case sdl.KEYDOWN:
                                    case sdl.K_TAB: // TEMPORARY
                                            if show_cmd_console_rect {
                                                show_cmd_console_rect = false
                                            }
                                            break
                                    case sdl.K_BACKSPACE: // TODO: @REFACTOR into a func
                                        if show_cmd_console_rect {
                                            if cmd_console_cursor_block.X <= 0 {
                                                cmd_console_cursor_block.X = 0
                                            } else {
                                                temp_string := cmd_text_buffer.String()[0:len(cmd_text_buffer.String())-1]
                                                cmd_text_buffer.Reset()
                                                cmd_text_buffer.WriteString(temp_string)

                                                cmd_console_ttf_texture.Destroy()

                                                if len(cmd_text_buffer.String()) > 0 {
                                                    cmd_console_ttf_texture = make_ttf_texture(renderer, font, temp_string, &cmd_rand_color)
                                                }

                                                if len(temp_string) != 0 {
                                                    curr_char_w = CHAR_W * len(string(temp_string[len(temp_string)-1]))

                                                    cmd_console_cursor_block.X -= int32(curr_char_w)

                                                    cmd_console_ttf_rect.W = int32(CHAR_W *len(cmd_text_buffer.String()))
                                                    cmd_console_ttf_rect.H = int32(CHAR_H)

                                                    println(temp_string)
                                                } else {
                                                    cmd_console_cursor_block.X = 0
                                                }
                                            }
                                        }
                                        break
                                    case sdl.K_RETURN: // TODO: @REFACTOR into a func
                                        // TODO: I need to add a command_history and a command_buffer here!
                                        //       I'm just not sure which data structure to use, at the moment.
                                        if show_cmd_console_rect {
                                            if len(cmd_text_buffer.String()) > 0 {
                                                fmt.Printf("[debug] PRE-Reset Buffer len %d \n", len(cmd_text_buffer.String()))
                                                cmd_text_buffer.Reset()
                                                cmd_console_ttf_texture.Destroy()
                                                cmd_console_cursor_block.X = 0
                                                fmt.Printf("[debug] Reset Buffer len %d \n", len(cmd_text_buffer.String()))
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

        // @TEST RENDERING TTF LINE
        if first_pass {
            for ln := range all_lines[START_INDEX:MAX_INDEX] {
                for index := range all_lines[ln].word_rects {
                    //renderer.SetDrawColor(100, 10, 100, uint8(cmd_console_anim_alpha))
                    renderer.SetDrawColor(0, 0, 0, 0)
                    renderer.FillRect(&all_lines[ln].word_rects[index])
                    renderer.DrawRect(&all_lines[ln].word_rects[index])
                }
                renderer.Copy(all_lines[ln].texture, nil, &all_lines[ln].bg_rect)
            }
            first_pass = false
        } else {
            for i := range all_lines[START_INDEX:MAX_INDEX] {
                renderer.Copy(all_lines[i].texture, nil, &all_lines[i].bg_rect)
            }
        }

        for i := range mouseover_word_texture {
            if mouseover_word_texture[i] == true {
                engage_loop = true
            }
        }

        if engage_loop  && !show_cmd_console_rect {
            for index := range _RECTS_ {
                if mouseover_word_texture[index] {
                    if _WORDS_[index] != "\n" {
                        renderer.SetDrawColor(255, 100, 200, 100)
                        renderer.FillRect(&_RECTS_[index])
                        renderer.DrawRect(&_RECTS_[index])

                        if print_word {
                            if _WORDS_[index] != "\n" {
                                fmt.Printf("%s\n", _WORDS_[index])
                                print_word = false
                            }
                        }
                    }
                } else {
                    renderer.SetDrawColor(0, 0, 0, 0)
                    renderer.FillRect(&_RECTS_[index])
                    renderer.DrawRect(&_RECTS_[index])
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
                renderer.SetDrawColor(100, 255, 255, 100)
                renderer.FillRect(&all_lines[index].bg_rect)
                renderer.DrawRect(&all_lines[index].bg_rect)
            }
        }
        // @TEST RENDERING TTF LINE

        // DRAWING_CMD_CONSOLE
        if show_cmd_console_rect {
            renderer.SetDrawColor(255, 10, 100, uint8(cmd_console_anim_alpha))
            //renderer.SetDrawColor(255, 255, 255, 255)
            renderer.FillRect(&cmd_console_rect)
            renderer.DrawRect(&cmd_console_rect)

            // renderer.SetDrawColor(100, 25, 90, 255)  // @TEMPORARY
            renderer.SetDrawColor(255, 255, 255, 0)
            renderer.DrawRect(&cmd_console_ttf_rect)
            //renderer.FillRect(&cmd_console_ttf_rect)
            renderer.Copy(cmd_console_ttf_texture, nil, &cmd_console_ttf_rect)

            renderer.SetDrawColor(0, 0, 0, uint8(cmd_console_anim_alpha))
            renderer.FillRect(&cmd_console_cursor_block)
            renderer.DrawRect(&cmd_console_cursor_block)

        // ...............
            draw_rect_without_border(renderer, &global_font_selector.bg_rect, &sdl.Color{255, 0, 255, 255})

            for i := 0; i < len(global_font_selector.textures); i++ {
                renderer.Copy(global_font_selector.textures[i], nil, &global_font_selector.ttf_rects[i]) // why nil?
            }

            clr := sdl.Color{255, 0, 255, 100}
            for index := 0; index < len(global_font_selector.ttf_rects); index++ {
                if (mouseover_word_texture_FONT[index] == true) {
                    draw_rect_without_border(renderer, &global_font_selector.ttf_rects[index], &clr)
                } else {
                    // debug
                    draw_rect_without_border(renderer, &global_font_selector.ttf_rects[index], &sdl.Color{255, 255, 255, 200})
                }
            }
        }

        // ...............
        // DRAWING_CMD_CONSOLE

        // WRAPLINE
        renderer.SetDrawColor(255, 100, 0, uint8(cmd_console_anim_alpha))
        renderer.DrawLine(wrapline.x1+int32(X_OFFSET), wrapline.y1, wrapline.x2+int32(X_OFFSET), wrapline.y2)
        // WRAPLINE

        // -----------------
        // ANIMATIONS
        // -----------------

        if !cmd_move_left {
            cmd_console_anim_alpha += 4
            if cmd_console_anim_alpha >= 80 {
                cmd_move_left = true
            }
        } else {
            cmd_console_anim_alpha -= 4
            if cmd_console_anim_alpha == 0 {
                cmd_move_left = false
            }
        }

        // -----------------
        // ANIMATIONS
        // -----------------

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

    if cmd_console_ttf_texture != nil {
        cmd_console_ttf_texture.Destroy()
        cmd_console_ttf_texture = nil
    }

	for index := range ttf_font_list {
		allfonts[index].data.Close() // @TEMPORARY HACK @SLOW
        allfonts[index].data = nil

        global_font_selector.fonts[index].data.Close()
        global_font_selector.current_font.Close()
        global_font_selector.fonts[index].data = nil
        global_font_selector.textures[index].Destroy()
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
    var surface *sdl.Surface

    if tex != nil {
        tex.Destroy()
        surface, _ = f.RenderUTF8Blended(s, c)

        tex, _ = r.CreateTextureFromSurface(surface)
        surface.Free()
        return tex
    }
    return tex
}

func generate_and_populate_lines(r *sdl.Renderer, font *ttf.Font, tokens *[]string, x int, y int, skipline int) (line []Line) {
    all_lines := make([]Line, len(*tokens))
    for index := 0; index < len(*tokens); index++ {
        new_ttf_texture_line(r, font, &all_lines[index], (*tokens)[index], int32(index), x, y, skipline)
    }
    return all_lines
}

func new_ttf_texture_line(rend *sdl.Renderer, font *ttf.Font, line *Line, line_text string, skip_nr int32, x int, y int, lineskip int) {
	assert_if(len(line_text) == 0)

    line.texture = make_ttf_texture(rend, font, line_text, &sdl.Color{0, 0, 0, 0})

    text := strings.Split(line_text, " ")
    text_len := len(text)
    line.word_rects = make([]sdl.Rect, text_len)

    tw := x * len(line_text)

    skipline := int32(lineskip) // @TEMPORARY HACK
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
    //var result []string
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

// [NOTE]
// This function will fail if MAX_LEN 
// is small enough to trigger is_space ifinite loop!
func determine_nwrap_lines(str []string, max_len int, xsize int) int32 {
    var result int32

    //println(len(str))
    for index := 0; index < len(str); index++ {
        if (len(str[index]) * xsize) + X_OFFSET <= max_len {
            result += 1
            //return result
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

// TODO: not sure we need this
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
