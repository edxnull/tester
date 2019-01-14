package main

import (
    "os"
    "log"
    "fmt"
    "time"
    "flag"
    "bytes"
	"errors"
    "strings"
    "runtime"
    "io/ioutil"
    "math/rand"
    "runtime/pprof"
    "github.com/veandco/go-sdl2/sdl"
    "github.com/veandco/go-sdl2/ttf"
)

const WIN_TITLE string = "GO_TEXT_APPLICATION"

const WIN_W int32 = 800
const WIN_H int32 = 600

const OFFSCREEN_W int32 = 800
const OFFSCREEN_H int32 = 1200

const TTF_FONT_SIZE int = 13

const MAX_TOKENS int = 20
const MAX_LINES  int = 18
const MAX_TEXT_WIDTH int32 = 100
const MAX_LINE_LEN int = 413 // @TEMPORARY

const TEXT_SCROLL_SPEED int32 = 5

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to 'file'")
var memprofile = flag.String("memprofile", "", "write mem profile to 'file'")

// @GLOBAL MUT VARS 
var global_win_w int32
var global_win_h int32

//TODO: https://www-archive.mozilla.org/projects/intl/input-method-spec.html
//TODO: https://pavelfatin.com/scrolling-with-pleasure/
//TODO: https://github.com/benbjohnson/testing 
//TODO: https://golang.org/pkg/runtime/pprof/
//TODO: https://www.ardanlabs.com/blog/2018/01/escape-analysis-flaws.html
//TODO: We need to make sure we render 1 page as a texture, otherwise we are just wasting a lot of
//      everything.
//TODO: https://austburn.me/blog/go-profile.html  // IMPORANT!
//TODO: https://segment.com/blog/allocation-efficiency-in-high-performance-go-services/
//TODO: https://motion-express.com/blog/organizing-a-go-project 
//TODO: https://justinas.org/best-practices-for-errors-in-go 
//TODO: https://www.joeshaw.org/dont-defer-close-on-writable-files/

type Texture struct {
    width  int32
    height int32
    data *sdl.Texture
}

type Font struct {
    size int
    name string
    data *ttf.Font
}

type Line struct {
    text string
    color sdl.Color
    texture Texture
    bg_rect sdl.Rect
    word_rects []sdl.Rect
}

