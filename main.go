package main

import (
    "os"
    "log"
    "fmt"
    "time"
    "flag"
    "bytes"
	"errors"
    "reflect"
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

type Line struct { // should probably add Texture struct here
    width int32
    height int32
    text string
    color sdl.Color
    texture Texture
    bg_rect sdl.Rect
    word_rects []sdl.Rect
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

    // ------ TEXT_MANIP

    file_stat, err := os.Stat("HP01.txt")
    if err != nil {
        panic(err)
    }

    file_size := file_stat.Size()

    file, err := os.Open("HP01.txt")
    if err != nil {
        panic(err)
    }
    defer file.Close() // NOTE: why are we deferring here?

    //TODO: do we even need make(...)? 
    // we need to use a buffer here, perhaps it would be better?
    // otherwise, according to golang, we are leaking this on the heap
    // I wonder if Buffer.Read() would help it somehow not to leak.
    file_data := make([]byte, file_size)

    file.Read(file_data)

    string_tokens := strings.Split(string(file_data), " ")

    fmt.Printf("number of tokens: %d\n", len(string_tokens))

    // ------ TEXT_MANIP END

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
		fmt.Printf("[debug] ~> %#v\n", allfonts[index])
		defer allfonts[index].data.Close() // @TEMPORARY HACK
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

    var ttf_textures []*sdl.Texture
    var ttf_texture_rects []*sdl.Rect // this should be [][]*sdl.Rect

    // TODO: @SLOW: Make it better!
    // I should try using a texture per line, instead of a texture
    // per word basis.
    ttf_textures, ttf_texture_rects = generate_and_populate_ttf_textures_and_rects(renderer, string_tokens, font)

    //var foobar_t []*sdl.Texture
    //var foobar_r []*sdl.Rect

    //// TODO: SUPER SLOOOOW! Make it bettter!
    //foobar_t, foobar_r = generate_all_textures(renderer, string_tokens, font)

    //fmt.Println(foobar_t[0], foobar_r[0])

    fmt.Printf("length is: %d, size is: %d\n", len(ttf_textures), reflect.TypeOf(ttf_textures).Size())

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

    // TEST RENDERING TTF LINE 
    MAX_LINE_LEN := 400 //actual: 413
    test_text := "This is some random boonkers text that we are dealing with."
    test_strings := strings.Split(test_text, " ")
    clr := sdl.Color{0, 0, 0, 0}
    test_line_texture := make_ttf_texture(renderer, font, test_text, clr)
    tx, ty := get_text_size(font, test_text)
    test_line_rect := sdl.Rect{0, 0, int32(tx), int32(ty)}
    defer test_line_texture.Destroy()

    // type Line test
    // ....
    line := Line{}
    line.text = "Another type Line struct for our testing purposes. That's all, folks"
    line.color = sdl.Color{0, 0, 0, 0}

    new_ttf_texture_line(renderer, font, &line)

    defer line.texture.data.Destroy()
    fmt.Printf("%#v\n", line)

    //newline := Line{}
	//new_ttf_texture_line(renderer, font, &newline)
    //fmt.Printf("%#v\n", newline)

    // ....
    // type Line test

    test_rects := make([]sdl.Rect, len(test_strings))
    test_mouse_over := make([]bool, len(test_strings))

    move_x  := 0
    move_y  := 0
    x_adder := 0
    add_nl := false
    space_x, _ := get_text_size(font, " ")
    for index, str := range test_strings {
        ix, iy := get_text_size(font, str)
        x_adder = move_x + ix
        if (x_adder) > MAX_LINE_LEN {
            // TODO: MAX_LINE_LEN here should, in fact, be the global (window_size x and y)
            // I would have to set proper line positioning and with in order for it to work.
            // TODO: Since the line is bigger than allowed, we'll have to "wrap"
            // global_win_w, global_win_h
            println("MAX_LINE_LEN diff:", MAX_LINE_LEN-(x_adder), (x_adder))
            move_y += font.LineSkip()
            move_x = 0
            add_nl = true
            println(len(test_text))
            println(test_text[0:len(test_text)-(-1 * (MAX_LINE_LEN - x_adder))])
            // create a new texture line here.
            // new_texture_line := make_ttf_texture(renderer, font, test_text, clor)
        }
        test_rects[index] = sdl.Rect{int32(move_x), int32(move_y), int32(ix), int32(iy)}
        if !add_nl {
            move_x += (ix + space_x)
        } else {
            add_nl = false
        }
    }
    // TEST RENDERING TTF LINE 

    renderer_info, err := renderer.GetInfo()
    if err != nil {
        panic(err)
    }

    fmt.Println(renderer_info)
    fmt.Println(sdl.GetPlatform())

    renderer.SetDrawBlendMode(sdl.BLENDMODE_BLEND)

    running := true
    print_word := false
    mouseover_word_texture := make([]bool, len(string_tokens))

    fmt.Println("FONT_FIXED_WIDTH: ", font.FaceIsFixedWidth())

    wrap_line := false

    // OFFSCREEN STUFF
    // -----------------
    move_text_up := false
    move_text_down := false
    // -----------------
    // OFFSCREEN STUFF
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
                        //case sdl.WINDOWEVENT_RESIZED: //NOTE: what is this event for?
                        case sdl.WINDOWEVENT_SIZE_CHANGED:
                            global_win_w, global_win_h = t.Data1, t.Data2
                            fmt.Printf("g_ww: %d, g_wh: %d, t.Data1: %d, t.Data2: %d\n",
                                            global_win_w, global_win_h, t.Data1, t.Data2)
                            fmt.Printf("tx: %d ty: %d\n", tx, ty)

                            if global_win_w <= int32(tx) {
                                println("We have to implement WRAP!")
                                wrap_line = true
                            } else {
                                wrap_line = false
                            }
                            break
                        default:
                            break
                    }
                    break
                case *sdl.MouseMotionEvent:
                    //fmt.Printf("~> %d %d\n", t.X, t.Y)
                    // TODO: @TEMPORARY @REFACTOR
                    for index := range ttf_textures {
                        mx_gt_rx :=    t.X > ttf_texture_rects[index].X
                        mx_lt_rx_rw := t.X < ttf_texture_rects[index].X + ttf_texture_rects[index].W
                        my_gt_ry :=    t.Y > ttf_texture_rects[index].Y
                        my_lt_ry_rh := t.Y < ttf_texture_rects[index].Y + ttf_texture_rects[index].H

                        if ((mx_gt_rx && mx_lt_rx_rw) && (my_gt_ry && my_lt_ry_rh)) {
                            mouseover_word_texture[index] = true
                        } else {
                            mouseover_word_texture[index] = false
                        }
                    }

                    // TODO: @TEMPORARY @REFACTOR
                    for index := range test_rects {
                        mx_gt_rx :=    t.X > test_rects[index].X
                        mx_lt_rx_rw := t.X < test_rects[index].X + test_rects[index].W
                        my_gt_ry :=    t.Y > test_rects[index].Y
                        my_lt_ry_rh := t.Y < test_rects[index].Y + test_rects[index].H

                        if ((mx_gt_rx && mx_lt_rx_rw) && (my_gt_ry && my_lt_ry_rh)) {
                            test_mouse_over[index] = true
                        } else {
                            test_mouse_over[index] = false
                        }
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

        // TODO: this works, but we still have to make sure we can do mouse interaction
        // NOTE: for some reason our text is being rendered as bold
        // @TEST RENDERING TTF LINE

        renderer.SetDrawColor(255, 10, 100, uint8(cmd_console_anim_alpha))
        renderer.FillRect(&line.bg_rect)
        renderer.DrawRect(&line.bg_rect)
        renderer.Copy(line.texture.data, nil, &line.bg_rect)

        for index := range test_rects {
            if test_mouse_over[index] {
                renderer.SetDrawColor(255, 10, 100, uint8(cmd_console_anim_alpha))
                renderer.FillRect(&test_rects[index])
                renderer.DrawRect(&test_rects[index])
                if index == 0 {
                    renderer.Copy(test_line_texture, nil, &test_line_rect)
                }
            } else {
                if index == 0 {
                    renderer.SetDrawColor(255, 255, 255, 0)
                    renderer.FillRect(&test_rects[index])
                    renderer.DrawRect(&test_rects[index])
                    renderer.Copy(test_line_texture, nil, &test_line_rect)
                }
            }
        }

        if wrap_line {
            renderer.SetDrawColor(100, 255, 255, 100)
            renderer.FillRect(&test_line_rect)
            renderer.DrawRect(&test_line_rect)
            renderer.Copy(test_line_texture, nil, &test_line_rect)
        }
        // @TEST RENDERING TTF LINE

        if move_text_down {
            move_text_down = false
            for index := range ttf_textures {
                ttf_texture_rects[index].Y += TEXT_SCROLL_SPEED
            }
        }

        if move_text_up {
            move_text_up = false
            for index := range ttf_textures {
                ttf_texture_rects[index].Y -= TEXT_SCROLL_SPEED
            }
        }

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

        //DRAW TEXT
        for index := range ttf_textures {
            if mouseover_word_texture[index] {
                // TODO: we are not animating anymore
                renderer.SetDrawColor(255, 0, 255, 100)
                renderer.FillRect(ttf_texture_rects[index])
                renderer.DrawRect(ttf_texture_rects[index])
                renderer.Copy(ttf_textures[index], nil, ttf_texture_rects[index])

                // NOTE
                // using strings.TrimSpace just for debugging purposes
                // there is no string_tokens array that would be cleaned up of all spaces

                if print_word {
                    fmt.Printf("%d, %#v\n", index, strings.TrimSpace(string_tokens[index]))
                    print_word = false
                }

                // NOTE
                // should we break out of the loop
                // and check if the index is the same as prev. index?
                // so that we don't render over and over again???
            } else {
                renderer.SetDrawColor(255, 255, 255, 0)
                renderer.FillRect(ttf_texture_rects[index])
                renderer.DrawRect(ttf_texture_rects[index])
                renderer.Copy(ttf_textures[index], nil, ttf_texture_rects[index])
            }
        }

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

    for index := range ttf_textures {
        ttf_textures[index].Destroy()
    }

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

    // NOTE: @TEMPORARY HACK
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

// TODO: @TEMPORARY: this is just testing things out. Eventually this will have to go!
func generate_all_textures(r *sdl.Renderer, string_tokens []string, font *ttf.Font) ([]*sdl.Texture, []*sdl.Rect) {

    ttf_index := 0
    ttf_textures := make([]*sdl.Texture, len(string_tokens))
    ttf_texture_rects := make([]*sdl.Rect, len(string_tokens))

    temp_x := 0
    temp_y := 0
    add_new_line := false
    already_added_new_line := false
    esc_seq_map := map[string]int{"nl": 0, "tab": 0, "vtab": 0, "cret": 0}

    for _, element := range string_tokens {
        var solid_ttf_texture *sdl.Texture

        esc_seq_map["nl"] = strings.Count(element, "\n")

        if strings.Count(element, "\n") > 0 {
            element = strings.TrimSpace(element)
        }

        color := sdl.Color{0,0,0,255}

        // -----------------------------
        // GUARDS!!! GUARDS!!! GUARDS!!!
        // -----------------------------
        if temp_x < int(WIN_W-MAX_TEXT_WIDTH) {
            add_new_line = false
            already_added_new_line = false
        }
        // -----------------------------
        // GUARDS!!! GUARDS!!! GUARDS!!!
        // -----------------------------

        solid_ttf_texture = make_ttf_texture(r, font, element, color)

        //ttf_textures = append(ttf_textures, solid_ttf_texture)
        ttf_textures[ttf_index] = solid_ttf_texture

        ttf_w, ttf_h := get_text_size(font, element)
        font_line_skip := font.LineSkip()

        // TODO: we also need to handle tabs, vtabs and carriage return... maybe others as well?
        if esc_seq_map["nl"] > 0 {
            for i := 0; i < esc_seq_map["nl"]; i++ {
                temp_y += font_line_skip
            }
            temp_x = 0
            esc_seq_map["nl"] = 0
            already_added_new_line = true
        }

        if add_new_line {
            temp_y += font_line_skip
            temp_x = 0
            add_new_line = false
        }

        //ttf_texture_rects = append(ttf_texture_rects, &sdl.Rect{int32(temp_x), int32(temp_y), int32(ttf_w), int32(ttf_h)})
        ttf_texture_rects[ttf_index] = &sdl.Rect{int32(temp_x), int32(temp_y), int32(ttf_w), int32(ttf_h)}

        if temp_x >= int(WIN_W-MAX_TEXT_WIDTH) {
            temp_y += font_line_skip
            temp_x = 0
        } else {
            temp_x += (ttf_w + 4)
        }

        ttf_index += 1 // NEXT_ELEMENT

        if already_added_new_line {
            add_new_line = false
            already_added_new_line = false
        } else {
            add_new_line = true
        }
    }
    return ttf_textures, ttf_texture_rects
}

func generate_and_populate_ttf_textures_and_rects(r *sdl.Renderer, string_tokens []string, font *ttf.Font) ([]*sdl.Texture, []*sdl.Rect) {
    var ttf_textures []*sdl.Texture
    var ttf_texture_rects []*sdl.Rect

    temp_x := 0
    temp_y := 0
    add_new_line := false
    already_added_new_line := false
    start_index := 0
    end_index := MAX_TOKENS
    esc_seq_map := map[string]int{"nl": 0, "tab": 0, "vtab": 0, "cret": 0}

    for newline := 0; newline < MAX_LINES; newline++ {
        for _, element := range string_tokens[start_index:end_index] {
            var solid_ttf_texture *sdl.Texture

            esc_seq_map["nl"] = strings.Count(element, "\n")

            if strings.Count(element, "\n") > 0 {
                element = strings.TrimSpace(element)
            }

            color := sdl.Color{0,0,0,255}

            // -----------------------------
            // GUARDS!!! GUARDS!!! GUARDS!!!
            // -----------------------------
            if temp_x < int(WIN_W-MAX_TEXT_WIDTH) {
                add_new_line = false
                already_added_new_line = false
            }
            // -----------------------------
            // GUARDS!!! GUARDS!!! GUARDS!!!
            // -----------------------------

            solid_ttf_texture = make_ttf_texture(r, font, element, color)

            ttf_textures = append(ttf_textures, solid_ttf_texture)

            ttf_w, ttf_h := get_text_size(font, element)
            font_line_skip := font.LineSkip()

            // TODO: we also need to handle tabs, vtabs and carriage return... maybe others as well?
            if esc_seq_map["nl"] > 0 {
                for i := 0; i < esc_seq_map["nl"]; i++ {
                    temp_y += font_line_skip
                }
                temp_x = 0
                esc_seq_map["nl"] = 0
                already_added_new_line = true
            }

            if add_new_line {
                temp_y += font_line_skip
                temp_x = 0
                add_new_line = false
            }

            ttf_texture_rects = append(ttf_texture_rects, &sdl.Rect{int32(temp_x), int32(temp_y), int32(ttf_w), int32(ttf_h)})
            if temp_x >= int(WIN_W-MAX_TEXT_WIDTH) {
                temp_y += font_line_skip
                temp_x = 0
            } else {
                temp_x += (ttf_w + 4)
            }
        }

        if already_added_new_line {
            add_new_line = false
            already_added_new_line = false
        } else {
            add_new_line = true
        }
        start_index += MAX_TOKENS
        end_index   += MAX_TOKENS
    }
    return ttf_textures, ttf_texture_rects
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
func new_ttf_texture_line(rend *sdl.Renderer, font *ttf.Font, line *Line) {
	assert_if(len(line.text) == 0, "line.text was empty")
	assert_if(font == nil, "font was nil")

    line.texture.data = make_ttf_texture(rend, font, line.text, line.color)

    line.word_rects = make([]sdl.Rect, len(strings.Split(line.text, " ")))

    tw, th := get_text_size(font, line.text)
    line.texture.width = int32(tw)
    line.texture.height = int32(th)

    skipline := int32(font.LineSkip()) // @TEMPORARY
    line.bg_rect = sdl.Rect{0, skipline, line.texture.width, line.texture.height}
}

func assert_if(cond bool, error_msg string) {
	if (cond) {
		println("")
		err := errors.New(error_msg)
		panic(err)
	}
}
