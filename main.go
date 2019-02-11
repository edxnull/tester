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

// TODO: rendering glyphs https://www.libsdl.org/projects/SDL_ttf/docs/SDL_ttf_38.html
// 		 should we just render some glyps onto a texture and just write them to a texture instead of rendering 1 texture per line?
//       https://www.libsdl.org/projects/SDL_ttf/docs/SDL_ttf_46.html#SEC46

// TODO(maybe): compare: rendering multiple lines per texture
// TODO(maybe): compare: rendering lines with glyphs
// TODO(maybe): compare: rendering lines like we do right now

// [ ] cleanup the code!
// [ ] try to optimize rendering/displaying rects with "enum" flags ~> [TypeActive; TypeInactive; TypePending]
// [ ] add equations of motion for nice animation effects https://easings.net/ 
// [ ] bezier curve easing functions
// [ ] imgui menu
// [ ] searching
// [ ] fuzzy search
// [ ] copy & pasting text
// [ ] copy & pasting commands
// [ ] save words to a trie tree?
// [ ] figure out what to do about languages like left to right and asian languages
// [ ] export/import csv
// [ ] smooth scrolling
// [ ] experiment with imgui
// [ ] grapical popup error messages like: error => your command is too long, etc...
// [ ] fix wrapping text
// [ ] make sure we handle utf8
// [ ] compare method call vs. function call overhead in golang: asm?
// [ ] cmd input commands + parsing

// NOTE: both of these would be easier if we wouldn't have to render the whole text at a time
// ---------------------------------------
// [ ] selecting and reloading text
// [ ] changing font size
// [ ] selecting and reloading fonts
// ---------------------------------------

const WIN_TITLE string = "GO_TEXT_APPLICATION"

const WIN_W int32 = 800
const WIN_H int32 = 600

const X_OFFSET int = 7
const TTF_FONT_SIZE int = 18
const TTF_FONT_SIZE_FOR_FONT_LIST int = 14
const LINE_LENGTH int = 730

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to 'file'")
var memprofile = flag.String("memprofile", "", "write mem profile to 'file'")

var MAX_INDEX int = 40
var START_INDEX int = 0

type RctStat uint8
const (
    TypeActive RctStat = 0
    TypePending RctStat = 1
    TypeInactive RctStat = 2
)

type v2 struct {
    x float32
    y float32
}

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
    word_rects []sdl.Rect
    slice []Line
}

type DebugWrapLine struct {
    x1, y1 int32
    x2, y2 int32
}

type CmdConsole struct {
    show bool
    move_left bool
    alpha_value uint8
    bg_rect sdl.Rect
    ttf_rect sdl.Rect
    cursor_rect sdl.Rect
    ttf_texture *sdl.Texture
    input_buffer bytes.Buffer
}