// type Tester struct
// ------------------
// TEXT:     [n1, n2, n3 ... n]
// WORDS:    [n1, n2, n3 ... n]
// RECTS:    [n1, n2, n3 ... n]
// BG_RECTS: [n1, n2, n3 ... n]
// TEXTURES: [n1, n2, n3 ... n]

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
    if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
        panic(err)
    }
    defer sdl.Quit()

    if err := ttf.Init(); err != nil {
        panic(err)
    }
    defer ttf.Quit()

    window, err := sdl.CreateWindow(WIN_TITLE, sdl.WINDOWPOS_CENTERED,
                                               sdl.WINDOWPOS_CENTERED,
                                               WIN_W, WIN_H,
                                               sdl.WINDOW_SHOWN | sdl.WINDOW_RESIZABLE | sdl.WINDOW_OPENGL)
    if err != nil {
        panic(err)
    }
    defer window.Destroy()

    // NOTE: I've heard that PRESENTVSYNC caps FPS
    renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED | sdl.RENDERER_PRESENTVSYNC)
    if err != nil {
        panic(err)
    }
    defer renderer.Destroy()

    file_stat, err := os.Stat("HP01.txt")
    if err != nil {
        panic(err)
    }

    file_size := file_stat.Size()

    file, err := os.Open("HP01.txt")
    if err != nil {
        panic(err)
    }

    file_data := make([]byte, file_size)

    file.Read(file_data)
    file.Close()

    ticker := time.NewTicker(time.Second / 30)

    //////////////////////////
    // ------ CREATE_FONTS
    //////////////////////////

    var font *ttf.Font

    file_names, err := ioutil.ReadDir("./")
    if err != nil {
        panic(err)
    }

    var ttf_font_list []string
    for _, f := range file_names {
        if strings.Contains(f.Name(), ".ttf") {
            ttf_font_list = append(ttf_font_list, f.Name())
        }
    }

    allfonts := make([]Font, len(ttf_font_list))

    fmt.Println(ttf_font_list)

    font = load_font("Inconsolata-Regular.ttf", TTF_FONT_SIZE)

	// NOTE: maybe I should font = all_fonts[...]
	// and just interate over font = all_fonts[...]
	// so that I don't have to do extra allocations
	// basically we would keep them all in memory at all times

	for index, element := range ttf_font_list {
		// TODO: new_font(&Font[index])
		// TODO: close_fonts(&[]Font)
		allfonts[index].data = load_font(element, TTF_FONT_SIZE)
		allfonts[index].name = element
		allfonts[index].size = TTF_FONT_SIZE
		defer allfonts[index].data.Close() // @TEMPORARY HACK @SLOW
	}

    // font = allfonts[1].data
    //TODO: @FIND_USE_CASE: //font = reload_font(font, "Opensans-Bold.ttf", TTF_FONT_SIZE)
    //TODO: @NOT_IMPLEMENTED: I should be able to dynamically load font related functinos on demand

    // ----
    //var char rune = 0x41
    //fmt.Println(font.GlyphMetrics(char))
    //fmt.Printf("font ascend: %d\n", font.Ascent())
    //fmt.Printf("font descend: %d\n", font.Descent())
    //font.SetOutline(1)
    //font.SetStyle(ttf.STYLE_UNDERLINE) //STYLE_UNDERLINE; STYLE_BOLD; STYLE_ITALIC; STYLE_STRIKETHROUGH
    //font.SetKerning(true)

    line_tokens := strings.Split(string(file_data), "\n")

    // @TEMPORARY
    const LINE_LENGTH int = 640
    test_tokens := do_wrap_lines(font, &line_tokens[0], LINE_LENGTH)
    for index := 1; index < 15; index += 1 {
        if (len(line_tokens[index]) > 1) {
            current := do_wrap_lines(font, &line_tokens[index], LINE_LENGTH)
            for _, element := range current {
                test_tokens = append(test_tokens, element)
            }
        }
    }

    all_lines := generate_and_populate_lines(renderer, font, &test_tokens)
    //all_lines := generate_and_populate_lines(renderer, font, &line_tokens)

    //////////////////////////
    // CMD_CONSOLE_STUFF
    //////////////////////////

    cmd_win_h := int32(18)
    show_cmd_console_rect := false
    cmd_console_test_str := "cmd_console_engaged!"
    cmd_console_anim_alpha := 0
    cmd_move_left := false

    var cmd_text_buffer bytes.Buffer
    // TODO: we need to save our commands in a bytes.Buffer. We also need a command_list.
    var cmd_console_ttf_texture *sdl.Texture

    cmd_rand_color := sdl.Color{0, 0, 0, 255}

    cmd_console_ttf_texture = make_ttf_texture(renderer, font, cmd_console_test_str, cmd_rand_color)

    cmd_console_ttf_w, cmd_console_ttf_h := get_text_size(font, cmd_console_test_str)

    ttf_letter_w, ttf_letter_h := get_text_size(font, "A") // "A" is just a random letter for our usecase

    cmd_console_ttf_rect     := sdl.Rect{0, WIN_H-cmd_win_h, int32(cmd_console_ttf_w), int32(cmd_console_ttf_h)}
    cmd_console_rect         := sdl.Rect{0, WIN_H-cmd_win_h, WIN_W, int32(cmd_console_ttf_h)}
    cmd_console_cursor_block := sdl.Rect{0, WIN_H-cmd_win_h, int32(ttf_letter_w), int32(ttf_letter_h)}

    //////////////////////////
    // END_CMD_CONSOLE_STUFF
    //////////////////////////

    renderer_info, err := renderer.GetInfo()
    if err != nil {
        panic(err)
    }

    fmt.Println(renderer_info)
    fmt.Println(sdl.GetPlatform())

    renderer.SetDrawBlendMode(sdl.BLENDMODE_BLEND)

    running := true
    print_word := false

    total := 0
    for index := range all_lines {
        total += len(all_lines[index].word_rects)
    }
    mouseover_word_texture := make([]bool, total)

    _RECTS_ := make([]sdl.Rect, 0)
    for index := range all_lines {
        for _, rct := range all_lines[index].word_rects {
            _RECTS_ = append(_RECTS_, rct)
        }
    }
    println(len(_RECTS_), len(mouseover_word_texture))

    _WORDS_ := make([]string, 0)
    for index := range all_lines {
        for _, rct := range strings.Split(all_lines[index].text, " ") {
            _WORDS_ = append(_WORDS_, rct)
        }
    }

    println(len(_RECTS_), len(mouseover_word_texture), len(_WORDS_))

    fmt.Println("FONT_FIXED_WIDTH: ", font.FaceIsFixedWidth())

    wrap_line := false

    //move_text_up := false
    //move_text_down := false

    test_rand_color := sdl.Color{uint8(rand.Intn(255)),uint8(rand.Intn(255)),uint8(rand.Intn(255)),uint8(rand.Intn(255))}

    curr_char_w := 0
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
                            } else {
                                cmd_console_rect.W = global_win_w
                                cmd_console_rect.Y = global_win_h-cmd_win_h
                                cmd_console_ttf_rect.Y = global_win_h-cmd_win_h
                                cmd_console_cursor_block.Y = global_win_h-cmd_win_h
                            }
                            break
                        default:
                            break
                    }
                    break
                case *sdl.MouseMotionEvent:
                    //fmt.Printf("~> %d %d\n", t.X, t.Y)
                    check_collision_mouse_over_words(t, &_RECTS_, &mouseover_word_texture)
                    //check_collision_mouse_over_words(t, &line.word_rects, &test_mouse_over)
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
                        cmd_console_ttf_texture = make_ttf_texture(renderer, font, cmd_text_buffer.String(), test_rand_color)

                        temp_w, temp_h := get_text_size(font, cmd_text_buffer.String())

                        curr_char_w, _ = get_text_size(font, input_char)

                        cmd_console_ttf_rect.W = int32(temp_w)
                        cmd_console_ttf_rect.H = int32(temp_h)

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
                                    cmd_console_ttf_texture = make_ttf_texture(renderer, font, temp_string, cmd_rand_color)

                                    temp_w, temp_h := get_text_size(font, cmd_text_buffer.String())

                                    if len(temp_string) != 0 {
                                        curr_char_w, _ = get_text_size(font, string(temp_string[len(temp_string)-1]))
                                        cmd_console_cursor_block.X -= int32(curr_char_w)

                                        cmd_console_ttf_rect.W = int32(temp_w)
                                        cmd_console_ttf_rect.H = int32(temp_h)

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
                                                cmd_console_ttf_texture = make_ttf_texture(renderer, font, temp_string, cmd_rand_color)

                                                temp_w, temp_h := get_text_size(font, cmd_text_buffer.String())

                                                if len(temp_string) != 0 {
                                                    curr_char_w, _ = get_text_size(font, string(temp_string[len(temp_string)-1]))
                                                    cmd_console_cursor_block.X -= int32(curr_char_w)

                                                    cmd_console_ttf_rect.W = int32(temp_w)
                                                    cmd_console_ttf_rect.H = int32(temp_h)

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
                        //move_text_up = true
                    }
                    if t.Keysym.Sym == sdl.K_DOWN {
                        //move_text_down = true
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
        //for ln := range all_lines[0:18] {
        for ln := range all_lines {
            for index := range all_lines[ln].word_rects {
                //renderer.SetDrawColor(100, 10, 100, uint8(cmd_console_anim_alpha))
                renderer.SetDrawColor(0, 0, 0, 0)
                renderer.FillRect(&all_lines[ln].word_rects[index])
                renderer.DrawRect(&all_lines[ln].word_rects[index])
            }
            renderer.Copy(all_lines[ln].texture.data, nil, &all_lines[ln].bg_rect)
        }

        // @HIGHLIGHT WORDS
        for index := range _RECTS_ {
            if mouseover_word_texture[index] {
                renderer.SetDrawColor(255, 100, 200, 100)
                renderer.FillRect(&_RECTS_[index])
                renderer.DrawRect(&_RECTS_[index])
                if print_word {
                    fmt.Printf("%s\n", _WORDS_[index])
                    print_word = false
                }
            } else {
                renderer.SetDrawColor(0, 0, 0, 0)
                renderer.FillRect(&_RECTS_[index])
                renderer.DrawRect(&_RECTS_[index])
            }
        }

        if wrap_line {
            for index := range all_lines {
                renderer.SetDrawColor(100, 255, 255, 100)
                renderer.FillRect(&all_lines[index].bg_rect)
                renderer.DrawRect(&all_lines[index].bg_rect)
                renderer.Copy(all_lines[index].texture.data, nil, &all_lines[index].bg_rect)
            }
        }
        // @TEST RENDERING TTF LINE

        // DRAWING_CMD_CONSOLE
        if show_cmd_console_rect {
            renderer.SetDrawColor(255, 10, 100, uint8(cmd_console_anim_alpha))
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
        }
        // DRAWING_CMD_CONSOLE

        renderer.SetDrawColor(255, 100, 0, uint8(cmd_console_anim_alpha))
        renderer.DrawLine(int32(LINE_LENGTH), 0, int32(LINE_LENGTH), WIN_H)

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
        <-ticker.C
    }

    font.Close()

    //for index := range ttf_textures {
    //    ttf_textures[index].Destroy()
    //}

    destroy_lines(&all_lines) // @WIP

    if cmd_console_ttf_texture != nil {
        println("The texture was not <nil>")
        cmd_console_ttf_texture.Destroy()
    } else {
        println("ERROR!!!! The texture was already Destroyed somewhere")
    }

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

func make_ttf_texture(renderer *sdl.Renderer, font *ttf.Font, text string, color sdl.Color) (*sdl.Texture) {
    var surface *sdl.Surface
    var texture *sdl.Texture
    var err error

	assert_if(len(text) <= 0, "text: len(text) <= 0")

    if surface, err = font.RenderUTF8Blended(text, color); err != nil {
        panic(err)
    }

    if texture, err = renderer.CreateTextureFromSurface(surface); err != nil {
        panic(err)
    }
    surface.Free()

    return texture
}

func reload_ttf_texture(r *sdl.Renderer, tex *sdl.Texture, f *ttf.Font, s string, c sdl.Color) (*sdl.Texture) {
    var surface *sdl.Surface
    var err error

    if tex != nil {
        tex.Destroy()
        if surface, err = f.RenderUTF8Blended(s, c); err != nil {
            panic(err)
        }

        if tex, err = r.CreateTextureFromSurface(surface); err != nil {
            panic(err)
        }
        surface.Free()
        return tex
    }
    return tex
}

func generate_and_populate_lines(renderer *sdl.Renderer, font *ttf.Font, tokens *[]string) (line []Line) {
    all_lines := make([]Line, len(*tokens))
    for index, tk := range *tokens {
        all_lines[index].text = tk
        new_ttf_texture_line(renderer, font, &all_lines[index], int32(index))
    }
    return all_lines
}

func get_text_size(font *ttf.Font, chars string) (int, int) {
    var err error
    line_w := 0
    line_h := 0

    line_w, line_h, err = font.SizeUTF8(chars)
    if err != nil {
        panic(err)
    }

    return line_w, line_h
}

// @TEMPORARY: this is just a wrapper at the moment
// NOTE: I'm not sure I like this function!!
func new_ttf_texture_line(rend *sdl.Renderer, font *ttf.Font, line *Line, skip_nr int32) {
    // TODO: I also have to handle cases like '\r' and such with length of 1
	assert_if(len(line.text) == 0, "line.text was empty")
	assert_if(font == nil, "font was nil")

    line.texture.data = make_ttf_texture(rend, font, line.text, line.color)

    text := strings.Split(line.text, " ")
    line.word_rects = make([]sdl.Rect, len(text))

    tw, th := get_text_size(font, line.text)
    line.texture.width = int32(tw)
    line.texture.height = int32(th)

    skipline := int32(font.LineSkip()) // @TEMPORARY HACK
    if (skip_nr > 0) {
        skipline *= skip_nr
    } else {
        skipline = 0
    }
    generate_new_line_rects(&line.word_rects, font, &text, skip_nr)
    line.bg_rect = sdl.Rect{0, skipline, line.texture.width, line.texture.height}
}

func generate_new_line_rects(rects *[]sdl.Rect, font *ttf.Font, tokens *[]string, skip_nr int32) {
    move_x  := 0
    move_y  := skip_nr
    //x_adder := 0
    //add_nl := false
    space_x, _ := get_text_size(font, " ")
    for index, str := range *tokens {
        ix, iy := get_text_size(font, str)
        //x_adder = move_x + ix
        //if (x_adder) > MAX_LINE_LEN {
        //    // TODO: MAX_LINE_LEN here should, in fact, be the global (window_size x and y)
        //    // I would have to set proper line positioning and with in order for it to work.
        //    // TODO: Since the line is bigger than allowed, we'll have to "wrap"
        //    // global_win_w, global_win_h
        //    //println("MAX_LINE_LEN diff:", MAX_LINE_LEN-(x_adder), (x_adder))
        //    move_y += font.LineSkip()
        //    move_x = 0
        //    add_nl = true
        //    //println(len(*tokens))
        //    //println((*tokens)[0:len(*tokens)-(-1 * (MAX_LINE_LEN - x_adder))])
        //    // create a new texture line here.
        //     //new_texture_line := make_ttf_texture(renderer, font, tokens, clor)
        //}
        if index == 0 {
            move_y *= int32(font.LineSkip())
        }
        (*rects)[index] = sdl.Rect{int32(move_x), int32(move_y), int32(ix), int32(iy)}
        move_x += (ix + space_x)
        //if !add_nl {
        //    move_x += (ix + space_x)
        //} else {
        //    add_nl = false
        //}
    }
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

func do_wrap_lines(font *ttf.Font, str *string, max_len int) ([]string) {
    var buff bytes.Buffer
    var result []string
    tokens := strings.Split(*str, " ")
    size_x, _ := get_text_size(font, " ")
    current_len := 0
    save_token := ""
    buffstr := ""

    assert_if(len(*str) <= 1, "assert: do_wrap_lines str size <= 1!\n")

    for index, _ := range tokens {
        if len(save_token) > 0 {
            buff.WriteString(save_token + " ")
            current_len = len(buff.String()) * size_x
            save_token = ""
        }
        if (current_len + (len(tokens[index]) * size_x) <= max_len) {
            buff.WriteString(tokens[index] + " ")
            current_len = len(buff.String()) * size_x
        } else {
            save_token = tokens[index]
            buffstr = buff.String()
            result = append(result, buffstr[0:len(buffstr)-1])
            buff.Reset()
            current_len = 0
        }
    }
    if len(buff.String()) > 0 {
        buffstr = buff.String()
        end := len(buffstr)-1
        cut := 0
        for string(buffstr[end]) == " " || string(buffstr[end]) == "\r" {
            end -= 1
            cut += 1
        }
        result = append(result, buffstr[0:len(buffstr)-cut])
        buff.Reset()
    }
    return result
}

func destroy_lines(lines *[]Line) {
    for index := range *lines {
        if ((*lines)[index]).texture.data == nil { // @TEMPORARY HACK
            break
        }
        if err := ((*lines)[index]).texture.data.Destroy(); err != nil {
            println(index)
            panic(err)
        }
    }
}

//NOTE: According to [go build -gcflags=-m main.go] this call has been inlined.
//NOTE: It would be great to check if inlining calls are actually any good or note.
func assert_if(cond bool, error_msg string) {
	if (cond) {
		println("")
		err := errors.New(error_msg)
		panic(err)
	}
}

func is_ascii_alpha(char string) bool {
    return ((char >= "A") && (char <= "z"))
}
