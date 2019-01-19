package main

import (
    "os"
    "log"
    "fmt"
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

const WIN_TITLE string = "GO_TEXT_APPLICATION"

const WIN_W int32 = 800
const WIN_H int32 = 600

const X_OFFSET int = 7
const TTF_FONT_SIZE int = 16
const TEXT_SCROLL_SPEED int32 = 14
const LINE_LENGTH int = 640

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to 'file'")
var memprofile = flag.String("memprofile", "", "write mem profile to 'file'")

// @GLOBAL MUT VARS 
var global_win_w int32
var global_win_h int32

//TODO: https://syslog.ravelin.com/bytes-buffer-i-thought-you-were-my-friend-4148fd001229 
//TODO: https://syslog.ravelin.com/bytes-buffer-revisited-edee5a882030

//TODO: https://www.ardanlabs.com/blog/2013/09/iterating-over-slices-in-go.html 
//TODO: https://garbagecollected.org/2017/02/22/go-range-loop-internals/ 

//TODO: https://stackoverflow.com/questions/28432658/does-go-garbage-collect-parts-of-slices
//TODO: https://appliedgo.net/slices/ 

//TODO: https://www.ardanlabs.com/blog/2017/05/language-mechanics-on-escape-analysis.html 
//TODO: https://www.ardanlabs.com/blog/2017/05/language-mechanics-on-stacks-and-pointers.html 

//TODO: https://divan.github.io/posts/avoid_gotchas/ 

//TODO: http://devs.cloudimmunity.com/gotchas-and-common-mistakes-in-go-golang/index.html#slice_hidden_data 

//TODO: https://golang.org/pkg/sync/#Pool
// A Pool is a set of temporary objects that may be individually saved and retrieved.
// Any item stored in the Pool may be removed automatically at any time without notification.
// If the Pool holds the only reference when this happens, the item might be deallocated.
// A Pool is safe for use by multiple goroutines simultaneously.
// 
// Pool's purpose is to cache allocated but unused items for later reuse, relieving pressure on 
// the garbage collector. That is, it makes it easy to build efficient, thread-safe free lists. 
// However, it is not suitable for all free lists. 

type Font struct {
    size int
    name string
    data *ttf.Font
}

type Line struct {
    text string // DELETE
    texture *sdl.Texture
    bg_rect sdl.Rect
    word_rects []sdl.Rect  //DELETE
}

type DebugWrapLine struct {
    x1, y1 int32
    x2, y2 int32
    clicked bool
}

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

	// Shouldn't init everything!
    if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
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

    line_tokens := strings.Split(string(file_data), "\n")

    file_data = nil

    ticker := time.NewTicker(time.Second / 60)

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

    file_names = nil

    allfonts := make([]Font, len(ttf_font_list))

    //fmt.Println(ttf_font_list)

    font = load_font("Inconsolata-Regular.ttf", TTF_FONT_SIZE)

	// NOTE: maybe I should font = all_fonts[...]
	// and just interate over font = all_fonts[...]
	// so that I don't have to do extra allocations
	// basically we would keep them all in memory at all times

	for index, element := range ttf_font_list {
		allfonts[index].data = load_font(element, TTF_FONT_SIZE)
		allfonts[index].name = element
		allfonts[index].size = TTF_FONT_SIZE
	}

    CHAR_W, CHAR_H, _ := font.SizeUTF8(" ")
    SKIP_LINE := font.LineSkip()
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


    // @TEMPORARY
	// do_wrap_lines should return []*strings
	// we should append(test_tokens, &element) that way we won't copy elements over and over again.
    start := time.Now()
    //x_size, _ := get_text_size(font, " ")
    MAX_INDEX := 40
    test_tokens := do_wrap_lines(&line_tokens[0], LINE_LENGTH, CHAR_W)
    for index := 1; index < len(line_tokens); index += 1 {
        if (len(line_tokens[index]) > 1) {
            current := do_wrap_lines(&line_tokens[index], LINE_LENGTH, CHAR_W)
			// current and element are both copies, so we end up copying multiple times for no reason
            for _, element := range current {
                test_tokens = append(test_tokens, element)
            }
        } else {
            test_tokens = append(test_tokens, "\n")
        }
    }
    end_start := time.Now().Sub(start)
    fmt.Printf("[[do_wrap_lines loop took %s]]\n", end_start.String())


    now_gen := time.Now()
	//@PERFORMANCE SLOW
    all_lines := generate_and_populate_lines(renderer, font, &test_tokens, CHAR_W, CHAR_H, SKIP_LINE)
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

    cmd_console_ttf_texture = make_ttf_texture(renderer, font, cmd_console_test_str, cmd_rand_color)

    //cmd_console_ttf_w, cmd_console_ttf_h := get_text_size(font, cmd_console_test_str)

    //ttf_letter_w, ttf_letter_h := get_text_size(font, "A") // "A" is just a random letter for our usecase

    cmd_console_ttf_rect     := sdl.Rect{0, WIN_H-cmd_win_h, int32(CHAR_W * len(cmd_console_test_str)), int32(CHAR_H)}
    cmd_console_rect         := sdl.Rect{0, WIN_H-cmd_win_h, WIN_W, int32(CHAR_H)}
    cmd_console_cursor_block := sdl.Rect{0, WIN_H-cmd_win_h, int32(CHAR_W), int32(CHAR_H)}

    //////////////////////////
    // END_CMD_CONSOLE_STUFF
    //////////////////////////

    renderer.SetDrawBlendMode(sdl.BLENDMODE_BLEND)

    running := true
    first_pass := true
    print_word := false
    engage_loop := false


    total := 0
    for index := range all_lines[0:MAX_INDEX] {
        total += len(all_lines[index].word_rects)
    }

    mouseover_word_texture := make([]bool, total)

    _RECTS_ := make([]sdl.Rect, 0)
    for index := range all_lines[0:MAX_INDEX] {
        for _, rct := range all_lines[index].word_rects {
            _RECTS_ = append(_RECTS_, rct)
        }
    }

    _WORDS_ := make([]string, 0)
    for index := range all_lines[0:MAX_INDEX] {
        for _, rct := range strings.Split(all_lines[index].text, " ") {
            _WORDS_ = append(_WORDS_, rct)
        }
    }

    wrap_line := false

    move_text_up := false
    move_text_down := false

    test_rand_color := sdl.Color{uint8(rand.Intn(255)),uint8(rand.Intn(255)),uint8(rand.Intn(255)),uint8(rand.Intn(255))}

    curr_char_w := 0

    wrapline := DebugWrapLine{int32(LINE_LENGTH), 0, int32(LINE_LENGTH), WIN_H, false}

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
                        cmd_console_ttf_texture = make_ttf_texture(renderer, font, cmd_text_buffer.String(), test_rand_color)

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
                                        cmd_console_ttf_texture = make_ttf_texture(renderer, font, temp_string, cmd_rand_color)
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
                                                    cmd_console_ttf_texture = make_ttf_texture(renderer, font, temp_string, cmd_rand_color)
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
            for ln := range all_lines[0:MAX_INDEX] {
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
            for i := range all_lines[0:MAX_INDEX] {
                renderer.Copy(all_lines[i].texture, nil, &all_lines[i].bg_rect)
            }
        }

        for i := range mouseover_word_texture {
            if mouseover_word_texture[i] == true {
                engage_loop = true
            }
        }
        if engage_loop {
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
            for index := range all_lines[0:MAX_INDEX] {
                all_lines[index].bg_rect.Y -= TEXT_SCROLL_SPEED
            }
            for index := range _RECTS_ {
                _RECTS_[index].Y -= TEXT_SCROLL_SPEED
            }
        }
        if move_text_up {
            move_text_up = false
            for index := range all_lines[0:MAX_INDEX] {
                all_lines[index].bg_rect.Y += TEXT_SCROLL_SPEED
            }
            for index := range _RECTS_ {
                _RECTS_[index].Y += TEXT_SCROLL_SPEED
            }
        }

        if wrap_line {
            for index := range all_lines[0:MAX_INDEX] {
                renderer.SetDrawColor(100, 255, 255, 100)
                renderer.FillRect(&all_lines[index].bg_rect)
                renderer.DrawRect(&all_lines[index].bg_rect)
                renderer.Copy(all_lines[index].texture, nil, &all_lines[index].bg_rect)
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
        }
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
        println("The texture was not <nil>")
        cmd_console_ttf_texture.Destroy()
        cmd_console_ttf_texture = nil
    }

	for index := range ttf_font_list {
		allfonts[index].data.Close() // @TEMPORARY HACK @SLOW
        allfonts[index].data = nil
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

func make_ttf_texture(renderer *sdl.Renderer, font *ttf.Font, text string, color sdl.Color) (*sdl.Texture) {
    var surface *sdl.Surface
    var texture *sdl.Texture
    var err error

	assert_if(len(text) <= 0)

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

func generate_and_populate_lines(r *sdl.Renderer, font *ttf.Font, tokens *[]string, x int, y int, skipline int) (line []Line) {
    all_lines := make([]Line, len(*tokens))
    for index, tk := range *tokens {
        all_lines[index].text = tk //TODO: saving .text in unnecessary here. Need to find a better way...
        new_ttf_texture_line(r, font, &all_lines[index], int32(index), x, y, skipline)
    }
    return all_lines
}

// we should probably only call font.SizeUTF8 a couple of times for w and h
// then we can calculate whatever we need as w = len(char) * char_w; we can alos use ttf.Height()
// 
//func get_text_size(font *ttf.Font, chars string) (int, int) {
//    //var err error
//    //line_w := 0
//    //line_h := 0
//
//    line_w, line_h, _ := font.SizeUTF8(chars)
//    //if err != nil {
//    //    panic(err)
//    //}
//
//    return line_w, line_h
//}

// TODO: we are Spliting too much everywhere

// @TEMPORARY: this is just a wrapper at the moment
// NOTE: I'm not sure I like this function!!
func new_ttf_texture_line(rend *sdl.Renderer, font *ttf.Font, line *Line, skip_nr int32, x int, y int, lineskip int) {
    // TODO: I also have to handle cases like '\r' and such with length of 1
	assert_if(len(line.text) == 0)
	assert_if(font == nil)

    line.texture = make_ttf_texture(rend, font, line.text, sdl.Color{0, 0, 0, 0})

    text := strings.Split(line.text, " ")
    line.word_rects = make([]sdl.Rect, len(text))

    //tw, th := get_text_size(font, line.text)
    tw := x * len(line.text)

    skipline := int32(lineskip) // @TEMPORARY HACK
    if (skip_nr > 0) {
        skipline *= skip_nr
    } else {
        skipline = 0
    }
    generate_new_line_rects(&line.word_rects, font, &text, skip_nr, x, y, lineskip)
    line.bg_rect = sdl.Rect{int32(X_OFFSET), skipline, int32(tw), int32(y)}
    text = nil
}

func generate_new_line_rects(rects *[]sdl.Rect, font *ttf.Font, tokens *[]string, skip_nr int32, x int, y int, lineskip int) {
    move_x  := X_OFFSET
    move_y  := skip_nr
    space_x := x
    ix := 0
    for index, str := range *tokens {
        ix = x * len(str)
        if index == 0 {
            move_y *= int32(lineskip)
        }
        (*rects)[index] = sdl.Rect{int32(move_x), int32(move_y), int32(ix), int32(y)}
        move_x += (ix + space_x)
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
        }
    }
}

// TODO: @PERFORMANCE: apparently bytes.Buffer is slow
//@SPEED this is slow. Use strings.Builder{} instead, or something else.
// https://habr.com/en/company/intel/blog/422447/ 
func do_wrap_lines(str *string, max_len int, xsize int) ([]string) {
    var buff strings.Builder
    var result []string
    tokens := strings.Split(*str, " ")
    size_x := xsize
    current_len := 0
    save_token := ""
    buffstr := ""

    //buff.Grow((len(tokens) * size_x) + X_OFFSET)

    // 1) split string into word_sized tokens
    // 2) loop through each word token
    // 3) if save_token is not empty we write
    // 4) if ...

    assert_if(len(*str) <= 1)

    for index := range tokens {
        if len(save_token) > 0 {
            buff.WriteString(save_token + " ")
            current_len = len(buff.String()) * size_x
            save_token = ""
        }
        if (current_len + (len(tokens[index]) * size_x)+X_OFFSET <= max_len) {
            buff.WriteString(tokens[index] + " ")
            current_len = (len(buff.String()) * size_x)+X_OFFSET
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