type FontSelector struct {
    show bool
    fonts []Font
    current_font *ttf.Font
    current_font_w int
    current_font_h int
    current_font_skip int
    alpha_value uint8
    alpha_f32 float32
    bg_rect sdl.Rect
    ttf_rects []sdl.Rect
    highlight_rect []sdl.Rect
    cursor_rect sdl.Rect
    textures []*sdl.Texture
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
                                               sdl.WINDOW_SHOWN | sdl.WINDOW_RESIZABLE)
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

    filename := "HP01.txt"
    font_dir := "./fonts/"
    text_dir := "./text/"

    line_tokens := strings.Split(string(get_filedata(text_dir, filename)), "\n")

    ticker := time.NewTicker(time.Second / 60)

    var gfonts FontSelector = FontSelector{}

    ttf_font_list := get_filenames(font_dir, []string{"ttf", "otf"})
    txt_list := get_filenames(text_dir, []string{".txt"})
    fmt.Println(txt_list)

    allocate_font_space(&gfonts, len(ttf_font_list))
    generate_fonts(&gfonts, ttf_font_list, font_dir)

    font := gfonts.current_font

    generate_rects_for_fonts(renderer, &gfonts)

    // NOTE: should we keep fonts in memory? or free them instead?

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
    //generate_and_populate_lines(renderer, font, &all_lines, &test_tokens)

    LESS := START_INDEX
    MORE := MAX_INDEX

    INC := 2
    generate_lines(renderer, font, &all_lines, &test_tokens, MAX_INDEX+1)
    generate_lines(renderer, font, &all_lines, &test_tokens, (MAX_INDEX+1)*INC)

    __SLICE__ := all_lines[LESS:MORE]

    end_gen := time.Now().Sub(now_gen)
    fmt.Printf("[[generate_and_populate_lines took %s]]\n", end_gen.String())

    cmd_win_h := int32(18)
    cmd := CmdConsole{}
    cmd.alpha_value = 100
    cmd.ttf_texture = make_ttf_texture(renderer, font, " ", &sdl.Color{0, 0, 0, 255})
    cmd.ttf_rect    = sdl.Rect{0, WIN_H-cmd_win_h, int32(gfonts.current_font_w * len(" ")), int32(gfonts.current_font_h)}
    cmd.bg_rect     = sdl.Rect{0, WIN_H-cmd_win_h, WIN_W, int32(gfonts.current_font_h)}
    cmd.cursor_rect = sdl.Rect{0, WIN_H-cmd_win_h, int32(gfonts.current_font_w), int32(gfonts.current_font_h)}
    cmd.input_buffer.Grow(128) // we need to make sure we never write past this value?

    dbg_str := make_console_text(0, len(test_tokens))
    dbg_rect := sdl.Rect{0, WIN_H-cmd_win_h-cmd_win_h-2, int32(gfonts.current_font_w * len(dbg_str)), int32(gfonts.current_font_h)}
    dbg_ttf := make_ttf_texture(renderer, gfonts.current_font, dbg_str, &sdl.Color{0, 0, 0, 255})

    sdl.SetHint(sdl.HINT_FRAMEBUFFER_ACCELERATION, "1")
    sdl.SetHint(sdl.HINT_RENDER_SCALE_QUALITY, "1")

    renderer.SetDrawBlendMode(sdl.BLENDMODE_BLEND)

    running := true
    print_word := false
    engage_loop := false
    add_new_line := false
    del_new_line := false
    dirty_hack := true

    nlines := 0
    num_word_textures := 0
    for index := 0; index < len(all_lines); index++ {
        if all_lines[index].texture != nil {
            num_word_textures += len(all_lines[index].word_rects)
            nlines += 1
        } else {
            break
        }
    }

    mouseover_word_texture := make([]bool, num_word_textures)
    mouseover_word_texture_FONT := make([]bool, len(ttf_font_list))

    println(len(all_lines), num_word_textures, nlines)

    _RECTS_ := make([]sdl.Rect, num_word_textures)
    for index, apos := 0, 0; index < nlines; index++ {
        for pos := 0; pos < len(all_lines[index].word_rects); pos++ {
            _RECTS_[apos] = all_lines[index].word_rects[pos]
            apos += 1
        }
    }

    _WORDS_ := make([]string, num_word_textures)
    for index, apos := 0, 0; index < nlines; index++ {
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
    TEXT_SCROLL_SPEED := int32(all_lines[0].bg_rect.H)

    location := v2{0, 0}
    test_rectq := sdl.Rect{int32(location.x), int32(location.y), 10, 10}

    for running {
        for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
            switch t := event.(type) {
                case *sdl.QuitEvent:
                    running = false
                    break
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
                                //viewport_rect.H = new_win_h
                                //viewport_rect.W = WIN_W

                                cmd.bg_rect.W = new_win_w
                                cmd.bg_rect.Y = new_win_h-cmd_win_h
                                cmd.ttf_rect.Y = new_win_h-cmd_win_h
                                cmd.cursor_rect.Y = new_win_h-cmd_win_h

                                wrapline.y2 = new_win_h
                                //renderer.SetViewport(&viewport_rect)
                            } else {
                                //viewport_rect.W = WIN_W
                                //viewport_rect.H = WIN_H

                                cmd.bg_rect.W = WIN_W
                                cmd.bg_rect.Y = new_win_h-cmd_win_h
                                cmd.ttf_rect.Y = new_win_h-cmd_win_h
                                cmd.cursor_rect.Y = new_win_h-cmd_win_h

                                wrapline.y2 = new_win_h
                                //renderer.SetViewport(&viewport_rect)
                            }
                            break
                        default:
                            break
                    }
                    break
                case *sdl.MouseMotionEvent:
                    check_collision_mouse_over_words(t, &_RECTS_, &mouseover_word_texture)
                    check_collision_mouse_over_words(t, &gfonts.ttf_rects, &mouseover_word_texture_FONT)
                    break
                case *sdl.MouseWheelEvent:
                    switch t.Y {
                        case 1:
                            move_text_up = true
                            break
                        case -1:
                            move_text_down = true
                            break
                        default:
                            break
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
                    if cmd.show {
                        if t.Keysym.Sym == sdl.K_BACKSPACE {
                            if t.Repeat > 0 {
                                execute_cmd_write_to_buffer(renderer, &cmd, curr_char_w, gfonts.current_font, gfonts.current_font_w,
                                                                                                              gfonts.current_font_h)
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
                                            break
                                    case sdl.K_BACKSPACE:
                                        execute_cmd_write_to_buffer(renderer, &cmd, curr_char_w, gfonts.current_font, gfonts.current_font_w,
                                                                                                                      gfonts.current_font_h)
                                        break
                                    case sdl.K_RETURN:
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
                                    case sdl.K_UP:
                                        move_text_up = true
                                        break
                                    case sdl.K_DOWN:
                                        move_text_down = true
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
        //for i := range all_lines[START_INDEX:MAX_INDEX] {
        //    renderer.Copy(all_lines[i].texture, nil, &all_lines[i].bg_rect)
        //}

        for i := range __SLICE__ {
            renderer.Copy(__SLICE__[i].texture, nil, &__SLICE__[i].bg_rect)
        }

        for i := range mouseover_word_texture {
            if mouseover_word_texture[i] == true {
                engage_loop = true
            }
        }

        if engage_loop && !cmd.show {
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

        if add_new_line {
            LESS += 1
            MORE += 1
            __SLICE__ = all_lines[LESS:MORE]
            MAX_INDEX = MAX_INDEX + 1
            all_lines[MAX_INDEX].bg_rect.Y = all_lines[MAX_INDEX-1].bg_rect.Y + (all_lines[MAX_INDEX].bg_rect.H - TEXT_SCROLL_SPEED)
            all_lines[MAX_INDEX-1].bg_rect.Y -= TEXT_SCROLL_SPEED

            rect_count := 0 // NOTE: This is a dirty HACK
            for i := range all_lines[START_INDEX:MAX_INDEX] {
                rect_count += len(all_lines[i].word_rects)
            }
            for i := rect_count; i < len(_RECTS_); i++ {
                _RECTS_[i].Y += 1
            }

            // TEMP HACK
            dbg_str = make_console_text(MAX_INDEX, len(test_tokens))
            dbg_ttf = reload_ttf_texture(renderer, dbg_ttf, font, dbg_str, &sdl.Color{0, 0, 0, 255})

            add_new_line = false
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

        if del_new_line {
            LESS -= 1
            MORE -= 1
            __SLICE__ = all_lines[LESS:MORE]

            rect_count := 0 // NOTE: This is a dirty HACK
            for i := range all_lines[START_INDEX:MAX_INDEX] {
                rect_count += len(all_lines[i].word_rects)
            }
            for i := rect_count; i < len(_RECTS_); i++ {
                _RECTS_[i].Y -= 1
            }

            MAX_INDEX = MAX_INDEX - 1
            // TEMP HACK
            dbg_str = make_console_text(MAX_INDEX, len(test_tokens))
            dbg_ttf = reload_ttf_texture(renderer, dbg_ttf, font, dbg_str, &sdl.Color{0, 0, 0, 255})

            del_new_line = false
        }

        if wrap_line {
            for i := 0 ; i < len(all_lines[START_INDEX:MAX_INDEX]); i++ {
                draw_rect_without_border(renderer, &all_lines[i].bg_rect, &sdl.Color{100, 255, 255, 100})
            }
        }

        if cmd.show {
            draw_rect_with_border_filled(renderer, &cmd.bg_rect, &sdl.Color{255, 10, 100, cmd.alpha_value})
            draw_rect_with_border(renderer, &cmd.ttf_rect, &sdl.Color{255, 255, 255, 0})

            renderer.Copy(cmd.ttf_texture, nil, &cmd.ttf_rect)

            draw_rect_with_border_filled(renderer, &cmd.cursor_rect, &sdl.Color{0, 0, 0, cmd.alpha_value})

            draw_rect_without_border(renderer, &gfonts.bg_rect, &sdl.Color{255, 0, 255, uint8(gfonts.alpha_f32)})

            for i := 0; i < len(gfonts.textures); i++ {
                renderer.Copy(gfonts.textures[i], nil, &gfonts.ttf_rects[i])
                if (mouseover_word_texture_FONT[i] == true) {
                    draw_rect_without_border(renderer, &gfonts.highlight_rect[i], &sdl.Color{0, 0, 0, 100})
                }
            }

            if dirty_hack { // A DIRTY HACK
                dbg_str = make_console_text(MAX_INDEX, len(test_tokens))
                dbg_ttf = reload_ttf_texture(renderer, dbg_ttf, font, dbg_str, &sdl.Color{0, 0, 0, 255})
                dirty_hack = false
            }

            draw_rect_with_border_filled(renderer, &dbg_rect, &sdl.Color{180, 123, 55, 255})
            renderer.Copy(dbg_ttf, nil, &dbg_rect)

            test_rectq.X = int32(location.x)
            test_rectq.Y = int32(location.y)
            draw_rect_without_border(renderer, &test_rectq, &sdl.Color{55, 100, 155, 255})
            if location.x < 100-1 {
                location.x = lerp(location.x, 100.0, 0.05)
            }
            if gfonts.alpha_f32 < 255-1 {
                gfonts.alpha_f32 = lerp(gfonts.alpha_f32, 255.0, 0.123)
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

func reload_ttf_texture(r *sdl.Renderer, tex *sdl.Texture, f *ttf.Font, s string, c *sdl.Color) (*sdl.Texture) {
    if tex != nil {
        tex.Destroy()
        var surface *sdl.Surface
        surface, _ = f.RenderUTF8Blended(s, *c)
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

func __generate_and_populate_lines(r *sdl.Renderer, font *ttf.Font, dest *[]Line, tokens *[]string, end int) {
    for index := 0; index < len(*tokens); index++ {
        new_ttf_texture_line(r, font, &(*dest)[index], (*tokens)[index], int32(end+index))
    }
}

func generate_lines(renderer *sdl.Renderer, font *ttf.Font, lines *[]Line, str *[]string, max int) {
    end := 0
    for index := 0; index < len((*lines)); index++ {
        if (*lines)[index].texture != nil {
            end += 1
        } else {
            break
        }
    }
    ptr := (*lines)[end:max]
    slice := (*str)[end:max]
    __generate_and_populate_lines(renderer, font, &ptr, &slice, end)
}

func new_ttf_texture_line(rend *sdl.Renderer, font *ttf.Font, line *Line, line_text string, skip_nr int32) {
    assert_if(len(line_text) == 0)

    line.texture = make_ttf_texture(rend, font, line_text, &sdl.Color{0, 0, 0, 0})

    text := strings.Split(line_text, " ")
    text_len := len(text)

    assert_if(text_len == 0)

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

func check_collision(event *sdl.MouseMotionEvent, rect *sdl.Rect) bool {
    result := false
    mx_gt_rx :=    event.X > (*rect).X
    mx_lt_rx_rw := event.X < (*rect).X + (*rect).W
    my_gt_ry :=    event.Y > (*rect).Y
    my_lt_ry_rh := event.Y < (*rect).Y + (*rect).H

    if ((mx_gt_rx && mx_lt_rx_rw) && (my_gt_ry && my_lt_ry_rh)) {
        result = true
    }
    return result
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
        //if (string((*s)[index]) == "\n") {
        //    break
        //}
        //if (string((*s)[index]) == "\r") {
        //    break
        //}
        if (!is_space(string((*s)[index]))) {
            curr += 1
        } else {
            result = append(result, curr)
            curr = 0
        }
    }
    if (curr > 0) {
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

func execute_cmd_write_to_buffer(renderer *sdl.Renderer, cmd *CmdConsole, curr_char_w int, font *ttf.Font, fontw int, fonth int) {
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
            curr_char_w = fontw * len(string(temp_string[len(temp_string)-1]))

            cmd.cursor_rect.X -= int32(curr_char_w)

            cmd.ttf_rect.W = int32(fontw * len(cmd.input_buffer.String()))
            cmd.ttf_rect.H = int32(fonth)
            println(temp_string)
        } else {
            cmd.cursor_rect.X = 0
        }
    }
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
    return float32(math.Sqrt(float64((*v).x * (*v).x) + float64((*v).y * (*v).y)))
}

func lerp(a float32, b float32, t float32) float32 {
    if t > 1 || t < 0 {
        return 0.0
    }
    return (1-t) * a + t * b
}

func normalize(n float32, max float32) float32{
    return n / max
}

func get_filenames(path string, format []string) []string {
    var result []string

    list, err := ioutil.ReadDir(path)
    if err != nil {
        panic(err)
    }

    for _, f := range list {
        for i := 0 ; i < len(format); i++ {
            if strings.Contains(f.Name(), format[i]) {
                result = append(result, f.Name())
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
    (*font).fonts = make([]Font, size)
    (*font).textures = make([]*sdl.Texture, size)
    (*font).ttf_rects = make([]sdl.Rect, size)
    (*font).highlight_rect = make([]sdl.Rect, size)
}

func generate_fonts(font *FontSelector, ttf_font_list []string, font_dir string) {
    CURRENT := 6 // magic number
    for index, element := range ttf_font_list {
        if CURRENT == index {
            (*font).current_font = load_font(font_dir + element, TTF_FONT_SIZE)
            w, h, _ := (*font).current_font.SizeUTF8(" ")
            skp := (*font).current_font.LineSkip()
            (*font).current_font_w = w
            (*font).current_font_h = h
            (*font).current_font_skip = skp
        }
        (*font).fonts[index].data = load_font(font_dir + element, TTF_FONT_SIZE_FOR_FONT_LIST)
        (*font).fonts[index].name = element
    }
}

func generate_rects_for_fonts(renderer *sdl.Renderer, font *FontSelector) {
    (*font).bg_rect = sdl.Rect{}
    adder_y := 0
    for index, element := range (*font).fonts {
        gx, gy, _ := (*font).fonts[index].data.SizeUTF8(" ")
        (*font).fonts[index].size = gx * len(element.name)

        (*font).textures[index] = make_ttf_texture(renderer, (*font).fonts[index].data,
                                                             (*font).fonts[index].name,
                                                                &sdl.Color{0, 0, 0, 0})

        (*font).ttf_rects[index] = sdl.Rect{0, int32(adder_y), int32(gx*len(element.name)), int32(gy)}

        if (*font).bg_rect.W < (*font).ttf_rects[index].W {
            (*font).bg_rect.W = (*font).ttf_rects[index].W
        }

        (*font).highlight_rect[index] = (*font).ttf_rects[index]

        (*font).bg_rect.H += (*font).ttf_rects[index].H
        adder_y += gy

        if index == len((*font).fonts)-1 {
            for i := 0; i < len((*font).ttf_rects); i++ {
                (*font).highlight_rect[i].W = (*font).bg_rect.W
            }
        }
    }
}
